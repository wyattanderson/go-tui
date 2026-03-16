package tui

import (
	"testing"
	"time"
)

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

func TestRender_SkipsWhenNotDirty(t *testing.T) {
	mock := NewMockTerminal(80, 24)
	app := &App{
		terminal:     mock,
		buffer:       NewBuffer(80, 24),
		focus:        newFocusManager(),
		events:       make(chan Event, 1),
		watcherQueue: make(chan func(), 1),
		stopCh:       make(chan struct{}),
		mounts:       newMountState(),
		batch:        newBatchContext(),
	}
	root := New(WithText("hello"))
	app.SetRoot(root)

	// First render: dirty from SetRoot
	app.Render()

	// Capture buffer state after first render
	snap1 := app.buffer.StringTrimmed()

	// Second render without marking dirty: should be no-op
	root.SetText("changed")
	app.resetDirty()
	app.Render()

	snap2 := app.buffer.StringTrimmed()
	if snap1 != snap2 {
		t.Errorf("Render() should be no-op when not dirty\nbefore: %q\nafter: %q", snap1, snap2)
	}
}

func TestOpen_DoubleCallReturnsError(t *testing.T) {
	app := &App{
		terminal:     NewMockTerminal(80, 24),
		reader:       &MockEventReader{},
		buffer:       NewBuffer(80, 24),
		focus:        newFocusManager(),
		events:       make(chan Event, 256),
		watcherQueue: make(chan func(), 256),
		stopCh:       make(chan struct{}),
		mounts:       newMountState(),
		batch:        newBatchContext(),
	}

	if err := app.Open(); err != nil {
		t.Fatalf("first Open() should succeed: %v", err)
	}
	defer app.Close()

	if err := app.Open(); err == nil {
		t.Fatal("second Open() should return error")
	}
}

func TestClose_Idempotent(t *testing.T) {
	app := &App{
		terminal:     NewMockTerminal(80, 24),
		reader:       &MockEventReader{},
		buffer:       NewBuffer(80, 24),
		focus:        newFocusManager(),
		events:       make(chan Event, 256),
		watcherQueue: make(chan func(), 256),
		stopCh:       make(chan struct{}),
		mounts:       newMountState(),
		batch:        newBatchContext(),
	}

	if err := app.Open(); err != nil {
		t.Fatalf("Open() failed: %v", err)
	}

	// First close
	app.Close()

	// Second close should not panic
	app.Close()
}

// Placeholder for time import usage
var _ = time.Millisecond
