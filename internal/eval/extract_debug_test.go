// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"fmt"
	"testing"
)

func TestExtractStackOverflowDebug(t *testing.T) {
	e := New()

	// Simple EXTRACT test
	source := `VALENCE: SAME
AROUSAL: excited
CURRENT: curious`

	e.Eval("▽source " + source + " ◆")

	fmt.Println("=== Testing EXTRACT directly ===")
	result, err := e.Eval("▶EXTRACT VALENCE ▲source ◆")
	fmt.Printf("EXTRACT VALENCE result: %q, err: %v\n", result, err)

	result2, err := e.Eval("▶EXTRACT AROUSAL ▲source ◆")
	fmt.Printf("EXTRACT AROUSAL result: %q, err: %v\n", result2, err)

	result3, err := e.Eval("▶EXTRACT CURRENT ▲source ◆")
	fmt.Printf("EXTRACT CURRENT result: %q, err: %v\n", result3, err)
}

func TestExtractInDeferredStore(t *testing.T) {
	e := New()

	source := `VALENCE: SAME
AROUSAL: excited
CURRENT: curious`

	e.Eval("▽_ie_raw " + source + " ◆")

	fmt.Println("=== Testing ▼ with EXTRACT ===")
	// This is how npc.losp uses it
	_, err := e.Eval("▼_ie_valence ▶EXTRACT VALENCE ▲_ie_raw ◆ ◆")
	if err != nil {
		t.Fatalf("failed to define _ie_valence: %v", err)
	}

	// Check what's stored
	raw := e.namespace.Get("_ie_valence")
	fmt.Printf("_ie_valence stored: %T = %q\n", raw, raw.String())

	// Try to retrieve it
	fmt.Println("=== Retrieving _ie_valence ===")
	result, err := e.Eval("▲_ie_valence")
	fmt.Printf("▲_ie_valence result: %q, err: %v\n", result, err)
}
