package engine

import (
	"sync"
	"time"
)

// Standard system topics published on the event bus.
const (
	TopicClockTick   = "system:clock:tick"
	TopicPowerDim    = "system:power:dim"
	TopicNetwork     = "system:network:status"
	TopicModuleState = "module:state:change"
)

// Event is a single message broadcast on the bus.
type Event struct {
	Topic string
	Data  any
	Time  time.Time
}

// Bus is an asynchronous publish/subscribe router. Subscribers receive events
// on buffered channels; publishes are non-blocking so a slow or stalled
// subscriber can never stall the frame loop or another module.
type Bus struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{}
}

// NewBus creates an empty event bus.
func NewBus() *Bus {
	return &Bus{subs: make(map[string]map[chan Event]struct{})}
}

// Subscribe returns a buffered channel receiving events published on topic.
func (b *Bus) Subscribe(topic string) chan Event {
	ch := make(chan Event, 16)
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[chan Event]struct{})
	}
	b.subs[topic][ch] = struct{}{}
	return ch
}

// Unsubscribe stops delivering topic events to ch.
func (b *Bus) Unsubscribe(topic string, ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if subs, ok := b.subs[topic]; ok {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(b.subs, topic)
		}
	}
}

// Publish broadcasts an event to every subscriber of the topic. Delivery is
// best-effort: a subscriber whose buffer is full misses the event.
func (b *Bus) Publish(topic string, data any) {
	ev := Event{Topic: topic, Data: data, Time: time.Now()}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs[topic] {
		select {
		case ch <- ev:
		default:
		}
	}
}

// SubscriberCount reports how many channels are subscribed to topic.
func (b *Bus) SubscriberCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[topic])
}
