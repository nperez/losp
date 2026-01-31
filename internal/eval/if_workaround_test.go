package eval

import (
	"strings"
	"testing"
)

func TestIFWithTemplateNames(t *testing.T) {
	var output strings.Builder
	e := New(WithOutputWriter(func(text string) error {
		output.WriteString(text)
		return nil
	}))

	// Define templates that produce output
	e.Eval("▼ThenBranch ▶SAY THEN ◆ ◆")
	e.Eval("▼ElseBranch ▶SAY ELSE ◆ ◆")

	// Set condition to FALSE
	e.Eval("▽Flag FALSE ◆")

	// IF returns the NAME, not the executed result (per PRIMER)
	output.Reset()
	result, _ := e.Eval(`▶IF ▷COMPARE ▲Flag TRUE ◆
		ThenBranch
		ElseBranch
	◆`)

	t.Logf("Result: %q", result)

	// IF should return "ElseBranch" (the name), not execute it
	if result != "ElseBranch" {
		t.Errorf("IF should return branch name, got %q", result)
	}

	// No output should have occurred - IF doesn't execute branches
	if output.String() != "" {
		t.Errorf("IF should not execute branches, got output: %q", output.String())
	}

	// Use dynamic execution to execute only the selected branch
	output.Reset()
	result, _ = e.Eval(`▶▶IF ▷COMPARE ▲Flag TRUE ◆
		ThenBranch
		ElseBranch
	◆ ◆`)

	t.Logf("Dynamic exec result: %q", result)
	t.Logf("Dynamic exec output: %q", output.String())

	// Only ELSE branch should have executed
	hasThen := strings.Contains(output.String(), "THEN")
	hasElse := strings.Contains(output.String(), "ELSE")

	if hasThen {
		t.Error("THEN branch was executed when condition was FALSE!")
	}
	if !hasElse {
		t.Error("ELSE branch should have been executed via dynamic exec")
	}
}
