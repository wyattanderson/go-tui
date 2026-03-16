package tui

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

// Open initializes the event loop: registers signal handlers, starts the
// input reader goroutine, and performs the initial render. Call this instead
// of Run() when driving your own event loop. Returns an error if already open.
//
// After Open(), use Events(), Dispatch(), and Render() to process events.
// Call Close() when done to restore terminal state.
func (a *App) Open() error {
	if !a.opened.CompareAndSwap(false, true) {
		return fmt.Errorf("tui: app is already open")
	}

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		select {
		case <-sigCh:
			a.Stop()
		case <-a.stopCh:
		}
		signal.Stop(sigCh)
	}()

	// Handle SIGWINCH (terminal resize)
	cleanupResize := a.registerResizeSignal()

	// Handle Ctrl+Z / SIGTSTP for job control
	cleanupSuspend := a.registerSuspendSignals()

	// Store cleanup functions for Close()
	a.signalCleanup = func() {
		cleanupResize()
		cleanupSuspend()
	}

	// Start input reader in background
	go a.readInputEvents()

	// Initial render
	a.MarkDirty()
	a.renderFrame()
	a.rebuildDispatchTable()

	return nil
}

// Run starts the main event loop. Blocks until Stop() is called or SIGINT received.
// Rendering occurs only when the dirty flag is set (by mutations).
func (a *App) Run() error {
	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		select {
		case <-sigCh:
			a.Stop()
		case <-a.stopCh:
			// App already stopped, clean up signal handler
		}
		signal.Stop(sigCh)
	}()

	// Handle SIGWINCH (terminal resize)
	cleanupResize := a.registerResizeSignal()
	defer cleanupResize()

	// Handle Ctrl+Z / SIGTSTP for job control
	cleanupSuspend := a.registerSuspendSignals()
	defer cleanupSuspend()

	// Start input reader in background
	go a.readInputEvents()

	// Initial render
	a.MarkDirty()
	a.renderFrame()
	a.rebuildDispatchTable()

	// Frame-based loop with configurable frame timing
	for {
		frameStart := time.Now()

		// Process events for up to half the frame budget (non-blocking)
		eventDeadline := frameStart.Add(a.frameDuration / 2)
	drain:
		for time.Now().Before(eventDeadline) {
			select {
			case ev := <-a.events:
				a.Dispatch(ev)
			case <-a.stopCh:
				return nil
			default:
				// No more events, move to render phase
				break drain
			}
		}

		// Render if dirty (Render() checks and clears the dirty flag internally)
		a.Render()

		// Sleep for remaining frame time to maintain consistent framerate
		elapsed := time.Since(frameStart)
		if elapsed < a.frameDuration {
			select {
			case <-time.After(a.frameDuration - elapsed):
			case <-a.stopCh:
				return nil
			}
		}
	}
}

// Stop signals the Run loop to exit gracefully and stops all watchers.
// Watchers receive the stop signal via stopCh and exit their goroutines.
// Stop is idempotent - multiple calls are safe.
func (a *App) Stop() {
	a.stopOnce.Do(func() {
		a.stopped = true

		// Interrupt blocking reader before closing stopCh to wake it up
		if interruptible, ok := a.reader.(InterruptibleReader); ok {
			interruptible.Interrupt()
		}

		// Signal all watcher goroutines to stop
		close(a.stopCh)
	})
}

// QueueUpdate enqueues a function to run on the main loop.
// Safe to call from any goroutine. Use this for background thread safety.
func (a *App) QueueUpdate(fn func()) {
	if fn == nil {
		return
	}
	select {
	case a.events <- UpdateEvent{fn: fn}:
	case <-a.stopCh:
	}
}

// rebuildDispatchTable walks the rendered element tree and builds a new
// dispatch table from all mounted components' KeyMap() methods.
// If validation fails, the previous table is kept.
func (a *App) rebuildDispatchTable() {
	if a.root == nil {
		return
	}

	table, err := buildDispatchTable(a.rootComponent, a.root)
	if err != nil {
		// Validation error (e.g., conflicting Stop handlers).
		// Log and keep the previous valid table rather than crashing.
		fmt.Fprintf(os.Stderr, "tui: dispatch table error: %v\n", err)
		return
	}
	a.dispatchTable = table
}
