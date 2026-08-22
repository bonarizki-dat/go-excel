package excel

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type User struct {
	ID    int    `excel:"id"`
	Name  string `excel:"name"`
	Email string `excel:"email"`
	Admin bool   `excel:"is_admin"`
}

func TestRowMapper_MapRow(t *testing.T) {
	headers := []string{"id", "name", "email", "is_admin"}
	mapper := NewRowMapper(headers)

	row := []any{"1", "John Doe", "john@example.com", "true"}
	var user User

	err := mapper.MapRow(row, &user)
	if err != nil {
		t.Fatalf("MapRow failed: %v", err)
	}

	if user.ID != 1 {
		t.Errorf("Expected ID 1, got %d", user.ID)
	}
	if user.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got %q", user.Name)
	}
	if user.Email != "john@example.com" {
		t.Errorf("Expected Email 'john@example.com', got %q", user.Email)
	}
	if !user.Admin {
		t.Errorf("Expected Admin true, got false")
	}
}

func TestRowMapper_MissingRequired(t *testing.T) {
	type RequiredStruct struct {
		Name string `excel:"name,required"`
	}

	headers := []string{"other"}
	mapper := NewRowMapper(headers)
	row := []any{"value"}
	var dest RequiredStruct

	err := mapper.MapRow(row, &dest)
	if err == nil {
		t.Error("Expected error for missing required column, got nil")
	}
}

// TestRowMapper_Validation_MinMax verifies that EnableValidation(true)
// enforces the min:/max: tag bounds, and that they are not enforced when
// validation is left disabled (the default).
func TestRowMapper_Validation_MinMax(t *testing.T) {
	type Bounded struct {
		Age int `excel:"age,min:18,max:65"`
	}

	headers := []string{"age"}

	t.Run("within bounds passes", func(t *testing.T) {
		mapper := NewRowMapper(headers)
		mapper.EnableValidation(true)
		var dest Bounded
		if err := mapper.MapRow([]any{"30"}, &dest); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest.Age != 30 {
			t.Errorf("Age = %d, want 30", dest.Age)
		}
	})

	t.Run("below min rejected when validation enabled", func(t *testing.T) {
		mapper := NewRowMapper(headers)
		mapper.EnableValidation(true)
		var dest Bounded
		if err := mapper.MapRow([]any{"10"}, &dest); err == nil {
			t.Error("expected error for value below min, got nil")
		}
	})

	t.Run("above max rejected when validation enabled", func(t *testing.T) {
		mapper := NewRowMapper(headers)
		mapper.EnableValidation(true)
		var dest Bounded
		if err := mapper.MapRow([]any{"99"}, &dest); err == nil {
			t.Error("expected error for value above max, got nil")
		}
	})

	t.Run("out of bounds allowed when validation disabled", func(t *testing.T) {
		mapper := NewRowMapper(headers)
		var dest Bounded
		if err := mapper.MapRow([]any{"10"}, &dest); err != nil {
			t.Fatalf("unexpected error with validation disabled: %v", err)
		}
		if dest.Age != 10 {
			t.Errorf("Age = %d, want 10", dest.Age)
		}
	})
}

