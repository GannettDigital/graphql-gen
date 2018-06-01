# graphql-gen

The code in this library aids in building GraphQL schemas for use with https://github.com/graphql-go/graphql from Golang structs.

## Building/Testing
This project uses the Go package management tool [Dep](https://github.com/golang/dep) for dependencies.
To leverage this tool to install dependencies, run the following command from the project root:

    dep ensure

Testing is done using standard go tooling, ie `go test ./...`
