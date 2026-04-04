package agent

import (
	"testing"
	"time"
)

func TestEventBuffer_PushAndDrain(t *testing.T) {
	buf := newEventBuffer(5 * time.Minute)

	buf.push([]byte("event-1"))
	buf.push([]byte("event-2"))
	buf.push([]byte("event-3"))

	if buf.len() != 3 {
		t.Fatalf("expected 3 events, got %d", buf.len())
	}

	events := buf.drain()
	if len(events) != 3 {
		t.Fatalf("expected 3 drained events, got %d", len(events))
	}
	if string(events[0]) != "event-1" {
		t.Fatalf("expected event-1, got %s", events[0])
	}
	if string(events[2]) != "event-3" {
		t.Fatalf("expected event-3, got %s", events[2])
	}

	// Buffer should be empty after drain.
	if buf.len() != 0 {
		t.Fatalf("expected 0 after drain, got %d", buf.len())
	}
	events = buf.drain()
	if len(events) != 0 {
		t.Fatalf("expected 0 drained events after second drain, got %d", len(events))
	}
}

func TestEventBuffer_Expiry(t *testing.T) {
	// Use a tiny TTL so events expire quickly.
	buf := newEventBuffer(50 * time.Millisecond)

	buf.push([]byte("old-event"))
	time.Sleep(100 * time.Millisecond)
	buf.push([]byte("new-event"))

	events := buf.drain()
	if len(events) != 1 {
		t.Fatalf("expected 1 event (old expired), got %d", len(events))
	}
	if string(events[0]) != "new-event" {
		t.Fatalf("expected new-event, got %s", events[0])
	}
}

func TestEventBuffer_EvictsOnPush(t *testing.T) {
	buf := newEventBuffer(50 * time.Millisecond)

	buf.push([]byte("will-expire"))
	time.Sleep(100 * time.Millisecond)

	// Push triggers eviction of expired entries.
	buf.push([]byte("fresh"))

	if buf.len() != 1 {
		t.Fatalf("expected 1 after eviction, got %d", buf.len())
	}
}

func TestEventBuffer_CopiesData(t *testing.T) {
	buf := newEventBuffer(5 * time.Minute)

	// Verify the buffer copies the input — caller can reuse the slice.
	data := []byte("original")
	buf.push(data)
	data[0] = 'X' // mutate after push

	events := buf.drain()
	if string(events[0]) != "original" {
		t.Fatalf("buffer did not copy data, got %s", events[0])
	}
}

func TestEventBuffer_DrainOrder(t *testing.T) {
	buf := newEventBuffer(5 * time.Minute)

	for i := 0; i < 100; i++ {
		buf.push([]byte{byte(i)})
	}

	events := buf.drain()
	if len(events) != 100 {
		t.Fatalf("expected 100 events, got %d", len(events))
	}
	for i, ev := range events {
		if ev[0] != byte(i) {
			t.Fatalf("event %d: expected %d, got %d", i, i, ev[0])
		}
	}
}
