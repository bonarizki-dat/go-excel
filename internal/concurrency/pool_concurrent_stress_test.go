package concurrency

import (
	"testing"
)

func TestPoolManager_MemoryLeakPrevention(t *testing.T) {
	// Create pool manager for testing
	pm := NewPoolManager()

	// Create some objects that we'll store in row
	obj1 := &struct{ data string }{data: "test1"}
	obj2 := &struct{ data string }{data: "test2"}

	// Get row and add objects
	row := pm.GetRow()
	row = append(row, obj1, obj2)

	// Verify row has objects
	if len(row) != 2 {
		t.Fatalf("Row length is %d, expected 2", len(row))
	}

	// Return row to pool (should clear all references)
	pm.PutRow(row)

	// The references in the row should now be nil
	// We can't directly inspect the pooled row, but we can verify
	// that getting a new row doesn't contain old references
	row2 := pm.GetRow()

	// New row should be empty
	if len(row2) != 0 {
		t.Errorf("New row from pool has length %d, expected 0", len(row2))
	}

	// Append to new row and verify it doesn't contain old data
	row2 = append(row2, "new data")
	if len(row2) != 1 {
		t.Errorf("Row length is %d, expected 1", len(row2))
	}
	if row2[0] != "new data" {
		t.Errorf("Row contains %v, expected \"new data\"", row2[0])
	}
}

// TestPoolManager_MultipleGetPut tests multiple get/put cycles.
//
// This verifies that pools can handle multiple get/put operations
// in sequence without degrading or accumulating garbage.
func TestPoolManager_MultipleGetPut(t *testing.T) {
	// Create pool manager for testing
	pm := NewPoolManager()

	// Number of cycles to test
	const cycles = 100

	// Test row pool
	t.Run("RowPoolMultipleCycles", func(t *testing.T) {
		for i := range cycles {
			row := pm.GetRow()

			// Add some data
			row = append(row, i, "test", true)

			// Verify data is there
			if len(row) != 3 {
				t.Errorf("Cycle %d: row length is %d, expected 3", i, len(row))
			}

			// Return to pool
			pm.PutRow(row)
		}
	})

	// Test cell buffer pool
	t.Run("CellBufferMultipleCycles", func(t *testing.T) {
		for i := range cycles {
			buf := pm.GetCellBuffer()

			// Add some data
			buf = append(buf, byte(i))

			// Verify data is there
			if len(buf) != 1 {
				t.Errorf("Cycle %d: buffer length is %d, expected 1", i, len(buf))
			}

			// Return to pool
			pm.PutCellBuffer(buf)
		}
	})

	// Test builder pool
	t.Run("BuilderMultipleCycles", func(t *testing.T) {
		for i := range cycles {
			builder := pm.GetBuilder()

			// Write some data
			builder.WriteString("test")

			// Verify data is there
			if builder.Len() != 4 {
				t.Errorf("Cycle %d: builder length is %d, expected 4", i, builder.Len())
			}

			// Return to pool
			pm.PutBuilder(builder)
		}
	})

	// Test buffer pool
	t.Run("BufferMultipleCycles", func(t *testing.T) {
		for i := range cycles {
			buf := pm.GetBuffer()

			// Write some data
			buf.WriteString("test")

			// Verify data is there
			if buf.Len() != 4 {
				t.Errorf("Cycle %d: buffer length is %d, expected 4", i, buf.Len())
			}

			// Return to pool
			pm.PutBuffer(buf)
		}
	})

	// Test large buffer pool
	t.Run("LargeBufferMultipleCycles", func(t *testing.T) {
		for i := range cycles {
			buf := pm.GetLargeBuffer()

			// Add some data
			buf = append(buf, byte(i))

			// Verify data is there
			if len(buf) != 1 {
				t.Errorf("Cycle %d: buffer length is %d, expected 1", i, len(buf))
			}

			// Return to pool
			pm.PutLargeBuffer(buf)
		}
	})
}
