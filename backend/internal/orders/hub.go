package orders

import (
	"sync"

	"shopbot/internal/store"
)

type HubEvent struct {
	Type  string            `json:"type"` // order_paid, order_updated
	Order store.OrderDetail `json:"order"`
}

// Hub sends order events to open owner dashboards (Server-Sent Events).
type Hub struct {
	mu   sync.Mutex
	subs map[int64]map[chan HubEvent]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[int64]map[chan HubEvent]struct{}{}}
}

// Subscribe returns a channel with events for one shop and a function to stop.
func (h *Hub) Subscribe(shopID int64) (<-chan HubEvent, func()) {
	ch := make(chan HubEvent, 8)
	h.mu.Lock()
	if h.subs[shopID] == nil {
		h.subs[shopID] = map[chan HubEvent]struct{}{}
	}
	h.subs[shopID][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs[shopID], ch)
		h.mu.Unlock()
	}
}

// Publish never blocks: a slow dashboard just misses the event and catches up on its next refresh.
func (h *Hub) Publish(shopID int64, e HubEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[shopID] {
		select {
		case ch <- e:
		default:
		}
	}
}
