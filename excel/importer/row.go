package importer

import (
	"context"
	"strings"
)

// Per-row filtering used by the import pipeline.

// processRow applies column range filtering, trimming, and validation
// to a single row from the Excel sheet. Used by importRows in
// importer_pipeline.go.
func (i *BasicImporter) processRow(ctx context.Context, row []string) ([]any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Like StartRow/EndRow, StartCol and EndCol are absolute, 0-indexed
	// positions and EndCol is exclusive: columns in [StartCol, EndCol)
	// are kept.
	startCol := i.config.StartCol
	endCol := i.config.EndCol

	if startCol < 0 {
		startCol = 0
	}
	if startCol >= len(row) {
		return []any{}, nil
	}

	if endCol <= 0 || endCol > len(row) {
		endCol = len(row)
	}

	row = row[startCol:endCol]

	result := make([]any, len(row))
	for j, cell := range row {
		cellValue := cell
		if i.config.TrimSpaces {
			cellValue = strings.TrimSpace(cellValue)
		}
		result[j] = cellValue
	}

	return result, nil
}

// isEmptyRow reports whether every cell in row is nil or an empty
// string. Used by importRows in importer_pipeline.go.
func (i *BasicImporter) isEmptyRow(row []any) bool {
	if len(row) == 0 {
		return true
	}

	for _, cell := range row {
		if cell == nil {
			continue
		}

		if str, ok := cell.(string); ok {
			if str != "" {
				return false
			}
		} else {
			return false
		}
	}

	return true
}
