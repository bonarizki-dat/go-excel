package exporter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// TestExporter_ExportToFile_Success tests successful file export.
func TestExporter_ExportToFile_Success(t *testing.T) {
	ctx := context.Background()

	// Prepare test data
	data := [][]any{
		{"Name", "Age", "City"},
		{"Alice", 30, "New York"},
		{"Bob", 25, "Los Angeles"},
	}

	// Create temporary file path
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_export.xlsx")

	// Create exporter and export to file
	exporter := exporter.NewExporter(data)
	err := exporter.ExportToFile(ctx, filename)
	if err != nil {
		t.Fatalf("ExportToFile failed: %v", err)
	}

	// Verify file was created
	if _, statErr := os.Stat(filename); os.IsNotExist(statErr) {
		t.Error("Exported file should exist")
	}

	// Verify file is not empty
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Exported file should not be empty")
	}
}

// TestExporter_ExportToFile_EmptyFilename tests export with empty filename.
func TestExporter_ExportToFile_EmptyFilename(t *testing.T) {
	ctx := context.Background()
	data := [][]any{{"test"}}

	exporter := exporter.NewExporter(data)
	err := exporter.ExportToFile(ctx, "")
	if err == nil {
		t.Error("ExportToFile should fail with empty filename")
	}

	// Verify it's an excel.ExportError
	var exportErr *excel.ExportError
	if !errors.As(err, &exportErr) {
		t.Error("Error should be *excel.ExportError")
	}
}

// TestExporter_ExportToFile_ContextCancellation tests context cancellation.
func TestExporter_ExportToFile_ContextCancellation(t *testing.T) {
	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create large dataset to ensure context is checked
	data := make([][]any, 1000)
	for i := range data {
		data[i] = []any{i, "test", "data"}
	}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_cancelled.xlsx")

	exporter := exporter.NewExporter(data)
	err := exporter.ExportToFile(ctx, filename)
	if err == nil {
		t.Error("ExportToFile should fail with canceled context")
	}

	// Error should be or wrap context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error should be or wrap context.Canceled, got %v", err)
	}
}

// TestExporter_ExportToFile_WithSubdirectory tests export to subdirectory.
func TestExporter_ExportToFile_WithSubdirectory(t *testing.T) {
	ctx := context.Background()
	data := [][]any{{"test"}}

	// Create path with subdirectory that doesn't exist
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "subdir", "nested", "test.xlsx")

	exporter := exporter.NewExporter(data)
	err := exporter.ExportToFile(ctx, filename)
	if err != nil {
		t.Fatalf("ExportToFile should create subdirectories: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("File should be created in subdirectory")
	}
}

// TestExporter_ExportToFile_WithOptions tests export with various options.
func TestExporter_ExportToFile_WithOptions(t *testing.T) {
	ctx := context.Background()

	data := [][]any{
		{"ID", "Name", "Email"},
		{1, "Alice", "alice@example.com"},
		{2, "Bob", "bob@example.com"},
	}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_options.xlsx")

	// Create exporter with options
	exporter := exporter.NewExporter(data,
		excel.WithSheetName("Users"),
		excel.WithHeadings(true),
	)

	err := exporter.ExportToFile(ctx, filename)
	if err != nil {
		t.Fatalf("ExportToFile failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("Exported file should exist")
	}
}
