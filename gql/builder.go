// Package gql provides tooling for building and using GraphQL to serve content.
// Most notably it includes ObjectBuilder which enables easy building of GraphQL schema from Golang structs.
//
// The tooling in this package wraps the GraphQL server implementation from github.com/graphql-go/graphql.
//
// When paired with https://github.com/GannettDigital/jstransform/ this tooling allows for building Go based GraphQL
// servers from JSONschema quickly and then evolving the schema easily.
package gql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/GannettDigital/graphql"
)

const (
	// FieldPathSeparator is '_' because it is the only allowed non-alphanumeric character.
	// http://facebook.github.io/graphql/October2016/#sec-Names
	// Any _ occurences will be gracefully removed e.g. _my_odd_name => myoddname
	FieldPathSeparator = "_"

	// QueryReporterContextKey is the key used with context.WithValue to locate the QueryReporter.
	QueryReporterContextKey = "GraphQLQueryReporter"

	deprecationPrefix  = "DEPRECATED:"
	filterArgumentName = "filter"
	sortArgumentName   = "sort"
)

var graphqlKinds = map[reflect.Kind]graphql.Type{
	reflect.Bool:    graphql.Boolean,
	reflect.Float32: graphql.Float,
	reflect.Float64: graphql.Float,
	reflect.Int:     graphql.Int,
	reflect.Int64:   graphql.Int,
	reflect.String:  graphql.String,
}

// QueryReporter defines the interface used to report details on the GraphQL queries being performed.
// Implementations must be concurrency safe and added to the request context using QueryReporterContextKey as the
// context value key.
type QueryReporter interface {
	// QueriedField is called by the default resolve function used for fields within the GraphQL query.
	QueriedField(string) error
}

// ObjectBuilder is used to build GraphGL Objects based on the fields within a set of structs.
// A graphql.Schema is built with the parameters defined at https://godoc.org/github.com/graphql-go/graphql#SchemaConfig
// this code creates Objects which implement the graphql.Type interface and also graphql.Interface interface.
// These types and interfaces can be leveraged to build the Query and the other components of a schema.
//
// Fields from the structs are setup as GraphQL fields only if they are exported. In the GraphQL these fields are named
// to match the JSON struct tag name or if none is found the lowercase field name. If the JSON struct tag for a field
// specifies "omitempty" the field is nullable otherwise it is NonNullable.
//
// Interfaces for a type are built whenever the underlying struct has an embedded struct within it.
// The embedded struct is built as an interface for the type. Only root level embedded structs are handled this way.
//
// GraphQL objects contain fields and for each field GraphQL utilizes resolve functions to populate the data. The default
// resolve function (ResolveByField) is setup for auto-generated fields. This resolve function assumes that the resolve
// function used for the Query retrieved an entire object matching the given struct and the resolve for fields is
// simply to pull the correct field from that object. The default resolve function also looks for a QueryReporter in
// the context and if it exists reports the QueriedFields. If the field is a List the default function is
// ResolveListField which works the same way but adds a filter parameter optionally used to filter the list items.
//
// It is also possible to specify custom fields which can be setup with custom resolve functions. See fieldAdditions on
// the NewObjectBulider function and the AddCustomFields method.
//
// The go proverbs, "Clear is better than clever." and "Reflection is never clear."  both apply
// here as the reflection is not the easiest to follow. It was chosen specifically because adding this complexity
// here makes projects utilizing GraphQL simpler as it allows the complicated types to update and change without
// any of the implementation code being impacted.
type ObjectBuilder struct {
	fieldAdditions  map[string][]*graphql.Field // fieldAdditions allows for inserting additional fields at the named parent
	interfaces      map[string]*graphql.Interface
	interfaceFields map[string]graphql.Fields
	objects         map[string]*graphql.Object
	prefix          string
	structs         []interface{}
}

