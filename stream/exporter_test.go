package stream

import (
	"bytes"
	"runtime"
	"sync/atomic"
	"testing"
)

// TestNewStreamExporter tests StreamExporter creation with defaults.
func TestNewStreamExporter(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter with defaults
	exporter, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Verify exporter was created
	if exporter == nil {
		t.Fatal("NewStreamExporter returned nil")
	}

	// Verify default values
	if exporter.writer != &buf {
		t.Error("Writer not set correctly")
	}

	if exporter.sheetName != "Sheet1" {
		t.Errorf("Expected default sheet name 'Sheet1', got '%s'", exporter.sheetName)
	}

	if exporter.numWorkers != runtime.NumCPU() {
		t.Errorf("Expected default workers %d, got %d", runtime.NumCPU(), exporter.numWorkers)
	}

	if exporter.progressInterval != 100 {
		t.Errorf("Expected default progress interval 100, got %d", exporter.progressInterval)
	}

	if exporter.file == nil {
		t.Error("Excelize file not initialized")
	}

	if exporter.withHeadings {
		t.Error("Headings should be disabled by default")
	}
}

// TestNewStreamExporter_WithOptions tests StreamExporter creation with options.
func TestNewStreamExporter_WithOptions(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Track progress callback invocations
	var progressCalled atomic.Uint64

	// Create exporter with all options
	exporter, err := NewStreamExporter(&buf,
		WithExportWorkers(4),
		WithExportHeadings([]string{"ID", "Name", "Email"}),
		WithExportSheetName("Users"),
		WithExportProgress(func(_ uint64, _ string) {
			progressCalled.Add(1)
		}, 50),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Verify options applied
	if exporter.numWorkers != 4 {
		t.Errorf("Expected 4 workers, got %d", exporter.numWorkers)
	}

	if !exporter.withHeadings {
		t.Error("Headings should be enabled")
	}

	if len(exporter.headings) != 3 {
		t.Errorf("Expected 3 headings, got %d", len(exporter.headings))
	}

	if exporter.headings[0] != "ID" {
		t.Errorf("Expected heading 'ID', got '%s'", exporter.headings[0])
	}

	if exporter.sheetName != "Users" {
		t.Errorf("Expected sheet name 'Users', got '%s'", exporter.sheetName)
	}

	if exporter.progressCallback == nil {
		t.Error("Progress callback not set")
	}

	if exporter.progressInterval != 50 {
		t.Errorf("Expected progress interval 50, got %d", exporter.progressInterval)
	}
}

// TestWithExportWorkers tests WithExportWorkers option.
func TestWithExportWorkers(t *testing.T) {
	tests := []struct {
		name     string
		workers  int
		expected int
	}{
		{
			name:     "valid positive value",
			workers:  8,
			expected: 8,
		},
		{
			name:     "zero workers ignored",
			workers:  0,
			expected: runtime.NumCPU(), // Should keep default
		},
		{
			name:     "negative workers ignored",
			workers:  -5,
			expected: runtime.NumCPU(), // Should keep default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			exporter, err := NewStreamExporter(&buf, WithExportWorkers(tt.workers))
			if err != nil {
				t.Fatalf("NewStreamExporter: %v", err)
			}

			if exporter.numWorkers != tt.expected {
				t.Errorf("Expected %d workers, got %d", tt.expected, exporter.numWorkers)
			}
		})
	}
}

// TestWithExportHeadings tests WithExportHeadings option.
func TestWithExportHeadings(t *testing.T) {
	var buf bytes.Buffer

	// Test with headings
	headings := []string{"Col1", "Col2", "Col3"}
	exporter, err := NewStreamExporter(&buf, WithExportHeadings(headings))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Verify headings enabled
	if !exporter.withHeadings {
		t.Error("Headings should be enabled")
	}

	// Verify headings stored
	if len(exporter.headings) != 3 {
		t.Errorf("Expected 3 headings, got %d", len(exporter.headings))
	}

	for i, heading := range headings {
		if exporter.headings[i] != heading {
			t.Errorf("Heading[%d]: expected '%s', got '%s'", i, heading, exporter.headings[i])
		}
	}
}

// TestWithExportSheetName tests WithExportSheetName option.
func TestWithExportSheetName(t *testing.T) {
	tests := []struct {
		name      string
		sheetName string
		expected  string
	}{
		{
			name:      "valid sheet name",
			sheetName: "CustomSheet",
			expected:  "CustomSheet",
		},
		{
			name:      "empty sheet name ignored",
			sheetName: "",
			expected:  "Sheet1", // Should keep default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			exporter, err := NewStreamExporter(&buf, WithExportSheetName(tt.sheetName))
			if err != nil {
				t.Fatalf("NewStreamExporter: %v", err)
			}

			if exporter.sheetName != tt.expected {
				t.Errorf("Expected sheet name '%s', got '%s'", tt.expected, exporter.sheetName)
			}
		})
	}
}

// TestWithExportProgress tests WithExportProgress option.
func TestWithExportProgress(t *testing.T) {
	var buf bytes.Buffer

	// Test with valid interval
	callback := func(_ uint64, _ string) {
		// Progress callback
	}

	exporter, err := NewStreamExporter(&buf, WithExportProgress(callback, 25))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Verify callback set
	if exporter.progressCallback == nil {
		t.Error("Progress callback not set")
	}

	// Verify interval set
	if exporter.progressInterval != 25 {
		t.Errorf("Expected interval 25, got %d", exporter.progressInterval)
	}

	// Test with zero interval (should keep default)
	exporter2, err := NewStreamExporter(&buf, WithExportProgress(callback, 0))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	if exporter2.progressInterval != 100 {
		t.Errorf("Expected default interval 100, got %d", exporter2.progressInterval)
	}
}

// TestGetRowCount tests row counter getter.
func TestGetRowCount(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter
	exporter, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Initial count should be 0
	if exporter.GetRowCount() != 0 {
		t.Errorf("Expected initial count 0, got %d", exporter.GetRowCount())
	}
}

// TestGetStats tests worker pool statistics getter.
func TestGetStats(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter
	exporter, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Get stats before export (should return empty stats)
	stats := exporter.GetStats()
	if stats.TasksSubmitted != 0 {
		t.Errorf("Expected 0 tasks submitted before export, got %d", stats.TasksSubmitted)
	}

	// Verify stats structure is valid
	if stats.TotalWorkers != 0 {
		t.Errorf("Expected 0 total workers before export, got %d", stats.TotalWorkers)
	}
}
