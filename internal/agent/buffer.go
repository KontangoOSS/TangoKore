package agent

import (
	"sync"
	"time"
)

// eventBuffer is a time-bounded in-memory ring buffer for telemetry events.
// When NATS is unreachable, events accumulate here. On reconnect the caller
// drains them in order. Events older than maxAge are silently dropped on
// every append — no background goroutine needed.
type eventBuffer struct {
	mu     sync.Mutex
	buf    []bufferedEvent
	maxAge time.Duration
}

type bufferedEvent struct {
	data      []byte
	createdAt time.Time
}

func newEventBuffer(maxAge time.Duration) *eventBuffer {
	return &eventBuffer{maxAge: maxAge}
}

// push appends an event and evicts anything older than maxAge.
func (b *eventBuffer) push(data []byte) {
	now := time.Now()
	b.mu.Lock()
	defer b.mu.Unlock()

	// Evict expired entries from the front.
	cutoff := now.Add(-b.maxAge)
	i := 0
	for i < len(b.buf) && b.buf[i].createdAt.Before(cutoff) {
		i++
	}
	if i > 0 {
		b.buf = b.buf[i:]
	}

	// Copy data — the caller may reuse the slice.
	cp := make([]byte, len(data))
	copy(cp, data)
	b.buf = append(b.buf, bufferedEvent{data: cp, createdAt: now})
}

// drain returns all buffered events in order and clears the buffer.
// Expired events are excluded.
func (b *eventBuffer) drain() [][]byte {
	now := time.Now()
	cutoff := now.Add(-b.maxAge)

	b.mu.Lock()
	defer b.mu.Unlock()

	var out [][]byte
	for _, ev := range b.buf {
		if ev.createdAt.After(cutoff) {
			out = append(out, ev.data)
		}
	}
	b.buf = b.buf[:0]
	return out
}

// len returns the current number of buffered events.
func (b *eventBuffer) len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.buf)
}
