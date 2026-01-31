package eval

import (
	"strings"
	"testing"
)

func TestTraceMaybeDebug(t *testing.T) {
	var output strings.Builder
	e := New(WithOutputWriter(func(text string) error {
		output.WriteString(text)
		return nil
	}))
	
	// Define ShowDebug first
	e.Eval("▼ShowDebug ▶SAY [DEBUG] shown ◆ ◆")
	e.Eval("▼DoNothing ◆")
	
	// Set debug to TRUE
	e.Eval("▽NPC_System_Debug TRUE ◆")
	
	// Check what's stored in ShowDebug
	t.Logf("ShowDebug stored: %q", e.Namespace().Get("ShowDebug"))
	t.Logf("DoNothing stored: %q", e.Namespace().Get("DoNothing"))
	t.Logf("NPC_System_Debug: %q", e.Namespace().Get("NPC_System_Debug"))
	
	// Define MaybeShowDebug
	e.Eval(`▼MaybeShowDebug
    ▶IF ▷COMPARE ▲NPC_System_Debug TRUE ◆
        ▶ShowDebug ◆
        ▶DoNothing ◆
    ◆
◆`)
	
	// Check what's stored in MaybeShowDebug
	t.Logf("MaybeShowDebug stored: %q", e.Namespace().Get("MaybeShowDebug"))
	
	// Now execute
	output.Reset()
	result, err := e.Eval("▶MaybeShowDebug ◆")
	if err != nil {
		t.Fatalf("MaybeShowDebug error: %v", err)
	}
	t.Logf("MaybeShowDebug result: %q", result)
	t.Logf("Output: %q", output.String())
}
