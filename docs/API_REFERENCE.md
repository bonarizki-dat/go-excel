# API Reference - Go Excel Library

> Complete API documentation for go-lib-matwebsite

**Covers**: `v0.3.0` (see [CHANGELOG.md](../CHANGELOG.md))
**Last audited against real signatures**: 2026-08-23

---

## Table of Contents

1. [Package: excel](#package-excel)
   - [BasicExporter](#basicexporter)
   - [BasicImporter](#basicimporter)
   - [Config & Options](#config--options)
   - [Events](#events)
   - [Style](#style)
   - [MultiSheet](#multisheet)
   - [Struct Mapping](#struct-mapping)
   - [Interfaces](#interfaces)
   - [Error Types](#error-types)
2. [Package: stream](#package-stream)
   - [StreamExporter](#streamexporter)
   - [StreamImporter](#streamimporter)

---

## Package: excel

### BasicExporter

#### NewExporter

```go
func NewExporter(data any, opts ...Option) *BasicExporter
```

Creates a new basic exporter with the given data and options.

**Parameters:**
- `data`: Data to export. Can be:
  - `[][]any` - 2D slice of data
  - `[]any` - 1D slice (converted to rows)
  - Any type implementing `Exportable` interface
- `opts`: Optional configuration options (see [Options](#config--options))

**Returns:**
- `*BasicExporter`: Configured exporter instance

**Example:**
```go
data := [][]any{
    {"Name", "Age"},
    {"John", 30},
}
exporter := exporter.NewExporter(data, excel.WithHeadings(true))
```

---

#### Export

```go
func (e *BasicExporter) Export(ctx context.Context) error
```

Performs the core export operation. Prepares data and writes to internal excelize file.

**Parameters:**
- `ctx`: Context for cancellation and timeout

**Returns:**
- `error`: Error if export fails, nil on success

**Example:**
```go
ctx := context.Background()
if err := exporter.Export(ctx); err != nil {
    log.Fatal(err)
}
```

---

#### ExportToFile

```go
func (e *BasicExporter) ExportToFile(ctx context.Context, filename string) error
```

Exports data to a file. Convenience method that calls `Export()` and saves to file.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `filename`: Path to output file. The format is derived from the extension: `.csv` writes CSV, anything else (`.xlsx` and other OOXML variants excelize accepts, such as `.xlsm`, `.xltx`, `.xltm`, `.xlam`) writes XLSX. `.xls` is not supported

**Returns:**
- `error`: Error if export fails or file cannot be saved

**Example:**
```go
err := exporter.ExportToFile(ctx, "output.xlsx")
```

---

#### ExportToWriter

```go
func (e *BasicExporter) ExportToWriter(ctx context.Context, w io.Writer) error
```

Exports data to an io.Writer. Framework-agnostic method for HTTP responses. There is no filename here for the format to be inferred from, so it writes whatever `excel.WithFormat` selected (default: `FileFormatXLSX`); pass `excel.WithFormat(excel.FileFormatCSV)` to write CSV instead.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `w`: io.Writer destination (e.g., http.ResponseWriter, gin.Context.Writer)

**Returns:**
- `error`: Error if export fails or writing fails

**Example:**
```go
// Gin framework
err := exporter.ExportToWriter(ctx, c.Writer)

// Echo framework
err := exporter.ExportToWriter(ctx, c.Response().Writer)

// CSV instead of XLSX
exporter := exporter.NewExporter(data, excel.WithFormat(excel.FileFormatCSV))
err := exporter.ExportToWriter(ctx, c.Writer)
```

---

#### ExportToBuffer

```go
func (e *BasicExporter) ExportToBuffer(ctx context.Context) ([]byte, error)
```

Exports data to a byte buffer. Useful for in-memory operations. Like `ExportToWriter`, there is no filename to infer format from, so it uses whatever `excel.WithFormat` selected (default: `FileFormatXLSX`).

**Parameters:**
- `ctx`: Context for cancellation and timeout

**Returns:**
- `[]byte`: Buffer contents
- `error`: Error if export fails

**Example:**
```go
buf, err := exporter.ExportToBuffer(ctx)
if err != nil {
    log.Fatal(err)
}
```

---

#### ExportToSheets

```go
func (e *BasicExporter) ExportToSheets(ctx context.Context, sheets []excel.Sheet, options excel.MultiSheetExportOptions) error
```

Exports multiple sheets to a single Excel file. CSV has no concept of multiple sheets, so when the exporter's format is `FileFormatCSV`, only the first sheet is written and a warning is logged (once, via `excel.WithLogger`) if any others were dropped; use `GetSink()` instead of `GetFile()` to retrieve CSV output afterward.

For XLSX, header/data/column styles and column widths set via `SetHeaderStyle`/`SetDataStyle`/`SetColumnStyle`/`SetColumnWidth` apply to every sheet, the same as `Export`. Unlike `Export`, this path does not dispatch `BeforeRowExport`/`AfterRowExport` (or `BeforeExport`/`AfterExport`, since it is not `Export`) for any sheet; use `MultiSheetExportOptions.ProgressCallback` or a sheet's own `Config.ProgressCallback` to observe progress instead.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `sheets`: Slice of Sheet structs containing data and config
- `options`: MultiSheetExportOptions for customization

**Returns:**
- `error`: Error if export fails, nil on success

**Example:**
```go
sheets := []excel.Sheet{
    {Name: "Users", Data: userData},
    {Name: "Orders", Data: orderData},
}
err := exporter.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
```

---

#### SetHeaderStyle

```go
func (e *BasicExporter) SetHeaderStyle(style *excel.Style)
```

Sets the style for the header row.

**Parameters:**
- `style`: Style to apply to header row

**Example:**
```go
headerStyle := excel.NewStyle().
    Bold(true).
    BackgroundColor("#4472C4").
    FontColor("#FFFFFF")
exporter.SetHeaderStyle(headerStyle)
```

---

#### SetDataStyle

```go
func (e *BasicExporter) SetDataStyle(style *excel.Style)
```

Sets the default style for data rows.

**Parameters:**
- `style`: Style to apply to data rows

**Example:**
```go
dataStyle := excel.NewStyle().
    FontSize(10).
    BorderAll(excel.BorderThin)
exporter.SetDataStyle(dataStyle)
```

---

#### SetColumnStyle

```go
func (e *BasicExporter) SetColumnStyle(col string, style *excel.Style)
```

Sets the style for a specific column (e.g., "A", "B").

**Parameters:**
- `col`: Column identifier (e.g., "A", "B", "C")
- `style`: Style to apply to the column

**Example:**
```go
exporter.SetColumnStyle("A", excel.NewStyle().Bold(true))
```

---

#### SetColumnWidth

```go
func (e *BasicExporter) SetColumnWidth(col string, width float64)
```

Sets the width for a specific column.

**Parameters:**
- `col`: Column identifier (e.g., "A", "B")
- `width`: Column width in Excel units

**Example:**
```go
exporter.SetColumnWidth("A", 15.0)
exporter.SetColumnWidth("B", 30.0)
```

**Recommended widths:** short codes/IDs 8-10, numbers/dates 10-15,
names/categories 15-25, descriptions 30-50. Or skip this entirely and
use `excel.ShouldAutoSize(true)` (see [Available Options](#available-options)).

---

#### SetNumberFormat

```go
func (e *BasicExporter) SetNumberFormat(col string, fmtID int)
```

Sets the number format for a specific column by merging `fmtID` into
that column's existing style from `SetColumnStyle`, or creating a new
column style if none is set yet. Only takes effect for `FileFormatXLSX`
exports. See [Format Constants](#format-constants) below for common
built-in format IDs.

**Example:**
```go
exporter.SetNumberFormat("C", excel.FormatCurrency)
exporter.SetNumberFormat("D", excel.FormatPercentage)
```

---

#### On

```go
func (e *BasicExporter) On(eventType excel.EventType, hook excel.Hook)
```

Registers a hook for a specific event type.

**Parameters:**
- `eventType`: Type of event (`BeforeExport`, `AfterExport`, `BeforeSheetExport`, `AfterSheetExport`, `BeforeRowExport`, `AfterRowExport`, `OnError`)
- `hook`: Hook function to call when event occurs

**Example:**
```go
exporter.On(excel.BeforeExport, func(ctx context.Context, e *excel.Event) error {
    log.Println("Starting export...")
    return nil
})
```

---

### BasicImporter

#### NewImporter

```go
func NewImporter(opts ...Option) *BasicImporter
```

Creates a new basic importer with the given options.

**Parameters:**
- `opts`: Optional configuration options (see [Options](#config--options))

**Returns:**
- `*BasicImporter`: Configured importer instance

**Example:**
```go
importer := importer.NewImporter(excel.WithHeadings(true))
```

---

#### NewImporterWithFile

```go
func NewImporterWithFile(file *excelize.File, opts ...Option) *BasicImporter
```

Creates a new importer with an existing excelize file instance.

**Parameters:**
- `file`: Existing excelize file instance
- `opts`: Optional configuration options

**Returns:**
- `*BasicImporter`: Configured importer instance

---

#### Import

```go
func (i *BasicImporter) Import(ctx context.Context) error
```

Performs the core import operation. Reads data from the configured file.

**Parameters:**
- `ctx`: Context for cancellation and timeout

**Returns:**
- `error`: Error if import fails, nil on success

**Note:** After Import(), use `GetRows()` or `GetHeadings()` to access data.

---

#### ImportFromFile

```go
func (i *BasicImporter) ImportFromFile(ctx context.Context, filename string) error
```

Imports data from a file into the importer's internal state. Unlike
`Import`, this also opens the file for you. The format is derived from
the extension: `.csv` reads CSV, anything else (including no extension)
reads XLSX via excelize.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `filename`: Path to Excel or CSV file

**Returns:**
- `error`: Error if import fails, nil on success

**Note:** This returns only an error. Call `GetRows()` and `GetHeadings()`
afterward to retrieve the imported data.

**Example:**
```go
if err := importer.ImportFromFile(ctx, "input.xlsx"); err != nil {
    log.Fatal(err)
}
data := importer.GetRows()
```

---

#### ImportFromReader

```go
func (i *BasicImporter) ImportFromReader(ctx context.Context, r io.Reader) error
```

Imports data from an io.Reader. Framework-agnostic method for HTTP requests. There is no filename here for the format to be inferred from, so it reads whatever `excel.WithFormat` selected (default: `FileFormatXLSX`); pass `excel.WithFormat(excel.FileFormatCSV)` to read CSV instead.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `r`: io.Reader source (e.g., http.Request.Body)

**Returns:**
- `error`: Error if import fails, nil on success

**Note:** This returns only an error. Call `GetRows()` afterward to
retrieve the imported data.

**Example:**
```go
if err := importer.ImportFromReader(ctx, request.Body); err != nil {
    log.Fatal(err)
}
data := importer.GetRows()
```

---

#### ImportFromSheet

```go
func (i *BasicImporter) ImportFromSheet(ctx context.Context, filename string, sheetName string) ([][]any, error)
```

Imports data from a specific sheet in an Excel file.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `filename`: Path to Excel file
- `sheetName`: Name of the sheet to import from

**Returns:**
- `[][]any`: Imported data rows
- `error`: Error if import fails, nil on success

**Example:**
```go
rows, err := importer.ImportFromSheet(ctx, "report.xlsx", "Users")
```

---

#### ImportAllSheets

```go
func (i *BasicImporter) ImportAllSheets(ctx context.Context, filename string) (map[string][][]any, error)
```

Imports data from all sheets in an Excel file. CSV has no concept of
multiple sheets, so a `.csv` filename degrades to a single-entry map
keyed by the configured sheet name (`"Sheet1"` by default) and logs a
warning (once, via `excel.WithLogger`) instead of failing.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `filename`: Path to Excel or CSV file

**Returns:**
- `map[string][][]any`: Map of sheet names to data rows
- `error`: Error if import fails, nil on success

**Example:**
```go
sheetsData, err := importer.ImportAllSheets(ctx, "report.xlsx")
usersData := sheetsData["Users"]
ordersData := sheetsData["Orders"]
```

---

#### ImportToStructs

```go
func ImportToStructs[T any](ctx context.Context, filename string, opts ...Option) ([]T, error)
```

Imports data directly to Go structs with automatic type conversion and validation.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `filename`: Path to Excel file

**Type Parameters:**
- `T`: Struct type to import into (must have excel tags)

**Returns:**
- `[]T`: Slice of imported structs
- `error`: Error if import fails, nil on success

**Example:**
```go
type User struct {
    ID    int    `excel:"id,required"`
    Name  string `excel:"name,required"`
    Email string `excel:"email"`
}

users, err := importer.ImportToStructs[User](ctx, "users.xlsx")
```

---

#### GetHeadings

```go
func (i *BasicImporter) GetHeadings() []string
```

Returns the column headings extracted from the first row.

**Returns:**
- `[]string`: Column headings, empty if WithHeadings was false

**Example:**
```go
headings := importer.GetHeadings()
// ["Name", "Age", "Email"]
```

---

#### GetSheetNames

```go
func (i *BasicImporter) GetSheetNames() ([]string, error)
```

Returns all sheet names in the file already opened on this importer (via
`Import`, `ImportFromFile`, or `NewImporterWithFile`). Takes no filename;
it reads from the importer's own state. If the most recent import was
CSV, this reports a single logical sheet named after the configured
sheet name instead of erroring.

**Returns:**
- `[]string`: List of sheet names
- `error`: Error if the underlying file cannot be read

**Example:**
```go
if err := importer.ImportFromFile(ctx, "report.xlsx"); err != nil {
    log.Fatal(err)
}
names, err := importer.GetSheetNames()
// ["Users", "Orders", "Products"]
```

---

#### GetSheetCount

```go
func (i *BasicImporter) GetSheetCount() (int, error)
```

Returns the number of sheets in the file already opened on this importer.
Takes no filename; see `GetSheetNames` above.

**Returns:**
- `int`: Number of sheets
- `error`: Error if the underlying file cannot be read

---

#### SheetExists

```go
func (i *BasicImporter) SheetExists(sheetName string) (bool, error)
```

Checks if a sheet exists in the file already opened on this importer.
Takes no filename; see `GetSheetNames` above.

**Parameters:**
- `sheetName`: Name of sheet to check

**Returns:**
- `bool`: True if sheet exists
- `error`: Error if the underlying file cannot be read (a missing sheet
  is not itself an error; it is reported as `false, nil`)

---

#### On

```go
func (i *BasicImporter) On(eventType excel.EventType, hook excel.Hook)
```

Registers a hook for a specific event type.

**Parameters:**
- `eventType`: Type of event (`BeforeImport`, `AfterImport`, `BeforeSheetImport`, `AfterSheetImport`, `BeforeRowImport`, `AfterRowImport`, `OnError`)
- `hook`: Hook function to call when event occurs

---

### Config & Options

#### DefaultConfig

```go
package config // github.com/bonarizki-dat/go-excel/excel/config

func DefaultConfig() *Config
```

Lives in the `excel/config` subpackage, not `excel` itself; call it as
`config.DefaultConfig()`. Returns a new Config with default values.

**Returns:**
- `*config.Config`: Default configuration

---

#### Option

```go
type Option func(*config.Config)
```

Function type for configuring exporters/importers.

**Available Options** (all in the `excel` package, all apply via `config.Config`):

- `WithSheetName(name string)` - Set sheet name
- `WithHeadings(enabled bool)` - Enable/disable header row handling
- `WithRows(first, last int)` - 1-based, inclusive data-row range (row 1 is the first data row, independent of `WithHeadings`); `last == 0` means through the end of the sheet
- `WithColumns(first, last int)` - 1-based, inclusive column range; `last == 0` means through the end of the row
- `WithSkipEmptyRows(skip bool)` - Skip empty rows during import
- `WithTrimSpaces(trim bool)` - Trim whitespace from cell values
- `WithValidation(enabled bool)` - Enable struct-tag validation during `ImportToStructs` (requires `min:`/`max:`/`pattern:` tags)
- `WithStopOnError(stop bool)` - Stop on first error vs. collect all errors
- `WithTimeout(timeout time.Duration)` - Wrap the operation's context with a deadline; `0` means no timeout
- `WithPreallocation(enabled bool)` - Preallocate row slices based on expected row count
- `WithLogger(logger Logger)` - Set a logger for debug/info messages
- `WithMetrics(metrics Metric)` - Set a metrics collector for monitoring
- `WithProgress(callback func(processed, total int64))` - Periodic progress callback
- `WithErrorHandler(callback func(row int, err error) bool)` - Per-error callback; return `true` to continue, `false` to stop
- `WithConfig(cfg *config.Config)` - Replace the entire configuration with a custom one
- `WithFormat(format FileFormat)` - Select the document format (`FileFormatXLSX` or `FileFormatCSV`) for `ExportToWriter`/`ExportToBuffer`/`ImportFromReader`, which have no filename to infer it from; `ExportToFile`/`ImportFromFile` derive it from the extension instead and ignore this option
- `WithCSVDelimiter(r rune)` - Set the CSV field delimiter (default: `,`); rejected at validation if it is a newline, carriage return, or double quote
- `WithCSVBOM(enabled bool)` - Enable/disable the UTF-8 byte order mark on CSV output (default: enabled)
- `WithCSVLineEnding(ending string)` - Set the CSV line ending, e.g. `"\r\n"` or `"\n"` (default: `"\r\n"`)
- `WithCSVTimeLayout(layout string)` - Set the `time.Time` format layout used for CSV cells (default: `"2006-01-02 15:04:05"`)
- `WithConverters(map[string]Converter)` - Register per-column `Converter` functions (`func(raw string, dest reflect.Type) (any, error)`), keyed by column name, consulted by `RowMapper.mapField` during struct-mapped import instead of `ConvertToType` for that column
- `WithRowMap(fn func(row []any) ([]any, error))` - Transform each data row (not the header row) before `Export` writes it; an error aborts the export
- `WithCustomStartCell(cell string)` - Shift XLSX export to start writing at `cell` (e.g. `"B3"`) instead of `A1`; row/column numbers used for progress and error reporting are unaffected. Ignored for CSV export
- `ShouldAutoSize(enabled bool)` - Widen each XLSX column to fit its longest cell's character length, applied once every row for a sheet has been written; an explicit `SetColumnWidth` always wins over the measured width. Ignored for CSV export
- `WithProperties(props DocProperties)` - Set a subset of the exported workbook's OOXML core document properties (`Title`, `Subject`, `Creator`, `Keywords`, `Description`, `Category`, `Version`) via excelize's `SetDocProps`. Ignored for CSV export
- `WithStrictFormat(strict bool)` - When `true`, exporting to CSV with an XLSX-only feature configured (styles, column widths, a custom start cell, auto-sized columns, document properties, or more than one sheet passed to `ExportToSheets`) fails with an error wrapping `ErrStrictFormatUnsupported` instead of dropping that feature and logging a one-time warning via `Logger`. Has no effect on XLSX operations. Default: `false` (degrade and warn)

There is no `WithChunkSize`, `WithBatchSize`, `WithWorkerCount`, or `WithBufferSize` — these were removed from `excel/config` as unused/dead configuration (see [CHANGELOG.md](../CHANGELOG.md)). `WithCSVDelimiter` and `WithEncoding` were removed for the same reason but have since returned: `WithCSVDelimiter` is a real functional option again (see above), and `WithEncoding` was superseded by the more general `WithFormat`/CSV option set rather than reintroduced under its old name. Worker count for the `stream` package is set separately via `stream.WithExportWorkers`/`stream.WithImportWorkers`.

**Example:**
```go
exp := exporter.NewExporter(data,
    excel.WithSheetName("Data"),
    excel.WithHeadings(true),
    excel.WithTimeout(30*time.Second),
)
```

---

### Events

#### EventType

```go
type EventType int
```

Constants for event types:

- `BeforeExport` - Before export starts
- `AfterExport` - After export completes
- `BeforeSheetExport` - Before each sheet export
- `AfterSheetExport` - After each sheet export
- `BeforeRowExport` - Before each row export
- `AfterRowExport` - After each row export
- `BeforeImport` - Before import starts
- `AfterImport` - After import completes
- `BeforeSheetImport` - Before each sheet import
- `AfterSheetImport` - After each sheet import
- `BeforeRowImport` - Before each row import
- `AfterRowImport` - After each row import
- `OnError` - When an error occurs

---

#### Event

```go
type Event struct {
    Type  EventType
    Data  any   // Event-specific data
    Error error // Error (for OnError events)
}
```

Contains event context and data.

---

#### Hook

```go
type Hook func(ctx context.Context, event *Event) error
```

Function type for event hooks. Return error to cancel operation.

**Example:**
```go
exporter.On(excel.BeforeRowExport, func(ctx context.Context, e *excel.Event) error {
    row := e.Data.([]any)
    // Validate or transform row
    return nil
})
```

---

EventDispatcher, the type behind `On`, lives at `internal/events` as of
`v0.3.0` and is not part of this module's public API: there is no
constructor to call directly, since `NewExporter`/`NewImporter` always
create one internally and only expose it through `On`.

---

### Style

#### NewStyle

```go
func NewStyle() *Style
```

Creates a new Style builder.

**Returns:**
- `*Style`: Style builder instance

**Example:**
```go
style := excel.NewStyle().
    Bold(true).
    FontSize(12).
    BackgroundColor("#4472C4").
    FontColor("#FFFFFF").
    BorderAll(excel.BorderMedium)
```

---

#### Style Builder Methods

```go
func (s *Style) Bold(bold bool) *Style
func (s *Style) Italic(italic bool) *Style
func (s *Style) FontSize(size float64) *Style
func (s *Style) FontColor(color string) *Style
func (s *Style) FontFamily(family string) *Style
func (s *Style) BackgroundColor(color string) *Style
func (s *Style) BorderAll(style string) *Style
func (s *Style) BorderColor(color string) *Style
func (s *Style) AlignHorizontal(align string) *Style
func (s *Style) AlignVertical(align string) *Style
func (s *Style) WrapText(wrap bool) *Style
func (s *Style) Format(fmtID int) *Style
func (s *Style) FillPattern(n int) *Style
```

All methods return `*Style` for method chaining. `BorderAll` takes a plain
`string` (see [Border Style Constants](#border-style-constants) below) —
there is no `BorderStyle` type, and there is no way to set individual
sides (`BorderTop`/`BorderBottom`/`BorderLeft`/`BorderRight` do not
exist); use `BorderAll` for now (see [Limitations](#limitations) below).

`FillPattern(n int)` sets the excelize fill pattern index (0-18; 1 is a
solid fill, the default from `NewStyle()`). Has no effect unless a fill
color is also set.

---

#### Color Constants

```go
const (
    ColorBlack = "#000000"
    ColorWhite = "#FFFFFF"
    ColorRed   = "#FF0000"
    ColorBlue  = "#0000FF"
    ColorGreen = "#00FF00"
    ColorGray  = "#808080"
)
```

Convenience constants for `FontColor`, `BackgroundColor`, and
`BorderColor`; any hex string works too (e.g. `"#4472C4"`).

---

#### Alignment Constants

```go
const (
    AlignLeft   = "left"
    AlignCenter = "center"
    AlignRight  = "right"

    AlignTop    = "top"
    AlignMiddle = "center" // excelize's vertical-center value
    AlignBottom = "bottom"
)
```

Passed to `AlignHorizontal` / `AlignVertical`.

---

#### Format Constants

```go
const (
    FormatGeneral    = 0   // General (default)
    FormatNumber     = 1   // "0"
    FormatDecimal    = 2   // "0.00"
    FormatCurrency   = 164 // Custom currency format "$#,##0.00"
    FormatPercentage = 10  // "0.00%"
    FormatDate       = 14  // "mm-dd-yy"
    FormatTime       = 20  // "h:mm"
    FormatDateTime   = 22  // "m/d/yy h:mm"
    FormatText       = 49  // "@"
)
```

Number format constants for `Format()`.

---

#### Border Style Constants

```go
const (
    BorderNone   = "none"
    BorderThin   = "thin"
    BorderMedium = "medium"
    BorderThick  = "thick"
    BorderDashed = "dashed"
    BorderDotted = "dotted"
)
```

Plain `string` constants (not a distinct `BorderStyle` type) passed to `BorderAll()`.

---

#### Merge Cells

No built-in wrapper; use `GetFile()` to reach the underlying `*excelize.File`:

```go
exp := exporter.NewExporter(data)
exp.Export(ctx)

file := exp.GetFile()
sheetName := exp.GetConfig().SheetName

if err := file.MergeCell(sheetName, "A1", "F1"); err != nil {
    log.Fatal(err)
}
file.SetCellValue(sheetName, "A1", "Report Title") // use the top-left cell
file.SaveAs("output.xlsx")
```

See [examples/merge_cells/main.go](../examples/merge_cells/main.go).

---

#### Style Best Practices

- Define color/width constants once per application rather than repeating hex
  strings.
- Reuse `*Style` values across columns instead of rebuilding an equivalent
  style per call — excelize caches by style ID, but constructing the same
  `Style` repeatedly still costs allocations.
- Prefer `ShouldAutoSize(true)` over guessing `SetColumnWidth` values, unless
  a fixed layout is required.
- Apply number formats through `SetColumnStyle`/`SetNumberFormat`, not by
  formatting the value itself into a string.
- Keep header borders heavier (`BorderMedium`) than data borders
  (`BorderThin`) so headers stay visually distinct.

#### Limitations

- Individual border sides (`BorderTop`/`BorderBottom`/`BorderLeft`/`BorderRight`)
  are not supported; `BorderAll` is the only option.
- Conditional formatting is not available; use `GetFile()` for direct
  excelize access if needed.
- Merge cells requires `GetFile()` (see above); there is no wrapper method.

---

### MultiSheet

#### Sheet

```go
type Sheet struct {
    Name        string          // Sheet name (max 31 chars)
    Data        [][]any         // Sheet data
    DataChannel <-chan []any    // Optional: streaming data channel
    Config      *config.Config  // Optional: sheet-specific config
    Index       int             // Optional: sheet order index
}
```

Represents a single sheet in a multi-sheet export.

---

#### SheetCollection

```go
type SheetCollection struct {
    // ...
}

func NewSheetCollection() *SheetCollection
func (sc *SheetCollection) Add(sheet *Sheet) error
func (sc *SheetCollection) Get(name string) *Sheet
func (sc *SheetCollection) GetByIndex(index int) *Sheet
func (sc *SheetCollection) Exists(name string) bool
func (sc *SheetCollection) Count() int
func (sc *SheetCollection) All() []*Sheet
func (sc *SheetCollection) Names() []string
func (sc *SheetCollection) Clear()
```

Manages multiple sheets for export. `Get` and `GetByIndex` return `nil`
(not a `(value, bool)` pair) when the sheet is not found.

---

#### MultiSheetExportOptions

```go
type MultiSheetExportOptions struct {
    CreateDefaultSheet bool // Create empty sheet if no sheets provided
    DefaultSheetName   string // Name for default sheet
    SkipEmptySheets    bool // Skip sheets with no data
    ContinueOnError    bool // Continue exporting other sheets on error
    ProgressCallback   func(ctx context.Context, index int, name string) error // Called after each sheet is exported
}
```

Options for multi-sheet export.

---

#### DefaultMultiSheetExportOptions

```go
func DefaultMultiSheetExportOptions() MultiSheetExportOptions
```

Returns default multi-sheet export options.

---

### Struct Mapping

#### ImportToStructs

See [BasicImporter.ImportToStructs](#importtostructs) above. `required`
is always enforced. `min:`/`max:`/`pattern:` are only enforced when
validation is turned on, e.g.:

```go
users, err := importer.ImportToStructs[User](ctx, "users.xlsx", excel.WithValidation(true))
```

**Struct Tag Format:**

```go
type User struct {
    ID        int       `excel:"id,required"`
    Name      string    `excel:"name,required,min:3,max:100"`
    Email     string    `excel:"email,pattern:^[a-z@.]+$"`
    Age       int       `excel:"age,min:18,max:100"`
    Salary    float64   `excel:"salary,min:0"`
    Active    bool      `excel:"active"`
    CreatedAt time.Time `excel:"created_at"`
}
```

**Tag Options:**
- `required` - Field must not be empty
- `omitempty` - On export, a zero-value field (`0`, `""`, `false`, nil, and so on) renders as a blank cell instead of its literal value. Import-side behavior is unaffected.
- `min:N` - Minimum value (for numeric types, enforced only with `WithValidation(true)`)
- `max:N` - Maximum value (for numeric types, enforced only with `WithValidation(true)`)
- `pattern:regex` - Regex validation against the raw string value, enforced only with `WithValidation(true)`

There is no `format:` tag option; `time.Time` fields are parsed by trying
a fixed list of formats (`time.RFC3339`, `2006-01-02`,
`2006-01-02 15:04:05`, `02-01-2006`, `01/02/2006`, `2006/01/02`,
`01/02/06 15:04`, `1/2/06 15:04`, and a couple more — see
`excel/converter.go`'s `convertToTime`), not a per-field layout.

**Supported Type Conversions:**
- string → int, int64, int32, uint, uint64
- string → float32, float64
- string → time.Time (tries several fixed formats, see above)
- string → bool ("true"/"false", "yes"/"no", "1"/"0")

---

### Interfaces

`excel/interfaces.go` declares only the types below. Each has a real
implementation elsewhere in the module; this list is not a set of
extension points to implement speculatively.

#### RowSink

```go
type RowSink interface {
    WriteRow(ctx context.Context, rowNum int, row []any, isHeader bool) error
    Finalize(ctx context.Context, w io.Writer) error
    Close() error
}
```

Produces a document in one on-disk format (XLSX, CSV) from a sequence of
rows. `BasicExporter` writes through a `RowSink` instead of calling
`excelize` directly; implemented by `xlsxSink` and `csvSink` in
`excel/exporter`.

---

#### FileSaver

```go
type FileSaver interface {
    SaveAs(name string) error
}
```

Optional capability of a `RowSink` for formats with a native "save to
path" operation. `ExportToFile` checks for this interface and writes
directly to disk when present, falling back to `Finalize` otherwise.

---

#### Exportable

```go
type Exportable interface {
    Data() any
}
```

Implemented by a caller's type to hand its data to `NewExporter` without
that type needing to be a raw `[][]any` or struct slice itself.

---

#### Logger

```go
type Logger = config.Logger
```

Alias for `config.Logger`. Implement `Debug`/`Info`/`Warn`/`Error` and
pass the value to `WithLogger`, `WithStreamLogger`, or `WithImportLogger`.

---

#### Metric

```go
type Metric = config.Metric
```

Alias for `config.Metric`. Implement `Inc`/`Gauge`/`Histogram`/`Timing`
and pass the value to `WithMetrics`.

---

### Error Types

#### ExportError

```go
type ExportError struct {
    Op    string
    Sheet string
    Row   int
    Col   int
    Err   error
}

func NewExportError(op, sheet string, row, col int, err error) error
```

Error for export operations.

---

#### ImportError

```go
type ImportError struct {
    Op    string
    Sheet string
    Row   int
    Col   int
    Err   error
}

func NewImportError(op, sheet string, row, col int, err error) error
```

Error for import operations.

---

#### ValidationError

```go
type ValidationError struct {
    Row     int
    Field   string
    Value   any
    Message string
    Err     error // underlying cause, if any; unwraps via errors.Is/errors.As
}

func NewValidationError(row int, field string, value any, message string, cause error) error
```

Error for validation failures. `cause` is the underlying error that
triggered the failure (e.g. a type conversion error), or `nil` when the
message is self-contained.

---

#### ConfigError

```go
type ConfigError struct {
    Field   string
    Value   any
    Message string
}

func NewConfigError(field string, value any, message string) error
```

Error for configuration issues.

---

#### StreamError

```go
type StreamError struct {
    Op  string
    Row int
    Err error
}

func NewStreamError(op string, row int, err error) error
```

Error for streaming operations.

---

## Package: stream

### StreamExporter

#### NewStreamExporter

```go
func NewStreamExporter(writer io.Writer, opts ...StreamExportOption) (*StreamExporter, error)
```

Creates a new streaming exporter for large files.

**Parameters:**
- `writer`: Output destination (file, HTTP response, etc.)
- `opts`: Optional configuration options

**Returns:**
- `*StreamExporter`: Configured streaming exporter
- `error`: Non-nil if `WithExportSheetName` set a name excelize's
  `NewSheet` rejects, instead of silently falling back to the default
  sheet name

**Example:**
```go
file, _ := os.Create("output.xlsx")
defer file.Close()

exporter, err := stream.NewStreamExporter(file,
    stream.WithExportWorkers(8),
    stream.WithExportHeadings([]string{"Name", "Email"}),
)
if err != nil {
    log.Fatal(err)
}
```

---

#### ExportStream

```go
func (se *StreamExporter) ExportStream(ctx context.Context, rows <-chan []any) error
```

Exports rows from a channel to an Excel file. Memory usage scales with
worker count, not dataset size, but is not O(1)/constant relative to
dataset size overall — see [stream package godoc](../stream/doc.go) and
[Memory Requirements](../README.md#-memory-requirements).

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `rows`: Channel of rows to export (each row is []any)

**Returns:**
- `error`: Error if export fails

**Example:**
```go
rows := make(chan []any, 1000)

go func() {
    defer close(rows)
    for i := 0; i < 1000000; i++ {
        rows <- []any{fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@example.com", i)}
    }
}()

err := exporter.ExportStream(ctx, rows)
```

---

#### ExportToSheets

```go
func (se *StreamExporter) ExportToSheets(ctx context.Context, sheets []excel.Sheet, options excel.MultiSheetExportOptions) error
```

Exports multiple sheets to the `io.Writer` the exporter was constructed with, giving each sheet its own excelize `StreamWriter` so rows can be consumed from per-sheet channels instead of collected up front.

This is a different write path from `ExportStream`, with two consequences: the transformer set by `WithRowTransform` does not run, and neither does the worker pool, so `WithExportWorkers` has no effect here. With `WithExportFormat(config.FileFormatCSV)` only the first sheet is written, with one warning through `WithExportLogger`.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `sheets`: Slice of Sheet structs; each may carry `Data` or `DataChannel`
- `options`: MultiSheetExportOptions

**Returns:**
- `error`: Error if export fails

---

#### StreamExportOption Functions

```go
func WithExportWorkers(count int) StreamExportOption
func WithRowTransform(transform RowTransformer) StreamExportOption
func WithExportHeadings(headings []string) StreamExportOption
func WithExportSheetName(name string) StreamExportOption
func WithExportProgress(callback ProgressCallback, interval uint64) StreamExportOption
func WithExportFormat(format config.FileFormat) StreamExportOption
func WithExportCSVOptions(opts config.CSVOptions) StreamExportOption
func WithExportLogger(logger excel.Logger) StreamExportOption
func WithExportStrictFormat(strict bool) StreamExportOption
```

RowTransformer is defined alongside these options:

```go
type RowTransformer func(ctx context.Context, rowNum uint64, row []any) ([]any, error)
```

Configuration options for StreamExporter. `WithExportWorkers` only
matters when a `RowTransformer` is set via `WithRowTransform`; without a
transformer, rows still write through a single ordered writer goroutine.
`WithExportFormat` selects `FileFormatXLSX` (default) or `FileFormatCSV`
for `ExportStream` and `ExportToSheets`; `WithExportCSVOptions` overrides
CSV delimiter/BOM/line-ending/time-layout settings when the format is
CSV. `WithExportLogger` sets the logger used to warn once when an
XLSX-only feature (styles, column widths, multiple sheets) is dropped
because the format is CSV — without it, degradation happens silently.
`WithExportStrictFormat(true)` replaces that warning with an error
wrapping `excel.ErrStrictFormatUnsupported` when `ExportToSheets` is
given more than one sheet for a CSV export. Default: `false` (degrade
and warn).

---

### StreamImporter

#### NewStreamImporter

```go
func NewStreamImporter(reader io.Reader, opts ...StreamImportOption) (*StreamImporter, error)
```

Creates a new streaming importer for large files.

**Parameters:**
- `reader`: Input source (file, HTTP request body, etc.)
- `opts`: Optional configuration options

**Returns:**
- `*StreamImporter`: Configured streaming importer
- `error`: Error if creation fails

**Example:**
```go
file, _ := os.Open("large.xlsx")
defer file.Close()

importer, err := stream.NewStreamImporter(file,
    stream.WithImportWorkers(8),
    stream.WithImportHeadings(),
)
```

---

#### ImportStream

```go
func (si *StreamImporter) ImportStream(ctx context.Context, processor RowProcessor) error
```

Imports rows from a file using streaming row-by-row processing. Note:
`StreamImporter` still loads the whole workbook via `excelize.OpenReader`
when it opens the file, so this is not O(1)/constant memory relative to
file size — see [stream package godoc](../stream/doc.go).

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `processor`: Function to process each row

**Returns:**
- `error`: Error if import fails

**Example:**
```go
processor := func(ctx context.Context, rowNum int, row []any) ([]any, error) {
    log.Printf("Row %d: %v", rowNum, row)
    return row, nil
}

err := importer.ImportStream(ctx, processor)
```

---

#### ImportFromSheet

```go
func (si *StreamImporter) ImportFromSheet(ctx context.Context, filename string, sheetName string, rowHandler func([]any) error) error
```

Imports from a specific sheet using streaming. The filename is opened
independently of the reader passed to `NewStreamImporter`, so this does
not disturb `si`'s own file or sheet. The same row filters as
`ImportStream` apply (`WithImportHeadings`, `WithImportRows`,
`WithImportSkipEmptyRows`); the heading row is skipped when headings are
enabled but is not recorded into `GetHeadings()`. CSV files are
rejected. Note `rowHandler` here is a plain `func([]any) error`, not
the `RowProcessor` type used by `ImportStream` — it takes only the row
(no context or row number) and cannot modify the row before further
processing.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `filename`: Path to Excel file
- `sheetName`: Name of sheet to import
- `rowHandler`: Function called with each row's data

**Returns:**
- `error`: Error if import fails

---

#### ImportAllSheets

```go
func (si *StreamImporter) ImportAllSheets(ctx context.Context, filename string, sheetHandler func(string) error, rowHandler func([]any) error) error
```

Imports from all sheets using streaming, applying the same filters as
`ImportFromSheet`. Stops at the first handler or decode error.
`rowHandler` is the same plain `func([]any) error` as `ImportFromSheet`,
not `RowProcessor`. CSV files are rejected.

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `filename`: Path to Excel file
- `sheetHandler`: Function called for each sheet (optional, can be nil)
- `rowHandler`: Function to process each row

**Returns:**
- `error`: Error if import fails

---

#### GetSheetNames

```go
func (si *StreamImporter) GetSheetNames(filename string) ([]string, error)
```

Returns all sheet names in an Excel file, without touching `si`'s own
file or sheet. CSV files are rejected.

**Parameters:**
- `filename`: Path to Excel file

**Returns:**
- `[]string`: List of sheet names
- `error`: Error if file cannot be read

---

#### StreamImportOption Functions

```go
func WithImportWorkers(count int) StreamImportOption
func WithImportHeadings() StreamImportOption
func WithImportSheetName(name string) StreamImportOption
func WithImportSkipEmptyRows(skip bool) StreamImportOption
func WithImportRows(start, end int) StreamImportOption
func WithImportProgress(callback ProgressCallback, interval uint64) StreamImportOption
func WithImportFormat(format config.FileFormat) StreamImportOption
func WithImportCSVOptions(opts config.CSVOptions) StreamImportOption
func WithImportLogger(logger excel.Logger) StreamImportOption
```

Configuration options for StreamImporter. `WithImportFormat` selects
`FileFormatXLSX` (default) or `FileFormatCSV` for the reader passed to
`NewStreamImporter`; `WithImportCSVOptions` overrides CSV delimiter
settings when the format is CSV. `WithImportLogger` sets the logger used
to warn about XLSX-only features (such as multiple sheets) that a CSV
input cannot provide.

---

#### RowProcessor

```go
type RowProcessor func(ctx context.Context, rowNum int, row []any) ([]any, error)
```

Function type for processing rows during streaming import.

**Parameters:**
- `ctx`: Context (for cancellation)
- `rowNum`: A 1-based sequential counter over the rows handed to
  processors, **not** the row's absolute position in the sheet: rows
  skipped by `WithImportRows` or `WithImportSkipEmptyRows`, or the
  heading row when `WithImportHeadings` is set, are never counted
- `row`: Row data

**Returns:**
- `[]any`: Processed row (can be modified)
- `error`: Error to stop import, nil to continue

---

### WorkerStats

```go
type WorkerStats struct {
    TasksSubmitted uint64
    TasksCompleted uint64
    TasksFailed    uint64
    TotalWorkers   int
}
```

`StreamExporter.GetStats()` and `StreamImporter.GetStats()` return a
snapshot of the underlying worker pool's statistics, or the zero
`WorkerStats` if `ExportStream`/`ImportStream` has not yet run.

**Example:**
```go
stats := importer.GetStats()
log.Printf("submitted=%d completed=%d failed=%d workers=%d",
    stats.TasksSubmitted, stats.TasksCompleted, stats.TasksFailed, stats.TotalWorkers)
```

As of `v0.3.0`, `WorkerPool`, `PoolManager`, `Task`, `Result`,
`ProcessFunc`, and `WorkerError` — the primitives `StreamExporter` and
`StreamImporter` are built from — moved to `internal/concurrency` and
are no longer part of this module's public API. `WorkerStats` is the
one exception, kept as a type alias (`stream.WorkerStats =
concurrency.WorkerStats`) because `GetStats()` returns it. See
CHANGELOG.md.

---

## Additional Resources

- [Performance Guide](PERFORMANCE_GUIDE.md) - Performance optimization guide
- [Style section](#style) above - Styling and formatting, including best practices and limitations
- [Examples](../examples/) - Code examples
