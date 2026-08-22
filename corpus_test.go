// Package corpus_test exercises "dirty" real-world Excel/CSV inputs
// against this module's public API: files with quirks (a UTF-8 BOM, a
// non-comma delimiter, a quoted embedded newline, ragged rows, merged
// cells, date-serial-vs-text ambiguity, oddly named or empty sheets,
// and a row count past excelize's hard limit) that the rest of this
// module's tests never produce because they all build clean [][]any
// fixtures by hand.
//
// It lives at the module root rather than under excel/importer or
// stream so a single package can freely mix BasicImporter, StreamImporter,
// and BasicExporter without import cycles, and so testdata/corpus/ (its
// sibling directory) reads naturally as "the corpus", not "importer's
// private fixtures". Small .csv files are committed under
// testdata/corpus/; .xlsx fixtures are built in-test via excelize
// instead of committing binary workbooks, per docs/FEATURE_CHECKLIST.md.
//
// Each test either compares against a golden expected result or
// asserts that the failure is a specific sentinel/typed error via
// errors.Is - never a panic and never a bare "err != nil".
package corpus_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
	"github.com/bonarizki-dat/go-excel/excel/importer"
	"github.com/bonarizki-dat/go-excel/stream"
	"github.com/xuri/excelize/v2"
)

// openCorpusFile opens a committed fixture under testdata/corpus/ for
// StreamImporter, which takes an io.Reader rather than a filename.
func openCorpusFile(t *testing.T, path string) (*os.File, error) {
	t.Helper()
	return os.Open(path)
}

// corpusPerson is the struct-mapping target for the BOM CSV corpus
// case, exercising the "struct atau GetRows" comparison the harden
// plan calls for alongside plain GetRows() checks elsewhere in this
// file.
type corpusPerson struct {
	Name string `excel:"name"`
	Age  int    `excel:"age"`
	City string `excel:"city"`
}

// TestCorpus_CSV_BOM_StripsAndImports verifies a leading UTF-8 BOM
// (common when a file was exported from Excel on Windows) never leaks
// into the first heading or the first row's first cell.
func TestCorpus_CSV_BOM_StripsAndImports(t *testing.T) {
	imp := importer.NewImporter(excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), "testdata/corpus/bom.csv"); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	headings := imp.GetHeadings()
	wantHeadings := []string{"name", "age", "city"}
	if len(headings) != len(wantHeadings) {
		t.Fatalf("headings = %v, want %v", headings, wantHeadings)
	}
	for i, h := range wantHeadings {
		if headings[i] != h {
			t.Errorf("headings[%d] = %q, want %q (BOM leaked into heading?)", i, headings[i], h)
		}
	}

	rows := imp.GetRows()
	wantRows := [][]any{
		{"Alice", "30", "Jakarta"},
		{"Bob", "25", "Bandung"},
	}
	if len(rows) != len(wantRows) {
		t.Fatalf("rows = %v, want %v", rows, wantRows)
	}
	for i, want := range wantRows {
		for j, wantCell := range want {
			if rows[i][j] != wantCell {
				t.Errorf("rows[%d][%d] = %v, want %v", i, j, rows[i][j], wantCell)
			}
		}
	}
}

// TestCorpus_CSV_BOM_ImportToStructs is the struct-mapped counterpart
// to TestCorpus_CSV_BOM_StripsAndImports: same dirty file, but through
// ImportToStructs instead of GetRows, confirming the BOM-stripped
// heading still matches the "name" excel tag.
func TestCorpus_CSV_BOM_ImportToStructs(t *testing.T) {
	people, err := importer.ImportToStructs[corpusPerson](context.Background(),
		"testdata/corpus/bom.csv", excel.WithHeadings(true))
	if err != nil {
		t.Fatalf("ImportToStructs: %v", err)
	}

	want := []corpusPerson{
		{Name: "Alice", Age: 30, City: "Jakarta"},
		{Name: "Bob", Age: 25, City: "Bandung"},
	}
	if len(people) != len(want) {
		t.Fatalf("people = %+v, want %+v", people, want)
	}
	for i := range want {
		if people[i] != want[i] {
			t.Errorf("people[%d] = %+v, want %+v", i, people[i], want[i])
		}
	}
}

