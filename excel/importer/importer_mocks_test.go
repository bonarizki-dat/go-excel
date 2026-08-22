package importer

// Mock types for testing shared across test files

// mockLogger is a mock logger for testing.
type mockLogger struct {
	logs []string
}

func (m *mockLogger) Info(msg string, _ ...any) {
	m.logs = append(m.logs, "INFO: "+msg)
}

func (m *mockLogger) Debug(msg string, _ ...any) {
	m.logs = append(m.logs, "DEBUG: "+msg)
}

func (m *mockLogger) Error(msg string, _ ...any) {
	m.logs = append(m.logs, "ERROR: "+msg)
}

func (m *mockLogger) Warn(msg string, _ ...any) {
	m.logs = append(m.logs, "WARN: "+msg)
}

// mockMetrics is a mock metrics collector for testing.
type mockMetrics struct {
	timings map[string]int64
	counts  map[string]int64
}

func (m *mockMetrics) Timing(name string, value int64) {
	m.timings[name] = value
}

func (m *mockMetrics) Inc(name string, value int64) {
	m.counts[name] += value
}

func (m *mockMetrics) Gauge(_ string, _ float64) {
	// Not used in import
}

func (m *mockMetrics) Histogram(_ string, _ float64) {
	// Not used in import
}
