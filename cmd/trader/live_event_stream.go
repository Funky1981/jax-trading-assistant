package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// liveEvent is a message pushed over the SSE stream to connected frontends.
type liveEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	SentAt  time.Time       `json:"sentAt"`
}

// eventBroker fans out liveEvents to all registered SSE clients.
type eventBroker struct {
	mu      sync.RWMutex
	clients map[chan liveEvent]struct{}
}

var globalBroker = &eventBroker{
	clients: make(map[chan liveEvent]struct{}),
}

// subscribe registers a new client channel and returns it.
func (b *eventBroker) subscribe() chan liveEvent {
	ch := make(chan liveEvent, 32)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// unsubscribe removes and closes a client channel.
func (b *eventBroker) unsubscribe(ch chan liveEvent) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// publish sends an event to all current subscribers (non-blocking per client).
func (b *eventBroker) publish(e liveEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- e:
		default:
			// slow client: skip rather than block
		}
	}
}

// publishEvent is a convenience wrapper.
func publishEvent(eventType string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	globalBroker.publish(liveEvent{
		Type:    eventType,
		Payload: json.RawMessage(raw),
		SentAt:  time.Now().UTC(),
	})
}

// sseHandler returns an http.HandlerFunc that streams events to the browser.
// GET /api/v1/events/stream
func sseHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		ch := globalBroker.subscribe()
		defer globalBroker.unsubscribe(ch)

		// Send an initial ping so the browser knows the stream is live.
		fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
		flusher.Flush()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				// keepalive comment
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			case evt, ok := <-ch:
				if !ok {
					return
				}
				b, err := json.Marshal(evt)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, b)
				flusher.Flush()
			}
		}
	}
}
