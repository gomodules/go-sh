package sh

import (
	"bytes"
	"sync"
)

type SafeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends data to the buffer in a thread-safe manner.
func (sb *SafeBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

// Bytes returns a copy of the buffer's contents in a thread-safe manner.
func (sb *SafeBuffer) Bytes() []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	cp := make([]byte, sb.buf.Len())
	copy(cp, sb.buf.Bytes())
	return cp
}

func (sb *SafeBuffer) Reset() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.buf.Reset()
}
