package exporter

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/bonarizki-dat/go-excel/excel"
)

// ExportToFile exports data to a file with the given filename.
// This is a convenience method that calls Export() and then saves to file.
// The format is derived from filename's extension: ".csv" writes CSV,
// anything else (including no extension) writes XLSX via excelize -
// this overrides whatever WithFormat set, since the filename is the
// more specific signal when one is available. Use ExportToWriter or
// ExportToBuffer with WithFormat if you need to write CSV to a path
// that doesn't end in ".csv".
// Returns error if export fails or file cannot be saved.
func (e *BasicExporter) ExportToFile(ctx context.Context, filename string) error {
	if filename == "" {
		return excel.NewExportError("ExportToFile", e.config.SheetName, 0, 0,
			fmt.Errorf("filename cannot be empty"))
	}

	// The filename extension is the more specific signal when both it
	// and WithFormat are available, so it takes precedence here.
	e.config.Format = excel.FormatFromExtension(filename)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if e.config.Logger != nil {
		e.config.Logger.Info("Starting export to file",
			"filename", filename,
			"sheet", e.config.SheetName)
	}

	var startTime time.Time
	if e.config.Metrics != nil {
		startTime = time.Now()
	}

	if err := e.Export(ctx); err != nil {
		if e.config.Logger != nil {
			e.config.Logger.Error("Export failed", "error", err)
		}
		return fmt.Errorf("export failed: %w", err)
	}

	dir := filepath.Dir(filename)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return excel.NewExportError("MkdirAll", e.config.SheetName, 0, 0,
				fmt.Errorf("failed to create directory %s: %w", dir, err))
		}
	}

	// Sinks that implement FileSaver (xlsxSink) write directly to disk;
	// others (csvSink) fall back to Finalize into a newly created file.
	if err := e.saveSinkAs(ctx, filename); err != nil {
		if e.config.Logger != nil {
			e.config.Logger.Error("Failed to save file", "error", err)
		}
		return err
	}

	if e.config.Metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		e.config.Metrics.Timing("export.duration", duration)
		e.config.Metrics.Inc("export.rows", e.totalRows)
		e.config.Metrics.Inc("export.success", 1)
	}

	if e.config.Logger != nil {
		e.config.Logger.Info("Export completed successfully",
			"filename", filename,
			"rows", e.totalRows)
	}

	return nil
}

// ExportToWriter exports data to an io.Writer.
// This method makes the library framework-agnostic:
// - Gin: exporter.ExportToWriter(ctx, c.Writer)
// - Echo: exporter.ExportToWriter(ctx, c.Response().Writer)
// - Fiber: exporter.ExportToWriter(ctx, c.Response().BodyWriter())
// - stdlib: exporter.ExportToWriter(ctx, w)
// There is no filename here for the format to be inferred from, so it
// uses whatever excel.WithFormat set on the exporter (default:
// FileFormatXLSX).
// Returns error if export fails or writing fails.
func (e *BasicExporter) ExportToWriter(ctx context.Context, w io.Writer) error {
	if w == nil {
		return excel.NewExportError("ExportToWriter", e.config.SheetName, 0, 0,
			fmt.Errorf("writer cannot be nil"))
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if e.config.Logger != nil {
		e.config.Logger.Info("Starting export to writer",
			"sheet", e.config.SheetName)
	}

	var startTime time.Time
	if e.config.Metrics != nil {
		startTime = time.Now()
	}

	if err := e.Export(ctx); err != nil {
		if e.config.Logger != nil {
			e.config.Logger.Error("Export failed", "error", err)
		}
		return fmt.Errorf("export failed: %w", err)
	}

	if err := e.sink.Finalize(ctx, w); err != nil {
		if e.config.Logger != nil {
			e.config.Logger.Error("Failed to write to writer", "error", err)
		}
		return err
	}

	if e.config.Metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		e.config.Metrics.Timing("export.duration", duration)
		e.config.Metrics.Inc("export.rows", e.totalRows)
		e.config.Metrics.Inc("export.success", 1)
	}

	if e.config.Logger != nil {
		e.config.Logger.Info("Export to writer completed successfully",
			"rows", e.totalRows)
	}

	return nil
}

// ExportToBuffer exports data to a byte buffer.
// This is useful for in-memory operations or further processing. There
// is no filename here for the format to be inferred from, so it uses
// whatever excel.WithFormat set on the exporter (default: FileFormatXLSX).
// Returns the buffer contents and any error.
func (e *BasicExporter) ExportToBuffer(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if e.config.Logger != nil {
		e.config.Logger.Info("Starting export to buffer",
			"sheet", e.config.SheetName)
	}

	var startTime time.Time
	if e.config.Metrics != nil {
		startTime = time.Now()
	}

	if err := e.Export(ctx); err != nil {
		if e.config.Logger != nil {
			e.config.Logger.Error("Export failed", "error", err)
		}
		return nil, fmt.Errorf("export failed: %w", err)
	}

	var buf bytes.Buffer
	if err := e.sink.Finalize(ctx, &buf); err != nil {
		if e.config.Logger != nil {
			e.config.Logger.Error("Failed to write to buffer", "error", err)
		}
		return nil, err
	}

	if e.config.Metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		e.config.Metrics.Timing("export.duration", duration)
		e.config.Metrics.Inc("export.rows", e.totalRows)
		e.config.Metrics.Inc("export.bytes", int64(buf.Len()))
		e.config.Metrics.Inc("export.success", 1)
	}

	if e.config.Logger != nil {
		e.config.Logger.Info("Export to buffer completed successfully",
			"rows", e.totalRows,
			"bytes", buf.Len())
	}

	return buf.Bytes(), nil
}

// saveSinkAs writes the current sink's document to filename. Sinks that
// implement excel.FileSaver (xlsxSink) write directly to the path,
// matching the pre-CSV-support behavior of calling excelize's SaveAs
// exactly. Sinks without a native file writer (csvSink) fall back to
// creating the file and calling Finalize.
func (e *BasicExporter) saveSinkAs(ctx context.Context, filename string) error {
	if saver, ok := e.sink.(excel.FileSaver); ok {
		if err := saver.SaveAs(filename); err != nil {
			return excel.NewExportError("SaveAs", e.config.SheetName, 0, 0, err)
		}
		return nil
	}

	f, err := os.Create(filename)
	if err != nil {
		return excel.NewExportError("Create", e.config.SheetName, 0, 0, err)
	}
	defer func() {
		_ = f.Close()
	}()

	return e.sink.Finalize(ctx, f)
}
