// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"testing"
)

func TestExtractStackOverflow(t *testing.T) {
	e := New()

	source := `VALENCE: SAME
AROUSAL: excited
CURRENT: curious`

	e.Eval("▽source " + source + " ◆")

	result, err := e.Eval("▶EXTRACT VALENCE ▲source ◆")
	if err != nil {
		t.Fatalf("EXTRACT VALENCE failed: %v", err)
	}
	if result != "SAME" {
		t.Errorf("EXTRACT VALENCE = %q, want %q", result, "SAME")
	}

	result2, err := e.Eval("▶EXTRACT AROUSAL ▲source ◆")
	if err != nil {
		t.Fatalf("EXTRACT AROUSAL failed: %v", err)
	}
	if result2 != "excited" {
		t.Errorf("EXTRACT AROUSAL = %q, want %q", result2, "excited")
	}

	result3, err := e.Eval("▶EXTRACT CURRENT ▲source ◆")
	if err != nil {
		t.Fatalf("EXTRACT CURRENT failed: %v", err)
	}
	if result3 != "curious" {
		t.Errorf("EXTRACT CURRENT = %q, want %q", result3, "curious")
	}
}

func TestExtractInDeferredStore(t *testing.T) {
	e := New()

	source := `VALENCE: SAME
AROUSAL: excited
CURRENT: curious`

	e.Eval("▽_ie_raw " + source + " ◆")

	_, err := e.Eval("▼_ie_valence ▶EXTRACT VALENCE ▲_ie_raw ◆ ◆")
	if err != nil {
		t.Fatalf("failed to define _ie_valence: %v", err)
	}

	result, err := e.Eval("▶_ie_valence ◆")
	if err != nil {
		t.Fatalf("failed to execute _ie_valence: %v", err)
	}
	if result != "SAME" {
		t.Errorf("_ie_valence = %q, want %q", result, "SAME")
	}
}