// TestCorpus_CSV_SemicolonDelimiter_StreamImporter verifies a
// ';'-delimited CSV (common in locales where ',' is the decimal
// separator) reads correctly through StreamImporter, not just
// BasicImporter.
func TestCorpus_CSV_SemicolonDelimiter_StreamImporter(t *testing.T) {
	f, err := openCorpusFile(t, "testdata/corpus/semicolon.csv")
	if err != nil {
		t.Fatalf("open corpus file: %v", err)
	}
	defer func() { _ = f.Close() }()

	csvOpts := config.DefaultCSVOptions()
	csvOpts.Delimiter = ';'

	imp, err := stream.NewStreamImporter(f,
		stream.WithImportFormat(config.FileFormatCSV),
		stream.WithImportCSVOptions(csvOpts),
		stream.WithImportHeadings(),
	)
	if err != nil {
		t.Fatalf("NewStreamImporter: %v", err)
	}
	defer func() { _ = imp.Close() }()

	// ImportStream's processor runs concurrently across worker
	// goroutines, so rows must be collected keyed by rowNum under a
	// mutex rather than appended to a shared slice; see
	// stream/importer_csv_test.go for the same pattern.
	var mu sync.Mutex
	got := make(map[int][]any)
	processor := func(_ context.Context, rowNum int, row []any) ([]any, error) {
		mu.Lock()
		got[rowNum] = row
		mu.Unlock()
		return row, nil
	}

	if err := imp.ImportStream(context.Background(), processor); err != nil {
		t.Fatalf("ImportStream: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2: %v", len(got), got)
	}
	if got[1][0] != "Alice" || got[1][2] != "Jakarta" {
		t.Errorf("row 1 = %v, want [Alice 30 Jakarta]", got[1])
	}
	if got[2][0] != "Bob" || got[2][2] != "Bandung" {
		t.Errorf("row 2 = %v, want [Bob 25 Bandung]", got[2])
	}
}

// TestCorpus_CSV_QuotedNewline_PreservesEmbeddedNewline verifies a
// quoted CSV field containing a literal newline (a "notes" column
// pasted from another document, say) survives import intact instead
// of being split into two rows.
func TestCorpus_CSV_QuotedNewline_PreservesEmbeddedNewline(t *testing.T) {
	imp := importer.NewImporter(excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), "testdata/corpus/quoted_newline.csv"); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	rows := imp.GetRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %v, want 2 rows (embedded newline must not split a row)", rows)
	}
	if rows[0][0] != "Alice" || rows[0][1] != "line one\nline two" {
		t.Errorf("rows[0] = %v, want [Alice \"line one\\nline two\"]", rows[0])
	}
	if rows[1][0] != "Bob" || rows[1][1] != "single line" {
		t.Errorf("rows[1] = %v, want [Bob \"single line\"]", rows[1])
	}
}

// TestCorpus_CSV_RaggedSparseRows_NoPanicVariableWidth verifies a CSV
// with an empty heading cell and rows of varying field counts (a
// hand-edited export, missing trailing columns on some rows and an
// extra trailing column on another) imports without panicking, and
// that rows keep their own actual width rather than being padded or
// truncated to the heading's width.
func TestCorpus_CSV_RaggedSparseRows_NoPanicVariableWidth(t *testing.T) {
	imp := importer.NewImporter(excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), "testdata/corpus/ragged_sparse.csv"); err != nil {
		t.Fatalf("ImportFromFile: %v", err)
	}

	headings := imp.GetHeadings()
	if len(headings) != 3 || headings[0] != "name" || headings[1] != "" || headings[2] != "city" {
		t.Fatalf("headings = %v, want [name \"\" city] (empty heading cell tolerated)", headings)
	}

	rows := imp.GetRows()
	if len(rows) != 3 {
		t.Fatalf("rows = %v, want 3 rows", rows)
	}

	if len(rows[0]) != 3 || rows[0][0] != "Alice" || rows[0][1] != "30" || rows[0][2] != "Jakarta" {
		t.Errorf("rows[0] = %v, want [Alice 30 Jakarta]", rows[0])
	}
	// "Bob,,," has 4 comma-separated fields, one more than the header;
	// the row keeps that width rather than being clipped to 3.
	if len(rows[1]) != 4 || rows[1][0] != "Bob" || rows[1][1] != "" || rows[1][2] != "" || rows[1][3] != "" {
		t.Errorf("rows[1] = %v, want [Bob \"\" \"\" \"\"] (ragged row wider than header)", rows[1])
	}
	if len(rows[2]) != 4 || rows[2][0] != "Carol" || rows[2][3] != "Extra" {
		t.Errorf("rows[2] = %v, want [Carol 22 Bandung Extra]", rows[2])
	}
}

