package gql

import (
	"context"
	"errors"
	"log"

	"github.com/GannettDigital/graphql"
	"strings"
)

// The simple example shows how to use the object builder with a struct with no embeded fields.
// The main value of the object builder comes with many structs each with many fields as it greatly minimizes the
// amount of code needed to setup these scenarios. This example shows the bare minimum additional pieces needed
// from the graphql-go library to setup a working schema.
func ExampleObjectBuilder_simple() {
	type exampleStruct struct {
		fieldA string
	}

	exampleData := make(map[string]exampleStruct)

	ob := NewObjectBuilder([]interface{}{exampleStruct{}}, "", nil)
	types := ob.BuildTypes()

	queryCfg := graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"example": &graphql.Field{
				Type: types[0],
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Description: "ID of the object to retrieve",
						Type:        graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, ok := p.Args["id"].(string)
					if !ok {
						return nil, errors.New("failed to extract ID from argument.")
					}
					// replace with DB implementation
					example := exampleData[id]
					return example, nil
				},
			},
		},
	}

	query := graphql.NewObject(queryCfg)

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: query,
		Types: types,
	})
	if err != nil {
		log.Fatal(err)
	}

	// schema is now ready to use for resolving queries
	params := graphql.Params{
		Context:       context.Background(),
		Schema:        schema,
		RequestString: "",
	}

	graphql.Do(params)
}

// The full example expands on the simple example showing custom fields, GraphQL interfaces and an interface including
// itself as a way of pulling related items in a single query.
func ExampleObjectBuilder_full() {
	type ExampleStruct struct {
		fieldA string
		Links  []struct {
			ID string `json:"id,omitempty"`
		} `json:"links,omitempty"`
	}

	type exampleStruct2 struct {
		ExampleStruct

		fieldB string
	}

	type exampleStruct3 struct {
		ExampleStruct

		fieldB string
	}

	exampleData2 := make(map[string]exampleStruct2)
	exampleData3 := make(map[string]exampleStruct3)

	exampleStructResolver := func(p graphql.ResolveParams) (interface{}, error) {
		id, ok := p.Args["id"].(string)
		if !ok {
			return nil, errors.New("failed to extract ID from argument.")
		}
		// replace with DB implementation
		if example, ok := exampleData2[id]; ok {
			return example, nil
		}

		example := exampleData3[id]
		return example, nil
	}
	ob := NewObjectBuilder([]interface{}{exampleStruct2{}, exampleStruct3{}}, "", nil)

	// First create the interface so the interface can be used in adding a custom field
	ifaces := ob.BuildInterfaces()
	exampleInterface := ifaces["ExampleStruct"]

	// This add a new field in the ExampleStruct interface that allows resolving additional structs recursively.
	ob.AddCustomFields(map[string][]*graphql.Field{
		strings.Join([]string{"ExampleStruct", "links"}, FieldPathSeperator): {
			{
				Name:    "examplestruct",
				Type:    exampleInterface,
				Resolve: exampleStructResolver,
			},
		},
	})

	types := ob.BuildTypes()

	queryCfg := graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"example": &graphql.Field{
				Type: exampleInterface, // The Query returns the interface so either type matching it can be returned
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Description: "ID of the object to retrieve",
						Type:        graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: exampleStructResolver,
			},
		},
	}

	query := graphql.NewObject(queryCfg)

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: query,
		Types: types,
	})
	if err != nil {
		log.Fatal(err)
	}

	// schema is now ready to use for resolving queries
	params := graphql.Params{
		Context:       context.Background(),
		Schema:        schema,
		RequestString: "", // replace with real query
	}

	graphql.Do(params)
}

// The prefix example shows how a prefix could be used to create GraphQL schema distinctions between similar or
// identical types.
func ExampleObjectBuilder_prefix() {
	type exampleStruct struct {
		fieldA string
	}

	exampleData := make(map[string]exampleStruct)

	ob := NewObjectBuilder([]interface{}{exampleStruct{}}, "", nil)
	types := ob.BuildTypes()

	// A second object builder adds a prefix to the naming. This example is a contrived but should demonstrate how
	// naming collisions in the GraphQL schema are avoided by adding the prefix.
	sob := NewObjectBuilder([]interface{}{exampleStruct{}}, "staging", nil)
	stypes := sob.BuildTypes()

	queryCfg := graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"example": &graphql.Field{
				Type: types[0],
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Description: "ID of the object to retrieve",
						Type:        graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, ok := p.Args["id"].(string)
					if !ok {
						return nil, errors.New("failed to extract ID from argument.")
					}
					// replace with DB implementation
					example := exampleData[id]
					return example, nil
				},
			},
			"stagingExample": &graphql.Field{
				Type: stypes[0],
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Description: "ID of the object to retrieve",
						Type:        graphql.NewNonNull(graphql.String),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id, ok := p.Args["id"].(string)
					if !ok {
						return nil, errors.New("failed to extract ID from argument.")
					}
					// replace with DB implementation, this one coming from staging
					example := exampleData[id]
					return example, nil
				},
			},
		},
	}

	query := graphql.NewObject(queryCfg)

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: query,
		Types: append(types, stypes...),
	})
	if err != nil {
		log.Fatal(err)
	}

	// schema is now ready to use for resolving queries
	params := graphql.Params{
		Context:       context.Background(),
		Schema:        schema,
		RequestString: "",
	}

	graphql.Do(params)
}
