package excel

import (
	"testing"
)

func TestEventType_String(t *testing.T) {
	tests := []struct {
		typ  EventType
		want string
	}{
		{BeforeExport, "BeforeExport"},
		{AfterExport, "AfterExport"},
		{BeforeSheetExport, "BeforeSheetExport"},
		{AfterSheetExport, "AfterSheetExport"},
		{BeforeRowExport, "BeforeRowExport"},
		{AfterRowExport, "AfterRowExport"},
		{BeforeImport, "BeforeImport"},
		{AfterImport, "AfterImport"},
		{BeforeSheetImport, "BeforeSheetImport"},
		{AfterSheetImport, "AfterSheetImport"},
		{BeforeRowImport, "BeforeRowImport"},
		{AfterRowImport, "AfterRowImport"},
		{OnError, "OnError"},
		{EventType(99), "Unknown"},
	}
	for _, tt := range tests {
		if got := tt.typ.String(); got != tt.want {
			t.Errorf("EventType(%d).String() = %q, want %q", tt.typ, got, tt.want)
		}
	}
}

// EventDispatcher's own tests moved to internal/events, which is
// where it now lives; see internal/events/dispatcher_test.go.
