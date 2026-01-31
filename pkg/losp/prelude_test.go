package losp

import (
	"strings"
	"testing"
)

func TestPreludeStartupExists(t *testing.T) {
	r := New(WithMemoryStore())
	defer r.Close()

	// Verify __startup__ is defined (though empty by default)
	result, err := r.Eval(`▲__startup__`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty startup should return empty
	if strings.TrimSpace(result) != "" {
		t.Errorf("expected empty __startup__, got '%s'", result)
	}
}

func TestNoStdlibOption(t *testing.T) {
	r := New(WithMemoryStore(), WithNoStdlib())
	defer r.Close()

	// Verify __startup__ is NOT available when stdlib is disabled
	result, err := r.Eval(`▲__startup__`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be empty since prelude wasn't loaded
	if result != "" {
		t.Errorf("expected empty (no prelude), got '%s'", result)
	}
}

func TestCustomPrelude(t *testing.T) {
	customPrelude := `▼MyCustomFunc hello world ◆`

	r := New(WithMemoryStore(), WithPrelude(customPrelude))
	defer r.Close()

	// Verify custom function is available
	result, err := r.Eval(`▲MyCustomFunc`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "hello world") {
		t.Errorf("expected 'hello world', got '%s'", result)
	}
}

func TestDatabasePreludeOverride(t *testing.T) {
	r := New(WithMemoryStore())
	defer r.Close()

	// Define custom stdlib and persist it
	_, err := r.Eval(`▼__stdlib__
▼CustomOverride overridden ◆
◆`)
	if err != nil {
		t.Fatalf("failed to define __stdlib__: %v", err)
	}

	_, err = r.Eval(`▶PERSIST __stdlib__ ◆`)
	if err != nil {
		t.Fatalf("failed to persist __stdlib__: %v", err)
	}

	// Note: In this test, we can't easily verify the override works on a fresh
	// runtime without keeping the same store. This is more of an integration test
	// that would require a more complex setup.
}
