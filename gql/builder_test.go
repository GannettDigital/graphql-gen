package gql

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/GannettDigital/graphql"
)

var (
	objectA = graphql.NewObject(graphql.ObjectConfig{Name: "objectA"})
	childA  = graphql.NewObject(graphql.ObjectConfig{
		Name: "childA",
		Fields: graphql.Fields{"objectA": &graphql.Field{
			Name: "objectA",
			Type: objectA,
		}},
	})
	parentA = graphql.NewObject(graphql.ObjectConfig{
		Name: "parentA",
		Fields: graphql.Fields{"childA": &graphql.Field{
			Name: "childA",
			Type: childA,
		}},
	})
)

type TestBase struct {
	Id string
}

type TestBase2 struct {
	Id2 string
}

type testUnexported struct {
	exported bool
}

type testEmbed struct {
	TestBase

	Extra string
}

type testDoubleEmbed struct {
	TestBase
	TestBase2
	testUnexported

	Extra string
}

type testQueryReporter struct {
	queriedField string
}

func (tqr *testQueryReporter) QueriedField(field string) error {
	tqr.queriedField = field
	return nil
}

func TestAddCustomFields(t *testing.T) {
	testBaseFields := graphql.Fields{
		"id": &graphql.Field{
			Name: "id",
			Type: graphql.String,
		},
	}

	ob := &ObjectBuilder{
		fieldAdditions:  make(map[string][]*graphql.Field),
		interfaceFields: map[string]graphql.Fields{"TestBase": testBaseFields},
	}
	testBaseInterface := graphql.NewInterface(graphql.InterfaceConfig{Name: "TestBase", Fields: ob.interfaceFields["TestBase"]})
	ob.interfaces = map[string]*graphql.Interface{"TestBase": testBaseInterface}

	ob.AddCustomFields(map[string][]*graphql.Field{"TestBase": {
		{
			Name: "childa",
			Type: childA,
		},
	}})

	if ob.fieldAdditions == nil || len(ob.fieldAdditions) != 1 {
		t.Errorf("Expeccted fieldAdditions to now equal 1")
	}

	if len(ob.interfaceFields["TestBase"]) != 2 {
		t.Errorf("Expeccted interface fields to now equal 2")
	}

	ob.AddCustomFields(map[string][]*graphql.Field{"TestBase_childa": {
		{
			Name: "added",
			Type: graphql.String,
		},
	}})

	if len(ob.fieldAdditions) != 2 {
		t.Errorf("Expeccted fieldAdditions to now equal 2")
	}

	iface := ob.interfaces["TestBase"]
	childfields := findObjectField(iface.Fields(), []string{"childa"})
	if childfields == nil || len(childfields.Fields()) != 2 {
		t.Errorf("Expeccted interface fields to now equal 2")
	}
}

func TestObjectBuilder_BuildInterfaces(t *testing.T) {
	tests := []struct {
		description string
		structs     []interface{}
		want        []string
	}{
		{
			description: "No interfaces",
			structs:     []interface{}{TestBase{}},
			want:        []string{},
		},
		{
			description: "One interfaces",
			structs:     []interface{}{testEmbed{}},
			want:        []string{"TestBase"},
		},
		{
			description: "Two interfaces",
			structs:     []interface{}{testDoubleEmbed{}},
			want:        []string{"TestBase", "TestBase2"},
		},
	}

	for _, test := range tests {
		ob := NewObjectBuilder(test.structs, "", nil)

		got := ob.BuildInterfaces()

		if len(got) != len(test.want) {
			t.Errorf("Test %q - got %d interfaces, want %d", test.description, len(got), len(test.want))
		}

		for _, key := range test.want {
			if _, ok := got[key]; !ok {
				t.Errorf("Test %q - interface name %q not in got", test.description, key)
			}
		}
	}
}

