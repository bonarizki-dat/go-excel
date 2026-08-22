package exporter_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// MockLogger implements a simple logger for testing.
type MockLogger struct {
	infoCalls  int
	errorCalls int
	warnCalls  int
	warnMsgs   []string
	lastMsg    string
}

func (m *MockLogger) Debug(_ string, _ ...any) {
	// Debug messages for testing
}

func (m *MockLogger) Info(msg string, _ ...any) {
	m.infoCalls++
	m.lastMsg = msg
}

func (m *MockLogger) Warn(msg string, _ ...any) {
	m.warnCalls++
	m.warnMsgs = append(m.warnMsgs, msg)
}

func (m *MockLogger) Error(msg string, _ ...any) {
	m.errorCalls++
	m.lastMsg = msg
}

// MockMetrics implements a simple metrics collector for testing.
type MockMetrics struct {
	timings    map[string]int64
	counts     map[string]int64
	gauges     map[string]float64
	histograms map[string]float64
}

func NewMockMetrics() *MockMetrics {
	return &MockMetrics{
		timings:    make(map[string]int64),
		counts:     make(map[string]int64),
		gauges:     make(map[string]float64),
		histograms: make(map[string]float64),
	}
}

func (m *MockMetrics) Inc(key string, value int64) {
	m.counts[key] += value
}

func (m *MockMetrics) Gauge(key string, value float64) {
	m.gauges[key] = value
}

func (m *MockMetrics) Histogram(key string, value float64) {
	m.histograms[key] = value
}

func (m *MockMetrics) Timing(key string, value int64) {
	m.timings[key] = value
}

// TestExporter_ExportToFile_WithLogger tests ExportToFile with logger.
func TestExporter_ExportToFile_WithLogger(t *testing.T) {
	ctx := context.Background()

	data := [][]any{
		{"Name", "Age"},
		{"Alice", 30},
	}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_logger.xlsx")

	logger := &MockLogger{}

	// Create exporter with logger
	exp := exporter.NewExporter(data, excel.WithLogger(logger))

	err := exp.ExportToFile(ctx, filename)
	if err != nil {
		t.Fatalf("ExportToFile with logger failed: %v", err)
	}

	// Verify logger was called
	if logger.infoCalls == 0 {
		t.Error("Logger Info should be called")
	}
}

// TestExporter_ExportToFile_WithMetrics tests ExportToFile with metrics.
func TestExporter_ExportToFile_WithMetrics(t *testing.T) {
	ctx := context.Background()

	data := [][]any{
		{"Product", "Price"},
		{"Widget", 9.99},
		{"Gadget", 19.99},
	}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_metrics.xlsx")

	metrics := NewMockMetrics()

	// Create exporter with metrics
	exp := exporter.NewExporter(data, excel.WithMetrics(metrics))

	err := exp.ExportToFile(ctx, filename)
	if err != nil {
		t.Fatalf("ExportToFile with metrics failed: %v", err)
	}

	// Verify metrics were recorded
	if _, ok := metrics.timings["export.duration"]; !ok {
		t.Error("Metrics should record export.duration")
	}
	if _, ok := metrics.counts["export.rows"]; !ok {
		t.Error("Metrics should record export.rows")
	}
	if _, ok := metrics.counts["export.success"]; !ok {
		t.Error("Metrics should record export.success")
	}

	// Verify row count (total rows in data, not just data rows)
	rowCount := metrics.counts["export.rows"]
	if rowCount != 3 {
		t.Errorf("Metrics export.rows = %d, want 3", rowCount)
	}
}

// TestExporter_ExportToFile_WithLoggerAndMetrics tests both logger and metrics.
func TestExporter_ExportToFile_WithLoggerAndMetrics(t *testing.T) {
	ctx := context.Background()

	data := [][]any{{"test"}}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_both.xlsx")

	logger := &MockLogger{}
	metrics := NewMockMetrics()

	// Create exporter with both logger and metrics
	exp := exporter.NewExporter(data,
		excel.WithLogger(logger),
		excel.WithMetrics(metrics))

	err := exp.ExportToFile(ctx, filename)
	if err != nil {
		t.Fatalf("ExportToFile failed: %v", err)
	}

	// Verify both were used
	if logger.infoCalls == 0 {
		t.Error("Logger should be called")
	}
	if len(metrics.timings) == 0 {
		t.Error("Metrics should be recorded")
	}
}

