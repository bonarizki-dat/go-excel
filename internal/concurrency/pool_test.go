package concurrency

import (
	"testing"
)

// TestNewPoolManager tests that PoolManager is initialized correctly.
func TestNewPoolManager(t *testing.T) {
	// Create new pool manager instance
	pm := NewPoolManager()

	// Verify pool manager is not nil
	if pm == nil {
		t.Fatal("NewPoolManager returned nil")
	}

	// Verify all internal pools are initialized
	if pm.rowPool == nil {
		t.Error("rowPool is nil")
	}
	if pm.cellPool == nil {
		t.Error("cellPool is nil")
	}
	if pm.builderPool == nil {
		t.Error("builderPool is nil")
	}
	if pm.bufferPool == nil {
		t.Error("bufferPool is nil")
	}
	if pm.largeBufferPool == nil {
		t.Error("largeBufferPool is nil")
	}
}

// TestPoolManager_GetRow tests retrieving rows from the row pool.
func TestPoolManager_GetRow(t *testing.T) {
	// Create pool manager for testing
	pm := NewPoolManager()

	// Test retrieving a row from the pool
	row := pm.GetRow()

	// Row should not be nil, have zero length, and capacity of 100
	if row == nil {
		t.Fatal("GetRow returned nil")
	}
	if len(row) != 0 {
		t.Errorf("GetRow returned row with length %d, expected 0", len(row))
	}
	if cap(row) != 100 {
		t.Errorf("GetRow returned row with capacity %d, expected 100", cap(row))
	}
}

// TestPoolManager_PutRow tests returning rows to the row pool.
func TestPoolManager_PutRow(t *testing.T) {
	// Create pool manager for testing
	pm := NewPoolManager()

	// Get a row from pool
	row := pm.GetRow()

	// Add some data to the row
	row = append(row, "value1", "value2", 123, true)

	// Verify row has data
	if len(row) != 4 {
		t.Fatalf("Row length is %d, expected 4", len(row))
	}

	// Return row to pool (should clear data)
	pm.PutRow(row)

	// Get another row from pool (should be the same buffer, cleared)
	row2 := pm.GetRow()

	// Row should have zero length after being returned to pool
	if len(row2) != 0 {
		t.Errorf("Row from pool has length %d, expected 0", len(row2))
	}

	// Capacity should be preserved
	if cap(row2) < 100 {
		t.Errorf("Row capacity is %d, expected >= 100", cap(row2))
	}
}

// TestPoolManager_GetCellBuffer tests retrieving cell buffers from pool.
func TestPoolManager_GetCellBuffer(t *testing.T) {
	// Create pool manager for testing
	pm := NewPoolManager()

	// Get cell buffer from pool
	buf := pm.GetCellBuffer()

	// Buffer should not be nil, have zero length, and 1KB capacity
	if buf == nil {
		t.Fatal("GetCellBuffer returned nil")
	}
	if len(buf) != 0 {
		t.Errorf("GetCellBuffer returned buffer with length %d, expected 0", len(buf))
	}
	if cap(buf) != 1024 {
		t.Errorf("GetCellBuffer returned buffer with capacity %d, expected 1024", cap(buf))
	}
}

// TestPoolManager_PutCellBuffer tests returning cell buffers to pool.
func TestPoolManager_PutCellBuffer(t *testing.T) {
	// Create pool manager for testing
	pm := NewPoolManager()

	// Get buffer from pool
	buf := pm.GetCellBuffer()

	// Write some data to buffer
	buf = append(buf, []byte("test data")...)

	// Verify buffer has data
	if len(buf) == 0 {
		t.Fatal("Buffer should have data")
	}

	// Return buffer to pool
	pm.PutCellBuffer(buf)

	// Get another buffer (should be same buffer, cleared)
	buf2 := pm.GetCellBuffer()

	// Buffer should be empty after being returned to pool
	if len(buf2) != 0 {
		t.Errorf("Buffer from pool has length %d, expected 0", len(buf2))
	}

	// Capacity should be preserved
	if cap(buf2) < 1024 {
		t.Errorf("Buffer capacity is %d, expected >= 1024", cap(buf2))
	}
}
