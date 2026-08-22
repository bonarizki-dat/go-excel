package exporter

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/bonarizki-dat/go-excel/excel"
)

// Input-to-row conversion for BasicExporter: type switches, Exportable
// unwrapping, and the reflection fallback used for structs.

// maxExportableDepth bounds how many times prepareData will unwrap a
// nested excel.Exportable before giving up. Without a limit, a Data()
// implementation that returns itself (directly or through a cycle of
// Exportable values) would recurse until the goroutine stack overflows.
const maxExportableDepth = 10

// isEmptyExportData reports whether data should be treated as empty
// input to Export: a nil interface, or a non-nil slice/array/map with
// zero elements. Anything else (structs, scalars, pointers to either)
// is left for prepareData to interpret.
func isEmptyExportData(data any) bool {
	if data == nil {
		return true
	}

	val := reflect.ValueOf(data)
	switch val.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return val.Len() == 0
	default:
		return false
	}
}

// prepareData converts the input data to a uniform row format, where
// each row is []interface{}.
func (e *BasicExporter) prepareData(ctx context.Context) ([][]any, error) {
	return e.prepareDataAtDepth(ctx, 0)
}

// prepareDataAtDepth is the recursive implementation behind prepareData;
// depth counts how many Exportable.Data() calls have been unwrapped so
// far in this chain.
func (e *BasicExporter) prepareDataAtDepth(ctx context.Context, depth int) ([][]any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if depth > maxExportableDepth {
		return nil, fmt.Errorf("excel.Exportable nesting exceeds max depth of %d; check for a Data() implementation that returns itself or forms a cycle", maxExportableDepth)
	}

	switch v := e.data.(type) {
	case [][]any:
		return v, nil

	case []any:
		return e.convertSliceToRows(v)

	case excel.Exportable:
		e.data = v.Data()
		return e.prepareDataAtDepth(ctx, depth+1)

	default:
		return e.convertUsingReflection(v)
	}
}

// convertSliceToRows converts []interface{} to [][]interface{}, where
// each element becomes a single-cell row.
func (e *BasicExporter) convertSliceToRows(slice []any) ([][]any, error) {
	var rows [][]any
	if e.config.PreallocateRows && len(slice) > 0 {
		rows = make([][]any, 0, len(slice))
	}

	for _, item := range slice {
		rows = append(rows, []any{item})
	}

	return rows, nil
}

// convertUsingReflection is the fallback for data types not explicitly
// handled by prepareDataAtDepth: pointers are dereferenced, slices and
// arrays become one row per element, structs become a single row, and
// anything else becomes a single-cell row.
func (e *BasicExporter) convertUsingReflection(data any) ([][]any, error) {
	val := reflect.ValueOf(data)

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		var rows [][]any
		if e.config.PreallocateRows && val.Len() > 0 {
			rows = make([][]any, 0, val.Len())
		}

		for i := 0; i < val.Len(); i++ {
			item := val.Index(i)
			row, err := e.reflectStructToRow(item)
			if err != nil {
				return nil, err
			}
			rows = append(rows, row)
		}
		return rows, nil
	}

	if val.Kind() == reflect.Struct {
		row, err := e.reflectStructToRow(val)
		if err != nil {
			return nil, err
		}
		return [][]any{row}, nil
	}

	return [][]any{{data}}, nil
}

// reflectStructToRow converts a struct (or pointer to struct) to a row,
// one cell per exported, non-`excel:"-"` field; embedded structs are
// flattened into the parent row instead of becoming a nested cell.
func (e *BasicExporter) reflectStructToRow(item reflect.Value) ([]any, error) {
	if item.Kind() == reflect.Pointer {
		item = item.Elem()
	}

	if item.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct or pointer to struct, got %s", item.Kind())
	}

	var row []any
	itemType := item.Type()

	for i := 0; i < item.NumField(); i++ {
		field := itemType.Field(i)
		fieldValue := item.Field(i)

		if !field.IsExported() {
			continue
		}

		tagValue, hasTag := field.Tag.Lookup("excel")
		if hasTag && tagValue == "-" {
			continue
		}

		if isEmbeddedStructField(&field, fieldValue) {
			embeddedRow, err := e.reflectStructToRow(fieldValue)
			if err != nil {
				return nil, err
			}
			row = append(row, embeddedRow...)
			continue
		}

		if hasOmitEmptyTag(tagValue) && fieldValue.IsZero() {
			row = append(row, nil)
			continue
		}

		row = append(row, fieldValue.Interface())
	}
	return row, nil
}

// isEmbeddedStructField reports whether field is an anonymous struct or
// pointer-to-struct field, in which case its columns are flattened into
// the parent row instead of being treated as a single cell value.
func isEmbeddedStructField(field *reflect.StructField, fieldValue reflect.Value) bool {
	if !field.Anonymous {
		return false
	}
	if fieldValue.Kind() == reflect.Struct {
		return true
	}
	return fieldValue.Kind() == reflect.Pointer && fieldValue.Elem().Kind() == reflect.Struct
}

// hasOmitEmptyTag reports whether tag's comma-separated `excel:"..."`
// options include "omitempty", mirroring encoding/json's option syntax.
// A zero-value field with this option renders as a blank cell instead
// of its literal zero value (0, "", false, and so on).
func hasOmitEmptyTag(tag string) bool {
	for _, part := range strings.Split(tag, ",") {
		if strings.TrimSpace(part) == "omitempty" {
			return true
		}
	}
	return false
}
