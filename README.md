# graphql-gen
[![Go Docs](https://godoc.org/github.com/GannettDigital/graphql-gen?status.svg)](https://godoc.org/github.com/GannettDigital/graphql-gen)
[![Build Status](https://travis-ci.org/GannettDigital/graphql-gen.svg)](https://travis-ci.org/GannettDigital/graphql-gen)
[![Go Report Card](https://goreportcard.com/badge/github.com/GannettDigital/graphql-gen)](https://goreportcard.com/report/github.com/GannettDigital/graphql-gen)
[![Coverage Status](https://coveralls.io/repos/github/GannettDigital/graphql-gen/badge.svg?branch=master)](https://coveralls.io/github/GannettDigital/graphql-gen?branch=master)

[GraphQL](https://graphql.org/) and Golang make a great combination for building Go based GraphQL servers, however the
amount of boilerplate code that needs to be written for a reasonably comples GraphQL schema is cumbersome. This project
provides tooling to build the GraphQL schema in Go code so it can be used by the library
https://github.com/graphql-go/graphql

For details in using this tooling see the documentation and examples in the [Go docs](http://godoc.org/github.com/GannettDigital/graphql-gen)

## Building/Testing
Build and Testing are done using standard go tooling, ie `go test ./...`