// buildMergedCellsFixture builds an .xlsx workbook with one merged
// cell range (B2:C2) and returns it encoded as bytes, for
// ImportFromReader to consume without touching disk.
func buildMergedCellsFixture(t *testing.T) []byte {
	t.Helper()

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	must(t, f.SetCellValue(sheet, "A1", "Name"))
	must(t, f.SetCellValue(sheet, "B1", "Note"))
	must(t, f.SetCellValue(sheet, "C1", "Extra"))

	must(t, f.SetCellValue(sheet, "A2", "Alice"))
	must(t, f.SetCellValue(sheet, "B2", "Merged"))
	must(t, f.MergeCell(sheet, "B2", "C2"))

	must(t, f.SetCellValue(sheet, "A3", "Bob"))
	must(t, f.SetCellValue(sheet, "B3", "Plain"))
	must(t, f.SetCellValue(sheet, "C3", "Val"))

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}
	return buf.Bytes()
}

// TestCorpus_XLSX_MergedCells_ProducesRaggedRowLengths documents a real
// excelize gotcha rather than a bug in this module: excelize.GetRows
// (which BasicImporter.Import calls) does not pad a merged range's
// covered-but-unwritten cells with empty strings - it simply stops the
// row at the last cell that has an explicit value. A row spanned by a
// trailing merge is therefore *shorter* than an unmerged row with the
// same number of columns. Callers indexing columns positionally must
// account for this; there is no BasicImporter escape hatch to inspect
// merge ranges directly (unlike BasicExporter.GetFile()) - reading the
// source with excelize.OpenReader/GetMergeCells directly, as this test
// does implicitly by building the fixture, is the only way to recover
// that structure.
func TestCorpus_XLSX_MergedCells_ProducesRaggedRowLengths(t *testing.T) {
	data := buildMergedCellsFixture(t)

	imp := importer.NewImporter(excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromReader(context.Background(), bytes.NewReader(data)); err != nil {
		t.Fatalf("ImportFromReader: %v", err)
	}

	headings := imp.GetHeadings()
	if len(headings) != 3 || headings[0] != "Name" || headings[1] != "Note" || headings[2] != "Extra" {
		t.Fatalf("headings = %v, want [Name Note Extra]", headings)
	}

	rows := imp.GetRows()
	if len(rows) != 2 {
		t.Fatalf("rows = %v, want 2 rows", rows)
	}

	// Alice's row is shortened to 2 cells: C2 is covered by the B2:C2
	// merge and was never itself written, so GetRows drops it instead
	// of reporting "" for it.
	if len(rows[0]) != 2 || rows[0][0] != "Alice" || rows[0][1] != "Merged" {
		t.Errorf("rows[0] = %v, want [Alice Merged] (merge target cell omitted, not empty)", rows[0])
	}
	if len(rows[1]) != 3 || rows[1][0] != "Bob" || rows[1][1] != "Plain" || rows[1][2] != "Val" {
		t.Errorf("rows[1] = %v, want [Bob Plain Val] (unmerged row keeps full width)", rows[1])
	}
}

// buildDateAndTextAmbiguityFixture builds an .xlsx workbook with three
// rows that each look "numeric" but decode differently: a bare date
// serial with no number format, the same serial with a date format
// applied, and a numeric-looking string stored as text (preserving a
// leading zero a real numeric type would drop).
func buildDateAndTextAmbiguityFixture(t *testing.T) []byte {
	t.Helper()

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	must(t, f.SetCellValue(sheet, "A1", "Label"))
	must(t, f.SetCellValue(sheet, "B1", "Value"))

	must(t, f.SetCellValue(sheet, "A2", "date_no_format"))
	must(t, f.SetCellValue(sheet, "B2", 45311)) // bare serial, no date style

	must(t, f.SetCellValue(sheet, "A3", "date_formatted"))
	must(t, f.SetCellValue(sheet, "B3", 45311))
	numFmt := "yyyy-mm-dd"
	styleID, err := f.NewStyle(&excelize.Style{CustomNumFmt: &numFmt})
	if err != nil {
		t.Fatalf("NewStyle: %v", err)
	}
	must(t, f.SetCellStyle(sheet, "B3", "B3", styleID))

	must(t, f.SetCellValue(sheet, "A4", "number_as_text"))
	must(t, f.SetCellValue(sheet, "B4", "0100")) // text, keeps the leading zero

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}
	return buf.Bytes()
}

