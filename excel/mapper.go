package excel

import (
	"fmt"
	"reflect"
	"regexp"
	"sync"
)

// RowMapper handles mapping of row data to struct fields.
type RowMapper struct {
	headerMap  map[string]int       // Maps column name to column index (0-based)
	validate   bool                 // If true, enforce min/max/pattern tag constraints
	converters map[string]Converter // Per-column overrides for the default ConvertToType conversion
}

// NewRowMapper creates a new RowMapper with the given header row.
func NewRowMapper(headers []string) *RowMapper {
	m := make(map[string]int)
	for i, h := range headers {
		m[h] = i
	}
	return &RowMapper{headerMap: m}
}

// EnableValidation turns min:/max:/pattern: tag enforcement on or off.
// Required-column enforcement is unconditional and unaffected by this
// setting; it mirrors config.Config.ValidateData, which callers such as
// ImportToStructs forward here.
func (m *RowMapper) EnableValidation(enabled bool) {
	m.validate = enabled
}

// SetConverters registers per-column Converters, keyed by column name,
// that mapField consults before falling back to ConvertToType. Passing
// nil clears any converters previously set.
func (m *RowMapper) SetConverters(converters map[string]Converter) {
	m.converters = converters
}

// MapRow maps a row of data to a pointer to a struct.
// 'dest' must be a pointer to a struct.
func (m *RowMapper) MapRow(row []any, dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return NewConfigError("dest", dest, "dest must be a pointer to a struct")
	}

	elem := v.Elem()
	tags, err := GetFieldTags(elem.Type())
	if err != nil {
		return err
	}

	for i := range tags {
		if err := m.mapField(row, elem, &tags[i]); err != nil {
			return err
		}
	}

	return nil
}

// mapField maps the single row value described by tag onto the
// corresponding field of elem, applying type conversion and, if
// validation is enabled, min/max/pattern constraints. tag is a pointer
// only to avoid copying TagOptions on every call; mapField never
// mutates it.
func (m *RowMapper) mapField(row []any, elem reflect.Value, tag *TagOptions) error {
	colIdx, ok := m.headerMap[tag.Name]
	if !ok {
		if tag.Required {
			return NewValidationError(0, tag.Name, nil, fmt.Sprintf("missing required column: %s", tag.Name), nil)
		}
		return nil
	}

	if colIdx >= len(row) {
		return nil
	}

	val := row[colIdx]
	if val == nil {
		return nil
	}

	// The string form is the common denominator ConvertToType parses
	// from, regardless of the concrete type excelize/CSV handed back.
	strVal := fmt.Sprintf("%v", val)

	// FieldByIndexErr (rather than Field, which only accepts a single
	// depth) supports tag.Index paths longer than one element, which
	// GetFieldTags produces for fields reached through an embedded
	// struct. It also reports a nil embedded pointer as an error
	// instead of panicking.
	field, err := elem.FieldByIndexErr(tag.Index)
	if err != nil {
		return NewConfigError(tag.Name, strVal, fmt.Sprintf("cannot reach field for column %s: %v", tag.Name, err))
	}
	if !field.CanSet() {
		return nil
	}

	var converted any
	if custom, ok := m.converters[tag.Name]; ok {
		converted, err = custom(strVal, field.Type())
	} else {
		converted, err = ConvertToType(strVal, field.Type().String())
	}
	if err != nil {
		return NewValidationError(0, tag.Name, strVal, fmt.Sprintf("failed to convert field %s: %v", tag.Name, err), err)
	}

	if m.validate {
		if err := validateTagConstraints(tag, strVal, converted); err != nil {
			return err
		}
	}

	setFieldValue(field, converted)
	return nil
}

// patternCache stores compiled regular expressions for pattern: tag
// constraints, keyed by the pattern string, so repeated MapRow calls over
// many rows do not recompile the same pattern for every row.
var patternCache = sync.Map{}

// compilePattern returns a compiled regexp for pattern, using patternCache
// to avoid recompiling the same pattern on every call.
func compilePattern(pattern string) (*regexp.Regexp, error) {
	if cached, ok := patternCache.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	patternCache.Store(pattern, re)
	return re, nil
}

// validateTagConstraints enforces the min:, max:, and pattern: tag
// constraints already parsed into tag by ParseTag. min/max only apply to
// fields that convert to a numeric type; pattern is matched against the
// raw string value before conversion. tag is a pointer only to avoid
// copying TagOptions on every call; validateTagConstraints never
// mutates it.
func validateTagConstraints(tag *TagOptions, strVal string, converted any) error {
	if tag.Pattern != "" {
		re, err := compilePattern(tag.Pattern)
		if err != nil {
			return NewConfigError(tag.Name, tag.Pattern, fmt.Sprintf("invalid pattern: %v", err))
		}
		if !re.MatchString(strVal) {
			return NewValidationError(0, tag.Name, strVal,
				fmt.Sprintf("value %q does not match pattern %q", strVal, tag.Pattern), nil)
		}
	}

	if tag.Min == nil && tag.Max == nil {
		return nil
	}

	numeric, ok := toFloat64(converted)
	if !ok {
		// min/max only make sense for numeric fields; silently skip
		// non-numeric fields rather than rejecting a tag combination
		// the developer may have applied for other reasons.
		return nil
	}

	if tag.Min != nil && numeric < float64(*tag.Min) {
		return NewValidationError(0, tag.Name, strVal,
			fmt.Sprintf("value %v is below minimum %d", converted, *tag.Min), nil)
	}
	if tag.Max != nil && numeric > float64(*tag.Max) {
		return NewValidationError(0, tag.Name, strVal,
			fmt.Sprintf("value %v exceeds maximum %d", converted, *tag.Max), nil)
	}

	return nil
}

// toFloat64 extracts a float64 representation of v if v is a numeric
// kind, for uniform comparison against min/max bounds.
func toFloat64(v any) (float64, bool) {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(val.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(val.Uint()), true
	case reflect.Float32, reflect.Float64:
		return val.Float(), true
	default:
		return 0, false
	}
}

// setFieldValue sets the value of a reflect.Value from an interface{}.
// It handles basic type matching.
func setFieldValue(field reflect.Value, value any) {
	if value == nil {
		return
	}

	val := reflect.ValueOf(value)

	if field.Kind() != val.Kind() {
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if isSignedInt(val.Kind()) {
				field.SetInt(val.Int())
				return
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if isUnsignedInt(val.Kind()) {
				field.SetUint(val.Uint())
				return
			}
		case reflect.Float32, reflect.Float64:
			if isFloat(val.Kind()) {
				field.SetFloat(val.Float())
				return
			}
		}
	}

	if field.Type() == val.Type() {
		field.Set(val)
	}
}

func isSignedInt(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Int64
}

func isUnsignedInt(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uint64
}

func isFloat(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}
