package gql

import (
	"reflect"
	"testing"
)

func TestExtractField(t *testing.T) {
	tests := []struct {
		description string
		st          interface{}
		key         string
		want        interface{}
	}{
		{
			description: "Basic exposed field",
			st:          TestBase{Id: "id"},
			key:         "id",
			want:        "id",
		},
		{
			description: "Basic exposed field, but mismatched case in key",
			st:          TestBase{Id: "id"},
			key:         "Id",
			want:        nil,
		},
		{
			description: "key not found",
			st:          TestBase{Id: "id"},
			key:         "Bogus",
			want:        nil,
		},
		{
			description: "field in embedded base",
			st:          testEmbed{TestBase: TestBase{Id: "id"}},
			key:         "id",
			want:        "id",
		},
		{
			description: "field in first embedded base",
			st:          testDoubleEmbed{TestBase: TestBase{Id: "id"}, TestBase2: TestBase2{Id2: "id2"}},
			key:         "id",
			want:        "id",
		},
		{
			description: "field in second embedded base",
			st:          testDoubleEmbed{TestBase: TestBase{Id: "id"}, TestBase2: TestBase2{Id2: "id2"}},
			key:         "id2",
			want:        "id2",
		},
		{
			description: "field is unexported",
			st:          TestBase{},
			key:         "assets",
			want:        nil,
		},
	}

	for _, test := range tests {
		got := ExtractField(test.st, test.key)

		if got != test.want {
			t.Errorf("Test %q - got %v, want %v", test.description, got, test.want)
		}
	}
}

func TestExtractEmbeds(t *testing.T) {
	tb := TestBase{}
	tb2 := TestBase2{}
	oneEmbed := testEmbed{TestBase: tb}
	twoEmbeds := testDoubleEmbed{TestBase: tb, TestBase2: tb2}

	tests := []struct {
		description string
		in          interface{}
		want        interface{}
	}{
		{
			description: "Not a struct",
			in:          []string{"a"},
			want:        nil,
		},
		{
			description: "No embeds",
			in:          TestBase{},
			want:        nil,
		},
		{
			description: "One embed",
			in:          oneEmbed,
			want:        map[string]interface{}{"TestBase": tb},
		},
		{
			description: "two embeds",
			in:          twoEmbeds,
			want:        map[string]interface{}{"TestBase": tb, "TestBase2": tb2},
		},
	}

	for _, test := range tests {
		got := extractEmbeds(test.in)
		if test.want == nil && got == nil {
			continue
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("Test %q - Got %#v, want %#v", test.description, got, test.want)
		}
	}
}

func TestFieldName(t *testing.T) {
	tests := []struct {
		description string
		field       reflect.StructField
		want        string
	}{
		{
			description: "Empty",
			field:       reflect.StructField{},
			want:        "",
		},
		{
			description: "Name from basic JSON tag",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"jsonName"`)},
			want:        "jsonName",
		},
		{
			description: "Name from advanced JSON tag",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"jsonName,omitempty"`)},
			want:        "jsonName",
		},
		{
			description: "Name from field name when JSON tag exists but is nameless",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:",omitempty"`)},
			want:        "name",
		},
		{
			description: "Empty Name JSON field is -",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"-"`)},
			want:        "",
		},
		{
			description: "JSON name is -",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"-,"`)},
			want:        "-",
		},
		{
			description: "Name from field name",
			field:       reflect.StructField{Name: "name"},
			want:        "name",
		},
		{
			description: "Name from capital field name",
			field:       reflect.StructField{Name: "Name"},
			want:        "name",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"_name"`)},
			want:        "name",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"__name"`)},
			want:        "name",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"name_"`)},
			want:        "name",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"_name_"`)},
			want:        "name",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"_my_longer_name_"`)},
			want:        "mylongername",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "name", Tag: reflect.StructTag(`json:"__my__longer__name__"`)},
			want:        "mylongername",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "_name"},
			want:        "name",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "__name",},
			want:        "name",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "_name"},
			want:        "name",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "_name_"},
			want:        "name",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "_my_longer_name_"},
			want:        "mylongername",
		},
		{
			description: "Field contains internal building delimiter char `_`",
			field:       reflect.StructField{Name: "__my__longer__name__"},
			want:        "mylongername",
		},
	}

	for _, test := range tests {
		got := fieldName(test.field)

		if got != test.want {
			t.Errorf("Test %q - got %q, want %q", test.description, got, test.want)
		}
	}
}
