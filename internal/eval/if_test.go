// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"testing"
)

func TestIfWithFalse(t *testing.T) {
	e := New()
	
	// Set up templates
	e.Eval("▼ShowDebug DebugShown ◆")
	e.Eval("▼DoNothing ◆")
	
	// Set debug to FALSE
	e.Eval("▽DebugFlag FALSE ◆")
	
	// Test IF with FALSE condition
	result, err := e.Eval(`▶IF ▷COMPARE ▲DebugFlag TRUE ◆
		▶ShowDebug ◆
		▶DoNothing ◆
	◆`)
	
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	
	t.Logf("Result: %q", result)
	
	if result == "DebugShown" {
		t.Error("IF executed then-branch when condition was FALSE")
	}
}

func TestIfWithTrue(t *testing.T) {
	e := New()
	
	// Set up templates
	e.Eval("▼ShowDebug DebugShown ◆")
	e.Eval("▼DoNothing ◆")
	
	// Set debug to TRUE
	e.Eval("▽DebugFlag TRUE ◆")
	
	// Test IF with TRUE condition
	result, err := e.Eval(`▶IF ▷COMPARE ▲DebugFlag TRUE ◆
		▶ShowDebug ◆
		▶DoNothing ◆
	◆`)
	
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	
	t.Logf("Result: %q", result)
	
	if result != "DebugShown" {
		t.Errorf("IF should execute then-branch, got: %q", result)
	}
}
