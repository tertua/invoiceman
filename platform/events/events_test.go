package events

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublishDeliversToSubscriber(t *testing.T) {
	b := New()
	ch, unsub := b.Subscribe("u1")
	defer unsub()

	b.Publish("u1", Event{Type: "payment.created", Data: "{}"})
	select {
	case ev := <-ch:
		require.Equal(t, "payment.created", ev.Type)
	default:
		t.Fatal("expected event, got none")
	}
}

func TestPublishIsolatesUsers(t *testing.T) {
	b := New()
	ch1, unsub1 := b.Subscribe("u1")
	defer unsub1()
	ch2, unsub2 := b.Subscribe("u2")
	defer unsub2()

	b.Publish("u1", Event{Type: "payment.created", Data: "{}"})
	select {
	case <-ch2:
		t.Fatal("user u2 must not receive u1 events")
	default:
	}
	select {
	case <-ch1:
	default:
		t.Fatal("expected event for u1, got none")
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	b := New()
	ch, unsub := b.Subscribe("u1")
	unsub()

	b.Publish("u1", Event{Type: "payment.created", Data: "{}"})
	select {
	case <-ch:
		t.Fatal("unsubscribed channel must not receive events")
	default:
	}
}

func TestPublishNeverBlocksOnSlowReader(t *testing.T) {
	b := New()
	ch, unsub := b.Subscribe("u1")
	defer unsub()

	for i := 0; i < 64; i++ {
		b.Publish("u1", Event{Type: "payment.created", Data: "{}"})
	}
	require.Len(t, ch, 16)
}
