package stream

import (
	"runtime"
	"testing"
)

func TestWithImportWorkers(t *testing.T) {
	testData := [][]any{{1, "data"}}

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
			expected: runtime.NumCPU(),
		},
		{
			name:     "negative workers ignored",
			workers:  -5,
			expected: runtime.NumCPU(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := createImporterTestFile(testData)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
			importer, err := NewStreamImporter(buf, WithImportWorkers(tt.workers))
			if err != nil {
				t.Fatalf("NewStreamImporter failed: %v", err)
			}
			defer importer.Close()

			if importer.numWorkers != tt.expected {
				t.Errorf("Expected %d workers, got %d", tt.expected, importer.numWorkers)
			}
		})
	}
}

// TestGetHeadings tests GetHeadings method.
func TestGetHeadings(t *testing.T) {
	// Create test file with headings
	testData := [][]any{
		{"Col1", "Col2", "Col3"},
		{1, 2, 3},
	}
	buf, err := createImporterTestFile(testData)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create importer with headings enabled
	importer, err := NewStreamImporter(buf, WithImportHeadings())
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Get headings
	headings := importer.GetHeadings()

	// Verify headings
	if len(headings) != 3 {
		t.Errorf("Expected 3 headings, got %d", len(headings))
	}

	expectedHeadings := []string{"Col1", "Col2", "Col3"}
	for i, expected := range expectedHeadings {
		if headings[i] != expected {
			t.Errorf("Heading[%d]: expected '%s', got '%s'", i, expected, headings[i])
		}
	}
}

// TestGetHeadings_ReturnsIndependentCopy verifies that mutating the
// slice returned by GetHeadings does not affect a later call.
func TestGetHeadings_ReturnsIndependentCopy(t *testing.T) {
	testData := [][]any{
		{"Col1", "Col2", "Col3"},
		{1, 2, 3},
	}
	buf, err := createImporterTestFile(testData)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	importer, err := NewStreamImporter(buf, WithImportHeadings())
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	first := importer.GetHeadings()
	first[0] = "Mutated"

	second := importer.GetHeadings()
	if second[0] != "Col1" {
		t.Errorf("GetHeadings() second call returned %q, expected \"Col1\" to be unaffected by mutating the first call's result", second[0])
	}
}

// TestImporter_GetRowCount tests GetRowCount method.
func TestImporter_GetRowCount(t *testing.T) {
	testData := [][]any{{1, "data"}}
	buf, _ := createImporterTestFile(testData)

	importer, err := NewStreamImporter(buf)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Initial count should be 0
	if importer.GetRowCount() != 0 {
		t.Errorf("Expected initial count 0, got %d", importer.GetRowCount())
	}
}

// TestImporter_GetStats tests GetStats method.
func TestImporter_GetStats(t *testing.T) {
	testData := [][]any{{1, "data"}}
	buf, _ := createImporterTestFile(testData)

	importer, err := NewStreamImporter(buf)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Get stats before import
	stats := importer.GetStats()
	if stats.TasksSubmitted != 0 {
		t.Errorf("Expected 0 tasks submitted before import, got %d", stats.TasksSubmitted)
	}
}
