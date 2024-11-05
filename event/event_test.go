package event

import (
	"sync"
	"testing"
)

type DoneEvent struct{}
type NewMinedBlockEvent struct{}

func TestSubscribe(t *testing.T) {
	var mux TypeMux
	defer mux.Stop()

	sub := mux.Subscribe(DoneEvent{})

	go func() {
		if err := mux.Post(DoneEvent{}); err != nil {
			t.Errorf("Post return unexpected error: %v", err)
		}
	}()

	ev := <-sub.Chan()

	event, ok := ev.Data.(DoneEvent)
	if !ok || event != (DoneEvent{}) {
		t.Errorf("Got: %v (%T), expect event: %v (%T)", ev, ev, DoneEvent{}, DoneEvent{})
	}
}

func TestDuplicateSubscribe(t *testing.T) {
	var mux TypeMux
	expected := "event: duplicate type event.DoneEvent in Subscribe"

	defer func() {
		err := recover()
		if err == nil {
			t.Errorf("Subscribe did not panic for duplicate type")
		} else if err != expected {
			t.Errorf("Got: %#v, expected error: %#v", err, expected)
		}
	}()

	mux.Subscribe(DoneEvent{}, DoneEvent{})
}

func TestSubUnSubAfterStop(t *testing.T) {
	var mux TypeMux
	mux.Stop()
	sub := mux.Subscribe(DoneEvent{})
	sub.Unsubscribe()
}

func TestPostAfterStop(t *testing.T) {
	var mux TypeMux
	mux.Stop()

	sub := mux.Subscribe(DoneEvent{})

	if _, isOpen := <-sub.Chan(); isOpen {
		t.Errorf("Subscription channel was not closed")
	}

	if err := mux.Post(DoneEvent{}); err != ErrMuxClosed {
		t.Errorf("Got: %s, expected error: %s", err, ErrMuxClosed)
	}
}

func TestSimulate(t *testing.T) {
	var mux TypeMux
	minedBlockSub := mux.Subscribe(NewMinedBlockEvent{}) // Correct subscription type

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer minedBlockSub.Unsubscribe()
		err := mux.Post(NewMinedBlockEvent{})
		if err != nil {
			t.Errorf("Post error: %v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for obj := range minedBlockSub.Chan() {
			event, ok := obj.Data.(NewMinedBlockEvent)
			if !ok || event != (NewMinedBlockEvent{}) {
				t.Errorf("Got: %v (%T), expect event: %v (%T)", obj, obj, NewMinedBlockEvent{}, NewMinedBlockEvent{})
				break
			} else {
				t.Log("Got new mined block event")
			}
		}
	}()
	wg.Wait()
}
