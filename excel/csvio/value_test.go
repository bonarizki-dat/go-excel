package csvio

import (
	"testing"
	"time"
)

type stringerValue struct{ s string }

func (v stringerValue) String() string { return v.s }

func TestFormatValue_VariousTypes_FormatsCorrectly(t *testing.T) {
	fixedTime := time.Date(2024, 3, 15, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		value  any
		layout string
		want   string
	}{
		{name: "nil", value: nil, want: ""},
		{name: "string", value: "hello", want: "hello"},
		{name: "empty string", value: "", want: ""},
		{name: "bytes", value: []byte("bytes"), want: "bytes"},
		{name: "bool true", value: true, want: "true"},
		{name: "bool false", value: false, want: "false"},
		{name: "int", value: 42, want: "42"},
		{name: "negative int", value: -7, want: "-7"},
		{name: "int8", value: int8(-128), want: "-128"},
		{name: "int16", value: int16(-32768), want: "-32768"},
		{name: "int32", value: int32(2147483647), want: "2147483647"},
		{name: "int64", value: int64(9223372036854775807), want: "9223372036854775807"},
		{name: "uint", value: uint(42), want: "42"},
		{name: "uint8", value: uint8(255), want: "255"},
		{name: "uint16", value: uint16(65535), want: "65535"},
		{name: "uint32", value: uint32(4294967295), want: "4294967295"},
		{name: "uint64", value: uint64(18446744073709551615), want: "18446744073709551615"},
		{name: "float64 integral", value: float64(1000000), want: "1000000"},
		{name: "float64 fractional", value: 3.14159, want: "3.14159"},
		{name: "float64 large no scientific notation", value: 1234567.89, want: "1234567.89"},
		{name: "float32", value: float32(2.5), want: "2.5"},
		{name: "time.Time default layout", value: fixedTime, want: "2024-03-15T09:30:00Z"},
		{name: "time.Time custom layout", value: fixedTime, layout: "2006-01-02 15:04:05", want: "2024-03-15 09:30:00"},
		{name: "stringer", value: stringerValue{s: "custom"}, want: "custom"},
		{name: "fallback via Sprintf", value: []int{1, 2, 3}, want: "[1 2 3]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatValue(tt.value, tt.layout)
			if got != tt.want {
				t.Errorf("FormatValue(%#v, %q) = %q, want %q", tt.value, tt.layout, got, tt.want)
			}
		})
	}
}
