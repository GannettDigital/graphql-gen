package gql

import "testing"

func TestOperators(t *testing.T) {
	tests := []struct {
		description          string
		operation            NewListOperation
		arguments            map[string]interface{}
		operand              interface{}
		wantInvalidOperation bool
		want                 bool
	}{
		{
			description: "string equal valid and true",
			operation:   NewEqualComparator,
			arguments:   map[string]interface{}{"Value": "a"},
			operand:     "a",
			want:        true,
		},
		{
			description: "string equal valid and true, empty string",
			operation:   NewEqualComparator,
			arguments:   map[string]interface{}{"Value": ""},
			operand:     "",
			want:        true,
		},
		{
			description: "string equal valid and false",
			operation:   NewEqualComparator,
			arguments:   map[string]interface{}{"Value": "a"},
			operand:     "b",
			want:        false,
		},
		{
			description: "string equal invalid, mismatched operand",
			operation:   NewEqualComparator,
			arguments:   map[string]interface{}{"Value": "a"},
			operand:     3,
			want:        false,
		},
		{
			description:          "NewEqualComparator invalid argument, nil",
			operation:            NewEqualComparator,
			arguments:            nil,
			wantInvalidOperation: true,
		},
		{
			description:          "NewEqualComparator invalid argument, missing value",
			operation:            NewEqualComparator,
			arguments:            map[string]interface{}{"nothing": "a"},
			wantInvalidOperation: true,
		},
		{
			description:          "NewEqualComparator invalid argument, float",
			operation:            NewEqualComparator,
			arguments:            map[string]interface{}{"Value": 3.14},
			wantInvalidOperation: true,
		},
		{
			description: "int equal valid and true",
			operation:   NewEqualComparator,
			arguments:   map[string]interface{}{"Value": 1},
			operand:     1,
			want:        true,
		},
		{
			description: "int equal valid and true, 0",
			operation:   NewEqualComparator,
			arguments:   map[string]interface{}{"Value": 0},
			operand:     0,
			want:        true,
		},
		{
			description: "int equal valid and true, -1",
			operation:   NewEqualComparator,
			arguments:   map[string]interface{}{"Value": -1},
			operand:     -1,
			want:        true,
		},
		{
			description: "int equal valid and false",
			operation:   NewEqualComparator,
			arguments:   map[string]interface{}{"Value": 1},
			operand:     2,
			want:        false,
		},
		{
			description: "int equal invalid, mismatched operand",
			operation:   NewEqualComparator,
			arguments:   map[string]interface{}{"Value": 1},
			operand:     "a",
			want:        false,
		},
		{
			description: "string not equal valid and false",
			operation:   NewNotEqualComparator,
			arguments:   map[string]interface{}{"Value": "a"},
			operand:     "a",
			want:        false,
		},
		{
			description: "string not equal valid and false",
			operation:   NewNotEqualComparator,
			arguments:   map[string]interface{}{"Value": "a"},
			operand:     "b",
			want:        true,
		},
		{
			description: "int not equal valid and true",
			operation:   NewNotEqualComparator,
			arguments:   map[string]interface{}{"Value": 1},
			operand:     1,
			want:        false,
		},
		{
			description: "int not equal valid and false",
			operation:   NewNotEqualComparator,
			arguments:   map[string]interface{}{"Value": 1},
			operand:     2,
			want:        true,
		},
		{
			description:          "NewNotEqualComparator invalid argument, missing value",
			operation:            NewNotEqualComparator,
			arguments:            map[string]interface{}{"nothing": "a"},
			wantInvalidOperation: true,
		},
		{
			description: "limitLength to 0", // Note limitLength is better tested as part of TestResolveListField
			operation:   NewLimitLengthComparator,
			arguments:   map[string]interface{}{"Value": 0},
			want:        false,
		},
		{
			description: "limitLength > 0",
			operation:   NewLimitLengthComparator,
			arguments:   map[string]interface{}{"Value": 1},
			want:        true,
		},
		{
			description:          "NewLimitLengthComparator invalid argument, missing value",
			operation:            NewLimitLengthComparator,
			arguments:            map[string]interface{}{"nothing": "a"},
			wantInvalidOperation: true,
		},
		{
			description:          "NewLimitLengthComparator invalid argument, string value",
			operation:            NewLimitLengthComparator,
			arguments:            map[string]interface{}{"Value": "a"},
			wantInvalidOperation: true,
		},
		{
			description: "IN expect true",
			operation:   NewInComparator,
			arguments:   map[string]interface{}{"Values": []string{"a", "b"}},
			operand:     "a",
			want:        true,
		},
		{
			description: "IN expect false",
			operation:   NewInComparator,
			arguments:   map[string]interface{}{"Values": []string{"a", "b"}},
			operand:     "c",
			want:        false,
		},
		{
			description: "IN compare against number expect false",
			operation:   NewInComparator,
			arguments:   map[string]interface{}{"Values": []string{"a", "b"}},
			operand:     3,
			want:        false,
		},
		{
			description: "NOT IN expect true",
			operation:   NewNotInComparator,
			arguments:   map[string]interface{}{"Values": []string{"a", "b"}},
			operand:     "c",
			want:        true,
		},
		{
			description: "NOT IN expect false",
			operation:   NewNotInComparator,
			arguments:   map[string]interface{}{"Values": []string{"a", "b"}},
			operand:     "a",
			want:        false,
		},
		{
			description:          "NOT IN invalid argument, missing values",
			operation:            NewNotInComparator,
			arguments:            map[string]interface{}{"nothing": "a"},
			wantInvalidOperation: true,
		},
		{
			description:          "NOT IN invalid argument, string values",
			operation:            NewNotInComparator,
			arguments:            map[string]interface{}{"Values": "a, b, c"},
			wantInvalidOperation: true,
		},
		{
			description:          "NOT IN invalid argument, integer in list",
			operation:            NewNotInComparator,
			arguments:            map[string]interface{}{"Values": []interface{}{"a", 1}},
			wantInvalidOperation: true,
		},
		{
			description: "< expect true",
			operation:   NewIntegerComparator("<"),
			arguments:   map[string]interface{}{"Value": 5},
			operand:     4,
			want:        true,
		},
		{
			description: "< expect false",
			operation:   NewIntegerComparator("<"),
			arguments:   map[string]interface{}{"Value": 5},
			operand:     5,
			want:        false,
		},
		{
			description: "<= expect true",
			operation:   NewIntegerComparator("<="),
			arguments:   map[string]interface{}{"Value": 5},
			operand:     5,
			want:        true,
		},
		{
			description: "<= expect false",
			operation:   NewIntegerComparator("<="),
			arguments:   map[string]interface{}{"Value": 5},
			operand:     6,
			want:        false,
		},
		{
			description: "> expect true",
			operation:   NewIntegerComparator(">"),
			arguments:   map[string]interface{}{"Value": 5},
			operand:     6,
			want:        true,
		},
		{
			description: "> expect false",
			operation:   NewIntegerComparator(">"),
			arguments:   map[string]interface{}{"Value": 5},
			operand:     5,
			want:        false,
		},
		{
			description: ">= expect true",
			operation:   NewIntegerComparator(">="),
			arguments:   map[string]interface{}{"Value": 5},
			operand:     5,
			want:        true,
		},
		{
			description: ">= expect false",
			operation:   NewIntegerComparator(">="),
			arguments:   map[string]interface{}{"Value": 5},
			operand:     4,
			want:        false,
		},
		{
			description: "Integer Comparator, non-string Operand",
			operation:   NewIntegerComparator(">="),
			arguments:   map[string]interface{}{"Value": 5},
			operand:     "a",
			want:        false,
		},
		{
			description:          "Integrer Comparator invalid argument, missing value",
			operation:            NewIntegerComparator(">"),
			arguments:            map[string]interface{}{"nothing": "a"},
			wantInvalidOperation: true,
		},
		{
			description:          "Integrer Comparator invalid argument, string value",
			operation:            NewIntegerComparator(">"),
			arguments:            map[string]interface{}{"Value": "five"},
			wantInvalidOperation: true,
		},
	}

	for _, test := range tests {
		op, err := test.operation(test.arguments)
		switch {
		case err != nil && test.wantInvalidOperation:
			continue
		case err == nil && test.wantInvalidOperation:
			t.Errorf("Test %q - got nil error want error on invalid operation", test.description)
			continue
		case err != nil && !test.wantInvalidOperation:
			t.Errorf("Test %q - got invalid operation error want nil: %v", test.description, err)
			continue
		}

		got := op.Match(test.operand)
		if got != test.want {
			t.Errorf("Test %q - got %t, want %t", test.description, got, test.want)
		}
	}
}
