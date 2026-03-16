package tui

import "testing"

func TestUpdateEvent_ImplementsEvent(t *testing.T) {
	var ev Event = UpdateEvent{fn: func() {}}
	if ev == nil {
		t.Fatal("UpdateEvent should implement Event")
	}
}

func TestUpdateEvent_RunsClosure(t *testing.T) {
	called := false
	ev := UpdateEvent{fn: func() { called = true }}
	ev.fn()
	if !called {
		t.Fatal("UpdateEvent closure should have been called")
	}
}
