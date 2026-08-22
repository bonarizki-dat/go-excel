package excel

import (
	"errors"
	"reflect"
	"testing"
)

type TestStruct struct {
	Name     string `excel:"name"`
	Age      int    `excel:"age,required,min:18"`
	Email    string `excel:"email,omitempty,pattern:^.+$"`
	Ignored  string `excel:"-"`
	NoTag    string
	Implicit string // Should use field name "Implicit"
}

func TestParseTag(t *testing.T) {
	tests := []struct {
		tag      string
		expected TagOptions
	}{
		{
			tag:      "name",
			expected: TagOptions{Name: "name"},
		},
		{
			tag: "age,required,min:18",
			expected: TagOptions{
				Name:     "age",
				Required: true,
				Min:      intPtr(18),
			},
		},
		{
			tag: "email,omitempty",
			expected: TagOptions{
				Name:      "email",
				OmitEmpty: true,
			},
		},
		{
			tag:      "",
			expected: TagOptions{},
		},
	}

	for _, tt := range tests {
		got, err := ParseTag(tt.tag, []int{0})
		if err != nil {
			t.Fatalf("ParseTag(%q) returned unexpected error: %v", tt.tag, err)
		}
		if got.Name != tt.expected.Name {
			t.Errorf("ParseTag(%q).Name = %v, want %v", tt.tag, got.Name, tt.expected.Name)
		}
		if got.Required != tt.expected.Required {
			t.Errorf("ParseTag(%q).Required = %v, want %v", tt.tag, got.Required, tt.expected.Required)
		}
		if got.OmitEmpty != tt.expected.OmitEmpty {
			t.Errorf("ParseTag(%q).OmitEmpty = %v, want %v", tt.tag, got.OmitEmpty, tt.expected.OmitEmpty)
		}
		if !intPtrEqual(got.Min, tt.expected.Min) {
			t.Errorf("ParseTag(%q).Min = %v, want %v", tt.tag, got.Min, tt.expected.Min)
		}
	}
}

// TestParseTag_InvalidMinMax verifies that malformed min:/max: values in a
// struct tag return an error instead of silently parsing as 0, since a
// malformed tag is a struct definition bug and 0 is often itself a valid
// minimum/maximum, which would otherwise hide the mistake entirely.
func TestParseTag_InvalidMinMax(t *testing.T) {
	tests := []string{
		"age,min:abc",
		"age,max:1.5",
		"age,min:",
	}

	for _, tag := range tests {
		if _, err := ParseTag(tag, []int{0}); err == nil {
			t.Errorf("ParseTag(%q) expected error for malformed bound, got nil", tag)
		}
	}
}

func TestGetFieldTags(t *testing.T) {
	tags, err := GetFieldTags(reflect.TypeFor[TestStruct]())
	if err != nil {
		t.Fatalf("GetFieldTags failed: %v", err)
	}

	// Expected fields: Name, Age, Email, NoTag, Implicit
	// Ignored field should be skipped
	if len(tags) != 5 {
		t.Errorf("Expected 5 tags, got %d", len(tags))
	}

	m := make(map[string]TagOptions)
	for _, tag := range tags {
		m[tag.Name] = tag
	}

	if _, ok := m["name"]; !ok {
		t.Error("Missing tag for 'name'")
	}
	if _, ok := m["age"]; !ok {
		t.Error("Missing tag for 'age'")
	}
	if tag, ok := m["age"]; ok && !tag.Required {
		t.Error("Expected 'age' to be required")
	}
	if _, ok := m["NoTag"]; !ok {
		t.Error("Missing tag for 'NoTag'")
	}
}

// TestGetFieldTags_AcceptsPointerType verifies GetFieldTags dereferences a
// pointer-to-struct type before inspecting fields, since callers other than
// RowMapper.MapRow (which already passes elem.Type()) may hand it one.
func TestGetFieldTags_AcceptsPointerType(t *testing.T) {
	tags, err := GetFieldTags(reflect.TypeFor[*TestStruct]())
	if err != nil {
		t.Fatalf("GetFieldTags(*TestStruct) failed: %v", err)
	}
	if len(tags) != 5 {
		t.Errorf("len(tags) = %d, want 5", len(tags))
	}
}

// TestGetFieldTags_RejectsNonStruct verifies a non-struct, non-pointer type
// (once dereferenced) is rejected with a ConfigError rather than panicking.
func TestGetFieldTags_RejectsNonStruct(t *testing.T) {
	_, err := GetFieldTags(reflect.TypeFor[int]())
	if err == nil {
		t.Fatal("GetFieldTags(int) = nil error, want error")
	}
	var cfgErr *ConfigError
	if !errors.As(err, &cfgErr) {
		t.Errorf("GetFieldTags(int) error = %T, want *ConfigError", err)
	}
}

