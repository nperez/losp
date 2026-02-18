// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"strings"
	"testing"
)

// =============================================================================
// Dynamic Execute as Argument Tests
//
// These tests verify that ▶▲name ◆ (dynamic execute via retrieve) works
// correctly when used as an argument to builtins like COMPARE, EXTRACT,
// UPPER, LOWER, TRIM — not just SAY.
// =============================================================================

// TestDynamicExecArgCompare tests ▶▲name ◆ as argument to COMPARE
func TestDynamicExecArgCompare(t *testing.T) {
	e := New()

	e.Eval("▼test hello ◆")
	e.Eval("▼myname test ◆")

	// Direct execute works
	result, err := e.Eval("▶COMPARE ▶test ◆ hello ◆")
	if err != nil {
		t.Fatalf("direct execute compare failed: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("direct: expected 'TRUE', got %q", result)
	}

	// Dynamic execute should also work
	result, err = e.Eval("▶COMPARE ▶▲myname ◆ hello ◆")
	if err != nil {
		t.Fatalf("dynamic execute compare failed: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("dynamic: expected 'TRUE', got %q", result)
	}
}

// TestDynamicExecArgCompareFalse tests ▶▲name ◆ returns correct value for FALSE comparison
func TestDynamicExecArgCompareFalse(t *testing.T) {
	e := New()

	e.Eval("▼test hello ◆")
	e.Eval("▼myname test ◆")

	result, err := e.Eval("▶COMPARE ▶▲myname ◆ world ◆")
	if err != nil {
		t.Fatalf("dynamic execute compare false failed: %v", err)
	}
	if result != "FALSE" {
		t.Errorf("expected 'FALSE', got %q", result)
	}
}

// TestDynamicExecArgExtract tests ▶▲name ◆ as argument to EXTRACT
func TestDynamicExecArgExtract(t *testing.T) {
	e := New()

	e.Eval("▼response SENTIMENT: positive ◆")
	e.Eval("▼respname response ◆")

	result, err := e.Eval("▶EXTRACT SENTIMENT ▶▲respname ◆ ◆")
	if err != nil {
		t.Fatalf("dynamic execute extract failed: %v", err)
	}
	if result != "positive" {
		t.Errorf("expected 'positive', got %q", result)
	}
}

// TestDynamicExecArgUpper tests ▶▲name ◆ as argument to UPPER
func TestDynamicExecArgUpper(t *testing.T) {
	e := New()

	e.Eval("▼test hello ◆")
	e.Eval("▼myname test ◆")

	result, err := e.Eval("▶UPPER ▶▲myname ◆ ◆")
	if err != nil {
		t.Fatalf("dynamic execute upper failed: %v", err)
	}
	if result != "HELLO" {
		t.Errorf("expected 'HELLO', got %q", result)
	}
}

// TestDynamicExecArgLower tests ▶▲name ◆ as argument to LOWER
func TestDynamicExecArgLower(t *testing.T) {
	e := New()

	e.Eval("▼test HELLO ◆")
	e.Eval("▼myname test ◆")

	result, err := e.Eval("▶LOWER ▶▲myname ◆ ◆")
	if err != nil {
		t.Fatalf("dynamic execute lower failed: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

// TestDynamicExecArgTrim tests ▶▲name ◆ as argument to TRIM
func TestDynamicExecArgTrim(t *testing.T) {
	e := New()

	e.Eval("▼test    hello    ◆")
	e.Eval("▼myname test ◆")

	result, err := e.Eval("▶TRIM ▶▲myname ◆ ◆")
	if err != nil {
		t.Fatalf("dynamic execute trim failed: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

// TestDynamicExecArgSay tests ▶▲name ◆ as argument to SAY (should still work)
func TestDynamicExecArgSay(t *testing.T) {
	var output strings.Builder
	e := New(WithOutputWriter(func(text string) error {
		output.WriteString(text)
		return nil
	}))

	e.Eval("▼test hello ◆")
	e.Eval("▼myname test ◆")

	_, err := e.Eval("▶SAY ▶▲myname ◆ ◆")
	if err != nil {
		t.Fatalf("dynamic execute say failed: %v", err)
	}
	if output.String() != "hello\n" {
		t.Errorf("expected output 'hello\\n', got %q", output.String())
	}
}

// TestDynamicExecArgTopLevel tests ▶▲name ◆ at top level (should work)
func TestDynamicExecArgTopLevel(t *testing.T) {
	e := New()

	e.Eval("▼test hello ◆")
	e.Eval("▼myname test ◆")

	result, err := e.Eval("▶▲myname ◆")
	if err != nil {
		t.Fatalf("dynamic execute top level failed: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

// TestDynamicExecArgStoreThenUse tests workaround: store in temp then use
func TestDynamicExecArgStoreThenUse(t *testing.T) {
	e := New()

	e.Eval("▼test hello ◆")
	e.Eval("▼myname test ◆")
	e.Eval("▽temp ▶▲myname ◆ ◆")

	result, err := e.Eval("▶COMPARE ▶temp ◆ hello ◆")
	if err != nil {
		t.Fatalf("store-then-use compare failed: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("expected 'TRUE', got %q", result)
	}
}

// TestDynamicExecArgExtractMultiLevel tests the MUDDN pattern:
// ▶EXTRACT FIELD ▶▲name ◆ ◆
func TestDynamicExecArgExtractMultiLevel(t *testing.T) {
	e := New()

	e.Eval("▼response SENTIMENT: positive\nCONFIDENCE: high ◆")
	e.Eval("▼respname response ◆")

	result, err := e.Eval("▶EXTRACT SENTIMENT ▶▲respname ◆ ◆")
	if err != nil {
		t.Fatalf("extract multi-level failed: %v", err)
	}
	if result != "positive" {
		t.Errorf("expected 'positive', got %q", result)
	}

	result, err = e.Eval("▶EXTRACT CONFIDENCE ▶▲respname ◆ ◆")
	if err != nil {
		t.Fatalf("extract confidence failed: %v", err)
	}
	if result != "high" {
		t.Errorf("expected 'high', got %q", result)
	}
}

// TestDynamicExecArgIF tests ▶▲name ◆ as condition in IF
func TestDynamicExecArgIF(t *testing.T) {
	e := New()

	e.Eval("▼test TRUE ◆")
	e.Eval("▼myname test ◆")

	result, err := e.Eval("▶IF ▶▲myname ◆\nmatched\nnot matched\n◆")
	if err != nil {
		t.Fatalf("dynamic exec as IF condition failed: %v", err)
	}
	if result != "matched" {
		t.Errorf("expected 'matched', got %q", result)
	}
}

// TestDynamicExecArgCompareWithImmRetrieve tests both args as dynamic execute
func TestDynamicExecArgCompareBothDynamic(t *testing.T) {
	e := New()

	e.Eval("▼val1 hello ◆")
	e.Eval("▼val2 hello ◆")
	e.Eval("▼name1 val1 ◆")
	e.Eval("▼name2 val2 ◆")

	result, err := e.Eval("▶COMPARE ▶▲name1 ◆ ▶▲name2 ◆ ◆")
	if err != nil {
		t.Fatalf("both dynamic compare failed: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("expected 'TRUE', got %q", result)
	}
}

// TestDynamicRetrieveArgCompare tests ▲▲name as argument to COMPARE via parseArgs
func TestDynamicRetrieveArgCompare(t *testing.T) {
	e := New()

	e.Eval("▽target hello ◆")
	e.Eval("▽ref target ◆")

	// ▲▲ref should resolve to value of target ("hello")
	result, err := e.Eval("▶COMPARE ▲▲ref hello ◆")
	if err != nil {
		t.Fatalf("dynamic retrieve arg compare failed: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("expected 'TRUE', got %q", result)
	}
}
