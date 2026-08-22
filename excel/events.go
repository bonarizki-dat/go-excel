package excel

import "context"

// EventType represents the type of event that occurred.
type EventType int

const (
	// BeforeExport fires once before an export operation starts writing
	// any rows.
	BeforeExport EventType = iota
	// AfterExport fires once after an export operation has finished
	// writing all rows.
	AfterExport
	// BeforeSheetExport fires before a sheet's rows are written in a
	// multi-sheet export.
	BeforeSheetExport
	// AfterSheetExport fires after a sheet's rows have been written in a
	// multi-sheet export.
	AfterSheetExport
	// BeforeRowExport fires before each row is written.
	BeforeRowExport
	// AfterRowExport fires after each row is written.
	AfterRowExport

	// BeforeImport fires once before an import operation starts reading
	// any rows.
	BeforeImport
	// AfterImport fires once after an import operation has finished
	// reading all rows.
	AfterImport
	// BeforeSheetImport fires before a sheet's rows are read in a
	// multi-sheet import.
	BeforeSheetImport
	// AfterSheetImport fires after a sheet's rows have been read in a
	// multi-sheet import.
	AfterSheetImport
	// BeforeRowImport fires before each row is read.
	BeforeRowImport
	// AfterRowImport fires after each row is read.
	AfterRowImport

	// OnError fires whenever an export or import operation encounters an
	// error, regardless of which stage produced it.
	OnError
)

// String returns the string representation of the EventType.
func (e EventType) String() string {
	switch e {
	case BeforeExport:
		return "BeforeExport"
	case AfterExport:
		return "AfterExport"
	case BeforeSheetExport:
		return "BeforeSheetExport"
	case AfterSheetExport:
		return "AfterSheetExport"
	case BeforeRowExport:
		return "BeforeRowExport"
	case AfterRowExport:
		return "AfterRowExport"
	case BeforeImport:
		return "BeforeImport"
	case AfterImport:
		return "AfterImport"
	case BeforeSheetImport:
		return "BeforeSheetImport"
	case AfterSheetImport:
		return "AfterSheetImport"
	case BeforeRowImport:
		return "BeforeRowImport"
	case AfterRowImport:
		return "AfterRowImport"
	case OnError:
		return "OnError"
	default:
		return "Unknown"
	}
}

// Event holds the context and data for an event.
type Event struct {
	Type  EventType
	Data  any   // The data associated with the event (e.g. row data, sheet name)
	Error error // The error associated with the event (for OnError)
}

// Hook is a function that is called when an event is triggered.
// It receives the context and the event details.
// If it returns an error, the operation may be stopped depending on configuration.
type Hook func(ctx context.Context, event *Event) error