// NewObjectBuilder creates an ObjectBuilder for the given structs and fieldAdditions.
//
// namePrefix is an optional string to prefix the name of each generated type and interface with, it becomes part of the
// parent name of a field name but otherwise does not affect field names.
//
// fieldAdditions allows for adding into GraphQL objects fields which don't show up in the underlying structs.
// The key to the map is a path for the field parent, this starts with the FieldPathRoot and adds the struct name for
// any embedded structs joined with FieldPathSeparator. Each nested object within the GraphQL object has its own
// path name. For example `fmt.Sprintf("%surl%ssitename", FieldPathRoot, FieldPathSeparator)` for fields added to the
// sitename object which is within the url object at the root.
// Be aware that these fields are added to all structs that have a matching path, this
// includes any interfaces build from embedded structs as well.
func NewObjectBuilder(structs []interface{}, namePrefix string, fieldAdditions map[string][]*graphql.Field) (*ObjectBuilder, error) {
	if strings.Contains(namePrefix, FieldPathSeparator) {
		return nil, fmt.Errorf("namePrefix can not include the FieldPathSeparator %q", FieldPathSeparator)
	}
	if fieldAdditions == nil {
		fieldAdditions = make(map[string][]*graphql.Field)
	}
	return &ObjectBuilder{fieldAdditions: fieldAdditions, prefix: namePrefix, structs: structs, objects: make(map[string]*graphql.Object)}, nil
}

// AddCustomFields configures more custom fields that will be used when building the types. This will overwrite custom
// fields with the same name that already exist in the object builder.
// This function is especially useful for adding fields that utilize an existing interface when run after
// BuildInterfaces but before BuildTypes.
func (ob *ObjectBuilder) AddCustomFields(fieldAdditions map[string][]*graphql.Field) {
	for key, fields := range fieldAdditions {
		ob.fieldAdditions[key] = fields

		splits := strings.Split(key, FieldPathSeparator)
		ifaceName := splits[0]
		if iface, ok := ob.interfaces[ifaceName]; ok {
			parent := findObjectField(iface.Fields(), splits[1:])
			if parent == nil {
				for _, field := range fields {
					iface.AddFieldConfig(field.Name, field)
				}
			} else {
				for _, field := range fields {
					parent.AddFieldConfig(field.Name, field)
				}
			}
		}
	}
}

// BuildInterfaces will create GraphQL interfaces out of embedded structs for source object builder structs.
// If not previously run this will be run automatically part of the BuildTypes method. It can be run independently
// before running BuildTypes to support using the returned interfaces in additional fields that will be added when
// BuildTypes creates the GraphQL types.
func (ob *ObjectBuilder) BuildInterfaces() map[string]*graphql.Interface {
	ob.interfaceFields = make(map[string]graphql.Fields)
	ob.interfaces = make(map[string]*graphql.Interface)

	allEmbeds := map[string]interface{}{}
	for _, srcStruct := range ob.structs {
		embeds := extractEmbeds(srcStruct)
		for name, value := range embeds {
			allEmbeds[name] = value
		}
	}

	for name, embed := range allEmbeds {
		sType := reflect.TypeOf(embed)
		iName := ob.prefix + name
		ob.interfaceFields[name] = ob.buildFields(sType, iName, nil)

		ob.interfaces[name] = graphql.NewInterface(graphql.InterfaceConfig{
			Name:        iName,
			Fields:      ob.interfaceFields[name],
			ResolveType: ob.resolveObjectByName,
		})
	}

	return ob.interfaces
}

// BuildTypes creates the GraphQL types from the sources structs. The output of this method is suitable for directly
// including in graphql.SchemaConfig which when coupled with a graphql.Query can be built into the a GraphQL schema.
func (ob *ObjectBuilder) BuildTypes() []graphql.Type {
	if ob.interfaceFields == nil {
		ob.BuildInterfaces()
	}

	gTypes := []graphql.Type{}
	for _, srcStruct := range ob.structs {
		gTypes = append(gTypes, ob.buildType(srcStruct))
	}

	return gTypes
}

// buildType will create a GraphQL type based on the given srcStruct, it includes fields from any embedded structs and
// sets those embedded structs up as interfaces in GraphQL.
func (ob *ObjectBuilder) buildType(srcStruct interface{}) graphql.Type {
	sType := reflect.TypeOf(srcStruct)
	name := ob.prefix + sType.Name()

	baseFields := make(graphql.Fields)
	// Find any defined interfaces that are relevant for this struct
	var gIfaces []*graphql.Interface
	for name := range extractEmbeds(srcStruct) {
		if iface, ok := ob.interfaces[name]; ok {
			gIfaces = append(gIfaces, iface)
		}
		for key, value := range ob.interfaceFields[name] {
			baseFields[key] = value
		}
	}

	if len(gIfaces) == 0 {
		return ob.buildObject(sType, name, nil, nil)
	}

	object := ob.buildObject(sType, name, gIfaces, baseFields)
	ob.objects[object.Name()] = object
	return object
}

