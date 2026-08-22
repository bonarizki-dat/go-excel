package excel

import "errors"

// Sentinel errors checked with errors.Is.

// Common sentinel errors that can be checked with errors.Is.
//
// This set is intentionally small. Config- and validation-shaped
// failures use the typed *ConfigError and *ValidationError instead (see
// errors.go), context cancellation and deadlines are reported via the
// idiomatic ctx.Err() rather than a package sentinel, and a handful of
// former sentinels (ErrRowNotFound, ErrColumnNotFound,
// ErrPermissionDenied, ErrWriteFailed, ErrReadFailed) were removed
// because they either had no production call site or were too generic
// to be meaningful — nearly any I/O error could be described as a
// "write failed" or "read failed".
var (
	// ErrInvalidFormat indicates the file format is not supported, or an
	// operation was attempted against a format that does not support it
	// (e.g. requesting multiple sheets from a CSV source).
	ErrInvalidFormat = errors.New("invalid file format")

	// ErrEmptyData indicates no data was provided for export: either a
	// nil value, or a non-nil slice/array/map with zero elements.
	ErrEmptyData = errors.New("empty data provided")

	// ErrInvalidSheetName indicates a sheet name failed Excel's naming
	// rules (empty, too long, disallowed characters, leading/trailing
	// apostrophe) or collided with an existing sheet name.
	ErrInvalidSheetName = errors.New("invalid sheet name")

	// ErrSheetNotFound indicates the requested sheet does not exist in
	// the workbook.
	ErrSheetNotFound = errors.New("sheet not found")

	// ErrFileNotFound indicates the file does not exist on disk.
	ErrFileNotFound = errors.New("file not found")

	// ErrInvalidFile indicates an operation was attempted before a
	// source file was opened or set on the importer.
	ErrInvalidFile = errors.New("invalid file")

	// ErrTypeMismatch indicates a value could not be converted to the
	// requested target type.
	ErrTypeMismatch = errors.New("type mismatch")

	// ErrStrictFormatUnsupported indicates WithStrictFormat(true) (or
	// stream.WithExportStrictFormat(true)) is set and an XLSX-only
	// feature was configured for a CSV export: the export fails
	// instead of silently dropping that feature.
	ErrStrictFormatUnsupported = errors.New("feature unsupported by the selected export format")
)
