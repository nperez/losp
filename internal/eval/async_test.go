// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"strings"
	"testing"
	"time"
)

func TestAsyncBasic(t *testing.T) {
	e := New()

	// Define an expression to run async
	_, err := e.Eval("▼Work result-value ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Launch async and await
	result, err := e.Eval("▶AWAIT ▶ASYNC Work ◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result-value" {
		t.Errorf("expected 'result-value', got '%s'", result)
	}
}

func TestAsyncWithHandle(t *testing.T) {
	e := New()

	_, err := e.Eval("▼Work async-output ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Store handle
	result, err := e.Eval("▽h ▶ASYNC Work ◆ ◆ ▶AWAIT ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "async-output") {
		t.Errorf("expected result to contain 'async-output', got '%s'", result)
	}
}

func TestAsyncParallel(t *testing.T) {
	e := New()

	// Two expressions
	_, err := e.Eval("▼A val-a ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = e.Eval("▼B val-b ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Launch both, await both
	result, err := e.Eval(`▽h1 ▶ASYNC A ◆ ◆
▽h2 ▶ASYNC B ◆ ◆
▶AWAIT ▲h1 ◆ ▶AWAIT ▲h2 ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "val-a") {
		t.Errorf("expected 'val-a' in result, got '%s'", result)
	}
	if !strings.Contains(result, "val-b") {
		t.Errorf("expected 'val-b' in result, got '%s'", result)
	}
}

func TestAsyncCheck(t *testing.T) {
	e := New()

	// Expression that takes a moment (uses SLEEP)
	_, err := e.Eval("▼SlowWork ▶SLEEP 50 ◆ done ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Launch async
	_, err = e.Eval("▽h ▶ASYNC SlowWork ◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CHECK should return FALSE initially (might be racy, but 50ms is enough)
	result, err := e.Eval("▶CHECK ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "FALSE" {
		t.Logf("CHECK returned %s (task may have completed fast)", result)
	}

	// Wait for completion
	_, err = e.Eval("▶AWAIT ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CHECK after AWAIT should return TRUE
	result, err = e.Eval("▶CHECK ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("expected TRUE after AWAIT, got '%s'", result)
	}
}

func TestAsyncDoubleAwait(t *testing.T) {
	e := New()

	_, err := e.Eval("▼Work double-val ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = e.Eval("▽h ▶ASYNC Work ◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First AWAIT
	r1, err := e.Eval("▶AWAIT ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Second AWAIT on same handle - should return same result immediately
	r2, err := e.Eval("▶AWAIT ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r1 != r2 {
		t.Errorf("double AWAIT returned different results: '%s' vs '%s'", r1, r2)
	}
}

func TestAsyncUnknownHandle(t *testing.T) {
	e := New()

	result, err := e.Eval("▶AWAIT nonexistent ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty for unknown handle, got '%s'", result)
	}
}

func TestAsyncNonexistentExpression(t *testing.T) {
	e := New()

	result, err := e.Eval("▶ASYNC NoSuchExpr ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty for nonexistent expression, got '%s'", result)
	}
}

func TestAsyncNamespaceIsolation(t *testing.T) {
	e := New()

	// Set a value, then launch async that reads it
	_, err := e.Eval("▽val before ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = e.Eval("▼ReadVal ▲val ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Launch async (clones namespace with val=before)
	_, err = e.Eval("▽h ▶ASYNC ReadVal ◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Change val after fork - forked evaluator shouldn't see this
	_, err = e.Eval("▽val after ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := e.Eval("▶AWAIT ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "before" {
		t.Errorf("expected 'before' (snapshot isolation), got '%s'", result)
	}
}

func TestAsyncSaySilenced(t *testing.T) {
	var output strings.Builder
	e := New(WithOutputWriter(func(text string) error {
		output.WriteString(text)
		return nil
	}))

	// SAY in forked evaluator should be silenced
	_, err := e.Eval("▼Talker ▶SAY should-not-appear ◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = e.Eval("▶AWAIT ▶ASYNC Talker ◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(output.String(), "should-not-appear") {
		t.Errorf("SAY output leaked from forked evaluator: '%s'", output.String())
	}
}

func TestTimer(t *testing.T) {
	e := New()

	_, err := e.Eval("▼Delayed timer-result ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Launch timer with 50ms delay
	_, err = e.Eval("▽h ▶TIMER\n50\nDelayed\n◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Await the timer result
	result, err := e.Eval("▶AWAIT ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "timer-result" {
		t.Errorf("expected 'timer-result', got '%s'", result)
	}
}

func TestTimerZeroMs(t *testing.T) {
	e := New()

	_, err := e.Eval("▼Immediate instant-result ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 0ms timer fires immediately
	result, err := e.Eval("▶AWAIT ▶TIMER\n0\nImmediate\n◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "instant-result" {
		t.Errorf("expected 'instant-result', got '%s'", result)
	}
}

func TestTicks(t *testing.T) {
	e := New()

	_, err := e.Eval("▼Later later-val ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Long timer so we can check ticks
	_, err = e.Eval("▽h ▶TIMER\n5000\nLater\n◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// TICKS should return a positive number
	result, err := e.Eval("▶TICKS ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "0" {
		t.Errorf("expected positive ticks, got 0")
	}

	// Cleanup: shut down to stop the pending timer
	e.asyncRegistry.Shutdown()
}

func TestTicksOnPromise(t *testing.T) {
	e := New()

	_, err := e.Eval("▼Work val ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = e.Eval("▽h ▶ASYNC Work ◆ ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// TICKS on a non-timer returns 0
	result, err := e.Eval("▶TICKS ▲h ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "0" {
		t.Errorf("expected '0' for non-timer, got '%s'", result)
	}
}

func TestSleep(t *testing.T) {
	start := time.Now()
	e := New()

	result, err := e.Eval("▶SLEEP 50 ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)

	if result != "" {
		t.Errorf("expected empty result, got '%s'", result)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("SLEEP didn't wait long enough: %v", elapsed)
	}
}

func TestCheckUnknownHandle(t *testing.T) {
	e := New()

	result, err := e.Eval("▶CHECK nonexistent ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "FALSE" {
		t.Errorf("expected FALSE for unknown handle, got '%s'", result)
	}
}

func TestAsyncRegistryShutdown(t *testing.T) {
	e := New()

	_, err := e.Eval("▼SlowWork ▶SLEEP 5000 ◆ slow-done ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Launch a slow async task
	_, err = e.Eval("▶ASYNC SlowWork ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Shutdown should complete without hanging (5s timeout)
	done := make(chan struct{})
	go func() {
		e.asyncRegistry.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(10 * time.Second):
		t.Fatal("Shutdown hung")
	}
}
