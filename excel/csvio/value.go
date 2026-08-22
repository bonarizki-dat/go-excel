// Package csvio implements CSV encoding and decoding shared by
// excel/exporter, excel/importer, and stream. It exists as its own
// package, rather than living inside exporter or importer, so both the
// batch and streaming code paths use one implementation instead of two
// diverging ones.
package csvio

import (
	"fmt"
	"strconv"
	"time"
)

// FormatValue renders v as the string that will be written to a CSV
// field. timeLayout formats time.Time values; pass an empty string to
// fall back to time.RFC3339.
//
// nil becomes an empty field, matching how a blank Excel cell reads
// back as "". Floats use strconv.FormatFloat with the 'f' verb and -1
// precision instead of fmt's default %v, which switches to scientific
// notation (e.g. "1.234568e+06") for large magnitudes - a spreadsheet
// opening the CSV would otherwise show exponent notation where the
// source data had a plain number.
func FormatValue(v any, timeLayout string) string { //nolint:gocyclo // flat type switch over primitive kinds, not meaningfully complex
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case bool:
		return strconv.FormatBool(val)
	case time.Time:
		layout := timeLayout
		if layout == "" {
			layout = time.RFC3339
		}
		return val.Format(layout)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.FormatInt(int64(val), 10)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", val)
	}
}
