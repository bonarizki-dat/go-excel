package importer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/bonarizki-dat/go-excel/excel"
)

type TestUser struct {
	ID        int       `excel:"id"`
	Name      string    `excel:"name"`
	Email     string    `excel:"email"`
	Active    bool      `excel:"active"`
	Score     float64   `excel:"score"`
	CreatedAt time.Time `excel:"created_at"`
}

func TestImportToStructs(t *testing.T) {
	// Create a temporary Excel file
	f := excelize.NewFile()
	sheet := "Sheet1"

	// Write headers
	headers := []string{"id", "name", "email", "active", "score", "created_at"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Write data
	now := time.Now().Truncate(time.Second) // Truncate for comparison
	data := []TestUser{
		{1, "Alice", "alice@example.com", true, 95.5, now},
		{2, "Bob", "bob@example.com", false, 80.0, now.Add(-24 * time.Hour)},
	}

	for i, u := range data {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), u.ID)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), u.Name)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), u.Email)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), u.Active)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), u.Score)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), u.CreatedAt)
	}

	tmpFile := "test_struct_import.xlsx"
	if err := f.SaveAs(tmpFile); err != nil {
		t.Fatalf("Failed to save temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	// Import
	ctx := context.Background()
	users, err := ImportToStructs[TestUser](ctx, tmpFile)
	if err != nil {
		t.Fatalf("ImportToStructs failed: %v", err)
	}

	// Verify
	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}

	if users[0].Name != "Alice" {
		t.Errorf("User 1 name mismatch: got %s, want Alice", users[0].Name)
	}
	if !users[0].Active {
		t.Errorf("User 1 active mismatch: got false, want true")
	}
	if users[1].Score != 80.0 {
		t.Errorf("User 2 score mismatch: got %f, want 80.0", users[1].Score)
	}

	// CreatedAt is intentionally not asserted here: excelize round-trips
	// time.Time through its own date serial format, so equality would
	// depend on timezone and sub-second precision that this test does
	// not otherwise control for.
}

// TestImportToStructs_ValidationOption verifies that ImportToStructs
// forwards options such as WithValidation through to the underlying
// RowMapper, and that a row violating a min: bound is rejected only when
// validation is enabled.
func TestImportToStructs_ValidationOption(t *testing.T) {
	type Person struct {
		Name string `excel:"name"`
		Age  int    `excel:"age,min:18"`
	}

	buildFile := func(t *testing.T) string {
		t.Helper()
		f := excelize.NewFile()
		f.SetCellValue("Sheet1", "A1", "name")
		f.SetCellValue("Sheet1", "B1", "age")
		f.SetCellValue("Sheet1", "A2", "Minor")
		f.SetCellValue("Sheet1", "B2", 10)

		path := filepath.Join(t.TempDir(), "people.xlsx")
		if err := f.SaveAs(path); err != nil {
			t.Fatalf("SaveAs: %v", err)
		}
		return path
	}

	t.Run("rejected when validation enabled", func(t *testing.T) {
		path := buildFile(t)
		if _, err := ImportToStructs[Person](context.Background(), path, excel.WithValidation(true)); err == nil {
			t.Error("expected error for under-age row when validation is enabled, got nil")
		}
	})

	t.Run("allowed when validation left at default", func(t *testing.T) {
		path := buildFile(t)
		people, err := ImportToStructs[Person](context.Background(), path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(people) != 1 || people[0].Age != 10 {
			t.Fatalf("unexpected result: %+v", people)
		}
	})
}

// TestImportToStructs_ConvertersOption verifies ImportToStructs forwards
// WithConverters through to the underlying RowMapper, letting a column
// populate a field type ConvertToType does not natively support (here,
// a []string split from a semicolon-delimited cell value).
func TestImportToStructs_ConvertersOption(t *testing.T) {
	// Named TaggedPerson, not Person: GetFieldTags' tagCache keys on
	// reflect.Type.String(), which for a named local type is just
	// "importer.Person" regardless of which test function declared
	// it, so reusing "Person" here would collide with the unrelated
	// local Person type in TestImportToStructs_ValidationOption.
	type TaggedPerson struct {
		Name string   `excel:"name"`
		Tags []string `excel:"tags"`
	}

	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "name")
	f.SetCellValue("Sheet1", "B1", "tags")
	f.SetCellValue("Sheet1", "A2", "Alice")
	f.SetCellValue("Sheet1", "B2", "admin;editor")

	path := filepath.Join(t.TempDir(), "people.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	converters := map[string]excel.Converter{
		"tags": func(raw string, _ reflect.Type) (any, error) {
			return strings.Split(raw, ";"), nil
		},
	}

	people, err := ImportToStructs[TaggedPerson](context.Background(), path, excel.WithConverters(converters))
	if err != nil {
		t.Fatalf("ImportToStructs: %v", err)
	}
	if len(people) != 1 {
		t.Fatalf("len(people) = %d, want 1", len(people))
	}

	want := []string{"admin", "editor"}
	if !reflect.DeepEqual(people[0].Tags, want) {
		t.Errorf("Tags = %v, want %v", people[0].Tags, want)
	}
}
