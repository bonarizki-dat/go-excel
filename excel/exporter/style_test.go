package exporter

import (
	"context"
	"os"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExporter_Styling_HeaderAndData(t *testing.T) {
	data := [][]any{
		{"Name", "Age", "Salary"},
		{"John", 30, 50000.0},
		{"Jane", 25, 60000.0},
	}

	exp := NewExporter(data, excel.WithHeadings(true))

	// Set header style
	headerStyle := excel.NewStyle().
		Bold(true).
		BackgroundColor(excel.ColorBlue).
		FontColor(excel.ColorWhite)
	exp.SetHeaderStyle(headerStyle)

	// Set data style
	dataStyle := excel.NewStyle().
		BorderAll(excel.BorderThin)
	exp.SetDataStyle(dataStyle)

	ctx := context.Background()
	err := exp.Export(ctx)
	require.NoError(t, err)
	assert.NotNil(t, exp.GetFile())
}

func TestExporter_Styling_ColumnWidth(t *testing.T) {
	data := [][]any{
		{"Short", "Very Long Column Name"},
		{"A", "B"},
	}

	exp := NewExporter(data)
	exp.SetColumnWidth("A", 10.0)
	exp.SetColumnWidth("B", 30.0)

	ctx := context.Background()
	err := exp.Export(ctx)
	require.NoError(t, err)
	assert.NotNil(t, exp.GetFile())
}

func TestExporter_Styling_ColumnSpecific(t *testing.T) {
	data := [][]any{
		{"Name", "Salary"},
		{"John", 50000.0},
		{"Jane", 60000.0},
	}

	exp := NewExporter(data, excel.WithHeadings(true))

	// Style for salary column (B)
	salaryStyle := excel.NewStyle().
		Format(excel.FormatCurrency).
		FontColor(excel.ColorGreen)
	exp.SetColumnStyle("B", salaryStyle)

	ctx := context.Background()
	err := exp.Export(ctx)
	require.NoError(t, err)
	assert.NotNil(t, exp.GetFile())
}

// TestExporter_SetNumberFormat_NewColumnStyle verifies SetNumberFormat
// creates a column style when none was set for that column yet.
func TestExporter_SetNumberFormat_NewColumnStyle(t *testing.T) {
	exp := NewExporter(nil)
	exp.SetNumberFormat("B", excel.FormatCurrency)

	style, ok := exp.columnStyles["B"]
	require.True(t, ok, "SetNumberFormat should register a column style for B")
	assert.Equal(t, excel.FormatCurrency, style.NumFmt)
}

// TestExporter_SetNumberFormat_MergesIntoExistingColumnStyle verifies
// SetNumberFormat merges into a column style already set via
// SetColumnStyle instead of overwriting its other properties.
func TestExporter_SetNumberFormat_MergesIntoExistingColumnStyle(t *testing.T) {
	exp := NewExporter(nil)
	exp.SetColumnStyle("B", excel.NewStyle().Bold(true))
	exp.SetNumberFormat("B", excel.FormatPercentage)

	style := exp.columnStyles["B"]
	assert.True(t, style.Font.Bold, "SetNumberFormat should preserve the existing style's other properties")
	assert.Equal(t, excel.FormatPercentage, style.NumFmt)
}

func TestExporter_Styling_ExportToFile(t *testing.T) {
	data := [][]any{
		{"Product", "Price", "Quantity"},
		{"Apple", 1.5, 100},
		{"Banana", 0.8, 200},
	}

	exp := NewExporter(data, excel.WithHeadings(true))

	// Header style
	headerStyle := excel.NewStyle().
		Bold(true).
		BackgroundColor("#4472C4").
		FontColor(excel.ColorWhite).
		AlignHorizontal(excel.AlignCenter)
	exp.SetHeaderStyle(headerStyle)

	// Column widths
	exp.SetColumnWidth("A", 20.0)
	exp.SetColumnWidth("B", 15.0)
	exp.SetColumnWidth("C", 15.0)

	// Price column style
	priceStyle := excel.NewStyle().Format(excel.FormatCurrency)
	exp.SetColumnStyle("B", priceStyle)

	ctx := context.Background()
	err := exp.Export(ctx)
	require.NoError(t, err)

	// Export to file
	tmpFile := "test_styled_export.xlsx"
	defer os.Remove(tmpFile)

	err = exp.GetFile().SaveAs(tmpFile)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(tmpFile)
	assert.NoError(t, err)
}

// TestExporter_Styling_BoundedStyleCount verifies that a large styled
// export registers only a handful of excelize styles rather than one per
// row/cell. It works by exporting 1000 rows with header, data, and a
// column style configured, then registering one more, deliberately
// distinct style on the same file: excelize.NewStyle() assigns
// sequential, zero-indexed IDs, so the ID it returns equals the number of
// styles already registered. Before style ID caching (Phase 4), this
// count grew with row count (roughly 2*rows); after caching, it stays
// fixed at the number of distinct style definitions actually used
// (header, data, one column override).
func TestExporter_Styling_BoundedStyleCount(t *testing.T) {
	const rowCount = 1000

	rows := make([][]any, 0, rowCount+1)
	rows = append(rows, []any{"Name", "Age", "Salary"})
	for i := 0; i < rowCount; i++ {
		rows = append(rows, []any{"person", i, float64(i) * 1.5})
	}

	exp := NewExporter(rows, excel.WithHeadings(true))
	exp.SetHeaderStyle(excel.NewStyle().Bold(true).BackgroundColor(excel.ColorBlue))
	exp.SetDataStyle(excel.NewStyle().BorderAll(excel.BorderThin))
	exp.SetColumnStyle("C", excel.NewStyle().FontColor(excel.ColorGreen))

	require.NoError(t, exp.Export(context.Background()))

	sentinelID, err := exp.GetFile().NewStyle(excel.NewStyle().Italic(true).ToExcelizeStyle())
	require.NoError(t, err)

	const maxExpectedStyles = 10
	assert.Lessf(t, sentinelID, maxExpectedStyles,
		"export registered %d styles before the sentinel; expected a small, row-count-independent number", sentinelID)
}

func TestExporter_Styling_NoStyles(t *testing.T) {
	data := [][]any{
		{"A", "B"},
		{"C", "D"},
	}

	exp := NewExporter(data)
	ctx := context.Background()
	err := exp.Export(ctx)
	require.NoError(t, err)
	assert.NotNil(t, exp.GetFile())
}

// TestExporter_Styling_HeaderStyle_WithHeadingsFalse tests that headerStyle
// is applied to first row even when WithHeadings(false) is used.
// This fixes the bug where background color didn't appear when WithHeadings(false).
func TestExporter_Styling_HeaderStyle_WithHeadingsFalse(t *testing.T) {
	data := [][]any{
		{"Sesi", "Test 1", "Test 2", "Test 3"},
		{1, "Rp10,000,000.00", "Rp10,000,000.00", "Rp10,000,000.00"},
		{2, "Rp10,000,000.00", "Rp10,000,000.00", "Rp10,000,000.00"},
	}

	// Create exporter with headings disabled
	exp := NewExporter(data, excel.WithHeadings(false))

	// Set header style with background color
	tableHeaderStyle := excel.NewStyle().
		Bold(true).
		BackgroundColor("#a8c444").
		FontColor(excel.ColorBlack).
		FontSize(11).
		AlignHorizontal(excel.AlignCenter).
		BorderAll(excel.BorderMedium)

	exp.SetHeaderStyle(tableHeaderStyle)

	ctx := context.Background()
	err := exp.Export(ctx)
	require.NoError(t, err)
	assert.NotNil(t, exp.GetFile())

	// Verify that headerStyle was applied (style should be created)
	// The actual background color verification would require reading the Excel file,
	// but at least we verify the export succeeds without errors
}