// buildObject does the heavy lifting in building a GraphQL object, it can be called recursively as Objects can have
// fields which are themselves objects.  This method relies heavily on the buildFields
// method which does reflection on the given type to discover the fields. If a name is given that name is used
// for the object, otherwise the name of the struct is used as the name. If the object is part of an interface the graphql.Interface and the set of base fields for
// that interface are expected to be provided as the fields for each type that implements an interface must match
// exactly.
func (ob *ObjectBuilder) buildObject(sType reflect.Type, name string, gInterfaces []*graphql.Interface, baseFields graphql.Fields) *graphql.Object {
	name = strings.ToLower(name) // TODO for v2 consider removing this and the similar line in resolveObjectByName

	gfields := ob.buildFields(sType, name, baseFields)

	cfg := graphql.ObjectConfig{
		Name:       name,
		Fields:     gfields,
		Interfaces: gInterfaces,
	}

	return graphql.NewObject(cfg)
}

// buildFields creates the GraphQL fields representing a Golang struct.
// The Resolve functions built for the fields all leverage the ResolveByField function. All public variables in the
// struct will be added as GraphQL fields excluding those from embedded structs. Any embedded struct fields should
// be provided in the baseFields argument as it is assumed embedded structs are expressed in GraphQL as an interface.
//
// The naming of the fields is based on the output of the fieldName function. If the field contains an inline object
// the name of that object needs to be unique so the parent name is passed to the creation of that object so a
// unique name is created. The name of the field is unaffected, only the name of the object in the field is changed.
// This function will panic if called on a non-struct.
//
// Fields will have a description set if a description struct tag exists. If this description begins with the
// deprecationPrefix it will be set as the DeprecationReason instead.
func (ob *ObjectBuilder) buildFields(sType reflect.Type, parent string, baseFields graphql.Fields) graphql.Fields {
	if sType.Kind() != reflect.Struct {
		// The function should be used on structs defined in the code, panic so misuse is caught in unit testing
		panic("graphQL buildFields used with a non-struct")
	}

	gfields := graphql.Fields{}
	for name, f := range baseFields {
		gfields[name] = f
	}
	for i := 0; i < sType.NumField(); i++ {
		field := sType.Field(i)

		if field.Anonymous { // Skip fields from embedded structs
			continue
		}

		gtype := ob.fieldGraphQLType(field, parent)
		if gtype == nil {
			continue
		}

		name := fieldName(field)
		description := field.Tag.Get("description")
		f := &graphql.Field{
			Name:    name,
			Type:    gtype,
			Resolve: ResolveByField(name, parent),
		}

		if strings.HasPrefix(description, deprecationPrefix) {
			f.DeprecationReason = description
		} else {
			f.Description = description
		}

		checkType := gtype
		if nn, ok := gtype.(*graphql.NonNull); ok {
			checkType = nn.OfType
		}
		if _, ok := checkType.(*graphql.List); ok {
			f.Args = graphql.FieldConfigArgument{
				filterArgumentName: &graphql.ArgumentConfig{
					Description: `A List Filter expression such as '{Field: "position", Operation: "<=", Argument: {Value: 10}}'`,
					Type:        graphqlListFilter,
				},
				sortArgumentName: &graphql.ArgumentConfig{
					Description: `Sort the list, ie '{Field: "position", Order: "ASC"}'`,
					Type:        graphqlSortFilter,
				},
			}
			f.Resolve = ResolveListField(name, parent)

			totalName := "total" + strings.Title(name)
			gfields[totalName] = &graphql.Field{
				Name:        totalName,
				Type:        graphql.Int,
				Resolve:     ResolveTotalCount(totalName, name, parent),
				Description: fmt.Sprintf("The total length of the %s list at this same level in the data, this number is unaffected by filtering.", name),
			}
		}

		gfields[name] = f
	}
	if fields, ok := ob.fieldAdditions[parent]; ok {
		for _, field := range fields {
			gfields[field.Name] = field
		}
	}

	return gfields
}

