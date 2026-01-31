package eval

import (
	"os"
	"strings"
	"testing"
)

func TestTraceExtractionPrompts(t *testing.T) {
	content, err := os.ReadFile("../../examples/simulation.losp")
	if err != nil {
		t.Fatal(err)
	}
	// Remove the startup to prevent auto-execution
	contentStr := strings.Replace(string(content), "▶Sim_Init ◆", "# REMOVED", 1)

	var prompts []struct{ system, user string }

	e := New(WithProvider(&testTraceProvider{
		handler: func(system, user string) string {
			prompts = append(prompts, struct{ system, user string }{system, user})
			// Return a mock response with the expected format for EXTRACT
			return `NAME: Test Character
TRAITS: curious, thoughtful, calm
BELIEFS: CORE: honesty matters, kindness is strength
GOALS: find meaning, help others
MOOD: contemplative
BACKSTORY_SUMMARY: A thoughtful individual seeking purpose.
BACKSTORY_FULL: Full backstory here.`
		},
	}))

	_, err = e.Eval(contentStr)
	if err != nil {
		t.Fatal(err)
	}

	// Set up character inputs
	e.Eval("▼Sim_Char_Ethnicity test ◆")
	e.Eval("▼Sim_Char_Age adult ◆")
	e.Eval("▼Sim_Char_Gender neutral ◆")
	e.Eval("▼Sim_Char_Archetype artist ◆")

	prompts = nil
	_, err = e.Eval("▶Sim_GenerateCharacter ◆")
	if err != nil {
		t.Fatalf("Sim_GenerateCharacter failed: %v", err)
	}

	t.Logf("GenerateCharacter prompts: %d", len(prompts))
	for i, p := range prompts {
		t.Logf("Prompt %d:", i)
		t.Logf("  System (first 100): %q", truncStr(p.system, 100))
		t.Logf("  User (first 100): %q", truncStr(p.user, 100))
	}

	// Verify character was parsed
	name, _ := e.Eval("▲Sim_Char_Name")
	t.Logf("Parsed name: %q", name)

	if name == "" {
		t.Error("Expected Sim_Char_Name to be set after parsing")
	}
}

type testTraceProvider struct {
	handler func(system, user string) string
}

func (t *testTraceProvider) Prompt(system, user string) (string, error) {
	return t.handler(system, user), nil
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
