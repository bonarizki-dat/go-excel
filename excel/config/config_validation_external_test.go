package config_test

import (
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestConfigValidate_Invalid tests validation with invalid configurations
// Uses table-driven tests to cover all validation rules.
func TestConfigValidate_Invalid(t *testing.T) {
	// Define test cases for invalid configurations
	tests := []struct {
		name      string               // Test case name
		setup     func(*config.Config) // Function to modify config to make it invalid
		wantField string               // Expected field name in error
	}{
		{
			name: "empty sheet name",
			setup: func(c *config.Config) {
				c.SheetName = "" // Set to empty string
			},
			wantField: "SheetName",
		},
		{
			name: "negative start row",
			setup: func(c *config.Config) {
				c.StartRow = -1 // Set to negative (invalid)
			},
			wantField: "StartRow",
		},
		{
			name: "end row before start row",
			setup: func(c *config.Config) {
				c.StartRow = 100 // Start at row 100
				c.EndRow = 50    // End before start (invalid)
			},
			wantField: "EndRow",
		},
		{
			name: "negative start column",
			setup: func(c *config.Config) {
				c.StartCol = -1 // Set to negative (invalid)
			},
			wantField: "StartCol",
		},
		{
			name: "end column before start column",
			setup: func(c *config.Config) {
				c.StartCol = 10 // Start at column 10
				c.EndCol = 5    // End before start (invalid)
			},
			wantField: "EndCol",
		},
	}

	// Run all test cases
	for _, tt := range tests {
		// Create subtest for each case
		t.Run(tt.name, func(t *testing.T) {
			// Start with valid default config
			cfg := config.DefaultConfig()
			// Apply modification to make it invalid
			tt.setup(cfg)
			// Validate should return error
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			// Check if error is config.ConfigError type
			configErr := &config.ConfigError{}
			ok := errors.As(err, &configErr)
			if !ok {
				t.Fatalf("expected *config.ConfigError, got %T", err)
			}
			// Verify error contains expected field name
			if configErr.Field != tt.wantField {
				t.Errorf("error field = %v, want %v", configErr.Field, tt.wantField)
			}
		})
	}
}
