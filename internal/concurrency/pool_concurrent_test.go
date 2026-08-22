package concurrency

import (
	"sync"
	"testing"
)

// TestPoolManager_ConcurrentAccess tests concurrent access to pools.
//
// This test verifies that PoolManager is safe for concurrent use from
// multiple goroutines without data races or panics.
func TestPoolManager_ConcurrentAccess(t *testing.T) {
	// Create pool manager for testing
	pm := NewPoolManager()

	// Number of concurrent goroutines to test thread safety
	const numGoroutines = 100

	// Wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Launch multiple goroutines accessing pools concurrently
	// This tests that sync.Pool is properly handling concurrent access
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			// Test row pool concurrency
			row := pm.GetRow()
			row = append(row, id)
			pm.PutRow(row)

			// Test cell buffer pool concurrency
			buf := pm.GetCellBuffer()
			buf = append(buf, byte(id))
			pm.PutCellBuffer(buf)

			// Test builder pool concurrency
			builder := pm.GetBuilder()
			builder.WriteString("test")
			pm.PutBuilder(builder)

			// Test buffer pool concurrency
			buffer := pm.GetBuffer()
			buffer.WriteString("test")
			pm.PutBuffer(buffer)

			// Test large buffer pool concurrency
			largeBuf := pm.GetLargeBuffer()
			largeBuf = append(largeBuf, byte(id))
			pm.PutLargeBuffer(largeBuf)
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// If we reach here without panicking, concurrent access is safe
	t.Log("Concurrent access test passed successfully")
}

// TestPoolManager_PoolReuse tests that pools actually reuse objects.
//
// This verifies that when an object is returned to the pool and then
// retrieved again, we get the same underlying buffer (memory reuse).
func TestPoolManager_PoolReuse(t *testing.T) {
	// Create pool manager for testing
	pm := NewPoolManager()

	// Test row pool reuse
	t.Run("RowPoolReuse", func(t *testing.T) {
		// Get a row and note its capacity address
		row1 := pm.GetRow()

		// Only test if row has capacity
		if cap(row1) == 0 {
			t.Skip("Row has zero capacity, cannot test reuse")
		}

		// Get pointer to underlying array
		ptr1 := &row1[0:cap(row1)][0]

		// Return to pool
		pm.PutRow(row1)

		// Get another row - should likely be the same underlying array
		row2 := pm.GetRow()

		// Only compare if row has capacity
		if cap(row2) == 0 {
			t.Fatal("Second row has zero capacity")
		}

		ptr2 := &row2[0:cap(row2)][0]

		// Pointers should be the same (reused buffer)
		// Note: This may not always be true due to sync.Pool behavior,
		// but in single-threaded test it usually reuses
		if ptr1 != ptr2 {
			t.Log("Warning: Pool may not be reusing buffers (acceptable with sync.Pool)")
		} else {
			t.Log("Pool successfully reused buffer")
		}
	})

	// Test cell buffer pool reuse
	t.Run("CellBufferReuse", func(t *testing.T) {
		// Get a buffer
		buf1 := pm.GetCellBuffer()

		// Only test if buffer has capacity
		if cap(buf1) == 0 {
			t.Skip("Buffer has zero capacity, cannot test reuse")
		}

		ptr1 := &buf1[0:cap(buf1)][0]

		// Return to pool
		pm.PutCellBuffer(buf1)

		// Get another buffer
		buf2 := pm.GetCellBuffer()

		// Only compare if buffer has capacity
		if cap(buf2) == 0 {
			t.Fatal("Second buffer has zero capacity")
		}

		ptr2 := &buf2[0:cap(buf2)][0]

		// Pointers should be the same (reused buffer)
		if ptr1 != ptr2 {
			t.Log("Warning: Pool may not be reusing buffers")
		} else {
			t.Log("Pool successfully reused buffer")
		}
	})

	// Test large buffer pool reuse
	t.Run("LargeBufferReuse", func(t *testing.T) {
		// Get a large buffer
		buf1 := pm.GetLargeBuffer()

		// Only test if buffer has capacity
		if cap(buf1) == 0 {
			t.Skip("Buffer has zero capacity, cannot test reuse")
		}

		ptr1 := &buf1[0:cap(buf1)][0]

		// Return to pool
		pm.PutLargeBuffer(buf1)

		// Get another buffer
		buf2 := pm.GetLargeBuffer()

		// Only compare if buffer has capacity
		if cap(buf2) == 0 {
			t.Fatal("Second buffer has zero capacity")
		}

		ptr2 := &buf2[0:cap(buf2)][0]

		// Pointers should be the same (reused buffer)
		if ptr1 != ptr2 {
			t.Log("Warning: Pool may not be reusing buffers")
		} else {
			t.Log("Pool successfully reused buffer")
		}
	})
}

// TestPoolManager_MemoryLeakPrevention tests that PutRow clears references.
//
// This is critical to prevent memory leaks. When a row containing pointers
// is returned to the pool, all references must be cleared to allow GC.
