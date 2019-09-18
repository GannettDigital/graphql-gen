package gql

import (
	"errors"
	"fmt"
	"sort"

	"github.com/GannettDigital/graphql"
	"github.com/GannettDigital/graphql/language/ast"
)

const (
	ascending  = "ASC"
	descending = "DESC"
)

var graphqlSortFilter = graphql.NewScalar(graphql.ScalarConfig{
	Name:         "SortFilter",
	Description:  "A JSON object used for sorting list items, includes optional field 'Field' and 'Order'.",
	Serialize:    func(value interface{}) interface{} { return nil },
	ParseValue:   func(value interface{}) interface{} { return nil },
	ParseLiteral: func(valueAST ast.Value) interface{} { return valueAST.GetValue() },
})

type lessFunc func(i, j int) bool

type sortParameters struct {
	field string
	order string
}

// parseSortParameters parses the given argument returning the sort parameters.
// If the argument is nil the returned value is nil.
func parseSortParameters(arg interface{}) (*sortParameters, error) {
	if arg == nil {
		return nil, nil
	}
	fields, ok := arg.([]*ast.ObjectField)
	if !ok {
		return nil, errors.New("unable to parse sort argument")
	}

	var params sortParameters
	for _, f := range fields {
		switch f.Name.Value {
		case "Field":
			v, ok := f.GetValue().(*ast.StringValue)
			if !ok {
				return nil, errors.New("unable to parse sort argument field Field")
			}
			params.field = v.Value
		case "Order":
			v, ok := f.GetValue().(*ast.StringValue)
			if !ok {
				return nil, errors.New("unable to parse sort argument field Order")
			}
			switch v.Value {
			case ascending, descending, "":
			default:
				return nil, fmt.Errorf("sort order must be %q or %q or undefined", ascending, descending)
			}
			params.order = v.Value
		}
	}

	return &params, nil
}

// listSort will sort the given list in place according to the params specified.
// The type of the specified field is essential information when sorting but can't be determined until the list to be
// sorted is available which is why this does not follow the golang standard sort interface.
// An error can occur if sort the fields in the list items is not consistently the same type or an unsupported type.
func listSort(params *sortParameters, list []interface{}) error {
	if len(list) < 2 {
		return nil
	}

	errChan := make(chan error)

	go unprotectedListSort(params, list, errChan)
	err := <-errChan
	return err
}

// unprotectedListSort does the work described in listSort but can panic and so defers a recover and with that always
// returns a error to the errChan.
func unprotectedListSort(params *sortParameters, list []interface{}, errChan chan<- error) {
	defer func() {
		err := recover()
		if err != nil {
			errChan <- fmt.Errorf("failed to sort: %v", err)
		}
	}()

	var less lessFunc
	extracFunc := func(index int) interface{} {
		value, err := DeepExtractField(list[index], params.field)
		if err != nil {
			panic(err)
		}
		return value
	}
	if params.field == "" {
		extracFunc = func(index int) interface{} {
			return list[index]
		}
	}

	field := extracFunc(0)
	switch field.(type) {
	case int:
		less = func(i, j int) bool {
			iItem := extracFunc(i).(int)
			jItem := extracFunc(j).(int)

			return iItem < jItem
		}
	case int64:
		less = func(i, j int) bool {
			iItem := extracFunc(i).(int64)
			jItem := extracFunc(j).(int64)

			return iItem < jItem
		}
	case string:
		less = func(i, j int) bool {
			iItem := extracFunc(i).(string)
			jItem := extracFunc(j).(string)

			return iItem < jItem
		}
	case float64:
		less = func(i, j int) bool {
			iItem := extracFunc(i).(float64)
			jItem := extracFunc(j).(float64)

			return iItem < jItem
		}
	case nil:
		errChan <- fmt.Errorf("unable to extract sort field %q", params.field)
		return
	default:
		errChan <- fmt.Errorf("unknown type for sort field %q", params.field)
		return
	}

	if params.order == descending {
		less = notLess(less)
	}

	sort.SliceStable(list, less) // This can panic if the less function encounters an inconsistent type

	errChan <- nil
	return
}

func notLess(parent lessFunc) lessFunc {
	return func(i, j int) bool {
		return !parent(i, j)
	}
}
