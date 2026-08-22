package excel_test

import (
	"testing"
	"time"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// stubLogger and stubMetric are minimal config.Logger/config.Metric
// implementations used only to verify WithLogger/WithMetrics assign the
// given instance; none of their methods are expected to be called here.
type stubLogger struct{}

func (stubLogger) Debug(string, ...any) {}
func (stubLogger) Info(string, ...any)  {}
func (stubLogger) Warn(string, ...any)  {}
func (stubLogger) Error(string, ...any) {}

type stubMetric struct{}

func (stubMetric) Inc(string, int64)         {}
func (stubMetric) Gauge(string, float64)     {}
func (stubMetric) Histogram(string, float64) {}
func (stubMetric) Timing(string, int64)      {}

// TestOptions tests all option functions using table-driven approach
// Each test case applies an option and verifies the config is modified correctly.
func TestOptions(t *testing.T) {
	tests := []struct {
		name   string                           // Test case name
		option excel.Option                     // Option to apply
		verify func(*testing.T, *config.Config) // Function to verify the config
	}{
		{
			name:   "excel.WithSheetName",
			option: excel.WithSheetName("CustomSheet"),
			verify: func(t *testing.T, c *config.Config) {
				if c.SheetName != "CustomSheet" {
					t.Errorf("SheetName = %v, want CustomSheet", c.SheetName)
				}
			},
		},
		{
			name:   "excel.WithHeadings false",
			option: excel.WithHeadings(false),
			verify: func(t *testing.T, c *config.Config) {
				if c.WithHeadings != false {
					t.Error("WithHeadings should be false")
				}
			},
		},
		{
			name:   "excel.WithRows valid",
			option: excel.WithRows(11, 100),
			verify: func(t *testing.T, c *config.Config) {
				if c.StartRow != 10 {
					t.Errorf("StartRow = %v, want 10", c.StartRow)
				}
				if c.EndRow != 100 {
					t.Errorf("EndRow = %v, want 100", c.EndRow)
				}
			},
		},
		{
			name:   "excel.WithColumns valid",
			option: excel.WithColumns(3, 20),
			verify: func(t *testing.T, c *config.Config) {
				if c.StartCol != 2 {
					t.Errorf("StartCol = %v, want 2", c.StartCol)
				}
				if c.EndCol != 20 {
					t.Errorf("EndCol = %v, want 20", c.EndCol)
				}
			},
		},
		{
			name:   "excel.WithSkipEmptyRows false",
			option: excel.WithSkipEmptyRows(false),
			verify: func(t *testing.T, c *config.Config) {
				if c.SkipEmptyRows != false {
					t.Error("SkipEmptyRows should be false")
				}
			},
		},
		{
			name:   "excel.WithTrimSpaces false",
			option: excel.WithTrimSpaces(false),
			verify: func(t *testing.T, c *config.Config) {
				if c.TrimSpaces != false {
					t.Error("TrimSpaces should be false")
				}
			},
		},
		{
			name:   "excel.WithValidation true",
			option: excel.WithValidation(true),
			verify: func(t *testing.T, c *config.Config) {
				if !c.ValidateData {
					t.Error("ValidateData should be true")
				}
			},
		},
		{
			name:   "excel.WithStopOnError false",
			option: excel.WithStopOnError(false),
			verify: func(t *testing.T, c *config.Config) {
				if c.StopOnError != false {
					t.Error("StopOnError should be false")
				}
			},
		},
		{
			name:   "excel.WithTimeout",
			option: excel.WithTimeout(30 * time.Second),
			verify: func(t *testing.T, c *config.Config) {
				if c.Timeout != 30*time.Second {
					t.Errorf("Timeout = %v, want 30s", c.Timeout)
				}
			},
		},
		{
			name:   "excel.WithPreallocation false",
			option: excel.WithPreallocation(false),
			verify: func(t *testing.T, c *config.Config) {
				if c.PreallocateRows != false {
					t.Error("PreallocateRows should be false")
				}
			},
		},
		{
			name:   "excel.WithCSVLineEnding",
			option: excel.WithCSVLineEnding("\n"),
			verify: func(t *testing.T, c *config.Config) {
				if c.CSV.LineEnding != "\n" {
					t.Errorf("LineEnding = %q, want \\n", c.CSV.LineEnding)
				}
			},
		},
		{
			name:   "excel.WithCSVTimeLayout",
			option: excel.WithCSVTimeLayout("2006-01-02"),
			verify: func(t *testing.T, c *config.Config) {
				if c.CSV.TimeLayout != "2006-01-02" {
					t.Errorf("TimeLayout = %q, want 2006-01-02", c.CSV.TimeLayout)
				}
			},
		},
		{
			name:   "excel.WithLogger",
			option: excel.WithLogger(stubLogger{}),
			verify: func(t *testing.T, c *config.Config) {
				if _, ok := c.Logger.(stubLogger); !ok {
					t.Errorf("Logger = %T, want stubLogger", c.Logger)
				}
			},
		},
		{
			name:   "excel.WithMetrics",
			option: excel.WithMetrics(stubMetric{}),
			verify: func(t *testing.T, c *config.Config) {
				if _, ok := c.Metrics.(stubMetric); !ok {
					t.Errorf("Metrics = %T, want stubMetric", c.Metrics)
				}
			},
		},
		{
			name:   "excel.WithFormat",
			option: excel.WithFormat(config.FileFormatCSV),
			verify: func(t *testing.T, c *config.Config) {
				if c.Format != config.FileFormatCSV {
					t.Errorf("Format = %v, want FileFormatCSV", c.Format)
				}
			},
		},
		{
			name:   "excel.WithCSVDelimiter",
			option: excel.WithCSVDelimiter(';'),
			verify: func(t *testing.T, c *config.Config) {
				if c.CSV.Delimiter != ';' {
					t.Errorf("Delimiter = %q, want ';'", c.CSV.Delimiter)
				}
			},
		},
		{
			name:   "excel.WithCSVBOM false",
			option: excel.WithCSVBOM(false),
			verify: func(t *testing.T, c *config.Config) {
				if c.CSV.UseBOM {
					t.Error("UseBOM should be false")
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with default config
			cfg := config.DefaultConfig()
			// Apply the option
			tt.option(cfg)
			// Verify the result
			tt.verify(t, cfg)
		})
	}
}