func TestObjectBuilder_BuildTypes(t *testing.T) {
	testBaseInterface := graphql.NewInterface(graphql.InterfaceConfig{Name: "TestBase", Fields: graphql.Fields{
		"id": &graphql.Field{
			Name: "id",
			Type: graphql.NewNonNull(graphql.String),
		},
	},
	})
	testBase2Interface := graphql.NewInterface(graphql.InterfaceConfig{Name: "TestBase2", Fields: graphql.Fields{
		"id2": &graphql.Field{
			Name: "id2",
			Type: graphql.NewNonNull(graphql.String),
		},
	},
	})
	testBasePrefixInterface := graphql.NewInterface(graphql.InterfaceConfig{Name: "prefixTestBase", Fields: graphql.Fields{
		"id": &graphql.Field{
			Name: "id",
			Type: graphql.NewNonNull(graphql.String),
		},
	},
	})
	testBase2PrefixInterface := graphql.NewInterface(graphql.InterfaceConfig{Name: "prefixTestBase2", Fields: graphql.Fields{
		"id2": &graphql.Field{
			Name: "id2",
			Type: graphql.NewNonNull(graphql.String),
		},
	},
	})
	testEmbedType := graphql.NewObject(graphql.ObjectConfig{
		Name: "testembed",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Name: "id",
				Type: graphql.NewNonNull(graphql.String),
			},
			"extra": &graphql.Field{
				Name: "extra",
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Interfaces: []*graphql.Interface{testBaseInterface},
	})
	testEmbedPrefixType := graphql.NewObject(graphql.ObjectConfig{
		Name: "prefixtestembed",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Name: "id",
				Type: graphql.NewNonNull(graphql.String),
			},
			"extra": &graphql.Field{
				Name: "extra",
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Interfaces: []*graphql.Interface{testBasePrefixInterface},
	})
	testDoubleEmbedType := graphql.NewObject(graphql.ObjectConfig{
		Name: "testdoubleembed",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Name: "id",
				Type: graphql.NewNonNull(graphql.String),
			},
			"id2": &graphql.Field{
				Name: "id2",
				Type: graphql.NewNonNull(graphql.String),
			},
			"extra": &graphql.Field{
				Name: "extra",
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Interfaces: []*graphql.Interface{testBaseInterface, testBase2Interface},
	})
	testDoubleEmbedPrefixType := graphql.NewObject(graphql.ObjectConfig{
		Name: "prefixtestdoubleembed",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Name: "id",
				Type: graphql.NewNonNull(graphql.String),
			},
			"id2": &graphql.Field{
				Name: "id2",
				Type: graphql.NewNonNull(graphql.String),
			},
			"extra": &graphql.Field{
				Name: "extra",
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Interfaces: []*graphql.Interface{testBasePrefixInterface, testBase2PrefixInterface},
	})
	testEmbedType2 := graphql.NewObject(graphql.ObjectConfig{
		Name: "testembed",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Name: "id",
				Type: graphql.NewNonNull(graphql.String),
			},
			"extra": &graphql.Field{
				Name: "extra",
				Type: graphql.NewNonNull(graphql.String),
			},
			"extraid": &graphql.Field{
				Name: "extraid",
				Type: graphql.String,
			},
		},
		Interfaces: []*graphql.Interface{testBaseInterface},
	})
	testEmbedType3 := graphql.NewObject(graphql.ObjectConfig{
		Name: "testembed",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Name: "id",
				Type: graphql.Int,
			},
			"extra": &graphql.Field{
				Name: "extra",
				Type: graphql.NewNonNull(graphql.String),
			},
		},
		Interfaces: []*graphql.Interface{testBaseInterface},
	})

	tests := []struct {
		description    string
		prefix string
		structs        []interface{}
		fieldAdditions map[string][]*graphql.Field
		want           []graphql.Type
	}{
		{
			description: "Single type",
			structs:     []interface{}{testEmbed{}},
			want:        []graphql.Type{testEmbedType},
		},
		{
			description: "Mulitple types",
			structs:     []interface{}{testEmbed{}, testDoubleEmbed{}},
			want:        []graphql.Type{testEmbedType, testDoubleEmbedType},
		},
		{
			description: "Additional custom field in the interface",
			structs:     []interface{}{testEmbed{}},
			fieldAdditions: map[string][]*graphql.Field{
				"TestBase": {{
					Name: "extraid",
					Type: graphql.String,
				}},
			},
			want: []graphql.Type{testEmbedType2},
		},
		{
			description: "Additional custom field in the given object",
			structs:     []interface{}{testEmbed{}},
			fieldAdditions: map[string][]*graphql.Field{
				"testembed": {{
					Name: "extraid",
					Type: graphql.String,
				}},
			},
			want: []graphql.Type{testEmbedType2},
		},
		{
			description: "Custom field overwrites generated field",
			structs:     []interface{}{testEmbed{}},
			fieldAdditions: map[string][]*graphql.Field{
				"TestBase": {{
					Name: "id",
					Type: graphql.Int,
				}},
			},
			want: []graphql.Type{testEmbedType3},
		},
		{
			description: "Mulitple types with prefix",
			prefix: "prefix",
			structs:     []interface{}{testEmbed{}, testDoubleEmbed{}},
			want:        []graphql.Type{testEmbedPrefixType, testDoubleEmbedPrefixType},
		},
	}

	for _, test := range tests {
		ob := NewObjectBuilder(test.structs, test.prefix, test.fieldAdditions)

		gotTypes := ob.BuildTypes()

		if len(gotTypes) != len(test.want) {
			t.Errorf("Test %q - got %d types, want %d", test.description, len(gotTypes), len(test.want))
		}
		for i := range gotTypes {
			got, ok := gotTypes[i].(*graphql.Object)
			if !ok {
				t.Fatalf("Test %q - got at index %d is not an object", test.description, i)
			}
			want, ok := test.want[i].(*graphql.Object)
			if !ok {
				t.Fatalf("Test %q - want at index %d is not an object", test.description, i)
			}

			switch {
			case got.Name() != want.Name():
				t.Errorf("Test %q - got name %v, want %v", test.description, got.Name(), want.Name())
			case len(got.Interfaces()) != len(want.Interfaces()):
				t.Errorf("Test %q - got number of interfaces %v, want %v", test.description, len(got.Interfaces()), len(want.Interfaces()))
			case len(got.Fields()) != len(want.Fields()):
				t.Errorf("Test %q - got number of fields %v, want %v", test.description, len(got.Fields()), len(want.Fields()))
			}

			// Note only the names of the fields is verified in this test
			wantFields := want.Fields()
			for key, f := range got.Fields() {
				gotf, wantf := f, wantFields[key]
				if gotf == nil || wantf == nil {
					t.Errorf("Test %q - field %q got or want was nil", test.description, key)
					continue
				}
				if gotf.Name != wantf.Name {
					t.Errorf("Test %q - field %q got name %v want %v", test.description, key, gotf.Name, wantf.Name)
				}
				if gotf.Type.String() != wantf.Type.String() {
					t.Errorf("Test %q - field %q got type %q want %q", test.description, key, gotf.Type, wantf.Type)
				}
			}
		}
	}
}

