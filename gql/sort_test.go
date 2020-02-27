package gql

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/GannettDigital/graphql"
)

// TestListSort verifies the sort functionality including the parsing of arguments.
func TestListSort(t *testing.T) {
	s := testSchema(t)
	tests := []struct {
		description string
		query       string
		want        string
		wantErr     bool
		wantLF      *ListFunctions
	}{
		{
			description: "No sort argument",
			query:       `query { q(id: "1"){ items{name value}}}`,
			want:        `{"data":{"q":{"items":[{"name":"c","value":3},{"name":"a","value":1},{"name":"d","value":4},{"name":"b","value":2},{"name":"e","value":5}]}}}`,
		},
		{
			description: "invalid sort",
			query:       `query { q(id: "1"){ items(sort: "name == foo"){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid sort, field not a string",
			query:       `query { q(id: "1"){ items(sort: {Field: 2, Order: "ASC"}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid sort, order is invalid",
			query:       `query { q(id: "1"){ items(sort: {Field: "name", Order: "foo"}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "invalid sort, order not a string",
			query:       `query { q(id: "1"){ items(sort: {Field: "name", Order: 1.2}){name value}}}`,
			wantErr:     true,
		},
		{
			description: "string sort ascending",
			query:       `query { q(id: "1"){ items(sort: {Field: "name", Order: "ASC"}){name}}}`,
			want:        `{"data":{"q":{"items":[{"name":"a"},{"name":"b"},{"name":"c"},{"name":"d"},{"name":"e"}]}}}`,
		},
		{
			description: "string sort ascending by default",
			query:       `query { q(id: "1"){ items(sort: {Field: "name"}){name}}}`,
			want:        `{"data":{"q":{"items":[{"name":"a"},{"name":"b"},{"name":"c"},{"name":"d"},{"name":"e"}]}}}`,
		},
		{
			description: "string sort on field not in output, ascending by default",
			query:       `query { q(id: "1"){ items(sort: {Field: "name"}){value}}}`,
			want:        `{"data":{"q":{"items":[{"value":1},{"value":2},{"value":3},{"value":4},{"value":5}]}}}`,
		},
		{
			description: "string sort descending",
			query:       `query { q(id: "1"){ items(sort: {Field: "name", Order: "DESC"}){name}}}`,
			want:        `{"data":{"q":{"items":[{"name":"e"},{"name":"d"},{"name":"c"},{"name":"b"},{"name":"a"}]}}}`,
		},
		{
			description: "second level string sort",
			query:       `query { q(id: "1"){ items(sort: {Field: "leaf_name", Order: "ASC"}){name leaf{name}}}}`,
			want:        `{"data":{"q":{"items":[{"leaf":{"name":""},"name":"c"},{"leaf":{"name":""},"name":"d"},{"leaf":{"name":""},"name":"e"},{"leaf":{"name":"leafA"},"name":"a"},{"leaf":{"name":"leafB"},"name":"b"}]}}}`,
		},
		{
			description: "int sort ascending by default",
			query:       `query { q(id: "1"){ items(sort: {Field: "value"}){value}}}`,
			want:        `{"data":{"q":{"items":[{"value":1},{"value":2},{"value":3},{"value":4},{"value":5}]}}}`,
		},
		{
			description: "int sort descending",
			query:       `query { q(id: "1"){ items(sort: {Field: "value", Order: "DESC"}){value}}}`,
			want:        `{"data":{"q":{"items":[{"value":5},{"value":4},{"value":3},{"value":2},{"value":1}]}}}`,
		},
		{
			description: "float64 sort ascending by default",
			query:       `query { q(id: "1"){ items(sort: {Field: "floatvalue"}){floatvalue}}}`,
			want:        `{"data":{"q":{"items":[{"floatvalue":1.1},{"floatvalue":1.2},{"floatvalue":2.1},{"floatvalue":2.2},{"floatvalue":5.5}]}}}`,
		},
		{
			description: "float64 sort descending",
			query:       `query { q(id: "1"){ items(sort: {Field: "floatvalue", Order: "DESC"}){floatvalue}}}`,
			want:        `{"data":{"q":{"items":[{"floatvalue":5.5},{"floatvalue":2.2},{"floatvalue":2.1},{"floatvalue":1.2},{"floatvalue":1.1}]}}}`,
		},
		{
			description: "string sort and integer filter",
			query:       `query { q(id: "1"){ items(filter: {Field: "floatvalue", Operation: "<", Argument:{Value: 2}}, sort: {Field: "name"}){name floatvalue}}}`,
			want:        `{"data":{"q":{"items":[{"floatvalue":1.1,"name":"a"},{"floatvalue":1.2,"name":"b"}]}}}`,
		},
		{
			description: "string sort and limit filter",
			query:       `query { q(id: "1"){ items(filter: {Operation: "LIMIT", Argument:{Value: 2}}, sort: {Field: "name"}){name}}}`,
			want:        `{"data":{"q":{"items":[{"name":"a"},{"name":"b"}]}}}`,
		},
		{
			description: "string sort, no field, ascending by default",
			query:       `query { q(id: "1"){ stringlist(sort: {})}}`,
			want:        `{"data":{"q":{"stringlist":["a","b","c","d"]}}}`,
		},
		{
			description: "string sort, no field, ascending",
			query:       `query { q(id: "1"){ stringlist(sort: {Order: "ASC"})}}`,
			want:        `{"data":{"q":{"stringlist":["a","b","c","d"]}}}`,
		},
		{
			description: "string sort, no field, descending",
			query:       `query { q(id: "1"){ stringlist(sort: {Order: "DESC"})}}`,
			want:        `{"data":{"q":{"stringlist":["d","c","b","a"]}}}`,
		},
		{
			description: "int sort, no field, ascending",
			query:       `query { q(id: "1"){ intlist(sort: {Order: "ASC"})}}`,
			want:        `{"data":{"q":{"intlist":[1,2,3,4]}}}`,
		},
		{
			description: "int sort, no field, descending",
			query:       `query { q(id: "1"){ intlist(sort: {Order: "DESC"})}}`,
			want:        `{"data":{"q":{"intlist":[4,3,2,1]}}}`,
		},
		{
			description: "float sort, no field, ascending",
			query:       `query { q(id: "1"){ floatlist(sort: {Order: "ASC"})}}`,
			want:        `{"data":{"q":{"floatlist":[1.1,2.2,3.3,4.4]}}}`,
		},
		{
			description: "float sort, no field, descending",
			query:       `query { q(id: "1"){ floatlist(sort: {Order: "DESC"})}}`,
			want:        `{"data":{"q":{"floatlist":[4.4,3.3,2.2,1.1]}}}`,
		},
		{
			description: "int sort and limit filter, no field, ascending",
			query:       `query { q(id: "1"){ intlist(sort: {Order: "ASC"},filter: {Operation: "LIMIT", Argument:{Value: 2}})}}`,
			want:        `{"data":{"q":{"intlist":[1,2]}}}`,
		},
		{
			description: "int64 sort, no field, ascending",
			query:       `query { q(id: "1"){ int64list(sort: {Order: "ASC"})}}`,
			want:        `{"data":{"q":{"int64list":[11,22,33,44]}}}`,
		},
		{
			description: "int64 sort, no field, descending",
			query:       `query { q(id: "1"){ int64list(sort: {Order: "DESC"})}}`,
			want:        `{"data":{"q":{"int64list":[44,33,22,11]}}}`,
		},
		{
			description: "int64 sort and limit filter, no field, ascending",
			query:       `query { q(id: "1"){ int64list(sort: {Order: "ASC"},filter: {Operation: "LIMIT", Argument:{Value: 2}})}}`,
			want:        `{"data":{"q":{"int64list":[11,22]}}}`,
		},
		{
			description: "reporting string sort ascending",
			query:       `query { q(id: "1"){ items(sort: {Field: "name", Order: "ASC"}){name}}}`,
			want:        `{"data":{"q":{"items":[{"name":"a"},{"name":"b"},{"name":"c"},{"name":"d"},{"name":"e"}]}}}`,
			wantLF: &ListFunctions{
				SortField: "name",
				SortOrder: "ASC",
			},
		},
	}

	for _, test := range tests {
		qr := &testQueryReporter{}
		ctx := context.Background()
		if test.wantLF != nil {
			ctx = context.WithValue(ctx, QueryReporterContextKey, qr)
		}
		params := graphql.Params{
			Context:       ctx,
			Schema:        s,
			RequestString: test.query,
		}

		resp := graphql.Do(params)
		var err error
		if len(resp.Errors) != 0 {
			err = resp.Errors[0]
		}
		if test.wantLF != nil {
			if got, want := qr.sortField, test.wantLF.SortField; got != want {
				t.Errorf("Test %q - got sort field %q, want %q", test.description, got, want)
			}
			if got, want := qr.sortOrder, test.wantLF.SortOrder; got != want {
				t.Errorf("Test %q - got sort order %q, want %q", test.description, got, want)
			}
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

func TestListSortFailures(t *testing.T) {
	type testItem struct {
		A interface{}
	}

	tests := []struct {
		description string
		params      *sortParameters
		in          []interface{}
		errPrefix   string
	}{
		{
			description: "unsupported field type",
			params:      &sortParameters{field: "a"},
			in: []interface{}{
				testItem{A: []string{"b", "c"}},
				testItem{A: 2},
			},
			errPrefix: "unknown type",
		},
		{
			description: "unsupported type, no field",
			params:      &sortParameters{},
			in: []interface{}{
				testItem{A: 3},
				testItem{A: 2},
			},
			errPrefix: "unknown type",
		},
		{
			description: "field not found",
			params:      &sortParameters{field: "b"},
			in: []interface{}{
				testItem{A: []string{"b", "c"}},
				testItem{A: 2},
			},
			errPrefix: "failed to sort",
		},
		{
			description: "List fields is mix of int and string",
			params:      &sortParameters{field: "a"},
			in: []interface{}{
				testItem{A: 0},
				testItem{A: 2},
				testItem{A: "4"},
			},
			errPrefix: "failed to sort",
		},
		{
			description: "List fields is mix of int and slice",
			params:      &sortParameters{field: "a"},
			in: []interface{}{
				testItem{A: 0},
				testItem{A: []int{2}},
			},
			errPrefix: "failed to sort",
		},
		{
			description: "List fields is mix of string and float",
			params:      &sortParameters{field: "a"},
			in: []interface{}{
				testItem{A: "0"},
				testItem{A: 1.1},
			},
			errPrefix: "failed to sort",
		},
		{
			description: "List fields is mix of int and float",
			params:      &sortParameters{field: "a"},
			in: []interface{}{
				testItem{A: 0},
				testItem{A: 1.1},
			},
			errPrefix: "failed to sort",
		},
		{
			description: "mix of int and string, no field",
			params:      &sortParameters{},
			in:          []interface{}{0, 2, "4"},
			errPrefix:   "failed to sort",
		},
	}

	for _, test := range tests {
		err := listSort(test.params, test.in)
		if err == nil {
			t.Fatalf("Test %q - want error got none", test.description)
		}
		if !strings.HasPrefix(err.Error(), test.errPrefix) {
			t.Errorf("Test %q - want prefix %q got err: %v", test.description, test.errPrefix, err)
		}
	}
}
