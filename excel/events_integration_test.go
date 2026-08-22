package excel_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

func TestEventSystem_ExportImport(t *testing.T) {
	// Prepare data
	data := [][]any{
		{"Name", "Age"},
		{"Alice", 30},
		{"Bob", 25},
	}

	// --- Export Test ---
	exp := exporter.NewExporter(data)

	eventsCalled := make(map[excel.EventType]int)

	exp.On(excel.BeforeExport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.BeforeExport]++
		return nil
	})
	exp.On(excel.AfterExport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.AfterExport]++
		return nil
	})
	exp.On(excel.BeforeRowExport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.BeforeRowExport]++
		return nil
	})
	exp.On(excel.AfterRowExport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.AfterRowExport]++
		return nil
	})

	ctx := context.Background()
	if err := exp.Export(ctx); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify export events
	if eventsCalled[excel.BeforeExport] != 1 {
		t.Errorf("Expected 1 BeforeExport, got %d", eventsCalled[excel.BeforeExport])
	}
	if eventsCalled[excel.AfterExport] != 1 {
		t.Errorf("Expected 1 AfterExport, got %d", eventsCalled[excel.AfterExport])
	}
	// 3 rows (1 header + 2 data)
	if eventsCalled[excel.BeforeRowExport] != 3 {
		t.Errorf("Expected 3 BeforeRowExport, got %d", eventsCalled[excel.BeforeRowExport])
	}
	if eventsCalled[excel.AfterRowExport] != 3 {
		t.Errorf("Expected 3 AfterRowExport, got %d", eventsCalled[excel.AfterRowExport])
	}

	// Save to file for import test
	filename := filepath.Join(t.TempDir(), "test_events.xlsx")
	if err := exp.GetFile().SaveAs(filename); err != nil {
		t.Fatalf("Failed to save file: %v", err)
	}

	// --- Import Test ---
	imp := importer.NewImporter()

	// Reset counters
	eventsCalled = make(map[excel.EventType]int)

	imp.On(excel.BeforeImport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.BeforeImport]++
		return nil
	})
	imp.On(excel.AfterImport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.AfterImport]++
		return nil
	})
	imp.On(excel.BeforeRowImport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.BeforeRowImport]++
		return nil
	})
	imp.On(excel.AfterRowImport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.AfterRowImport]++
		return nil
	})

	if err := imp.ImportFromFile(ctx, filename); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify import events
	if eventsCalled[excel.BeforeImport] != 1 {
		t.Errorf("Expected 1 BeforeImport, got %d", eventsCalled[excel.BeforeImport])
	}
	if eventsCalled[excel.AfterImport] != 1 {
		t.Errorf("Expected 1 AfterImport, got %d", eventsCalled[excel.AfterImport])
	}
	// The fixture has a header row plus 2 data rows; WithHeadings is true
	// by default, so BeforeRowImport/AfterRowImport fire once per data
	// row, not once per row in the file.
	if eventsCalled[excel.BeforeRowImport] != 2 {
		t.Errorf("Expected 2 BeforeRowImport, got %d", eventsCalled[excel.BeforeRowImport])
	}
	if eventsCalled[excel.AfterRowImport] != 2 {
		t.Errorf("Expected 2 AfterRowImport, got %d", eventsCalled[excel.AfterRowImport])
	}
}

func TestEventSystem_CancelExport(t *testing.T) {
	data := [][]any{{"A"}}
	exp := exporter.NewExporter(data)

	exp.On(excel.BeforeExport, func(_ context.Context, _ *excel.Event) error {
		return fmt.Errorf("canceled by user")
	})

	if err := exp.Export(context.Background()); err == nil {
		t.Error("Expected error, got nil")
	} else if err.Error() != "canceled by user" {
		t.Errorf("Expected 'cancelled by user', got '%v'", err)
	}
}
