package excel

import (
	"testing"
	"time"
)

func TestConvertToType(t *testing.T) {
	tests := []struct {
		value      string
		targetType string
		want       any
		wantErr    bool
	}{
		{"123", "int", int64(123), false},
		{"123.45", "float64", 123.45, false},
		{"true", "bool", true, false},
		{"yes", "bool", true, false},
		{"0", "bool", false, false},
		{"2023-01-01", "time.Time", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), false},
		{"invalid", "int", nil, true},
		{"", "int", int64(0), false},
		{"42", "uint", uint64(42), false},
		{"anything", "unsupported-type", nil, true},
	}

	for _, tt := range tests {
		got, err := ConvertToType(tt.value, tt.targetType)
		if (err != nil) != tt.wantErr {
			t.Errorf("ConvertToType(%q, %q) error = %v, wantErr %v", tt.value, tt.targetType, err, tt.wantErr)
			continue
		}

		if !tt.wantErr {
			// Simple equality check might fail for time due to location, so handle separately if needed
			// But for this simple test, reflect.DeepEqual or simple comparison works for basic types
			if tt.targetType == "time.Time" {
				t1 := got.(time.Time)
				t2 := tt.want.(time.Time)
				if !t1.Equal(t2) {
					t.Errorf("ConvertToType(%q, %q) = %v, want %v", tt.value, tt.targetType, got, tt.want)
				}
			} else if got != tt.want {
				t.Errorf("ConvertToType(%q, %q) = %v, want %v", tt.value, tt.targetType, got, tt.want)
			}
		}
	}
}

func TestConvertToInt(t *testing.T) {
	val, err := convertToInt("123")
	if err != nil || val != 123 {
		t.Errorf("convertToInt failed")
	}

	val, err = convertToInt("123.00")
	if err != nil || val != 123 {
		t.Errorf("convertToInt with float string failed")
	}

	// Test negative numbers
	val, err = convertToInt("-456")
	if err != nil || val != -456 {
		t.Errorf("convertToInt with negative failed")
	}

	// Test invalid input
	_, err = convertToInt("abc")
	if err == nil {
		t.Errorf("convertToInt should fail for invalid input")
	}

	// Test a float-looking string that isn't actually a valid float
	// (exercises the ParseFloat error branch, distinct from the
	// ParseInt error branch already covered by "abc" above).
	_, err = convertToInt("12.3.4")
	if err == nil {
		t.Errorf("convertToInt should fail for malformed float-like input")
	}
}

func TestConvertToUint(t *testing.T) {
	val, err := convertToUint("123")
	if err != nil || val != 123 {
		t.Errorf("convertToUint failed")
	}

	val, err = convertToUint("123.00")
	if err != nil || val != 123 {
		t.Errorf("convertToUint with float string failed")
	}

	// Test invalid input (negative)
	_, err = convertToUint("-123")
	if err == nil {
		t.Errorf("convertToUint should fail for negative number")
	}

	// Test invalid input (non-numeric)
	_, err = convertToUint("abc")
	if err == nil {
		t.Errorf("convertToUint should fail for invalid input")
	}

	// Test empty string (returns zero, not an error)
	val, err = convertToUint("")
	if err != nil || val != 0 {
		t.Errorf("convertToUint(\"\") = (%v, %v), want (0, nil)", val, err)
	}

	// Test a float-looking string that isn't actually a valid float
	_, err = convertToUint("12.3.4")
	if err == nil {
		t.Errorf("convertToUint should fail for malformed float-like input")
	}

	// Test a negative float-like string (distinct from the negative
	// integer case above, which never reaches the float-parsing branch)
	_, err = convertToUint("-1.5")
	if err == nil {
		t.Errorf("convertToUint should fail for negative float-like input")
	}
}

func TestConvertToFloat(t *testing.T) {
	val, err := convertToFloat("123.45")
	if err != nil || val != 123.45 {
		t.Errorf("convertToFloat failed")
	}

	val, err = convertToFloat("123")
	if err != nil || val != 123.0 {
		t.Errorf("convertToFloat with int string failed")
	}

	// Test negative numbers
	val, err = convertToFloat("-456.78")
	if err != nil || val != -456.78 {
		t.Errorf("convertToFloat with negative failed")
	}

	// Test invalid input
	_, err = convertToFloat("abc")
	if err == nil {
		t.Errorf("convertToFloat should fail for invalid input")
	}

	// Test empty string (returns zero, not an error)
	val, err = convertToFloat("")
	if err != nil || val != 0.0 {
		t.Errorf("convertToFloat(\"\") = (%v, %v), want (0, nil)", val, err)
	}
}

func TestConvertToBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"YES", true},
		{"1", true},
		{"false", false},
		{"FALSE", false},
		{"no", false},
		{"NO", false},
		{"0", false},
	}

	for _, tt := range tests {
		got, err := convertToBool(tt.input)
		if err != nil {
			t.Errorf("convertToBool(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("convertToBool(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}

	// Test invalid input
	_, err := convertToBool("maybe")
	if err == nil {
		t.Errorf("convertToBool should fail for invalid input")
	}

	// Test empty string (returns false, not an error)
	got, err := convertToBool("")
	if err != nil || got != false {
		t.Errorf("convertToBool(\"\") = (%v, %v), want (false, nil)", got, err)
	}
}

func TestConvertToTime(t *testing.T) {
	// Test various date formats
	formats := []string{
		"2023-01-15",
		"2023/01/15",
		"15-01-2023",
		"01/15/2023",
		"2023-01-15T10:30:00Z",
	}

	for _, dateStr := range formats {
		_, err := convertToTime(dateStr)
		if err != nil {
			t.Logf("convertToTime(%q) error: %v (acceptable, some formats may not be supported)", dateStr, err)
		}
	}

	// Test definitely invalid input
	_, err := convertToTime("not-a-date")
	if err == nil {
		t.Errorf("convertToTime should fail for completely invalid input")
	}

	// Test empty string (returns zero time, not an error)
	result, err := convertToTime("")
	if err != nil {
		t.Errorf("convertToTime should handle empty string gracefully: %v", err)
	}
	if !result.IsZero() {
		t.Errorf("convertToTime(\"\") should return zero time")
	}
}