// fieldGraphQLType returns the graphql.Type which is appropriate for the kind of the struct field being examined.
// If the JSON struct tag specifies "omitempty" the field is nullable otherwise it is NonNullable.
// The function leverages graphQLType for the base type with the struct field specific options added to that.
func (ob *ObjectBuilder) fieldGraphQLType(field reflect.StructField, parent string) graphql.Type {
	gtype := ob.graphQLType(field.Type, fieldName(field), parent)

	if graphql.GetNullable(gtype) == nil { // Some GraphQL types can't be set NonNull
		return gtype
	}
	jsonTag := field.Tag.Get("json")
	splits := strings.Split(jsonTag, ",")
	var omitEmpty bool
	for _, sp := range splits {
		if sp == "omitempty" {
			omitEmpty = true
		}
	}
	if !omitEmpty {
		gtype = graphql.NewNonNull(gtype)
	}

	return gtype
}

// graphQLType returns the graphql.Type which matches reflect.Kind() of the given type.
// If the type is a struct then a GraphQL scalar can't be used so a new GraphQL object must be created. This is done
// by calling the buildObject function which will result in recursively walking the struct. It is important that
// each GraphQL type have an independent name but because a struct may be inline to another names are only unique
// when considering the entire chain. To make this work when buildObject is called from this function a new name
// derived from the parent name and the name of this type is passed as an argument.
func (ob *ObjectBuilder) graphQLType(rType reflect.Type, name, parent string) graphql.Type {
	var gtype graphql.Type
	kind := rType.Kind()
	switch kind {
	case reflect.Struct:
		if rType.PkgPath() == "time" {
			gtype = graphql.DateTime
			break
		}

		gtype = ob.buildObject(rType, fullFieldName(name, parent), nil, nil)
	case reflect.Slice:
		elemType := ob.graphQLType(rType.Elem(), name, parent)
		gtype = graphql.NewList(elemType)
	default:
		gtype = graphqlKinds[kind]
	}

	return gtype
}

// resolveObjectByName is a graphql.ResolveTypeFn used to determine the type a GraphQL interface resolves to based
// on the name of the struct.
func (ob *ObjectBuilder) resolveObjectByName(p graphql.ResolveTypeParams) *graphql.Object {
	sType := reflect.TypeOf(p.Value)
	name := sType.Name()
	name = ob.prefix + strings.ToLower(name) // TODO for v2 consider removing this and the similar line in buildObject
	return ob.objects[name]
}

// ResolveListField returns a FieldResolveFn that leverages ResolveByField to get the field value then applies an
// optional filter to the returned list of results.
//
// The optional filter is specified by a 'filter' argument which is a JSON object with a required string field
// named 'Operation' and optional object field 'Argument' and string field named 'Field'. Certain operations may
// require a valid 'Field' and/or 'Argument'. When provided 'Argument' should be a JSON object whose value will be
// the argument to NewListOperation functions. The value of 'Field' is the name of a field in the list items, if
// the list contains objects within it child field keys can be added using FieldPathSeparator.
//
// In addition to the filter argument as sort argument can be specified. The sort argument takes a string parameter
// Field which is the same as that for the filter, the field to be compared or the list itself if unspecified.
// It also takes an optional order parameter which is either "ASC" or "DESC", "ASC" is default.
//
// Sorting occurs before filtering as some filters limit the total returned size of the list.
//
// Example:
//  {
//    query(id: "blah") {
//      modules(filter: {Field: "moduleName", Operation: "==", Argument: {Value: "foo"}}, sort: {Field: "module_type", Order: "ASC"}) {
//        moduleName
//      }
//    }
//  }
//
func ResolveListField(name string, parent string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		filter, err := newListFilter(p.Args[filterArgumentName])
		if err != nil {
			return nil, err
		}

		sortParams, err := parseSortParameters(p.Args[sortArgumentName])
		if err != nil {
			return nil, err
		}

		resolve := ResolveByField(name, parent)

		resolvedValue, err := resolve(p)
		if err != nil {
			return nil, err
		}
		if filter == nil && sortParams == nil {
			// an optimization, skip further processing if neither filtering nor sorting is specified
			return resolvedValue, nil
		}

		value := reflect.ValueOf(resolvedValue)
		if value.Kind() != reflect.Slice {
			return nil, fmt.Errorf("value returned from field %q is not a list as expected", fullFieldName(name, parent))
		}

		values := make([]interface{}, value.Len())
		for i := 0; i < value.Len(); i++ {
			values[i] = value.Index(i).Interface()
		}

		// sort before filter because some filters are based on the count of items
		if sortParams != nil {
			if err := listSort(sortParams, values); err != nil {
				return nil, err
			}
		}

		if filter == nil {
			return values, nil
		}

		var filtered []interface{}
		for _, item := range values {
			matched, err := filter.match(item)
			if err != nil {
				return nil, fmt.Errorf("%v. Note: filtering and sorting is not available on hydrated items", err)
			}
			if matched {
				filtered = append(filtered, item)
			}
		}

		return filtered, nil
	}
}

