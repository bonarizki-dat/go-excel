package exporter

import (
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestExportToSheets_StylesApplyToEveryXLSXSheet verifies that
// SetHeaderStyle, SetDataStyle, SetColumnStyle, and SetColumnWidth take
// effect on every sheet of an XLSX multi-sheet export, the same as they
// do for Export. Before ExportToSheets was wired through xlsxSink, this
// path wrote cell values directly and ignored all four.
func TestExportToSheets_StylesApplyToEveryXLSXSheet(t *testing.T) {
	exp := NewExporter(nil)
	exp.SetHeaderStyle(excel.NewStyle().Bold(true))
	exp.SetDataStyle(excel.NewStyle().Italic(true))
	exp.SetColumnStyle("B", excel.NewStyle().FontColor(excel.ColorRed))
	exp.SetColumnWidth("A", 25.0)

	sheets := []excel.Sheet{
		{Name: "First", Data: [][]any{{"Name", "Age"}, {"Alice", 30}}},
		{Name: "Second", Data: [][]any{{"Name", "Age"}, {"Bob", 40}}},
	}

	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets: %v", err)
	}

	file := exp.GetFile()

	sheetNames := []string{"First", "Second"}
	headerStyleIDs := make([]int, 0, len(sheetNames))
	for _, sheetName := range sheetNames {
		headerID, err := file.GetCellStyle(sheetName, "A1")
		if err != nil {
			t.Fatalf("GetCellStyle(%s, A1): %v", sheetName, err)
		}
		if headerID == 0 {
			t.Errorf("sheet %s: header style ID = 0, want a registered style", sheetName)
		}
		headerStyleIDs = append(headerStyleIDs, headerID)

		dataID, err := file.GetCellStyle(sheetName, "A2")
		if err != nil {
			t.Fatalf("GetCellStyle(%s, A2): %v", sheetName, err)
		}
		if dataID == 0 {
			t.Errorf("sheet %s: data style ID = 0, want a registered style", sheetName)
		}
		if dataID == headerID {
			t.Errorf("sheet %s: data style ID equals header style ID, want distinct styles", sheetName)
		}

		colB, err := file.GetCellStyle(sheetName, "B2")
		if err != nil {
			t.Fatalf("GetCellStyle(%s, B2): %v", sheetName, err)
		}
		if colB == dataID {
			t.Errorf("sheet %s: column B style equals the plain data style, want the column-specific override", sheetName)
		}

		width, err := file.GetColWidth(sheetName, "A")
		if err != nil {
			t.Fatalf("GetColWidth(%s, A): %v", sheetName, err)
		}
		if width != 25.0 {
			t.Errorf("sheet %s: column A width = %v, want 25.0", sheetName, width)
		}
	}

	// The style-ID cache is scoped to the workbook, not to a sheet, so
	// reusing the sink across sheets must resolve the same header style
	// to the same ID rather than registering it twice.
	if headerStyleIDs[0] != headerStyleIDs[1] {
		t.Errorf("header style IDs across sheets = %v, want equal (cached, not re-registered)", headerStyleIDs)
	}
}

// TestExportToSheets_RepeatedCall_ClosesPreviousWorkbook verifies that
// calling ExportToSheets a second time on the same exporter succeeds
// and produces a fresh, independent *excelize.File rather than leaking
// the workbook from the first call. Before initWorkbook closed the
// previous file, GetFile() only ever exposed the current workbook and
// the prior one became unreachable without being closed.
func TestExportToSheets_RepeatedCall_ClosesPreviousWorkbook(t *testing.T) {
	ctx := context.Background()
	exp := NewExporter(nil)

	first := []excel.Sheet{{Name: "Sheet1", Data: [][]any{{"a"}}}}
	if err := exp.ExportToSheets(ctx, first, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("first ExportToSheets failed: %v", err)
	}
	firstFile := exp.GetFile()
	if firstFile == nil {
		t.Fatal("expected first ExportToSheets to produce a file")
	}

	second := []excel.Sheet{{Name: "Sheet2", Data: [][]any{{"b"}}}}
	if err := exp.ExportToSheets(ctx, second, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("second ExportToSheets failed: %v", err)
	}
	secondFile := exp.GetFile()
	if secondFile == nil {
		t.Fatal("expected second ExportToSheets to produce a file")
	}
	if secondFile == firstFile {
		t.Fatal("expected second ExportToSheets to produce a distinct *excelize.File")
	}

	sheetList := secondFile.GetSheetList()
	if len(sheetList) != 1 || sheetList[0] != "Sheet2" {
		t.Errorf("second workbook sheets = %v, want only 'Sheet2'", sheetList)
	}

	// A third call must also succeed, proving the close-then-replace
	// path is safe to repeat rather than just working once.
	third := []excel.Sheet{{Name: "Sheet3", Data: [][]any{{"c"}}}}
	if err := exp.ExportToSheets(ctx, third, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("third ExportToSheets failed: %v", err)
	}
}
