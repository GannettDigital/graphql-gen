package gql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GannettDigital/graphql"
)

// TestResolveListField covers ResolveListField but also proper parsing of a filter in newListFilter.
// The test builds up an entire GraphQL schema to utilize the filter argument parsing done by the GraphQL library and
// ensure the parsing code matches up with what will happen when it runs.
func TestResolveListField(t *testing.T) {
	type item struct {
		Name  string
		Value int
	}
	type testStruct struct {
		Items []item
	}
	fullList := []item{
		{Name: "a", Value: 1},
		{Name: "b", Value: 2},
		{Name: "c", Value: 3},
		{Name: "d", Value: 4},
	}
	testData := testStruct{Items: fullList}

	ob, err := NewObjectBuilder([]interface{}{testStruct{}}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	types := ob.BuildTypes()
	queryCfg := graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"q": &graphql.Field{
				Type: types[0],
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Description: "ID of the object to retrieve",
						Type:        graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return testData, nil
				},
			},
		},
	}
	query := graphql.NewObject(queryCfg)
	config := graphql.SchemaConfig{
		Query: query,
		Types: types,
	}
	s, err := graphql.NewSchema(config)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		description string
		query       string
		want        string
		wantErr     bool
	}{
		{
			description: "No filter argument",
			query:       `query { q(id: "1"){ items{name value}}}`,
			want:        `{"data":{"q":{"items":[{"name":"a","value":1},{"name":"b","value":2},{"name":"c","value":3},{"name":"d","value":4}]}}}`,
		},
		{
			description: "invalid filter",
			query:       `query { q(id: "1"){ items(filter: "name == foo"){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid filter, operation not a string",
			query:       `query { q(id: "1"){ items(filter: {Field: "name", Operation: ["==", "!="], Argument: {Value: "a"}}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid filter, field not a string",
			query:       `query { q(id: "1"){ items(filter: {Field: 2, Operation: "==", Argument: {Value: "a"}}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid filter, Argument not an object",
			query:       `query { q(id: "1"){ items(filter: {Field: "name", Operation: "==", Argument: ["a", "b"]}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid filter, missing operation",
			query:       `query { q(id: "1"){ items(filter: {Field: "name", Argument: {Value: "a"}}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid filter, unknown operation",
			query:       `query { q(id: "1"){ items(filter: {Field: "name", Operation: "unknown", Argument: {Value: "a"}}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid filter, arguments don't match operation",
			query:       `query { q(id: "1"){ items(filter: {Field: "name", Operation: "IN", Argument: {Value: "a"}}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid filter, object within the Argument object, currently unsupported",
			query:       `query { q(id: "1"){ items(filter: {Field: "name", Operation: "==", Argument: {Value: {SubValue: "a"}}}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "string equal filter",
			query:       `query { q(id: "1"){ items(filter: {Field: "name", Operation: "==", Argument: {Value: "a"}}){name value}}}`,
			want:        `{"data":{"q":{"items":[{"name":"a","value":1}]}}}`,
		},
		{
			description: "string equal filter - no match",
			query:       `query { q(id: "1"){ items(filter: {Field: "name", Operation: "==", Argument: {Value: "z"}}){name value}}}`,
			want:        `{"data":{"q":{"items":[]}}}`,
		},
		{
			description: "int equal filter",
			query:       `query { q(id: "1"){ items(filter: {Field: "value", Operation: "==", Argument: {Value: 1}}){name value}}}`,
			want:        `{"data":{"q":{"items":[{"name":"a","value":1}]}}}`,
		},
		{
			description: "limitLength filter",
			query:       `query { q(id: "1"){ items(filter: {Operation: "LIMIT", Argument: {Value: 2}}){name value}}}`,
			want:        `{"data":{"q":{"items":[{"name":"a","value":1},{"name":"b","value":2}]}}}`,
		},
		{
			description: "NOT IN filter",
			query:       `query { q(id: "1"){ items(filter: {Field: "name", Operation: "NOT IN", Argument: {Values: ["c", "d"]}}){name value}}}`,
			want:        `{"data":{"q":{"items":[{"name":"a","value":1},{"name":"b","value":2}]}}}`,
		},
	}

	for _, test := range tests {
		params := graphql.Params{
			Context:       context.Background(),
			Schema:        s,
			RequestString: test.query,
		}

		resp := graphql.Do(params)
		var err error
		if len(resp.Errors) != 0 {
			err = resp.Errors[0]
		}
		switch {
		case test.wantErr && err != nil:
			continue
		case test.wantErr && err == nil:
			t.Errorf("Test %q - got nil, want error", test.description)
		case !test.wantErr && err != nil:
			t.Errorf("Test %q - got err, want nil: %v", test.description, err)
		}

		gotBytes, err := json.Marshal(resp)
		if err != nil {
			t.Errorf("Test %q - failed to Marshal: %v", test.description, err)
		}

		if got, want := string(gotBytes), test.want; got != want {
			t.Errorf("Test %q - got %v, want %v", test.description, got, want)
		}
	}
}
