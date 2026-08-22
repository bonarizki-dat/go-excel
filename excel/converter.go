package excel

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ConvertToType attempts to convert a string value to the target Go type.
// It supports basic types: string, int, float, bool, and time.Time.
func ConvertToType(value, targetType string) (any, error) {
	value = strings.TrimSpace(value)

	switch targetType {
	case "string":
		return value, nil
	case "int", "int64", "int32", "int16", "int8":
		return convertToInt(value)
	case "uint", "uint64", "uint32", "uint16", "uint8":
		return convertToUint(value)
	case "float64", "float32":
		return convertToFloat(value)
	case "bool":
		return convertToBool(value)
	case "time.Time":
		return convertToTime(value)
	default:
		return nil, fmt.Errorf("%w: unsupported type: %s", ErrTypeMismatch, targetType)
	}
}

// convertToInt converts a string to an int64.
func convertToInt(value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	// Handle float-like strings (e.g. "123.00") by parsing as float first
	if strings.Contains(value, ".") {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, err
		}
		return int64(f), nil
	}
	return strconv.ParseInt(value, 10, 64)
}

// convertToUint converts a string to a uint64.
func convertToUint(value string) (uint64, error) {
	if value == "" {
		return 0, nil
	}
	if strings.Contains(value, ".") {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, err
		}
		if f < 0 {
			return 0, fmt.Errorf("%w: cannot convert negative value to uint", ErrTypeMismatch)
		}
		return uint64(f), nil
	}
	return strconv.ParseUint(value, 10, 64)
}

// convertToFloat converts a string to a float64.
func convertToFloat(value string) (float64, error) {
	if value == "" {
		return 0.0, nil
	}
	// Handle currency symbols or commas if simple
	clean := strings.ReplaceAll(value, ",", "")
	clean = strings.ReplaceAll(clean, "$", "")
	return strconv.ParseFloat(clean, 64)
}

// convertToBool converts a string to a boolean.
// Supports: "true", "1", "yes", "on" (case insensitive).
func convertToBool(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	lower := strings.ToLower(value)
	switch lower {
	case "true", "1", "yes", "on", "y", "t":
		return true, nil
	case "false", "0", "no", "off", "n", "f":
		return false, nil
	}
	return false, fmt.Errorf("%w: invalid boolean value: %s", ErrTypeMismatch, value)
}

// convertToTime converts a string to time.Time.
// Tries multiple common formats.
func convertToTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"02-01-2006",
		"01/02/2006",
		"2006/01/02",
		"01/02/06 15:04",
		"1/2/06 15:04",
		"01-02-06 15:04",
		"2006-01-02 15:04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("%w: could not parse time: %s", ErrTypeMismatch, value)
}
