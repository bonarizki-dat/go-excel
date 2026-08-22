/*
Package excel provides shared types (config, options, events, styling,
struct-tag mapping) used by the excel/exporter and excel/importer
subpackages for Excel import/export.

# Overview

The exporter and importer live in their own subpackages so that each can
depend on excelize without forcing that dependency onto callers who only
need the shared types. This package is framework-agnostic: it works with
any io.Writer/io.Reader, so it integrates with Gin, Echo, Fiber, or the
standard library equally.

# Basic Usage

Export data to Excel:

	data := [][]interface{}{
		{"Name", "Age", "Email"},
		{"John Doe", 30, "john@example.com"},
		{"Jane Smith", 25, "jane@example.com"},
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true))
	if err := exp.ExportToFile(ctx, "output.xlsx"); err != nil {
		log.Fatal(err)
	}

Import data from Excel:

	imp := importer.NewImporter(excel.WithHeadings(true))
	if err := imp.ImportFromFile(ctx, "input.xlsx"); err != nil {
		log.Fatal(err)
	}
	rows := imp.GetRows()
	headings := imp.GetHeadings()

# Streaming for Large Files

For files with millions of rows, use the stream package to process rows
one at a time instead of loading the whole sheet into memory:

	importer, err := stream.NewStreamImporter(reader, stream.WithImportWorkers(8))
	if err != nil {
		log.Fatal(err)
	}
	defer importer.Close()

	err = importer.ImportStream(ctx, func(ctx context.Context, rowNum int, row []interface{}) ([]interface{}, error) {
		// process row
		return row, nil
	})

See the stream package documentation for the corresponding streaming
export API.

# Framework Integration

The exporter writes to any io.Writer, so serving a download requires only
setting the response headers before calling ExportToWriter:

	// Standard library
	func exportHandler(w http.ResponseWriter, r *http.Request) {
		exp := exporter.NewExporter(data, excel.WithHeadings(true))
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=export.xlsx")
		if err := exp.ExportToWriter(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	// Gin
	func exportHandler(c *gin.Context) {
		exp := exporter.NewExporter(data, excel.WithHeadings(true))
		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", "attachment; filename=export.xlsx")
		if err := exp.ExportToWriter(c.Request.Context(), c.Writer); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
	}

# Custom Data Sources

Any value can be exported directly if it implements Exportable, or if it
is a struct/slice of structs (converted via reflection):

	type UserExport struct {
		Users []User
	}

	func (u *UserExport) Data() any {
		return u.Users
	}

	exp := exporter.NewExporter(&UserExport{Users: users})
	if err := exp.Export(ctx); err != nil {
		log.Fatal(err)
	}

# Struct Mapping

ImportToStructs maps rows directly onto a Go struct type using `excel`
tags, with optional min/max/pattern validation enforced when the
importer's ValidateData option is enabled:

	type User struct {
		ID    int    `excel:"id,required"`
		Name  string `excel:"name,required,min:3,max:100"`
		Email string `excel:"email,pattern:^[^@]+@[^@]+$"`
	}

	users, err := importer.ImportToStructs[User](ctx, "users.xlsx", excel.WithValidation(true))

# Events

Exporters and importers accept hooks for BeforeExport, AfterExport,
BeforeRowExport, AfterRowExport, BeforeRowImport, and AfterRowImport via
On(eventType, hook). A hook returning an error aborts the operation.

# Context Support

All long-running operations accept context.Context for cancellation, and
Config.Timeout (via WithTimeout) wraps the operation in its own deadline:

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := exp.Export(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("export timeout")
		}
		return err
	}

# Concurrency

BasicExporter and BasicImporter are not safe for concurrent use: each
instance owns a single in-progress excelize.File and mutable internal
state (buffered errors, row counters). Use one instance per operation, or
guard shared instances with your own synchronization.

# See Also

For runnable examples, see the examples/ directory in the repository.
*/
package excel
