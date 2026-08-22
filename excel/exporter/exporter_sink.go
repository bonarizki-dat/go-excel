package exporter

import (
	"fmt"
	"strings"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// Sink construction and teardown for BasicExporter, kept separate from
// Export() so the format-routing details do not mix into the public
// entry point.

// newSink builds the excel.RowSink for the exporter's configured
// format, using the style and column-width state accumulated via
// SetHeaderStyle/SetDataStyle/SetColumnStyle/SetColumnWidth.
func (e *BasicExporter) newSink() (excel.RowSink, error) {
	if e.config.Format == config.FileFormatCSV {
		return newCSVSink(e.config.CSV)
	}
	return newXLSXSink(e.config.SheetName, e.headerStyle, e.dataStyle, e.columnStyles, e.columnWidths, e.config.StartCell, e.config.AutoSizeColumns, e.config.DocProperties)
}

// warnCSVDegradation reports any XLSX-only features configured on this
// exporter that a CSV sink cannot honor. Called only once newSink() has
// confirmed the sink is not *xlsxSink.
//
// With WithStrictFormat(true), a non-empty feature list is returned as
// an error wrapping excel.ErrStrictFormatUnsupported instead of being
// dropped. Otherwise (the default) it is logged once per Export() call
// via Logger, if one is set, and the export proceeds with those
// features silently ignored.
func (e *BasicExporter) warnCSVDegradation() error {
	var dropped []string
	if e.headerStyle != nil || e.dataStyle != nil || len(e.columnStyles) > 0 {
		dropped = append(dropped, "styles")
	}
	if len(e.columnWidths) > 0 {
		dropped = append(dropped, "column widths")
	}
	if e.config.StartCell != "" {
		dropped = append(dropped, "start cell")
	}
	if e.config.AutoSizeColumns {
		dropped = append(dropped, "auto-sized columns")
	}
	if e.config.DocProperties != (excel.DocProperties{}) {
		dropped = append(dropped, "document properties")
	}
	if len(dropped) == 0 {
		return nil
	}

	if e.config.StrictFormat {
		return fmt.Errorf("%w: CSV export cannot honor: %s", excel.ErrStrictFormatUnsupported, strings.Join(dropped, ", "))
	}

	if e.config.Logger != nil {
		e.config.Logger.Warn("CSV export ignores unsupported features", "features", dropped)
	}
	return nil
}

// closeCurrent closes whatever resource a previous Export()-family call
// allocated. e.file is authoritative when set: it is the same
// excelize.File an xlsxSink holds, and ExportToSheets (which does not go
// through the sink abstraction) also assigns directly to e.file, so
// checking e.file first closes both cases with a single Close() call.
// Only when e.file is nil (a previous non-XLSX sink, e.g. CSV) does
// closing fall to the sink itself.
func (e *BasicExporter) closeCurrent() error {
	if e.file != nil {
		return e.file.Close()
	}
	if e.sink != nil {
		return e.sink.Close()
	}
	return nil
}
