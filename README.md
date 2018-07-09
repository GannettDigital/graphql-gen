# graphql-gen

[GraphQL](https://graphql.org/) and Golang make a great combination for building Go based GraphQL servers, however the
amount of boilerplate code that needs to be written for a reasonably comples GraphQL schema is cumbersome. This project
provides tooling to build the GraphQL schema in Go code so it can be used by the library
https://github.com/graphql-go/graphql

For details in using this tooling see the documentation and examples in the [Go docs](http://godoc.org/github.com/GannettDigital/graphql-gen)

## Building/Testing
This project uses the Go package management tool [Dep](https://github.com/golang/dep) for dependencies.
To leverage this tool to install dependencies, run the following command from the project root:

    dep ensure

Testing is done using standard go tooling, ie `go test ./...`
