package gql

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// A Comparator checks whether a value Matches.
// Each NewListOperation returns a comparator to be used in List filtering.
type Comparator interface {
	Match(interface{}) bool
}

// NotComparator simply does a logical not on the Match result of its child.
type NotComparator struct {
	Child Comparator
}

func (c NotComparator) Match(raw interface{}) bool {
	return !c.Child.Match(raw)
}

// NewEqualComparator returns a comparator for equality of both strings and ints.
// The comparator expects either a string or int with the key "Value" as part of the given argument.
func NewEqualComparator(arg map[string]interface{}) (Comparator, error) {
	raw, ok := arg["Value"]
	if !ok {
		return nil, errors.New("filter argument is missing 'Value'")
	}
	switch a := raw.(type) {
	case int:
		return intEqual{value: a}, nil
	case string:
		return stringEqual{value: a}, nil
	default:
		return nil, errors.New("unsupported argument value, strings and integers are supported")
	}
}

// NewNotEqualComparator returns a comparator for not equality of both strings and ints.
// The comparator expects either a string or int with the key "Value" as part of the given argument.
func NewNotEqualComparator(arg map[string]interface{}) (Comparator, error) {
	eq, err := NewEqualComparator(arg)
	if err != nil {
		return nil, err
	}
	return NotComparator{Child: eq}, nil
}

// NewInComparator returns a comparator which returns true if the operand matches any of the given values.
// The comparator expects a list of strings under the key "Values" as part of the given argument.
func NewInComparator(arg map[string]interface{}) (Comparator, error) {
	raw, ok := arg["Values"]
	if !ok {
		return nil, errors.New("filter argument is missing 'Values'")
	}
	value := reflect.ValueOf(raw)
	if value.Kind() != reflect.Slice {
		return nil, errors.New("filter argument 'Values' should be a list of strings")
	}

	values := make([]string, value.Len())
	for i := 0; i < value.Len(); i++ {
		rawItem := value.Index(i).Interface()
		item, ok := rawItem.(string)
		if !ok {
			return nil, errors.New("filter argument 'Values' should be a list of strings")
		}
		values[i] = item
	}

	return inComparator{values: values}, nil
}

// NewIntegerComparator returns a NewListOperator for integer operations supported by integerComparator,
// specifically <, <=, > and >=.
// The returned NewListOperation expects an int with the key "Value" as part of the given argument.
func NewIntegerComparator(op string) NewListOperation {
	switch op {
	case ">", ">=", "<", "<=":
		break
	default:
		panic(fmt.Sprintf("unsupported integer comparison operation %q", op))
	}
	return func(arg map[string]interface{}) (Comparator, error) {
		raw, ok := arg["Value"]
		if !ok {
			return nil, errors.New("filter argument is missing 'Value'")
		}
		value, ok := raw.(int)
		if !ok {
			return nil, errors.New("filter argument 'Value' must be an integer")
		}

		return integerComparator{operation: op, value: value}, nil
	}
}

// NewNotInComparator wraps NewInComparator returning the opposite boolean.
func NewNotInComparator(arg map[string]interface{}) (Comparator, error) {
	in, err := NewInComparator(arg)
	if err != nil {
		return nil, err
	}
	return NotComparator{Child: in}, nil
}

// NewLimitLengthComparator returns a comparator which simply limits the length of the results to the integer
// given with the key "Value" in the argument.
func NewLimitLengthComparator(arg map[string]interface{}) (Comparator, error) {
	raw, ok := arg["Value"]
	if !ok {
		return nil, errors.New("filter argument is missing 'Value'")
	}
	limit, ok := raw.(int)
	if !ok {
		return nil, errors.New("filter argument 'Value' must be an integer")
	}

	return &limitLength{limit: limit}, nil
}

type inComparator struct {
	values []string
}

func (c inComparator) Match(raw interface{}) bool {
	in, ok := raw.(string)
	if !ok {
		return false
	}
	for _, v := range c.values {
		if in == v {
			return true
		}
	}
	return false
}

type intEqual struct {
	value int
}

func (c intEqual) Match(raw interface{}) bool {
	in, ok := raw.(int)
	if !ok {
		return false
	}
	return in == c.value
}

// integerComparator supports comparisons between ints for a these operations <, <=, > and >=.
type integerComparator struct {
	operation string
	value     int
}

func (c integerComparator) Match(raw interface{}) bool {
	in, ok := raw.(int)
	if !ok {
		rawfloat, ok := raw.(float64)
		if !ok {
			return false
		}
		in = int(rawfloat)
		// This covers the case where the underlying field is a float but the comparison in the filter in an int
		// this can happen because the distinction between float and int is not strong in the JSON
	}
	switch c.operation {
	case ">":
		return in > c.value
	case ">=":
		return in >= c.value
	case "<":
		return in < c.value
	case "<=":
		return in <= c.value
	default:
		panic("unknown operation, this struct should always be initialized with NewIntegerComparator making this unreachable")
	}
}

type limitLength struct {
	limit int
	count int
	mtx   sync.Mutex
}

func (c *limitLength) Match(raw interface{}) bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.count++
	if c.count > c.limit {
		return false
	}
	return true
}

type stringEqual struct {
	value string
}

func (c stringEqual) Match(raw interface{}) bool {
	in, ok := raw.(string)
	if !ok {
		return false
	}
	return in == c.value
}