// TestExporter_ExportToWriter_WithLogger tests ExportToWriter with logger.
func TestExporter_ExportToWriter_WithLogger(t *testing.T) {
	ctx := context.Background()

	data := [][]any{
		{"ID", "Value"},
		{1, "test"},
	}

	var buf bytes.Buffer
	logger := &MockLogger{}

	// Create exporter with logger
	exp := exporter.NewExporter(data, excel.WithLogger(logger))

	err := exp.ExportToWriter(ctx, &buf)
	if err != nil {
		t.Fatalf("ExportToWriter with logger failed: %v", err)
	}

	// Verify logger was called
	if logger.infoCalls == 0 {
		t.Error("Logger Info should be called")
	}

	// Verify buffer has data
	if buf.Len() == 0 {
		t.Error("Buffer should not be empty")
	}
}

// TestExporter_ExportToWriter_WithMetrics tests ExportToWriter with metrics.
func TestExporter_ExportToWriter_WithMetrics(t *testing.T) {
	ctx := context.Background()

	data := [][]any{
		{"A", "B", "C"},
		{1, 2, 3},
		{4, 5, 6},
	}

	var buf bytes.Buffer
	metrics := NewMockMetrics()

	// Create exporter with metrics
	exp := exporter.NewExporter(data, excel.WithMetrics(metrics))

	err := exp.ExportToWriter(ctx, &buf)
	if err != nil {
		t.Fatalf("ExportToWriter with metrics failed: %v", err)
	}

	// Verify metrics were recorded
	if _, ok := metrics.timings["export.duration"]; !ok {
		t.Error("Metrics should record export.duration")
	}
	// Row count should be 3 (all rows in data)
	if metrics.counts["export.rows"] != 3 {
		t.Errorf("Metrics export.rows = %d, want 3", metrics.counts["export.rows"])
	}
}

// TestExporter_ExportToBuffer_WithLogger tests ExportToBuffer with logger.
func TestExporter_ExportToBuffer_WithLogger(t *testing.T) {
	ctx := context.Background()

	data := [][]any{{"test", "data"}}

	logger := &MockLogger{}

	// Create exporter with logger
	exp := exporter.NewExporter(data, excel.WithLogger(logger))

	buf, err := exp.ExportToBuffer(ctx)
	if err != nil {
		t.Fatalf("ExportToBuffer with logger failed: %v", err)
	}

	// Verify logger was called
	if logger.infoCalls == 0 {
		t.Error("Logger Info should be called")
	}

	// Verify buffer
	if len(buf) == 0 {
		t.Error("Buffer should not be empty")
	}
}

// TestExporter_ExportToBuffer_WithMetrics tests ExportToBuffer with metrics.
func TestExporter_ExportToBuffer_WithMetrics(t *testing.T) {
	ctx := context.Background()

	data := [][]any{
		{"Header1", "Header2"},
		{"Value1", "Value2"},
		{"Value3", "Value4"},
	}

	metrics := NewMockMetrics()

	// Create exporter with metrics
	exp := exporter.NewExporter(data, excel.WithMetrics(metrics))

	buf, err := exp.ExportToBuffer(ctx)
	if err != nil {
		t.Fatalf("ExportToBuffer with metrics failed: %v", err)
	}

	// Verify metrics were recorded
	if _, ok := metrics.timings["export.duration"]; !ok {
		t.Error("Metrics should record export.duration")
	}
	if _, ok := metrics.counts["export.bytes"]; !ok {
		t.Error("Metrics should record export.bytes")
	}

	// Verify byte count matches buffer length
	byteCount := metrics.counts["export.bytes"]
	if byteCount != int64(len(buf)) {
		t.Errorf("Metrics export.bytes = %d, want %d", byteCount, len(buf))
	}
}
