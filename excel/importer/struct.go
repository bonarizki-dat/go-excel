package importer

import (
	"context"
	"reflect"

	"github.com/bonarizki-dat/go-excel/excel"
)

// ImportToStructs imports data from an Excel file into a slice of structs.
// T must be a struct type.
// The file is read row by row, and each row is mapped to a new instance of T.
// Accepts the same options as NewImporter; in particular, WithValidation(true)
// enables enforcement of min:/max:/pattern: struct tag constraints.
func ImportToStructs[T any](ctx context.Context, filename string, opts ...Option) ([]T, error) {
	imp := NewImporter(opts...)
	defer func() {
		_ = imp.Close()
	}()

	// ImportFromFile loads the whole file into memory; see the stream
	// package for a row-at-a-time alternative on very large files.
	err := imp.ImportFromFile(ctx, filename)
	if err != nil {
		return nil, err
	}
	rows := imp.GetRows()

	if len(rows) == 0 {
		return []T{}, nil
	}

	headers := imp.GetHeadings()
	if len(headers) == 0 {
		return nil, excel.NewConfigError("headers", nil, "no headers found in file")
	}

	mapper := excel.NewRowMapper(headers)
	mapper.EnableValidation(imp.GetConfig().ValidateData)
	mapper.SetConverters(imp.GetConfig().Converters)

	var results []T

	for i, row := range rows {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		var item T

		if reflect.TypeOf(item).Kind() != reflect.Struct {
			return nil, excel.NewConfigError("type", reflect.TypeOf(item).String(), "ImportToStructs requires a struct type")
		}

		// MapRow takes a pointer since it sets fields via reflection.
		err := mapper.MapRow(row, &item)
		if err != nil {
			// Row index is 0-based and excludes the header row; +2
			// converts it to the 1-based row number a spreadsheet user
			// would see, accounting for the header row above it.
			return nil, excel.NewValidationError(i+2, "", nil, err.Error(), err)
		}

		results = append(results, item)
	}

	return results, nil
}