func TestFieldGraphQLType(t *testing.T) {
	ob := NewObjectBuilder(nil, "", nil)

	tests := []struct {
		description  string
		field        reflect.StructField
		wantNullable bool
	}{
		{
			description:  "No JSON tag",
			field:        reflect.StructField{Name: "name", Type: reflect.TypeOf("")},
			wantNullable: false,
		},
		{
			description:  "Basic JSON tag with omitempty",
			field:        reflect.StructField{Name: "name", Type: reflect.TypeOf(""), Tag: reflect.StructTag(`json:"jsonName,omitempty"`)},
			wantNullable: true,
		},
		{
			description:  "JSON tag is nameless, but has no omitempty",
			field:        reflect.StructField{Name: "name", Type: reflect.TypeOf(""), Tag: reflect.StructTag(`json:",string"`)},
			wantNullable: false,
		},
		{
			description:  "JSON tag is nameless, but has omitempty",
			field:        reflect.StructField{Name: "name", Type: reflect.TypeOf(""), Tag: reflect.StructTag(`json:",omitempty"`)},
			wantNullable: true,
		},
		{
			description:  "JSON tag is -",
			field:        reflect.StructField{Name: "name", Type: reflect.TypeOf(""), Tag: reflect.StructTag(`json:"-"`)},
			wantNullable: false,
		},
		{
			description:  "JSON tag is -,",
			field:        reflect.StructField{Name: "name", Type: reflect.TypeOf(""), Tag: reflect.StructTag(`json:"-,"`)},
			wantNullable: false,
		},
	}

	for _, test := range tests {
		got := ob.fieldGraphQLType(test.field, "")

		_, nonNull := got.(*graphql.NonNull)
		nullable := !nonNull

		if test.wantNullable != nullable {
			t.Errorf("Test %q - want nullable %t, got %t", test.description, test.wantNullable, nullable)
		}
	}
}

