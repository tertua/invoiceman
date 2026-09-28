package events

// Package events is a tiny in-process pub/sub fan-out for server-sent
// events (SSE): controllers publish per-user aggregate changes, the
// /events stream delivers them to that user's browsers. Delivery is
// best-effort (slow readers drop frames) and single-instance; Redis
// fan-out across instances is future work, not wired yet.

import "sync"

// Event is one SSE frame: Type selects the client listener (it reuses
// the models.NotifEvent strings), Data carries a small JSON object.
// Clients only need the nudge and refetch state themselves.
type Event struct {
	Type string
	Data string
}

// Broker fans events out to per-user subscribers.
type Broker struct {
	mu   sync.Mutex
	subs map[string]map[chan Event]struct{}
}

// New returns an empty broker (tests use their own; the server uses Default).
func New() *Broker {
	return &Broker{subs: make(map[string]map[chan Event]struct{})}
}

// Default is the server-wide broker (see StreamEvents, enqueueNotification).
var Default = New()

// Subscribe registers a buffered channel for userID; call unsubscribe to
// stop delivery and free the slot (deferred by the SSE handler).
func (b *Broker) Subscribe(userID string) (ch chan Event, unsubscribe func()) {
	ch = make(chan Event, 16)
	b.mu.Lock()
	set, ok := b.subs[userID]
	if !ok {
		set = make(map[chan Event]struct{})
		b.subs[userID] = set
	}
	set[ch] = struct{}{}
	b.mu.Unlock()
	unsubscribe = func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if set, ok := b.subs[userID]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(b.subs, userID)
			}
		}
	}
	return ch, unsubscribe
}

// Publish drops ev to every subscriber of userID. Never blocks: a reader
// that fell behind (full buffer) misses the frame and catches up on the
// next dashboard fetch instead of stalling the writer.
func (b *Broker) Publish(userID string, ev Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.subs[userID] {
		select {
		case ch <- ev:
		default:
		}
	}
}
