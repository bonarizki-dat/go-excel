# Examples - Go Excel Library

> Runnable examples showing how to use the library.

---

## 📚 Available Examples

- **[basic_export/](basic_export/)** - Export a slice of rows to an `.xlsx` file
- **[basic_import/](basic_import/)** - Import rows from an `.xlsx` file
- **[csv_format/](csv_format/)** - Export/import `.csv` files, select the format
  explicitly with `WithFormat` for writer/buffer targets, override CSV
  delimiter/BOM/line-ending options, and see the one-time degradation warning
  when styles are set on a CSV export
- **[multisheet/](multisheet/)** - Export multiple sheets to a single file, import a
  specific sheet, import all sheets, and query sheet names/count/existence
- **[struct_mapping/](struct_mapping/)** - Import Excel rows directly into Go structs,
  with automatic type conversion and `excel:"..."` tag-based validation
- **[events/](events/)** - Register `BeforeExport`/`AfterExport` and
  `BeforeRowExport`/`AfterRowExport` hooks
- **[styling/](styling/)** - Font, background color, border, alignment, column width,
  and number format styling
- **[merge_cells/](merge_cells/)** - Merge cells for titles and headers via
  `GetFile()` to access the underlying `excelize.File`
- **[web_export/](web_export/)** - Serve a generated `.xlsx` file directly from an
  `net/http` handler using `ExportToWriter`
- **[stream_export/](stream_export/)** - Row-at-a-time export of a large dataset with
  `stream.StreamExporter`: a concurrent `RowTransformer`, progress reporting, and
  correctly deferring `Close()` so resources are released even if the export fails
- **[stream_import/](stream_import/)** - Row-at-a-time import with `stream.StreamImporter`:
  a `RowProcessor` that handles per-row errors and context cancellation, and correctly
  deferring `Close()`

---

## 🚀 Quick Start

Each example is a self-contained `main.go`. Run any of them with:

```bash
cd examples/<name>
go run main.go
```

For example:

```bash
cd examples/basic_export
go run main.go
# Creates output.xlsx in the current directory
```

```bash
cd examples/web_export
go run main.go
# Visit http://localhost:8080/export to download the generated file
```

`stream_import` reads the file `stream_export` produces, so run them in
order from the same directory:

```bash
cd examples/stream_export && go run main.go   # writes stream_export_output.xlsx
cp stream_export_output.xlsx ../stream_import/
cd ../stream_import && go run main.go
```

---

## 💡 Common Patterns

### Export to an `io.Writer` (e.g. an HTTP response)

```go
exp := exporter.NewExporter(data, excel.WithHeadings(true))
if err := exp.ExportToWriter(ctx, w); err != nil {
    // handle error
}
```

### Import from an `io.Reader` (e.g. an HTTP request body)

```go
imp := importer.NewImporter(excel.WithHeadings(true))
if err := imp.ImportFromReader(ctx, r); err != nil {
    // handle error
}
rows := imp.GetRows()
```

See [basic_export/main.go](basic_export/main.go) and
[basic_import/main.go](basic_import/main.go) for complete, runnable versions.

---

**See individual example directories for detailed documentation.**
