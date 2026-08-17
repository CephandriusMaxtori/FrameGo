package engine

import (
	"testing"
	"time"
)

func TestBusPubSub(t *testing.T) {
	b := NewBus()
	ch := b.Subscribe(TopicClockTick)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case ev := <-ch:
				if ev.Topic != TopicClockTick {
					t.Errorf("got topic %q", ev.Topic)
				}
				close(done)
				return
			case <-time.After(time.Second):
				close(done)
				return
			}
		}
	}()
	b.Publish(TopicClockTick, nil)
	<-done
	if got := b.SubscriberCount(TopicClockTick); got != 1 {
		t.Errorf("subscriber count = %d", got)
	}
}

func TestBusUnsubscribe(t *testing.T) {
	b := NewBus()
	ch := b.Subscribe(TopicNetwork)
	b.Unsubscribe(TopicNetwork, ch)
	if got := b.SubscriberCount(TopicNetwork); got != 0 {
		t.Errorf("subscriber count after unsubscribe = %d", got)
	}
}

func TestBusPublishDoesNotBlock(t *testing.T) {
	b := NewBus()
	_ = b.Subscribe(TopicClockTick) // subscriber that never drains
	for i := 0; i < 1000; i++ {
		b.Publish(TopicClockTick, i)
	}
}

func TestBusIsolatedTopics(t *testing.T) {
	b := NewBus()
	ch := b.Subscribe(TopicPowerDim)
	b.Publish(TopicClockTick, nil) // wrong topic must not reach ch
	select {
	case ev := <-ch:
		t.Errorf("unexpected event on power:dim: %+v", ev)
	case <-time.After(20 * time.Millisecond):
	}
}