// ResolveByField returns a FieldResolveFn that leverages ExtractFields for the given field name to
// resolve the data. The resolve function assumes the entire object is available in the ResolveParams source.
// It will also report the queried field to the QueryReporter if one is found in the context.
// This is default resolve function used by the objectbuilder.
func ResolveByField(name string, parent string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		if qr, ok := p.Context.Value(QueryReporterContextKey).(QueryReporter); ok && qr != nil {
			if err := qr.QueriedField(fullFieldName(name, parent)); err != nil {
				return nil, err
			}
		}

		field := ExtractField(p.Source, name)
		if field == nil {
			return nil, graphql.NewLocatedError(
				fmt.Errorf("failed to extract field %q value from data", name),
				graphql.FieldASTsToNodeASTs(p.Info.FieldASTs),
			)
		}
		return field, nil
	}
}

// ResolveTotalCount accepts a total count field name, a name of the list field, and a parent name. It leverages
// ExtractField for the given list field name and will return the count of items in the extract field if it is an array
// or a slice. It will also report the queried field to the QueryReporter if one is found in the context.
func ResolveTotalCount(totalFieldName, listFieldName, parent string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		if qr, ok := p.Context.Value(QueryReporterContextKey).(QueryReporter); ok && qr != nil {
			if err := qr.QueriedField(fullFieldName(totalFieldName, parent)); err != nil {
				return nil, err
			}
		}

		field := ExtractField(p.Source, listFieldName)
		fieldValue := reflect.ValueOf(field)
		valueKind := fieldValue.Kind()
		if !fieldValue.IsValid() {
			// This will happen when the field doesn't exist at all in the resolved interface
			return 0, nil
		}
		if valueKind != reflect.Slice && valueKind != reflect.Array {
			return nil, graphql.NewLocatedError(
				fmt.Errorf("field value is not a valid list in the data"),
				graphql.FieldASTsToNodeASTs(p.Info.FieldASTs),
			)
		}
		return fieldValue.Len(), nil
	}
}

// findObjectField traverses the fields in the given GraphQL object and returns the value of the one matching the path.
// If the path contains multiple items it is assumed that each item represents a layer in a nested set of objects.
// This only handles fields that are themselves graphql.Objects other field types are ignored.
func findObjectField(fields graphql.FieldDefinitionMap, path []string) *graphql.Object {
	if len(path) == 0 {
		return nil
	}
	childDef, ok := fields[path[0]]
	if !ok {
		return nil
	}

	child, ok := resolveGraphQLObject(childDef.Type)
	if !ok {
		return nil
	}

	if len(path) == 1 {
		return child
	}
	return findObjectField(child.Fields(), path[1:])
}

// resolveGraphQLObject attempts to return an underlying graphql.Object found in the grapqhl.Output.
// It continues digging deeper past graphql.NonNull and graphql.List wrappers
func resolveGraphQLObject(object interface{}) (*graphql.Object, bool) {
	switch object := object.(type) {
	case *graphql.Object:
		return object, true
	case *graphql.NonNull:
		return resolveGraphQLObject(object.OfType)
	case *graphql.List:
		return resolveGraphQLObject(object.OfType)
	default:
		return nil, false
	}
}
