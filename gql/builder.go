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
	FieldPathRoot      = ""
	FieldPathSeperator = "_"
	// because _ is the only allowed non-alphanumeric character
	// http://facebook.github.io/graphql/October2016/#sec-Names
	// any _ occurances will be gracefully removed e.g. _my_odd_name => myoddname
	QueryReporterContextKey = "GraphQLQueryReporter"
)

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
// the context and if it exists reports the QueriedFields.
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
	structs         []interface{}
}

// NewObjectBuilder creates an ObjectBuilder for the given structs and fieldAdditions.
//
// fieldAdditions allows for adding into GraphQL objects fields which don't show up in the underlying structs.
// The key to the map is a path for the field parent, this starts with the FieldPathRoot and adds the struct name for
// any embedded structs joined with FieldPathSeperator. Each nested object within the GraphQL object has its own
// path name. For example `fmt.Sprintf("%surl%ssitename", FieldPathRoot, FieldPathSeperator)` for fields added to the
// sitename object which is within the url object at the root.
// Be aware that these fields are added to all structs that have a matching path, this
// includes any interfaces build from embeded structs as well.
func NewObjectBuilder(structs []interface{}, fieldAdditions map[string][]*graphql.Field) *ObjectBuilder {
	if fieldAdditions == nil {
		fieldAdditions = make(map[string][]*graphql.Field)
	}
	return &ObjectBuilder{fieldAdditions: fieldAdditions, structs: structs, objects: make(map[string]*graphql.Object)}
}

// AddCustomFields configures more custom fields that will be used when building the types. This will overwrite custom
// fields with the same name that already exist in the object builder.
// This function is especially useful for adding fields that utilize an existing interface when run after
// BuildInterfaces but before BuildTypes.
func (ob *ObjectBuilder) AddCustomFields(fieldAdditions map[string][]*graphql.Field) {
	for key, fields := range fieldAdditions {
		ob.fieldAdditions[key] = fields

		splits := strings.Split(key, FieldPathSeperator)
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
		ob.interfaceFields[name] = ob.buildFields(sType, name, nil)

		ob.interfaces[name] = graphql.NewInterface(graphql.InterfaceConfig{
			Name:        name,
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
	// Find any defined interfaces that are relevant for this struct
	var gIfaces []*graphql.Interface
	for name := range extractEmbeds(srcStruct) {
		if iface, ok := ob.interfaces[name]; ok {
			gIfaces = append(gIfaces, iface)
		}
	}

	if len(gIfaces) == 0 {
		return ob.buildObject(sType, "", nil, nil)
	}

	baseFields := make(graphql.Fields)
	for _, iface := range gIfaces {
		name := iface.Name()
		for key, value := range ob.interfaceFields[name] {
			baseFields[key] = value
		}
	}

	object := ob.buildObject(sType, "", gIfaces, baseFields)
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
	if name == "" {
		name = sType.Name()
	}
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
		gfields[name] = &graphql.Field{
			Name:    name,
			Type:    gtype,
			Resolve: ResolveByField(name, parent),
		}
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
	graphqlKinds := map[reflect.Kind]graphql.Type{
		reflect.Bool:    graphql.Boolean,
		reflect.Float32: graphql.Float,
		reflect.Float64: graphql.Float,
		reflect.Int:     graphql.Int,
		reflect.String:  graphql.String,
		// There are other reflect types like various int types that are not supported as there has been no need
		// if a struct uses one of these it should result in a schema creation error
	}

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
	name = strings.ToLower(name) // TODO for v2 consider removing this and the similar line in buildObject
	return ob.objects[name]
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
