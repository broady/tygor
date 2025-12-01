package tygor

import (
	"context"
	"encoding/json"
	"iter"
	"sync"
)

// Atom holds a single value that can be read, written, and subscribed to.
// Updates are broadcast to all subscribers via SSE streaming.
// Thread-safe for concurrent Get/Set operations.
//
// Unlike event streams, Atom represents current state - subscribers always
// receive the latest value, and intermediate updates may be skipped if
// a subscriber is slow.
//
// Example:
//
//	status := tygor.NewAtom(&Status{State: "idle"})
//
//	// Read current value
//	current := status.Get()
//
//	// Update and broadcast to all subscribers
//	status.Set(&Status{State: "running"})
//
//	// Register SSE endpoint
//	svc.Register("SubscribeStatus", status.SubscribeHandler())
type Atom[T any] struct {
	mu          sync.RWMutex
	value       T
	bytes       json.RawMessage // pre-serialized for efficient broadcast
	subscribers map[int64]chan json.RawMessage
	nextSubID   int64
}

// NewAtom creates a new Atom with the given initial value.
func NewAtom[T any](initial T) *Atom[T] {
	a := &Atom[T]{
		value:       initial,
		subscribers: make(map[int64]chan json.RawMessage),
	}
	// Pre-serialize initial value (ignore error - will be caught on first Set if invalid)
	a.bytes, _ = json.Marshal(initial)
	return a
}

// Get returns the current value.
func (a *Atom[T]) Get() T {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.value
}

// Set updates the value and broadcasts to all subscribers.
// The value is serialized once and the same bytes are sent to all subscribers.
func (a *Atom[T]) Set(value T) {
	// Serialize once before taking lock
	data, err := json.Marshal(value)
	if err != nil {
		// If serialization fails, still update in-memory value
		// but don't broadcast (subscribers would get stale data)
		a.mu.Lock()
		a.value = value
		a.mu.Unlock()
		return
	}

	// Update value and snapshot subscribers
	a.mu.Lock()
	a.value = value
	a.bytes = data
	subs := make([]chan json.RawMessage, 0, len(a.subscribers))
	for _, ch := range a.subscribers {
		subs = append(subs, ch)
	}
	a.mu.Unlock()

	// Broadcast outside lock with non-blocking sends
	for _, ch := range subs {
		select {
		case ch <- data:
			// Delivered
		default:
			// Channel full - drain old value and send new (latest-wins)
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- data:
			default:
			}
		}
	}
}

// Update atomically applies fn to the current value.
// Useful for read-modify-write operations.
func (a *Atom[T]) Update(fn func(T) T) {
	a.mu.Lock()
	newValue := fn(a.value)
	a.mu.Unlock()
	a.Set(newValue)
}

// Subscribe returns an iterator that yields the current value and all future
// updates until ctx is canceled. For use in Go code, not HTTP handlers.
func (a *Atom[T]) Subscribe(ctx context.Context) iter.Seq[T] {
	return func(yield func(T) bool) {
		// Send current value
		a.mu.RLock()
		current := a.value
		a.mu.RUnlock()

		if !yield(current) {
			return
		}

		// Create channel and subscribe
		ch := make(chan json.RawMessage, 1)
		subID := a.addSubscriber(ch)
		defer a.removeSubscriber(subID)

		for {
			select {
			case <-ctx.Done():
				return
			case data := <-ch:
				var val T
				if err := json.Unmarshal(data, &val); err != nil {
					continue // Skip malformed data
				}
				if !yield(val) {
					return
				}
			}
		}
	}
}

// SubscribeHandler returns a StreamHandler for SSE streaming.
// Sends the current value immediately, then streams updates.
//
// Example:
//
//	svc.Register("SubscribeStatus", statusAtom.SubscribeHandler())
func (a *Atom[T]) SubscribeHandler() *StreamHandler[Empty, json.RawMessage] {
	return StreamEmit(func(ctx context.Context, _ Empty, emit *Emitter[json.RawMessage]) error {
		// Send current value
		a.mu.RLock()
		current := a.bytes
		a.mu.RUnlock()

		if err := emit.Send(current); err != nil {
			return err
		}

		// Create channel and subscribe
		ch := make(chan json.RawMessage, 1)
		subID := a.addSubscriber(ch)
		defer a.removeSubscriber(subID)

		// Stream updates until disconnect
		for {
			select {
			case <-ctx.Done():
				return nil
			case data := <-ch:
				if err := emit.Send(data); err != nil {
					return err
				}
			}
		}
	})
}

// addSubscriber adds a channel to the subscriber list.
func (a *Atom[T]) addSubscriber(ch chan json.RawMessage) int64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	id := a.nextSubID
	a.nextSubID++
	a.subscribers[id] = ch
	return id
}

// removeSubscriber removes a channel from the subscriber list.
func (a *Atom[T]) removeSubscriber(id int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.subscribers, id)
}
