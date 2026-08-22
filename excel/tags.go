package excel

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// TagOptions represents the parsed options from a struct tag.
// It contains the column name and various validation/behavior flags.
type TagOptions struct {
	Name      string // The column name mapped to this field
	Required  bool   // If true, the field must not be empty
	OmitEmpty bool   // If true, a zero-value field renders as a blank cell during export
	Min       *int   // Minimum value for numeric fields (nil if not set)
	Max       *int   // Maximum value for numeric fields (nil if not set)
	Pattern   string // Regex pattern for validation (empty if not set)

	// Index is the field's path from the top-level struct, suitable
	// for reflect.Value.FieldByIndexErr. It has more than one element
	// only for a field reached through an anonymous (embedded) struct,
	// where each element is the field index at that nesting level.
	Index []int
}

// tagCache stores parsed tag options for struct fields to avoid repeated parsing.
// The key is the reflect.Type's string representation, unique per type
// even for two distinct anonymous struct types that would otherwise
// share an empty package path and name.
var tagCache = sync.Map{}

// maxEmbeddedFieldDepth bounds how many levels of anonymous
// (embedded) structs GetFieldTags will flatten. Go permits an
// embedded pointer field to reference its own struct type (unlike
// embedding by value, which would not compile), so without a limit a
// self-referential embedded pointer would recurse until the depth
// bound below is reached instead of overflowing the goroutine stack.
const maxEmbeddedFieldDepth = 10

// ParseTag parses the "excel" struct tag and returns TagOptions.
// Format: `excel:"column_name,option1,option2:value"`
// Example: `excel:"email,required,min:5"`
// Returns an error if a min: or max: value is not a valid integer, since a
// malformed tag is a struct definition bug that should surface immediately
// rather than silently behaving as min:0/max:0.
func ParseTag(tag string, fieldIndex []int) (TagOptions, error) {
	options := TagOptions{
		Index: fieldIndex,
	}

	if tag == "" {
		return options, nil
	}

	parts := strings.Split(tag, ",")
	if len(parts) > 0 {
		options.Name = strings.TrimSpace(parts[0])
	}

	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		switch {
		case part == "required":
			options.Required = true
		case part == "omitempty":
			options.OmitEmpty = true
		case strings.HasPrefix(part, "min:"):
			raw := strings.TrimPrefix(part, "min:")
			val, err := strconv.Atoi(raw)
			if err != nil {
				return options, fmt.Errorf("excel tag %q: invalid min value %q: %w", tag, raw, err)
			}
			options.Min = &val
		case strings.HasPrefix(part, "max:"):
			raw := strings.TrimPrefix(part, "max:")
			val, err := strconv.Atoi(raw)
			if err != nil {
				return options, fmt.Errorf("excel tag %q: invalid max value %q: %w", tag, raw, err)
			}
			options.Max = &val
		case strings.HasPrefix(part, "pattern:"):
			options.Pattern = strings.TrimPrefix(part, "pattern:")
		}
	}

	return options, nil
}

// GetFieldTags extracts and parses tags for all exported fields of a
// struct type, including fields reached through an anonymous
// (embedded) struct or pointer-to-struct field, which are flattened
// into the same result list rather than treated as one opaque field.
// This mirrors the flattening BasicExporter already applies on export
// (see isEmbeddedStructField in excel/exporter/exporter_data.go). It
// uses caching to improve performance for repeated calls.
func GetFieldTags(t reflect.Type) ([]TagOptions, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, NewConfigError("type", t.String(), "GetFieldTags requires a struct type")
	}

	cacheKey := t.String()
	if cached, ok := tagCache.Load(cacheKey); ok {
		return cached.([]TagOptions), nil
	}

	tags, err := collectFieldTags(t, nil, 0)
	if err != nil {
		return nil, err
	}

	tagCache.Store(cacheKey, tags)
	return tags, nil
}

// collectFieldTags walks t's exported fields and returns their parsed
// tags. parentIndex is the field-index path already taken to reach t
// (empty for the top-level struct passed to GetFieldTags); it is
// prepended to each field's own index so the result is usable directly
// with reflect.Value.FieldByIndexErr regardless of nesting depth.
func collectFieldTags(t reflect.Type, parentIndex []int, depth int) ([]TagOptions, error) {
	if depth > maxEmbeddedFieldDepth {
		return nil, NewConfigError("type", t.String(), "embedded struct nesting exceeds the supported depth")
	}

	var tags []TagOptions
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		index := make([]int, len(parentIndex)+1)
		copy(index, parentIndex)
		index[len(parentIndex)] = i

		if embedded, ok := anonymousStructType(&field); ok {
			nested, err := collectFieldTags(embedded, index, depth+1)
			if err != nil {
				return nil, err
			}
			tags = append(tags, nested...)
			continue
		}

		tagValue := field.Tag.Get("excel")
		if tagValue == "-" {
			continue // Explicitly ignored
		}

		parsed, err := ParseTag(tagValue, index)
		if err != nil {
			return nil, NewConfigError(field.Name, tagValue, err.Error())
		}

		// If no name specified in tag, use field name
		if parsed.Name == "" {
			parsed.Name = field.Name
		}

		tags = append(tags, parsed)
	}

	return tags, nil
}

// anonymousStructType reports whether field is an anonymous (embedded)
// struct or pointer-to-struct field and, if so, returns the struct
// type to flatten into the parent's field list. The pointer case
// matches isEmbeddedStructField's export-side rule so import and
// export flatten the same shapes. field is a pointer only to avoid
// copying reflect.StructField on every call; it is never mutated.
func anonymousStructType(field *reflect.StructField) (reflect.Type, bool) {
	if !field.Anonymous {
		return nil, false
	}
	t := field.Type
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false
	}
	return t, true
}
