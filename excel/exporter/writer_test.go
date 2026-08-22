package exporter

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportToFile_Basic(t *testing.T) {
	data := [][]any{
		{"Name", "Age"},
		{"Alice", 30},
		{"Bob", 25},
	}

	filename := "test_export_basic.xlsx"
	defer os.Remove(filename)

	exp := NewExporter(data, excel.WithHeadings(true))
	ctx := context.Background()

	err := exp.ExportToFile(ctx, filename)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filename)
	assert.NoError(t, err)
}

func TestExportToFile_WithStyling(t *testing.T) {
	data := [][]any{
		{"Product", "Price"},
		{"Apple", 1.50},
		{"Banana", 0.80},
	}

	filename := "test_export_styled.xlsx"
	defer os.Remove(filename)

	exp := NewExporter(data, excel.WithHeadings(true))

	// Apply styles
	headerStyle := excel.NewStyle().Bold(true).BackgroundColor(excel.ColorBlue)
	exp.SetHeaderStyle(headerStyle)
	exp.SetColumnWidth("A", 20.0)

	ctx := context.Background()
	err := exp.ExportToFile(ctx, filename)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filename)
	assert.NoError(t, err)
}

func TestExportToFile_EmptyFilename(t *testing.T) {
	data := [][]any{{"A", "B"}}
	exp := NewExporter(data)
	ctx := context.Background()

	err := exp.ExportToFile(ctx, "")
	assert.Error(t, err)
}

func TestExportToWriter_Basic(t *testing.T) {
	data := [][]any{
		{"Name", "Age"},
		{"Carol", 28},
	}

	exp := NewExporter(data, excel.WithHeadings(true))
	ctx := context.Background()

	var buf bytes.Buffer
	err := exp.ExportToWriter(ctx, &buf)
	require.NoError(t, err)

	// Verify buffer has data
	assert.Greater(t, buf.Len(), 0, "Buffer should contain Excel data")
}

func TestExportToWriter_WithStyling(t *testing.T) {
	data := [][]any{
		{"Category", "Count"},
		{"Electronics", 100},
		{"Furniture", 50},
	}

	exp := NewExporter(data, excel.WithHeadings(true))

	// Apply styling
	headerStyle := excel.NewStyle().
		Bold(true).
		BackgroundColor("#4472C4").
		FontColor(excel.ColorWhite)
	exp.SetHeaderStyle(headerStyle)

	dataStyle := excel.NewStyle().BorderAll(excel.BorderThin)
	exp.SetDataStyle(dataStyle)

	ctx := context.Background()
	var buf bytes.Buffer
	err := exp.ExportToWriter(ctx, &buf)
	require.NoError(t, err)

	assert.Greater(t, buf.Len(), 0)
}

func TestExportToWriter_NilWriter(t *testing.T) {
	data := [][]any{{"A"}}
	exp := NewExporter(data)
	ctx := context.Background()

	err := exp.ExportToWriter(ctx, nil)
	assert.Error(t, err)
}

func TestExportToBuffer_Basic(t *testing.T) {
	data := [][]any{
		{"Header1", "Header2"},
		{"Value1", "Value2"},
	}

	exp := NewExporter(data, excel.WithHeadings(true))
	ctx := context.Background()

	buf, err := exp.ExportToBuffer(ctx)
	require.NoError(t, err)
	require.NotNil(t, buf)

	assert.Greater(t, len(buf), 0, "Buffer should contain data")
}

func TestExportToBuffer_WithColumnWidths(t *testing.T) {
	data := [][]any{
		{"A", "B", "C"},
		{1, 2, 3},
	}

	exp := NewExporter(data)
	exp.SetColumnWidth("A", 15.0)
	exp.SetColumnWidth("B", 20.0)
	exp.SetColumnWidth("C", 25.0)

	ctx := context.Background()
	buf, err := exp.ExportToBuffer(ctx)
	require.NoError(t, err)

	assert.Greater(t, len(buf), 0)
}

func TestExportToBuffer_WithColumnStyles(t *testing.T) {
	data := [][]any{
		{"Name", "Amount", "Date"},
		{"Item1", 123.45, "2024-01-01"},
		{"Item2", 678.90, "2024-01-02"},
	}

	exp := NewExporter(data, excel.WithHeadings(true))

	// Style for Amount column
	amountStyle := excel.NewStyle().Format(excel.FormatCurrency)
	exp.SetColumnStyle("B", amountStyle)

	// Style for Date column
	dateStyle := excel.NewStyle().Format(excel.FormatDate)
	exp.SetColumnStyle("C", dateStyle)

	ctx := context.Background()
	buf, err := exp.ExportToBuffer(ctx)
	require.NoError(t, err)

	assert.Greater(t, len(buf), 0)
}

func TestExportToBuffer_EmptyData(t *testing.T) {
	exp := NewExporter(nil)
	ctx := context.Background()

	buf, err := exp.ExportToBuffer(ctx)
	assert.Error(t, err)
	assert.Nil(t, buf)
}

func TestExportToFile_LargeData(t *testing.T) {
	// Create large dataset
	data := make([][]any, 1001) // 1 header + 1000 rows
	data[0] = []any{"ID", "Name", "Value"}

	for i := 1; i <= 1000; i++ {
		data[i] = []any{i, "Item" + string(rune(i)), i * 10}
	}

	filename := "test_export_large.xlsx"
	defer os.Remove(filename)

	exp := NewExporter(data, excel.WithHeadings(true))
	ctx := context.Background()

	err := exp.ExportToFile(ctx, filename)
	require.NoError(t, err)

	// Verify file exists and has reasonable size
	stat, err := os.Stat(filename)
	assert.NoError(t, err)
	assert.Greater(t, stat.Size(), int64(5000), "File should have reasonable size for 1000 rows")
}

func TestExportToWriter_MultipleColumns(t *testing.T) {
	// Create data with many columns
	data := [][]any{
		{"Col1", "Col2", "Col3", "Col4", "Col5", "Col6", "Col7", "Col8", "Col9", "Col10"},
		{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		{11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
	}

	exp := NewExporter(data, excel.WithHeadings(true))

	// Set different widths for columns
	for i, width := range []float64{10, 12, 14, 16, 18, 20, 15, 13, 11, 9} {
		col := string(rune('A' + i))
		exp.SetColumnWidth(col, width)
	}

	ctx := context.Background()
	var buf bytes.Buffer
	err := exp.ExportToWriter(ctx, &buf)
	require.NoError(t, err)

	assert.Greater(t, buf.Len(), 0)
}
