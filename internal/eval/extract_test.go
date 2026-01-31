package eval

import (
	"testing"
)

func TestExtractSimple(t *testing.T) {
	e := New()

	// Store source text with labels
	e.Eval("▽Source NAME: Alice\nAGE: 30 ◆")

	// Extract NAME
	result, err := e.Eval(`▶EXTRACT NAME
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Alice" {
		t.Errorf("expected 'Alice', got '%s'", result)
	}

	// Extract AGE
	result, err = e.Eval(`▶EXTRACT AGE
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "30" {
		t.Errorf("expected '30', got '%s'", result)
	}
}

func TestExtractMultiLine(t *testing.T) {
	e := New()

	// Store source with multi-line value
	e.Eval("▽Source NAME: Alice\nBIO: She is a programmer.\nShe loves coding.\nAGE: 30 ◆")

	// Extract multi-line BIO
	result, err := e.Eval(`▶EXTRACT BIO
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "She is a programmer.\nShe loves coding."
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestExtractCaseInsensitive(t *testing.T) {
	e := New()

	// Store source with mixed case labels
	e.Eval("▽Source Name: Alice ◆")

	// Extract with uppercase label
	result, err := e.Eval(`▶EXTRACT NAME
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Alice" {
		t.Errorf("expected 'Alice', got '%s'", result)
	}

	// Extract with lowercase label
	result, err = e.Eval(`▶EXTRACT name
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Alice" {
		t.Errorf("expected 'Alice', got '%s'", result)
	}
}

func TestExtractMissingLabel(t *testing.T) {
	e := New()

	e.Eval("▽Source NAME: Alice ◆")

	// Extract non-existent label
	result, err := e.Eval(`▶EXTRACT MISSING
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for missing label, got '%s'", result)
	}
}

func TestExtractEmptyValue(t *testing.T) {
	e := New()

	// Label with empty value followed by another label
	e.Eval("▽Source EMPTY:\nNAME: Alice ◆")

	result, err := e.Eval(`▶EXTRACT EMPTY
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

func TestExtractLastLabel(t *testing.T) {
	e := New()

	// Last label in the source
	e.Eval("▽Source NAME: Alice\nAGE: 30 ◆")

	result, err := e.Eval(`▶EXTRACT AGE
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "30" {
		t.Errorf("expected '30', got '%s'", result)
	}
}

func TestExtractWithUnderscore(t *testing.T) {
	e := New()

	// Label with underscores
	e.Eval("▽Source FIRST_NAME: Alice\nLAST_NAME: Smith ◆")

	result, err := e.Eval(`▶EXTRACT FIRST_NAME
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Alice" {
		t.Errorf("expected 'Alice', got '%s'", result)
	}
}

func TestExtractIgnoresNonLabels(t *testing.T) {
	e := New()

	// Text that looks like labels but isn't (contains special chars)
	e.Eval("▽Source NAME: Alice\nNOT-A-LABEL: ignored\nAGE: 30 ◆")

	// Extract NAME - should get value including the "NOT-A-LABEL" line until AGE
	result, err := e.Eval(`▶EXTRACT NAME
▲Source ◆`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Alice\nNOT-A-LABEL: ignored" {
		t.Errorf("expected 'Alice\\nNOT-A-LABEL: ignored', got '%s'", result)
	}
}

func TestExtractInsufficientArgs(t *testing.T) {
	e := New()

	// Only one argument
	result, err := e.Eval("▶EXTRACT NAME ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for insufficient args, got '%s'", result)
	}
}
