package config

import "fmt"

// ConfigError represents a configuration validation error. Used when
// configuration values are invalid. The name is intentional (see
// CODING_STANDARDS.md naming decisions): it is the single, canonical error
// type for the config package, also aliased as excel.ConfigError.
type ConfigError struct { //nolint:revive // stutters intentionally, see comment above
	// Field is the configuration field that is invalid
	Field string

	// Value is the invalid value
	Value any

	// Message is the error message
	Message string
}

// Error implements the error interface
// Returns a formatted configuration error message.
func (e *ConfigError) Error() string {
	// Include field and value in error message
	return fmt.Sprintf("config error: field '%s' (value: %v): %s",
		e.Field, e.Value, e.Message)
}

// NewConfigError creates a new ConfigError with the given details
// Exported for use by validation functions.
func NewConfigError(field string, value any, message string) error {
	return &ConfigError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}
