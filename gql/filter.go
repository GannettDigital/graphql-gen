package gql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/GannettDigital/graphql"
	"github.com/GannettDigital/graphql/language/ast"
	"github.com/GannettDigital/graphql/language/kinds"
)

var (
	// ListOperations is the set of list filters operations available for use by ResolveListField.
	// The string is the operation, ie '==' which matches a particular NewListOperation function.
	// The default implementations are pre-populated and additional operations can be added before running the
	// ObjectBuilder.
	ListOperations = map[string]NewListOperation{
		"==":     NewEqualComparator,
		"!=":     NewNotEqualComparator,
		">":      NewIntegerComparator(">"),
		">=":     NewIntegerComparator(">="),
		"<":      NewIntegerComparator("<"),
		"<=":     NewIntegerComparator("<="),
		"LIMIT":  NewLimitLengthComparator,
		"IN":     NewInComparator,
		"NOT IN": NewNotInComparator,
	}

	graphqlListFilter = graphql.NewScalar(graphql.ScalarConfig{
		Name:         "ListFilter",
		Description:  "A JSON object used for filtering list items, includes a required field 'Operation' and optional fields 'Argument' and 'Field'.",
		Serialize:    func(value interface{}) interface{} { return nil },
		ParseValue:   func(value interface{}) interface{} { return nil },
		ParseLiteral: func(valueAST ast.Value) interface{} { return valueAST.GetValue() },
	})
)

// NewListOperation is a function which returns a Comparator given a set of list filter arguments.
type NewListOperation func(argument map[string]interface{}) (Comparator, error)

// listFilterJSON represents a list filter as defined as a JSON object.
type listFilterJSON struct {
	Field     string                 `json:"Field,omitempty"`
	Operation string                 `json:"Operation"`
	Argument  map[string]interface{} `json:"Argument,omitempty"`
}

// newListFilterJSON will parse an AST ObjectField returning the listFilterJSON	found within it.
// A newListFilterJSON can be made unmarshaled directly from JSON but in a GraphQL query it has already been parsed
// by the GraphQL library into an AST ObjectField.
func newListFilterJSON(fields []*ast.ObjectField) (*listFilterJSON, error) {
	var lf listFilterJSON
	for _, f := range fields {
		switch f.Name.Value {
		case "Field":
			v, ok := f.GetValue().(*ast.StringValue)
			if !ok {
				return nil, errors.New("unable to parse filter argument field Field")
			}
			lf.Field = v.Value
		case "Operation":
			v, ok := f.GetValue().(*ast.StringValue)
			if !ok {
				return nil, errors.New("unable to parse filter argument field Operation")
			}
			lf.Operation = v.Value
		case "Argument":
			arg, ok := f.GetValue().(*ast.ObjectValue)
			if !ok {
				return nil, errors.New("unable to parse filter argument field Argument")
			}
			argfields := make(map[string]interface{})
			for _, field := range arg.Fields {
				value, err := parseASTValue(field.GetValue())
				if err != nil {
					return nil, fmt.Errorf("unable to parse filter -> Argument -> %s: %v", field.Name.Value, err)
				}
				argfields[field.Name.Value] = value
			}
			lf.Argument = argfields
		}
	}

	return &lf, nil
}

func (lf listFilterJSON) String() string {
	var arguments []string
	for _, arg := range lf.Argument {
		arguments = append(arguments, fmt.Sprintf("%v", arg))
	}
	return fmt.Sprintf("Field:%v, Operation:%v, Arguments:%v", lf.Field, lf.Operation, strings.Join(arguments, ","))
}

// parseASTValue will recursively follow a AST value structure to build up a Golang object.
func parseASTValue(in interface{}) (interface{}, error) {
	value, ok := in.(ast.Value)
	if !ok {
		return nil, errors.New("unable to parse filter argument")
	}
	switch value.GetKind() {
	case kinds.StringValue:
		return value.GetValue(), nil
	case kinds.IntValue:
		// It isn't clear why the GraphQL library is written this way but the Value of a Intfield is stored as
		// a string so if needed convert
		strValue, ok := value.GetValue().(string)
		if !ok {
			if _, ok := value.GetValue().(int); ok { // In case the library is fixed at some point
				return value.GetValue(), nil
			}
			return nil, errors.New("unable to determine value of integer")
		}
		realValue, err := strconv.Atoi(strValue)
		if err != nil {
			return nil, errors.New("unable to determine value of integer")
		}
		return realValue, nil
	case kinds.ListValue:
		v, ok := value.(*ast.ListValue)
		if !ok {
			return nil, errors.New("failed to parse AST list")
		}
		var list []interface{}
		for _, item := range v.Values {
			itemValue, err := parseASTValue(item)
			if err != nil {
				return nil, err
			}
			list = append(list, itemValue)
		}
		return list, nil
	default:
		return nil, fmt.Errorf("unhandled AST value type %q", value.GetKind())
	}
}

// An listFilter is able to filter values in an array only returning one which match its comparator.
type listFilter struct {
	fieldName string
	op        Comparator
	json      *listFilterJSON
}

// newListFilter parses a given argument into a listFilter. The type of listFilter returned is based on the operation.
func newListFilter(arg interface{}) (*listFilter, error) {
	if arg == nil {
		return nil, nil
	}
	fields, ok := arg.([]*ast.ObjectField)
	if !ok {
		return nil, errors.New("unable to parse filter argument")
	}

	lf, err := newListFilterJSON(fields)
	if err != nil {
		return nil, err
	}

	if lf.Operation == "" {
		return nil, errors.New("filter Operation is undefined")
	}

	newOp, ok := ListOperations[lf.Operation]
	if !ok {
		return nil, fmt.Errorf("unknown filter operator %q", lf.Operation)
	}
	op, err := newOp(lf.Argument)
	if err != nil {
		return nil, err
	}

	return &listFilter{fieldName: lf.Field, op: op, json: lf}, nil
}

func (lf listFilter) match(raw interface{}) (bool, error) {
	// In the case  the fieldName is empty, DeepExtractField will always error.
	// So check for this and don't extract the field but just try to match it to nil.
	// This comes into play with filters such as limitLength
	if lf.fieldName == "" {
		return lf.op.Match(nil), nil
	}
	field, err := deepExtractFieldWithError(raw, lf.fieldName)
	if err != nil {
		return false, err
	}
	return lf.op.Match(field), nil
}