// unexportedFieldStruct has an unexported field, which GetFieldTags must
// skip instead of trying to read its (empty) excel tag.
type unexportedFieldStruct struct {
	Visible string `excel:"visible"`
	hidden  string //nolint:unused // exists only to exercise the unexported-field skip
}

func TestGetFieldTags_SkipsUnexportedFields(t *testing.T) {
	tags, err := GetFieldTags(reflect.TypeFor[unexportedFieldStruct]())
	if err != nil {
		t.Fatalf("GetFieldTags failed: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d, want 1 (unexported field should be skipped)", len(tags))
	}
	if tags[0].Name != "visible" {
		t.Errorf("tags[0].Name = %q, want %q", tags[0].Name, "visible")
	}
}

// invalidTagStruct has a malformed min: value, which should surface as an
// error from GetFieldTags rather than being silently parsed as min:0.
type invalidTagStruct struct {
	Age int `excel:"age,min:abc"`
}

func TestGetFieldTags_PropagatesParseTagError(t *testing.T) {
	_, err := GetFieldTags(reflect.TypeFor[invalidTagStruct]())
	if err == nil {
		t.Fatal("GetFieldTags with malformed tag = nil error, want error")
	}
}

// EmbeddedContact is used as both a value and pointer embed target
// below, to exercise both flattening shapes GetFieldTags supports. Its
// name must be exported: an anonymous field's export status follows
// its type name's capitalization, not the exported fields within it.
type EmbeddedContact struct {
	Email string `excel:"email"`
	Phone string `excel:"phone"`
}

type flattenedByValue struct {
	Name string `excel:"name"`
	EmbeddedContact
}

type flattenedByPointer struct {
	Name string `excel:"name"`
	*EmbeddedContact
}

// TestGetFieldTags_FlattensAnonymousStruct verifies fields reached
// through a value-embedded anonymous struct are flattened into the
// same result list as the parent's own fields, mirroring the export
// side's isEmbeddedStructField flattening.
func TestGetFieldTags_FlattensAnonymousStruct(t *testing.T) {
	tags, err := GetFieldTags(reflect.TypeFor[flattenedByValue]())
	if err != nil {
		t.Fatalf("GetFieldTags failed: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("len(tags) = %d, want 3 (name, email, phone)", len(tags))
	}

	byName := make(map[string]TagOptions, len(tags))
	for _, tag := range tags {
		byName[tag.Name] = tag
	}
	for _, name := range []string{"name", "email", "phone"} {
		if _, ok := byName[name]; !ok {
			t.Errorf("missing flattened tag for %q", name)
		}
	}
	if len(byName["email"].Index) != 2 {
		t.Errorf("email Index = %v, want a 2-element path through the embedded struct", byName["email"].Index)
	}
}

// TestGetFieldTags_FlattensAnonymousPointerStruct verifies the same
// flattening applies when the anonymous field is a pointer to a
// struct, not just a struct value.
func TestGetFieldTags_FlattensAnonymousPointerStruct(t *testing.T) {
	tags, err := GetFieldTags(reflect.TypeFor[flattenedByPointer]())
	if err != nil {
		t.Fatalf("GetFieldTags failed: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("len(tags) = %d, want 3 (name, email, phone)", len(tags))
	}
}

// TestGetFieldTags_AnonymousStructTypesDoNotShareCacheKey verifies the
// tagCache key distinguishes two different anonymous struct types.
// Both have an empty PkgPath and Name, so a cache key built from those
// two fields alone would collide and make the second call silently
// reuse the first call's tags.
func TestGetFieldTags_AnonymousStructTypesDoNotShareCacheKey(t *testing.T) {
	type1 := reflect.TypeOf(struct {
		A string `excel:"a"`
	}{})
	type2 := reflect.TypeOf(struct {
		B string `excel:"b"`
	}{})

	tags1, err := GetFieldTags(type1)
	if err != nil {
		t.Fatalf("GetFieldTags(type1) failed: %v", err)
	}
	tags2, err := GetFieldTags(type2)
	if err != nil {
		t.Fatalf("GetFieldTags(type2) failed: %v", err)
	}

	if len(tags1) != 1 || tags1[0].Name != "a" {
		t.Fatalf("tags1 = %+v, want a single tag named %q", tags1, "a")
	}
	if len(tags2) != 1 || tags2[0].Name != "b" {
		t.Fatalf("tags2 = %+v, want a single tag named %q, got the cache leaking type1's tags", tags2, "b")
	}
}

func intPtr(i int) *int {
	return &i
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