func TestFindObjectField(t *testing.T) {
	tests := []struct {
		description string
		fields      graphql.FieldDefinitionMap
		path        []string
		want        *graphql.Object
	}{
		{
			description: "no path",
			path:        []string{},
			want:        nil,
		},
		{
			description: "simple no nesting",
			fields: graphql.FieldDefinitionMap{"fieldA": &graphql.FieldDefinition{
				Name: "fieldA",
				Type: objectA,
			}},
			path: []string{"fieldA"},
			want: objectA,
		},
		{
			description: "nested path missing",
			fields: graphql.FieldDefinitionMap{"fieldA": &graphql.FieldDefinition{
				Name: "fieldA",
				Type: objectA,
			}},
			path: []string{"fieldA", "childA"},
			want: nil,
		},
		{
			description: "nested path",
			fields: graphql.FieldDefinitionMap{"fieldA": &graphql.FieldDefinition{
				Name: "fieldA",
				Type: parentA,
			}},
			path: []string{"fieldA", "childA"},
			want: childA,
		},
	}

	for _, test := range tests {
		got := findObjectField(test.fields, test.path)

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("Test %q - got %v, want %v", test.description, got, test.want)
		}
	}
}

func TestGraphQLType(t *testing.T) {
	ob := NewObjectBuilder(nil, "", nil)

	tests := []struct {
		description string
		sType       reflect.Type
		want        graphql.Type
	}{
		{
			description: "scalar string",
			sType:       reflect.TypeOf(""),
			want:        graphql.String,
		},
		{
			description: "scalar int",
			sType:       reflect.TypeOf(0),
			want:        graphql.Int,
		},
		{
			description: "scalar Float32",
			sType:       reflect.TypeOf(float32(0.0)),
			want:        graphql.Float,
		},
		{
			description: "scalar Float64",
			sType:       reflect.TypeOf(float64(0.0)),
			want:        graphql.Float,
		},
		{
			description: "scalar bool",
			sType:       reflect.TypeOf(true),
			want:        graphql.Boolean,
		},
		{
			description: "scalar time.Time",
			sType:       reflect.TypeOf(time.Now()),
			want:        graphql.DateTime,
		},
		{
			description: "slice of time.Time",
			sType:       reflect.TypeOf([]time.Time{time.Now()}),
			want:        graphql.NewList(graphql.DateTime),
		},
		{
			description: "slice of strings",
			sType:       reflect.TypeOf([]string{""}),
			want:        graphql.NewList(graphql.String),
		},
		{
			description: "slice of slide of strings",
			sType:       reflect.TypeOf([][]string{{""}}),
			want:        graphql.NewList(graphql.NewList(graphql.String)),
		},
		{
			description: "Unsupported type expect nil",
			sType:       reflect.TypeOf(func() {}),
			want:        nil,
		},
	}

	for _, test := range tests {
		got := ob.graphQLType(test.sType, "name", "parent")

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("Test %q - got %+v want %+v", test.description, got, test.want)

		}
	}
}

