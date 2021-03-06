package gql

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/GannettDigital/graphql"
)

// DeepExtractField executed the deepExtractFieldWithError method and swallows the error, if any.
func DeepExtractField(s interface{}, key string) interface{} {
	value, _ := deepExtractFieldWithError(s, key)
	return value
}

// deepExtractFieldWithError utilizes ExtractField multiple times to retrieve the value of a field in a multilevel object.
// The key is expected to use FieldPathSeparator to distinguish the multiple levels.
// This will throw an error if the value for the first split cannot be found.
func deepExtractFieldWithError(s interface{}, key string) (interface{}, error) {
	splits := strings.Split(key, FieldPathSeparator)

	value := s
	for i, split := range splits {
		value = ExtractField(value, split)
		if value == nil {
			if i == 0 {
				return nil, fmt.Errorf("unable to find field to extract: %q", split)
			}
			return nil, nil
		}
	}

	return value, nil
}

// ExtractField returns the value of a field from a struct, the key is the field name, which is matched
// to the output from the fieldName function. This function also handles searching any root level embedded structs.
// If the key does not match a field in the struct or the provided interface is not a struct nil is returned.
func ExtractField(s interface{}, key string) interface{} {
	sType := reflect.TypeOf(s)
	if sType.Kind() != reflect.Struct {
		return nil
	}

	sValue := reflect.ValueOf(s)
	var embeddedFields []int

	for i := 0; i < sType.NumField(); i++ {
		field := sType.Field(i)
		if field.Anonymous {
			embeddedFields = append(embeddedFields, i)
		}
		name := fieldName(field)
		if name == key {
			fieldValue := sValue.Field(i)
			return fieldValue.Interface()
		}
	}

	// try pulling from any embedded structs if the field was not found
	for _, i := range embeddedFields {
		embed := sValue.Field(i)
		if !embed.IsValid() {
			continue
		}
		if result := ExtractField(embed.Interface(), key); result != nil {
			return result
		}
	}

	return nil
}

// extractEmbeds will parse a struct looking for embedded struct and it will return a mapping of the names to the
// interface{} of any that are found
func extractEmbeds(parent interface{}) map[string]interface{} {
	sType := reflect.TypeOf(parent)
	if sType.Kind() != reflect.Struct {
		return nil
	}
	sValue := reflect.ValueOf(parent)

	embeds := make(map[string]interface{})
	for i := 0; i < sType.NumField(); i++ {
		field := sType.Field(i)
		if field.Anonymous {
			if field.PkgPath != "" { // This is empty for exported fields but not for unexported
				continue
			}
			embed := sValue.Field(i)
			if !embed.IsValid() {
				continue
			}
			embeds[embed.Type().Name()] = embed.Interface()
		}
	}

	if len(embeds) == 0 {
		return nil
	}

	return embeds
}

// fieldName extracts the name of a struct field from the JSON struct tag or if none uses field.Name transformed to
// be lowercase.
// NOTE: Thought this is primarily used in building GraphQL fields it does no checking or enforcing of GraphQL
// allowed characters for field names, that will happen during the GraphQL schema creation.
// All delimiter occurrences of _ will be stripped
func fieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	splits := strings.Split(jsonTag, ",")
	if len(splits) > 0 {
		name := splits[0]
		if name == "-" && len(splits) == 1 {
			// for details on this behavior see https://golang.org/pkg/encoding/json/#Marshal
			return ""
		}
		name = strings.Replace(name, "_", "", -1)
		if name != "" && nameIsValidGraphQL(name) {
			return name
		}
	}

	return strings.ToLower(strings.Replace(field.Name, "_", "", -1))
}

// nameIsValidGraphQL is used to determine if a name is a valid GraphQL field name
func nameIsValidGraphQL(name string) bool {
	match := graphql.NameRegExp.MatchString(name)
	return match
}

// fullFieldName returns the name of the field with its parent name included.
func fullFieldName(name, parent string) string {
	return strings.Join([]string{parent, name}, FieldPathSeparator)
}
