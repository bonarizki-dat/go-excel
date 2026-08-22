package concurrency

import (
	"bytes"
	"strings"
	"sync"
)

// PoolManager manages multiple object pools for different data types.
//
// It provides centralized management of sync.Pool instances to minimize
// memory allocations and GC pressure during high-throughput streaming operations.
//
// Usage:
//
//	pm := NewPoolManager()
//	row := pm.GetRow()
//	defer pm.PutRow(row)
type PoolManager struct {
	// rowPool holds []any row buffers, preallocated for ~100 columns to
	// cover most use cases without reallocation.
	rowPool *sync.Pool

	// cellPool holds []byte buffers sized for a single serialized cell
	// value (typical cell: 10-1000 bytes).
	cellPool *sync.Pool

	// builderPool holds *strings.Builder instances reused across
	// concatenation-heavy operations.
	builderPool *sync.Pool

	// bufferPool holds general-purpose *bytes.Buffer instances sized to
	// the OS page size (4KB), used for I/O and temporary storage.
	bufferPool *sync.Pool

	// largeBufferPool holds 64KB []byte buffers for batch operations
	// that would otherwise thrash bufferPool's smaller buffers.
	largeBufferPool *sync.Pool
}

// NewPoolManager returns a PoolManager with all pools initialized and
// ready for use. Each pool's initial capacity is sized from benchmark
// results to minimize allocations during normal operation.
func NewPoolManager() *PoolManager {
	return &PoolManager{
		rowPool: &sync.Pool{
			New: func() any {
				// Pointer avoids the boxing allocation staticcheck SA6002
				// warns about when a sync.Pool stores a slice by value.
				row := make([]any, 0, 100)
				return &row
			},
		},

		cellPool: &sync.Pool{
			New: func() any {
				buf := make([]byte, 0, 1024)
				return &buf
			},
		},

		builderPool: &sync.Pool{
			New: func() any {
				return &strings.Builder{}
			},
		},

		bufferPool: &sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 4096))
			},
		},

		largeBufferPool: &sync.Pool{
			New: func() any {
				buf := make([]byte, 0, 65536)
				return &buf
			},
		},
	}
}

// GetRow returns a []any row buffer from the pool, with zero length but
// possibly non-zero capacity. Append to it and return it via PutRow
// when done.
func (pm *PoolManager) GetRow() []any {
	rowPtr := pm.rowPool.Get().(*[]any)
	row := *rowPtr
	return row[:0]
}

// PutRow returns row to the pool for reuse by a future GetRow call.
func (pm *PoolManager) PutRow(row []any) {
	// Nil out every element before returning to the pool; otherwise the
	// pool would keep the row's old contents (including any large
	// objects they reference) alive until the buffer is reused.
	for i := range row {
		row[i] = nil
	}

	row = row[:0]
	pm.rowPool.Put(&row)
}

// GetCellBuffer returns a []byte buffer from the pool for serializing a
// single cell value, with zero length but possibly non-zero capacity.
func (pm *PoolManager) GetCellBuffer() []byte {
	bufPtr := pm.cellPool.Get().(*[]byte)
	buf := *bufPtr
	return buf[:0]
}

// PutCellBuffer returns buf to the pool for reuse by a future
// GetCellBuffer call.
func (pm *PoolManager) PutCellBuffer(buf []byte) {
	buf = buf[:0]
	pm.cellPool.Put(&buf)
}

// GetBuilder returns a reset, ready-to-use *strings.Builder from the
// pool.
func (pm *PoolManager) GetBuilder() *strings.Builder {
	builder := pm.builderPool.Get().(*strings.Builder)
	// Reset defensively in case a previous caller returned it dirty.
	builder.Reset()
	return builder
}

// PutBuilder resets builder and returns it to the pool for reuse by a
// future GetBuilder call.
func (pm *PoolManager) PutBuilder(builder *strings.Builder) {
	builder.Reset()
	pm.builderPool.Put(builder)
}

// GetBuffer returns a reset, ready-to-use *bytes.Buffer from the pool,
// for I/O and other temporary byte storage.
func (pm *PoolManager) GetBuffer() *bytes.Buffer {
	buf := pm.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer resets buf and returns it to the pool for reuse by a
// future GetBuffer call.
func (pm *PoolManager) PutBuffer(buf *bytes.Buffer) {
	buf.Reset()
	pm.bufferPool.Put(buf)
}

// GetLargeBuffer returns a []byte buffer with 64KB capacity from the
// pool, for batch operations spanning multiple rows.
func (pm *PoolManager) GetLargeBuffer() []byte {
	bufPtr := pm.largeBufferPool.Get().(*[]byte)
	buf := *bufPtr
	return buf[:0]
}

// PutLargeBuffer returns buf to the pool for reuse by a future
// GetLargeBuffer call.
func (pm *PoolManager) PutLargeBuffer(buf []byte) {
	buf = buf[:0]
	pm.largeBufferPool.Put(&buf)
}