func TestResolveByField(t *testing.T) {
	extra := "extra"
	tb := TestBase{Id: "id"}
	te := testEmbed{TestBase: tb, Extra: extra}

	tests := []struct {
		description      string
		source           interface{}
		want             interface{}
		wantQueriedField string
		wantErr          bool
	}{
		{
			description: "Simple found test",
			source:      te,
			want:        extra,
		},
		{
			description: "Simple not found test",
			source:      tb,
			wantErr:     true,
		},
		{
			description:      "found with queried field",
			source:           te,
			wantQueriedField: "TestBase_extra",
			want:             extra,
		},
		{
			description:      "not found with queried field",
			source:           tb,
			wantQueriedField: "TestBase_extra",
			wantErr:          true,
		},
	}

	resolveFn := ResolveByField("extra", "TestBase")

	for _, test := range tests {
		qr := &testQueryReporter{}
		ctx := context.Background()
		if test.wantQueriedField != "" {
			ctx = context.WithValue(ctx, QueryReporterContextKey, qr)
		}

		params := graphql.ResolveParams{
			Source:  test.source,
			Context: ctx,
		}

		got, err := resolveFn(params)

		if got, want := qr.queriedField, test.wantQueriedField; got != want {
			t.Errorf("Test %q - got queried field %q, want %q", test.description, got, want)
		}

		switch {
		case test.wantErr && err != nil:
			continue
		case test.wantErr && err == nil:
			t.Errorf("Test %q - got nil, want error", test.description)
		case !test.wantErr && err != nil:
			t.Errorf("Test %q - got err, want nil: %v", test.description, err)
		case !reflect.DeepEqual(got, test.want):
			t.Errorf("Test %q - got %v, want %v", test.description, got, test.want)
		}
	}
}

func TestResolveGraphQLObject(t *testing.T) {
	tests := []struct {
		description string
		in          interface{}
		want        *graphql.Object
		wantBool    bool
	}{
		{
			description: "In is a graphql object",
			in:          objectA,
			want:        objectA,
			wantBool:    true,
		},
		{
			description: "nonnull",
			in:          graphql.NewNonNull(objectA),
			want:        objectA,
			wantBool:    true,
		},
		{
			description: "list",
			in:          graphql.NewList(objectA),
			want:        objectA,
			wantBool:    true,
		},
		{
			description: "nonnull and list",
			in:          graphql.NewList(graphql.NewNonNull(objectA)),
			want:        objectA,
			wantBool:    true,
		},
		{
			description: "something else",
			in:          "",
			want:        nil,
			wantBool:    false,
		},
	}

	for _, test := range tests {
		got, gotBool := resolveGraphQLObject(test.in)

		if gotBool != test.wantBool {
			t.Errorf("Test %q - got bool %t want %t", test.description, gotBool, test.wantBool)
		}

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("Test %q - got %v want %v", test.description, got, test.want)
		}
	}
}

func TestResolveObjectByName(t *testing.T) {
	ob := &ObjectBuilder{
		objects: map[string]*graphql.Object{
			"objecta": objectA,
		},
	}

	type ObjectA string

	tests := []struct {
		description string
		value       interface{}
		want        *graphql.Object
	}{
		{
			description: "found",
			value:       ObjectA(""),
			want:        objectA,
		},
		{
			description: "not found",
			value:       childA,
			want:        nil,
		},
	}

	for _, test := range tests {
		params := graphql.ResolveTypeParams{
			Value: test.value,
		}

		got := ob.resolveObjectByName(params)

		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("Test %q - got %v, want %v", test.description, got, test.want)
		}
	}
}