// TestCorpus_XLSX_DateSerialVsFormattedVsTextNumber documents that this
// module does not reinterpret cell values beyond what excelize.GetRows
// already decodes: a date-typed cell with no number format applied
// comes back as its bare serial number, the same value with a date
// format comes back as a formatted date string, and a numeric-looking
// value stored as text comes back verbatim (leading zero intact).
// Callers that need calendar dates from the first case must convert
// the serial themselves; this module does not guess.
func TestCorpus_XLSX_DateSerialVsFormattedVsTextNumber(t *testing.T) {
	data := buildDateAndTextAmbiguityFixture(t)

	imp := importer.NewImporter(excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromReader(context.Background(), bytes.NewReader(data)); err != nil {
		t.Fatalf("ImportFromReader: %v", err)
	}

	rows := imp.GetRows()
	want := [][]any{
		{"date_no_format", "45311"},
		{"date_formatted", "2024-01-20"},
		{"number_as_text", "0100"},
	}
	if len(rows) != len(want) {
		t.Fatalf("rows = %v, want %v", rows, want)
	}
	for i, w := range want {
		if rows[i][0] != w[0] || rows[i][1] != w[1] {
			t.Errorf("rows[%d] = %v, want %v", i, rows[i], w)
		}
	}
}

// buildMultiSheetWeirdAndEmptyFixture builds an .xlsx workbook with an
// ASCII-named sheet, a sheet with a non-ASCII name (a real pattern for
// international users' workbooks), and a trailing sheet with no cells
// written at all, then saves it to a file under t.TempDir() (both
// ImportAllSheets and ImportFromSheet require a filename, not a
// reader). Returns the file path and the weird sheet's name.
func buildMultiSheetWeirdAndEmptyFixture(t *testing.T) (path, weirdSheetName string) {
	t.Helper()

	f := excelize.NewFile()
	dataSheet := f.GetSheetName(0)
	must(t, f.SetSheetName(dataSheet, "Data"))
	must(t, f.SetCellValue("Data", "A1", "ID"))
	must(t, f.SetCellValue("Data", "A2", 1))

	weirdSheetName = "Data_日本語"
	if _, err := f.NewSheet(weirdSheetName); err != nil {
		t.Fatalf("NewSheet(%q): %v", weirdSheetName, err)
	}
	must(t, f.SetCellValue(weirdSheetName, "A1", "ID"))
	must(t, f.SetCellValue(weirdSheetName, "A2", 2))

	if _, err := f.NewSheet("Empty"); err != nil {
		t.Fatalf("NewSheet(Empty): %v", err)
	}
	// Deliberately no cells written to "Empty".

	path = t.TempDir() + "/multisheet_weird_and_empty.xlsx"
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	return path, weirdSheetName
}

// TestCorpus_XLSX_ImportAllSheets_EmptySheetAbortsWithTypedError
// documents that ImportAllSheets does not skip an empty sheet - it
// propagates that sheet's excel.ErrEmptyData and aborts the whole
// call, even though earlier sheets (including one with a non-ASCII
// name) decoded fine. This is a real, typed failure mode, not a panic
// and not a silently-dropped sheet.
func TestCorpus_XLSX_ImportAllSheets_EmptySheetAbortsWithTypedError(t *testing.T) {
	path, _ := buildMultiSheetWeirdAndEmptyFixture(t)

	imp := importer.NewImporter()
	defer func() { _ = imp.Close() }()

	_, err := imp.ImportAllSheets(context.Background(), path)
	if err == nil {
		t.Fatal("ImportAllSheets with a trailing empty sheet = nil error, want error wrapping excel.ErrEmptyData")
	}
	if !errors.Is(err, excel.ErrEmptyData) {
		t.Errorf("ImportAllSheets error = %v, want it to wrap excel.ErrEmptyData", err)
	}
}

// TestCorpus_XLSX_ImportFromSheet_WeirdNameSucceeds is the positive
// counterpart to the empty-sheet case: a sheet with a non-ASCII name
// imports normally through ImportFromSheet, targeted explicitly by
// that name.
func TestCorpus_XLSX_ImportFromSheet_WeirdNameSucceeds(t *testing.T) {
	path, weirdSheetName := buildMultiSheetWeirdAndEmptyFixture(t)

	imp := importer.NewImporter(excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	rows, err := imp.ImportFromSheet(context.Background(), path, weirdSheetName)
	if err != nil {
		t.Fatalf("ImportFromSheet(%q): %v", weirdSheetName, err)
	}
	if len(rows) != 1 || rows[0][0] != "2" {
		t.Errorf("rows = %v, want [[2]]", rows)
	}
}

// TestCorpus_XLSX_ImportFromSheet_NonExistentSheet_TypedError verifies
// requesting a sheet absent from a real multi-sheet workbook returns
// excel.ErrSheetNotFound specifically, checkable via errors.Is, not
// just a generic non-nil error.
func TestCorpus_XLSX_ImportFromSheet_NonExistentSheet_TypedError(t *testing.T) {
	path, _ := buildMultiSheetWeirdAndEmptyFixture(t)

	imp := importer.NewImporter()
	defer func() { _ = imp.Close() }()

	_, err := imp.ImportFromSheet(context.Background(), path, "DoesNotExist")
	if err == nil {
		t.Fatal("ImportFromSheet with a nonexistent sheet = nil error, want error wrapping excel.ErrSheetNotFound")
	}
	if !errors.Is(err, excel.ErrSheetNotFound) {
		t.Errorf("ImportFromSheet error = %v, want it to wrap excel.ErrSheetNotFound", err)
	}
}

// TestCorpus_XLSX_ExportBeyondMaxRows_TypedErrMaxRows verifies writing
// past row 1,048,576 (excelize's hard limit, TotalRows) surfaces
// excelize.ErrMaxRows through this module's export path, checkable via
// errors.Is, rather than a generic wrapped error or a panic. The start
// cell is set to the last valid row so only the header (row
// 1,048,576) and a single data row (row 1,048,577, past the limit)
// need to be written - no large dataset required to hit the limit
// cheaply and deterministically.
func TestCorpus_XLSX_ExportBeyondMaxRows_TypedErrMaxRows(t *testing.T) {
	data := [][]any{
		{"Name"},
		{"Alice"},
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true), excel.WithCustomStartCell("A1048576"))
	err := exp.Export(context.Background())
	if err == nil {
		t.Fatal("Export past TotalRows = nil error, want error wrapping excelize.ErrMaxRows")
	}
	if !errors.Is(err, excelize.ErrMaxRows) {
		t.Errorf("Export error = %v, want it to wrap excelize.ErrMaxRows", err)
	}
}

// must fails the test immediately if err is non-nil, for the
// excelize fixture-building helpers above where every call is
// expected to succeed and a failure means the fixture itself is
// broken, not the code under test.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("fixture setup: %v", err)
	}
}