// TestRowMapper_Validation_Pattern verifies that EnableValidation(true)
// enforces the pattern: tag against the raw cell value.
func TestRowMapper_Validation_Pattern(t *testing.T) {
	type Emailed struct {
		Email string `excel:"email,pattern:^[^@]+@[^@]+\\.[^@]+$"`
	}

	headers := []string{"email"}

	t.Run("matching pattern passes", func(t *testing.T) {
		mapper := NewRowMapper(headers)
		mapper.EnableValidation(true)
		var dest Emailed
		if err := mapper.MapRow([]any{"user@example.com"}, &dest); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("non-matching pattern rejected when validation enabled", func(t *testing.T) {
		mapper := NewRowMapper(headers)
		mapper.EnableValidation(true)
		var dest Emailed
		if err := mapper.MapRow([]any{"not-an-email"}, &dest); err == nil {
			t.Error("expected error for value not matching pattern, got nil")
		}
	})

	t.Run("non-matching pattern allowed when validation disabled", func(t *testing.T) {
		mapper := NewRowMapper(headers)
		var dest Emailed
		if err := mapper.MapRow([]any{"not-an-email"}, &dest); err != nil {
			t.Fatalf("unexpected error with validation disabled: %v", err)
		}
	})
}

// TestRowMapper_MapRow_InvalidDest verifies MapRow rejects a dest that is
// not a pointer to a struct, instead of panicking inside reflect.
func TestRowMapper_MapRow_InvalidDest(t *testing.T) {
	mapper := NewRowMapper([]string{"name"})

	t.Run("not a pointer", func(t *testing.T) {
		var notAPointer struct{ Name string }
		if err := mapper.MapRow([]any{"x"}, notAPointer); err == nil {
			t.Error("expected error for non-pointer dest, got nil")
		}
	})

	t.Run("pointer to non-struct", func(t *testing.T) {
		n := 0
		if err := mapper.MapRow([]any{"x"}, &n); err == nil {
			t.Error("expected error for pointer-to-non-struct dest, got nil")
		}
	})
}

// TestRowMapper_MapRow_PropagatesFieldTagError verifies a malformed excel
// tag on the destination struct surfaces as an error from MapRow (via
// GetFieldTags/ParseTag) rather than being silently ignored.
func TestRowMapper_MapRow_PropagatesFieldTagError(t *testing.T) {
	type BadTag struct {
		Age int `excel:"age,min:notanumber"`
	}
	mapper := NewRowMapper([]string{"age"})
	var dest BadTag
	if err := mapper.MapRow([]any{"30"}, &dest); err == nil {
		t.Error("expected error for malformed min: tag, got nil")
	}
}

// TestRowMapper_MapRow_UnmappedOptionalColumnSkipped verifies a field
// whose tag name has no corresponding header entry is left untouched
// when it is not marked required (the required case is already covered
// by TestRowMapper_MissingRequired).
func TestRowMapper_MapRow_UnmappedOptionalColumnSkipped(t *testing.T) {
	type Sparse struct {
		Name     string `excel:"name"`
		Nickname string `excel:"nickname"` // no "nickname" column in headers
	}
	mapper := NewRowMapper([]string{"name"})
	dest := Sparse{Nickname: "preexisting"}
	if err := mapper.MapRow([]any{"Alice"}, &dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Name != "Alice" || dest.Nickname != "preexisting" {
		t.Errorf("dest = %+v, want Name=Alice Nickname=preexisting (untouched)", dest)
	}
}

// TestRowMapper_MapRow_RowShorterThanHeader verifies a row with fewer
// columns than the header map leaves the corresponding field untouched
// instead of panicking on an out-of-range index.
func TestRowMapper_MapRow_RowShorterThanHeader(t *testing.T) {
	type Wide struct {
		Name string `excel:"name"`
		Age  int    `excel:"age"`
	}
	mapper := NewRowMapper([]string{"name", "age"})
	var dest Wide
	if err := mapper.MapRow([]any{"Alice"}, &dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Name != "Alice" || dest.Age != 0 {
		t.Errorf("dest = %+v, want Name=Alice Age=0", dest)
	}
}

// TestRowMapper_MapRow_NilCellLeftUntouched verifies a nil cell value is
// skipped rather than converted, leaving the field at its zero value.
func TestRowMapper_MapRow_NilCellLeftUntouched(t *testing.T) {
	type Named struct {
		Name string `excel:"name"`
	}
	mapper := NewRowMapper([]string{"name"})
	dest := Named{Name: "preexisting"}
	if err := mapper.MapRow([]any{nil}, &dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Name != "preexisting" {
		t.Errorf("Name = %q, want unchanged %q", dest.Name, "preexisting")
	}
}

// TestRowMapper_MapRow_UnconvertibleValueReturnsError verifies a cell value
// that cannot be converted to the field's type surfaces as a ValidationError
// from ConvertToType, rather than silently zeroing the field.
func TestRowMapper_MapRow_UnconvertibleValueReturnsError(t *testing.T) {
	type Numeric struct {
		Age int `excel:"age"`
	}
	mapper := NewRowMapper([]string{"age"})
	var dest Numeric
	if err := mapper.MapRow([]any{"not-a-number"}, &dest); err == nil {
		t.Error("expected error for unconvertible cell value, got nil")
	}
}

// TestRowMapper_Validation_InvalidPattern verifies a malformed pattern:
// regex surfaces as a ConfigError from validateTagConstraints instead of
// panicking inside regexp.Compile.
func TestRowMapper_Validation_InvalidPattern(t *testing.T) {
	type BadPattern struct {
		Name string `excel:"name,pattern:[invalid("`
	}
	mapper := NewRowMapper([]string{"name"})
	mapper.EnableValidation(true)
	var dest BadPattern
	if err := mapper.MapRow([]any{"anything"}, &dest); err == nil {
		t.Error("expected error for malformed pattern, got nil")
	}
}

// TestRowMapper_Validation_MinMax_NonNumericFieldSkipped verifies min:/max:
// on a field whose converted value is not numeric (toFloat64 returns
// ok=false) is silently skipped rather than rejected.
func TestRowMapper_Validation_MinMax_NonNumericFieldSkipped(t *testing.T) {
	// bool is deliberately not a supported numeric kind for toFloat64.
	type Flagged struct {
		Active bool `excel:"active,min:1"`
	}
	mapper := NewRowMapper([]string{"active"})
	mapper.EnableValidation(true)
	var dest Flagged
	if err := mapper.MapRow([]any{"true"}, &dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dest.Active {
		t.Error("Active = false, want true")
	}
}

// TestRowMapper_Validation_MinMax_UnsignedAndFloatFields exercises the
// uint/float branches of toFloat64 via validateTagConstraints, which the
// existing int-only min/max tests never reach.
func TestRowMapper_Validation_MinMax_UnsignedAndFloatFields(t *testing.T) {
	// Named BoundedMixed, not Bounded, to avoid colliding with the
	// unrelated local "Bounded" type in TestRowMapper_Validation_MinMax:
	// GetFieldTags' tagCache keys only on package path + type name, so
	// two distinct local types sharing a name would collide and one
	// would silently get the other's cached tags.
	type BoundedMixed struct {
		Count uint    `excel:"count,min:1,max:10"`
		Score float64 `excel:"score,min:0,max:100"`
	}

	t.Run("uint within bounds", func(t *testing.T) {
		mapper := NewRowMapper([]string{"count", "score"})
		mapper.EnableValidation(true)
		var dest BoundedMixed
		if err := mapper.MapRow([]any{"5", "50.0"}, &dest); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("uint above max rejected", func(t *testing.T) {
		mapper := NewRowMapper([]string{"count", "score"})
		mapper.EnableValidation(true)
		var dest BoundedMixed
		if err := mapper.MapRow([]any{"99", "50.0"}, &dest); err == nil {
			t.Error("expected error for uint field above max, got nil")
		}
	})

	t.Run("float above max rejected", func(t *testing.T) {
		mapper := NewRowMapper([]string{"count", "score"})
		mapper.EnableValidation(true)
		var dest BoundedMixed
		if err := mapper.MapRow([]any{"5", "150.0"}, &dest); err == nil {
			t.Error("expected error for float field above max, got nil")
		}
	})
}

// TestRowMapper_MapField_UnsettableFieldSkipped exercises the CanSet()
// guard in mapField directly. GetFieldTags already filters out unexported
// fields before MapRow ever calls mapField, so this branch is otherwise
// unreachable through the public API; it exists purely so a future change
// to that filtering can't accidentally introduce a reflect panic here.
func TestRowMapper_MapField_UnsettableFieldSkipped(t *testing.T) {
	var dest unexportedFieldStruct
	mapper := NewRowMapper([]string{"hidden"})
	tag := TagOptions{Name: "hidden", Index: []int{1}} // index of the unexported "hidden" field

	elem := reflect.ValueOf(&dest).Elem()
	if err := mapper.mapField([]any{"value"}, elem, &tag); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRowMapper_MapRow_FlattensEmbeddedStruct verifies MapRow writes
// into fields reached through a value-embedded anonymous struct, using
// the multi-element Index path GetFieldTags produces for them.
func TestRowMapper_MapRow_FlattensEmbeddedStruct(t *testing.T) {
	type Contact struct {
		Email string `excel:"email"`
	}
	type Person struct {
		Name string `excel:"name"`
		Contact
	}

	mapper := NewRowMapper([]string{"name", "email"})
	var dest Person
	if err := mapper.MapRow([]any{"Alice", "alice@example.com"}, &dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Name != "Alice" || dest.Email != "alice@example.com" {
		t.Errorf("dest = %+v, want Name=Alice Email=alice@example.com", dest)
	}
}

// TestRowMapper_MapRow_EmbeddedPointerStruct_Preallocated verifies
// MapRow writes into a pointer-embedded anonymous struct when the
// caller has already allocated it, the same precondition Go itself
// requires to read or write such a field directly.
func TestRowMapper_MapRow_EmbeddedPointerStruct_Preallocated(t *testing.T) {
	type Contact struct {
		Email string `excel:"email"`
	}
	type Person struct {
		Name string `excel:"name"`
		*Contact
	}

	dest := Person{Contact: &Contact{}}
	mapper := NewRowMapper([]string{"name", "email"})
	if err := mapper.MapRow([]any{"Alice", "alice@example.com"}, &dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Name != "Alice" || dest.Email != "alice@example.com" {
		t.Errorf("dest = %+v, want Name=Alice Email=alice@example.com", dest)
	}
}

// TestRowMapper_MapRow_EmbeddedNilPointer_ReturnsError verifies a nil
// pointer-embedded struct surfaces as an error from MapRow instead of
// panicking, since FieldByIndexErr cannot step through it to reach the
// destination field.
func TestRowMapper_MapRow_EmbeddedNilPointer_ReturnsError(t *testing.T) {
	type Contact struct {
		Email string `excel:"email"`
	}
	type Person struct {
		Name string `excel:"name"`
		*Contact
	}

	var dest Person // Contact left nil
	mapper := NewRowMapper([]string{"name", "email"})
	if err := mapper.MapRow([]any{"Alice", "alice@example.com"}, &dest); err == nil {
		t.Error("expected error for nil embedded pointer, got nil")
	}
}

// TestRowMapper_SetConverters_OverridesConvertToType verifies a
// registered Converter runs for its column instead of ConvertToType,
// letting MapRow populate a field type ConvertToType cannot produce on
// its own (here, a []string split from a delimited cell value).
func TestRowMapper_SetConverters_OverridesConvertToType(t *testing.T) {
	type Contact struct {
		Name string   `excel:"name"`
		Tags []string `excel:"tags"`
	}

	mapper := NewRowMapper([]string{"name", "tags"})
	mapper.SetConverters(map[string]Converter{
		"tags": func(raw string, _ reflect.Type) (any, error) {
			return strings.Split(raw, ";"), nil
		},
	})

	var dest Contact
	if err := mapper.MapRow([]any{"Alice", "a;b;c"}, &dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Name != "Alice" {
		t.Errorf("Name = %q, want Alice", dest.Name)
	}
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(dest.Tags, want) {
		t.Errorf("Tags = %v, want %v", dest.Tags, want)
	}
}

// TestRowMapper_SetConverters_ColumnWithoutEntryUsesConvertToType
// verifies a column with no registered Converter still goes through
// ConvertToType, so SetConverters only affects the columns it names.
func TestRowMapper_SetConverters_ColumnWithoutEntryUsesConvertToType(t *testing.T) {
	type Numbered struct {
		Age int `excel:"age"`
	}

	mapper := NewRowMapper([]string{"age"})
	mapper.SetConverters(map[string]Converter{
		"other": func(_ string, _ reflect.Type) (any, error) {
			t.Fatal("converter for unrelated column should not run")
			return nil, errors.New("unreachable")
		},
	})

	var dest Numbered
	if err := mapper.MapRow([]any{"42"}, &dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest.Age != 42 {
		t.Errorf("Age = %d, want 42", dest.Age)
	}
}

// TestRowMapper_SetConverters_PropagatesConverterError verifies an
// error returned by a Converter surfaces from MapRow as a
// ValidationError, the same as a ConvertToType failure would.
func TestRowMapper_SetConverters_PropagatesConverterError(t *testing.T) {
	type Numbered struct {
		Age int `excel:"age"`
	}

	mapper := NewRowMapper([]string{"age"})
	mapper.SetConverters(map[string]Converter{
		"age": func(_ string, _ reflect.Type) (any, error) {
			return nil, errors.New("boom")
		},
	})

	var dest Numbered
	if err := mapper.MapRow([]any{"42"}, &dest); err == nil {
		t.Error("expected error from failing converter, got nil")
	}
}

// TestSetFieldValue_CrossWidthNumericKinds exercises the isSignedInt/
// isUnsignedInt/isFloat branches directly. MapRow never produces a
// field/value kind mismatch within the same numeric family (ConvertToType
// always returns the field's own type), so these branches are only
// reachable by calling setFieldValue with mismatched widths directly.
func TestSetFieldValue_CrossWidthNumericKinds(t *testing.T) {
	t.Run("uint8 field from uint64 value", func(t *testing.T) {
		var dest struct{ N uint8 }
		field := reflect.ValueOf(&dest).Elem().Field(0)
		setFieldValue(field, uint64(200))
		if dest.N != 200 {
			t.Errorf("N = %d, want 200", dest.N)
		}
	})

	t.Run("float32 field from float64 value", func(t *testing.T) {
		var dest struct{ F float32 }
		field := reflect.ValueOf(&dest).Elem().Field(0)
		setFieldValue(field, float64(1.5))
		if dest.F != 1.5 {
			t.Errorf("F = %v, want 1.5", dest.F)
		}
	})

	t.Run("int8 field from int64 value", func(t *testing.T) {
		var dest struct{ N int8 }
		field := reflect.ValueOf(&dest).Elem().Field(0)
		setFieldValue(field, int64(-5))
		if dest.N != -5 {
			t.Errorf("N = %d, want -5", dest.N)
		}
	})

	t.Run("incompatible kinds left untouched", func(t *testing.T) {
		var dest struct{ N int }
		field := reflect.ValueOf(&dest).Elem().Field(0)
		setFieldValue(field, "not a number")
		if dest.N != 0 {
			t.Errorf("N = %d, want 0 (untouched)", dest.N)
		}
	})

	t.Run("nil value is a no-op", func(t *testing.T) {
		var dest struct{ N int }
		field := reflect.ValueOf(&dest).Elem().Field(0)
		setFieldValue(field, nil)
		if dest.N != 0 {
			t.Errorf("N = %d, want 0 (untouched)", dest.N)
		}
	})
}
