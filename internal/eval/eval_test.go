package eval

import (
	"strings"
	"testing"

	"nickandperla.net/losp/internal/expr"
	"nickandperla.net/losp/internal/store"
)

func TestBasicSay(t *testing.T) {
	var output strings.Builder
	e := New(WithOutputWriter(func(text string) error {
		output.WriteString(text)
		return nil
	}))

	result, err := e.Eval("▶SAY Hello ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// SAY returns empty (output already happened via outputWriter)
	if result != "" {
		t.Errorf("expected empty result, got '%s'", result)
	}
	if output.String() != "Hello\n" {
		t.Errorf("expected output 'Hello\\n', got '%s'", output.String())
	}
}

func TestStoreRetrieve(t *testing.T) {
	e := New()

	// Test immediate store and retrieve
	_, err := e.Eval("▽X hello ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := e.Eval("▲X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestDeferredStore(t *testing.T) {
	e := New()

	// Store a template with deferred retrieve
	_, err := e.Eval("▼Template Value is: ▲X ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Set X
	_, err = e.Eval("▽X first ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Execute template
	result, err := e.Eval("▶Template ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "first") {
		t.Errorf("expected result to contain 'first', got '%s'", result)
	}

	// Change X and re-execute
	_, err = e.Eval("▽X second ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err = e.Eval("▶Template ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "second") {
		t.Errorf("expected result to contain 'second', got '%s'", result)
	}
}

func TestPlaceholder(t *testing.T) {
	e := New()

	// Define a template with placeholder
	_, err := e.Eval("▼Greet □name Hello, ▲name! ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Execute with argument
	result, err := e.Eval("▶Greet Alice ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Alice") {
		t.Errorf("expected result to contain 'Alice', got '%s'", result)
	}
}

func TestCompare(t *testing.T) {
	e := New()

	tests := []struct {
		input    string
		expected string
	}{
		{"▶COMPARE\nhello\nhello\n◆", "TRUE"},
		{"▶COMPARE\nhello\nworld\n◆", "FALSE"},
		{"▶COMPARE\n  hello  \n  hello  \n◆", "TRUE"}, // whitespace trimmed
	}

	for _, tt := range tests {
		result, err := e.Eval(tt.input)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("for %s: expected '%s', got '%s'", tt.input, tt.expected, result)
		}
	}
}

func TestIf(t *testing.T) {
	e := New()

	// Set up expressions
	e.Eval("▼Yes yes-result ◆")
	e.Eval("▼No no-result ◆")

	// Test TRUE branch
	result, err := e.Eval("▶IF TRUE ▲Yes ▲No ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "yes-result" {
		t.Errorf("expected 'yes-result', got '%s'", result)
	}

	// Test FALSE branch
	result, err = e.Eval("▶IF FALSE ▲Yes ▲No ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "no-result" {
		t.Errorf("expected 'no-result', got '%s'", result)
	}
}

func TestForeach(t *testing.T) {
	var output strings.Builder
	e := New(WithOutputWriter(func(text string) error {
		output.WriteString(text)
		return nil
	}))

	// Define item handler and items list
	e.Eval("▼ShowItem □item - ▲item ◆")
	e.Eval("▼Items\napple\nbanana\n◆")

	result, err := e.Eval("▶FOREACH\n▲Items\nShowItem\n◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "apple") || !strings.Contains(result, "banana") {
		t.Errorf("expected result to contain items, got '%s'", result)
	}
}

func TestCount(t *testing.T) {
	e := New()

	e.Eval("▽Text line1\nline2\nline3 ◆")

	result, err := e.Eval("▶COUNT ▲Text ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "3" {
		t.Errorf("expected '3', got '%s'", result)
	}
}

func TestAppend(t *testing.T) {
	e := New()

	e.Eval("▽List first ◆")
	e.Eval(`▶APPEND List
second ◆`)

	result, err := e.Eval("▲List")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "first") || !strings.Contains(result, "second") {
		t.Errorf("expected result to contain both items, got '%s'", result)
	}
}

func TestTrueFalseEmpty(t *testing.T) {
	e := New()

	result, _ := e.Eval("▶TRUE ◆")
	if result != "TRUE" {
		t.Errorf("expected 'TRUE', got '%s'", result)
	}

	result, _ = e.Eval("▶FALSE ◆")
	if result != "FALSE" {
		t.Errorf("expected 'FALSE', got '%s'", result)
	}

	result, _ = e.Eval("▶EMPTY ◆")
	if result != "" {
		t.Errorf("expected empty, got '%s'", result)
	}
}

func TestMockProvider(t *testing.T) {
	e := New(WithProvider(&mockProvider{response: "test-response"}))

	result, err := e.Eval("▶PROMPT system user ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test-response" {
		t.Errorf("expected 'test-response', got '%s'", result)
	}
}

type mockProvider struct {
	response string
}

func (m *mockProvider) Prompt(system, user string) (string, error) {
	return m.response, nil
}

// Tests for semantic fixes: load operators re-parse retrieved values

func TestDeferOperatorWithRetrieve(t *testing.T) {
	// Test case from PRIMER: defer prevents immediate parsing,
	// but when later retrieved, the expression is parsed
	e := New()

	// Store "△X" using defer to prevent immediate resolution
	// ◯ needs its own ◆, then ▽ needs its own ◆
	_, err := e.Eval("▽Template ◯△X ◆ ◆")
	if err != nil {
		t.Fatalf("failed to store template: %v", err)
	}

	// Set X to "first"
	_, err = e.Eval("▽X first ◆")
	if err != nil {
		t.Fatalf("failed to set X: %v", err)
	}

	// Retrieve Template - should re-parse "△X" and resolve to "first"
	result, err := e.Eval("▲Template")
	if err != nil {
		t.Fatalf("failed to retrieve template: %v", err)
	}
	if result != "first" {
		t.Errorf("expected 'first', got '%s'", result)
	}

	// Change X to "second"
	_, err = e.Eval("▽X second ◆")
	if err != nil {
		t.Fatalf("failed to update X: %v", err)
	}

	// Retrieve Template again - should STILL return "first" because
	// △X fired on first retrieve and was replaced by its result (ephemeral semantic)
	result, err = e.Eval("▲Template")
	if err != nil {
		t.Fatalf("failed to retrieve template again: %v", err)
	}
	if result != "first" {
		t.Errorf("expected 'first' (cached from first retrieve), got '%s'", result)
	}
}

func TestChainedDefer(t *testing.T) {
	// Test chained defer: a contains "△b", b contains "X"
	e := New()

	// a = "△b" (deferred)
	// ◯ needs its own ◆, then ▽ needs its own ◆
	_, err := e.Eval("▽a ◯△b ◆ ◆")
	if err != nil {
		t.Fatalf("failed to store a: %v", err)
	}

	// b = "X"
	_, err = e.Eval("▽b X ◆")
	if err != nil {
		t.Fatalf("failed to store b: %v", err)
	}

	// ▲a should re-parse "△b", which resolves to "X"
	result, err := e.Eval("▲a")
	if err != nil {
		t.Fatalf("failed to retrieve a: %v", err)
	}
	if result != "X" {
		t.Errorf("expected 'X', got '%s'", result)
	}
}

func TestDynamicNamingWithReparse(t *testing.T) {
	// Test dynamic naming where the target contains a deferred expression
	e := New()

	// target = "△varname" (deferred)
	// ◯ needs its own ◆, then ▽ needs its own ◆
	_, err := e.Eval("▽target ◯△varname ◆ ◆")
	if err != nil {
		t.Fatalf("failed to store target: %v", err)
	}

	// varname = "X"
	_, err = e.Eval("▽varname X ◆")
	if err != nil {
		t.Fatalf("failed to store varname: %v", err)
	}

	// ▽▲target hello ◆ should:
	// 1. ▲target retrieves "△varname"
	// 2. Re-parse "△varname" → resolves to "X"
	// 3. Store "hello" to X
	_, err = e.Eval("▽▲target hello ◆")
	if err != nil {
		t.Fatalf("failed to store with dynamic naming: %v", err)
	}

	// X should now be "hello"
	result, err := e.Eval("▲X")
	if err != nil {
		t.Fatalf("failed to retrieve X: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestImmediateVsDeferredTiming(t *testing.T) {
	// Test that △ captures at parse time while ▲ resolves at execution time
	e := New()

	// X = "first"
	_, err := e.Eval("▽X first ◆")
	if err != nil {
		t.Fatalf("failed to set X: %v", err)
	}

	// Snapshot captures △X NOW (at parse time) = "first"
	_, err = e.Eval("▽Snapshot △X ◆")
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	// X = "second"
	_, err = e.Eval("▽X second ◆")
	if err != nil {
		t.Fatalf("failed to update X: %v", err)
	}

	// Snapshot should still be "first" (captured at parse time)
	result, err := e.Eval("▲Snapshot")
	if err != nil {
		t.Fatalf("failed to retrieve snapshot: %v", err)
	}
	if result != "first" {
		t.Errorf("expected 'first', got '%s'", result)
	}

	// X should now be "second"
	result, err = e.Eval("▲X")
	if err != nil {
		t.Fatalf("failed to retrieve X: %v", err)
	}
	if result != "second" {
		t.Errorf("expected 'second', got '%s'", result)
	}
}

func TestDynamicNamingWithDeferredRetrieve(t *testing.T) {
	// Test that ▲ works in the name position of store operators
	e := New()

	// fieldName = "MyField"
	_, err := e.Eval("▽fieldName MyField ◆")
	if err != nil {
		t.Fatalf("failed to store fieldName: %v", err)
	}

	// ▽▲fieldName hello ◆ should store "hello" to MyField
	_, err = e.Eval("▽▲fieldName hello ◆")
	if err != nil {
		t.Fatalf("failed to store with dynamic naming: %v", err)
	}

	// MyField should be "hello"
	result, err := e.Eval("▲MyField")
	if err != nil {
		t.Fatalf("failed to retrieve MyField: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestNestedExecuteInStore(t *testing.T) {
	// Test that ▶ inside ▼ is preserved and executed on retrieval
	var output strings.Builder
	e := New(WithOutputWriter(func(text string) error {
		output.WriteString(text)
		return nil
	}))

	// Store a template with nested execute
	_, err := e.Eval("▼foo before▶SAY hello ◆after ◆")
	if err != nil {
		t.Fatalf("failed to store: %v", err)
	}
	
	// Should NOT have printed anything yet
	if output.String() != "" {
		t.Errorf("expected no output during store, got '%s'", output.String())
	}

	// Execute the template - NOW it should print
	output.Reset()
	result, err := e.Eval("▶foo ◆")
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}
	
	// Should have printed "hello" via SAY
	if output.String() != "hello\n" {
		t.Errorf("expected 'hello\\n', got '%s'", output.String())
	}
	
	// Result should be "beforeafter" (SAY returns empty)
	if result != "beforeafter" {
		t.Errorf("expected 'beforeafter', got '%s'", result)
	}
}

// =============================================================================
// PRIMER.md Conformance Tests
// These tests verify behavior matches the examples in PRIMER.md exactly.
// =============================================================================

// PRIMER.md lines 47-57: Parse-Time Examples
// △X resolves NOW at parse time, so Snapshot captures "first"
func TestPRIMER_ParseTimeSnapshot(t *testing.T) {
	e := New()

	// ▽X first ◆
	_, err := e.Eval("▽X first ◆")
	if err != nil {
		t.Fatalf("failed to set X: %v", err)
	}

	// ▽Snapshot △X ◆ - △X resolves NOW to "first", stored in Snapshot
	_, err = e.Eval("▽Snapshot △X ◆")
	if err != nil {
		t.Fatalf("failed to store Snapshot: %v", err)
	}

	// ▽X second ◆
	_, err = e.Eval("▽X second ◆")
	if err != nil {
		t.Fatalf("failed to update X: %v", err)
	}

	// ▲Snapshot → "first" (captured at parse time)
	result, err := e.Eval("▲Snapshot")
	if err != nil {
		t.Fatalf("failed to retrieve Snapshot: %v", err)
	}
	if result != "first" {
		t.Errorf("PRIMER.md line 55: expected 'first', got '%s'", result)
	}

	// ▲X → "second" (current value)
	result, err = e.Eval("▲X")
	if err != nil {
		t.Fatalf("failed to retrieve X: %v", err)
	}
	if result != "second" {
		t.Errorf("PRIMER.md line 56: expected 'second', got '%s'", result)
	}
}

// PRIMER.md lines 64-76: Execution-Time Examples
// ▲X inside Expression is not resolved until ▶Expression ◆ executes
func TestPRIMER_ExecutionTimeDeferred(t *testing.T) {
	e := New()

	// ▼Expression Current value: ▲X ◆
	_, err := e.Eval("▼Expression Current value: ▲X ◆")
	if err != nil {
		t.Fatalf("failed to store Expression: %v", err)
	}

	// ▽X first ◆
	_, err = e.Eval("▽X first ◆")
	if err != nil {
		t.Fatalf("failed to set X: %v", err)
	}

	// ▶Expression ◆ → "Current value: first"
	result, err := e.Eval("▶Expression ◆")
	if err != nil {
		t.Fatalf("failed to execute Expression: %v", err)
	}
	if result != "Current value: first" {
		t.Errorf("PRIMER.md line 71: expected 'Current value: first', got '%s'", result)
	}

	// ▽X second ◆
	_, err = e.Eval("▽X second ◆")
	if err != nil {
		t.Fatalf("failed to update X: %v", err)
	}

	// ▶Expression ◆ → "Current value: second"
	result, err = e.Eval("▶Expression ◆")
	if err != nil {
		t.Fatalf("failed to execute Expression again: %v", err)
	}
	if result != "Current value: second" {
		t.Errorf("PRIMER.md line 73: expected 'Current value: second', got '%s'", result)
	}
}

// PRIMER.md lines 78-90: The Defer Operator
// ◯ prevents parse-time resolution, allowing △X to fire on each ▲Expression
func TestPRIMER_DeferOperator(t *testing.T) {
	e := New()

	// ▽Expression ◯△X ◆ ◆ - Stores the expression △X itself, not its value
	_, err := e.Eval("▽Expression ◯△X ◆ ◆")
	if err != nil {
		t.Fatalf("failed to store Expression: %v", err)
	}

	// ▽X first ◆
	_, err = e.Eval("▽X first ◆")
	if err != nil {
		t.Fatalf("failed to set X: %v", err)
	}

	// ▲Expression - NOW △X resolves → "first"
	result, err := e.Eval("▲Expression")
	if err != nil {
		t.Fatalf("failed to retrieve Expression: %v", err)
	}
	if result != "first" {
		t.Errorf("PRIMER.md line 85: expected 'first', got '%s'", result)
	}

	// NOTE: Due to ephemeral semantic, the second ▲Expression returns "first"
	// because △X was consumed on first retrieve and replaced by "first".
	// This is expected behavior per CLAUDE.md ephemeral semantics.
	// To get "second", use ▶ (execute) not ▲ (retrieve).
}

// PRIMER.md lines 190-213: Dynamic Naming
// Store operators support dynamic naming - name can be computed at runtime
func TestPRIMER_DynamicNaming(t *testing.T) {
	e := New()

	// ▼FieldName X ◆
	_, err := e.Eval("▼FieldName X ◆")
	if err != nil {
		t.Fatalf("failed to store FieldName: %v", err)
	}

	// ▼▲FieldName hello ◆ - ▲FieldName resolves to "X", stores "hello" to X
	_, err = e.Eval("▼▲FieldName hello ◆")
	if err != nil {
		t.Fatalf("failed to store with dynamic naming: %v", err)
	}

	// ▲X → "hello"
	result, err := e.Eval("▲X")
	if err != nil {
		t.Fatalf("failed to retrieve X: %v", err)
	}
	if result != "hello" {
		t.Errorf("PRIMER.md line 197: expected 'hello', got '%s'", result)
	}
}

// PRIMER.md lines 219-243: Placeholder Arguments
func TestPRIMER_PlaceholderArguments(t *testing.T) {
	e := New()

	// ▼Greet □name Hello, ▲name! ◆
	_, err := e.Eval("▼Greet □name Hello, ▲name! ◆")
	if err != nil {
		t.Fatalf("failed to store Greet: %v", err)
	}

	// ▶Greet Alice ◆ → "Hello, Alice!"
	result, err := e.Eval("▶Greet Alice ◆")
	if err != nil {
		t.Fatalf("failed to execute Greet: %v", err)
	}
	if result != "Hello, Alice!" {
		t.Errorf("PRIMER.md line 228: expected 'Hello, Alice!', got '%s'", result)
	}

	// ▲name → "Alice" (still in global namespace)
	result, err = e.Eval("▲name")
	if err != nil {
		t.Fatalf("failed to retrieve name: %v", err)
	}
	if result != "Alice" {
		t.Errorf("PRIMER.md line 229: expected 'Alice', got '%s'", result)
	}
}

// PRIMER.md lines 282-299: Placeholder Clobbering
func TestPRIMER_PlaceholderClobbering(t *testing.T) {
	e := New()

	// ▼Outer □x ▶Inner one ◆ ▲x ◆
	_, err := e.Eval("▼Outer □x ▶Inner one ◆ ▲x ◆")
	if err != nil {
		t.Fatalf("failed to store Outer: %v", err)
	}

	// ▼Inner □x ▲x ◆
	_, err = e.Eval("▼Inner □x ▲x ◆")
	if err != nil {
		t.Fatalf("failed to store Inner: %v", err)
	}

	// ▶Outer two ◆ - Inner sets x="one" and returns "one", then Outer's ▲x returns "one"
	// Results are concatenated: "one" + "one" = "oneone" (no space - direct concat)
	result, err := e.Eval("▶Outer two ◆")
	if err != nil {
		t.Fatalf("failed to execute Outer: %v", err)
	}
	// Inner returns "one", then ▲x returns "one" - both clobbered to same value
	if result != "one one" {
		t.Errorf("PRIMER.md clobbering: expected 'one one', got '%s'", result)
	}
}

// PRIMER.md lines 305-331: Control Flow - IF
func TestPRIMER_ControlFlowIF(t *testing.T) {
	e := New()

	// Test with TRUE condition
	_, err := e.Eval("▽State new ◆")
	if err != nil {
		t.Fatalf("failed to set State: %v", err)
	}

	result, err := e.Eval(`▶IF ▶COMPARE ▲State new ◆
    Setting up...
    Already initialized
◆`)
	if err != nil {
		t.Fatalf("failed to execute IF: %v", err)
	}
	if result != "Setting up..." {
		t.Errorf("PRIMER.md IF with TRUE: expected 'Setting up...', got '%s'", result)
	}

	// Test with FALSE condition
	_, err = e.Eval("▽State old ◆")
	if err != nil {
		t.Fatalf("failed to update State: %v", err)
	}

	result, err = e.Eval(`▶IF ▶COMPARE ▲State new ◆
    Setting up...
    Already initialized
◆`)
	if err != nil {
		t.Fatalf("failed to execute IF: %v", err)
	}
	if result != "Already initialized" {
		t.Errorf("PRIMER.md IF with FALSE: expected 'Already initialized', got '%s'", result)
	}
}

// Test that ▲ (retrieve) returns the stored body, and ▶ (execute) evaluates it
// ▲ retrieves content as text; ▶ executes and evaluates deferred operators
func TestRetrieveVsExecute(t *testing.T) {
	e := New()

	// Store an expression containing a deferred execute
	_, err := e.Eval("▼_response ▶COMPARE\nhello\nhello\n◆ ◆")
	if err != nil {
		t.Fatalf("failed to store _response: %v", err)
	}

	// ▲_response retrieves the body as text (does NOT execute deferred operators)
	result, err := e.Eval("▲_response")
	if err != nil {
		t.Fatalf("failed to retrieve _response: %v", err)
	}
	if result != "▶COMPARE\nhello\nhello\n◆" {
		t.Errorf("▲ should return body text: expected '▶COMPARE\\nhello\\nhello\\n◆', got '%s'", result)
	}

	// ▶_response ◆ executes the body (evaluates deferred operators)
	result, err = e.Eval("▶_response ◆")
	if err != nil {
		t.Fatalf("failed to execute _response: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("▶ should execute and return TRUE, got '%s'", result)
	}
}

// Test expression execution pattern from npc.losp
// ▼Expr □arg ▼_result ▶BUILTIN ▲arg ◆ ◆ ▲_result ◆
func TestExpressionWithInternalStoreAndRetrieve(t *testing.T) {
	e := New()

	// Define an expression that stores a COMPARE result and executes it
	// NOTE: Use ▶_tf_result ◆ not ▲_tf_result because ▲ on Stored expressions
	// has ephemeral semantics (body updated to result after evaluation)
	_, err := e.Eval("▼TestExpr □_tf_input ▼_tf_result ▶COMPARE ▲_tf_input hello ◆ ◆ ▶_tf_result ◆ ◆")
	if err != nil {
		t.Fatalf("failed to store TestExpr: %v", err)
	}

	// Execute with "hello" - should return TRUE
	result, err := e.Eval("▶TestExpr hello ◆")
	if err != nil {
		t.Fatalf("failed to execute TestExpr: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("expected 'TRUE', got '%s'", result)
	}

	// Execute again with "world" - should return FALSE (repeatable)
	result, err = e.Eval("▶TestExpr world ◆")
	if err != nil {
		t.Fatalf("failed to execute TestExpr again: %v", err)
	}
	if result != "FALSE" {
		t.Errorf("expected 'FALSE', got '%s'", result)
	}

	// Execute again with "hello" - should return TRUE (still repeatable)
	result, err = e.Eval("▶TestExpr hello ◆")
	if err != nil {
		t.Fatalf("failed to execute TestExpr third time: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("expected 'TRUE', got '%s'", result)
	}
}

// =============================================================================
// Ephemeral Body Tests
// Bodies are ephemeral: immediate operators are consumed when they fire.
// =============================================================================

// Test that immediate operators in expression bodies are consumed after execution
func TestEphemeralBody_ImmediateOperatorsConsumed(t *testing.T) {
	e := New()

	// Define expression with deferred immediate store: ◯▽X hello ◆
	// The ◯ is consumed during definition, so body becomes "▽X hello ◆"
	_, err := e.Eval("▼Expr ◯▽X hello ◆ ◆◆")
	if err != nil {
		t.Fatalf("failed to store Expr: %v", err)
	}

	// First execution: ▽ fires, X is set to "hello", body becomes empty
	_, err = e.Eval("▶Expr ◆")
	if err != nil {
		t.Fatalf("failed to execute Expr first time: %v", err)
	}

	// Verify X was set
	result, err := e.Eval("▲X")
	if err != nil {
		t.Fatalf("failed to retrieve X: %v", err)
	}
	if result != "hello" {
		t.Errorf("after first exec: expected X='hello', got '%s'", result)
	}

	// Change X to something else
	_, err = e.Eval("▽X changed ◆")
	if err != nil {
		t.Fatalf("failed to change X: %v", err)
	}

	// Second execution: body is now empty, ▽ does NOT fire again
	_, err = e.Eval("▶Expr ◆")
	if err != nil {
		t.Fatalf("failed to execute Expr second time: %v", err)
	}

	// X should still be "changed" because the ▽ was consumed
	result, err = e.Eval("▲X")
	if err != nil {
		t.Fatalf("failed to retrieve X after second exec: %v", err)
	}
	if result != "changed" {
		t.Errorf("after second exec: expected X='changed' (▽ consumed), got '%s'", result)
	}
}

// Test that deferred operators are NOT consumed - expressions with only deferred ops are repeatable
func TestEphemeralBody_DeferredOperatorsNotConsumed(t *testing.T) {
	e := New()

	// Define expression with only deferred operators
	_, err := e.Eval("▼Repeatable ▶COMPARE ▲X hello ◆ ◆")
	if err != nil {
		t.Fatalf("failed to store Repeatable: %v", err)
	}

	// Set X to "hello"
	_, err = e.Eval("▽X hello ◆")
	if err != nil {
		t.Fatalf("failed to set X: %v", err)
	}

	// First execution: should return TRUE
	result, err := e.Eval("▶Repeatable ◆")
	if err != nil {
		t.Fatalf("failed to execute Repeatable first time: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("first exec: expected 'TRUE', got '%s'", result)
	}

	// Change X
	_, err = e.Eval("▽X world ◆")
	if err != nil {
		t.Fatalf("failed to change X: %v", err)
	}

	// Second execution: should return FALSE (body still has ▶COMPARE)
	result, err = e.Eval("▶Repeatable ◆")
	if err != nil {
		t.Fatalf("failed to execute Repeatable second time: %v", err)
	}
	if result != "FALSE" {
		t.Errorf("second exec: expected 'FALSE', got '%s'", result)
	}

	// Change X back
	_, err = e.Eval("▽X hello ◆")
	if err != nil {
		t.Fatalf("failed to change X back: %v", err)
	}

	// Third execution: should return TRUE again (repeatable!)
	result, err = e.Eval("▶Repeatable ◆")
	if err != nil {
		t.Fatalf("failed to execute Repeatable third time: %v", err)
	}
	if result != "TRUE" {
		t.Errorf("third exec: expected 'TRUE', got '%s'", result)
	}
}

// Test double defer (◯◯) - matches conformance test double_defer.losp
// Each ◯ is consumed on one level of access:
// - Outer ◯ consumed at definition time
// - Inner ◯ consumed on first retrieve
// - Content fires on second retrieve
func TestEphemeralBody_NestedDeferForNesting(t *testing.T) {
	e := New()

	// ▽Template ◯◯△X ◆ ◆ ◆
	// Outer ◯ consumed at definition, Template = ◯△X ◆
	_, err := e.Eval("▽X first ◆")
	if err != nil {
		t.Fatalf("failed to set X: %v", err)
	}

	_, err = e.Eval("▽Template ◯◯△X ◆ ◆ ◆")
	if err != nil {
		t.Fatalf("failed to store Template: %v", err)
	}

	// Change X
	_, err = e.Eval("▽X second ◆")
	if err != nil {
		t.Fatalf("failed to update X: %v", err)
	}

	// ▽Inner ▲Template ◆
	// Retrieve Template, inner ◯ consumed, Inner = △X (not fired yet)
	_, err = e.Eval("▽Inner ▲Template ◆")
	if err != nil {
		t.Fatalf("failed to store Inner: %v", err)
	}

	// Change X again
	_, err = e.Eval("▽X third ◆")
	if err != nil {
		t.Fatalf("failed to update X again: %v", err)
	}

	// ▲Inner - NOW △X fires with current X value
	result, err := e.Eval("▲Inner")
	if err != nil {
		t.Fatalf("failed to retrieve Inner: %v", err)
	}
	if result != "third" {
		t.Errorf("expected 'third', got '%s'", result)
	}
}

// Test that nested ◯ is for NESTING levels, not execution counts
// Use case: expression defined inside another expression
func TestEphemeralBody_NestedDeferForNestedDefinition(t *testing.T) {
	e := New()

	// Outer defines Inner with ◯◯▽
	// - Outer definition: outer ◯ consumed, Outer body has ▼Inner ◯▽X fired ◆ ◆◆
	// - Outer execution: Inner is defined, inner ◯ consumed, Inner body has ▽X fired ◆
	// - Inner execution: ▽ fires
	_, err := e.Eval("▼Outer ▼Inner ◯◯▽X fired ◆ ◆◆◆ ◆◆")
	if err != nil {
		t.Fatalf("failed to store Outer: %v", err)
	}

	// Initialize X
	_, err = e.Eval("▽X initial ◆")
	if err != nil {
		t.Fatalf("failed to set initial X: %v", err)
	}

	// Execute Outer - this defines Inner
	_, err = e.Eval("▶Outer ◆")
	if err != nil {
		t.Fatalf("failed to exec Outer: %v", err)
	}
	// X should still be initial (▽ hasn't fired yet)
	result, _ := e.Eval("▲X")
	if result != "initial" {
		t.Errorf("after Outer exec: expected X='initial', got '%s'", result)
	}

	// Execute Inner - NOW ▽ fires
	_, err = e.Eval("▶Inner ◆")
	if err != nil {
		t.Fatalf("failed to exec Inner: %v", err)
	}
	result, _ = e.Eval("▲X")
	if result != "fired" {
		t.Errorf("after Inner exec: expected X='fired', got '%s'", result)
	}
}

// =============================================================================
// SYSTEM Builtin Tests
// =============================================================================

func TestSystemGetSetModel(t *testing.T) {
	e := New(WithProvider(&mockConfigurable{model: "test-model", params: map[string]string{}}))

	result, err := e.Eval("▶SYSTEM MODEL ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test-model" {
		t.Errorf("expected 'test-model', got '%s'", result)
	}

	_, err = e.Eval("▶SYSTEM\nMODEL\nnew-model\n◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err = e.Eval("▶SYSTEM MODEL ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "new-model" {
		t.Errorf("expected 'new-model', got '%s'", result)
	}
}

func TestSystemGetSetTemperature(t *testing.T) {
	e := New(WithProvider(&mockConfigurable{model: "m", params: map[string]string{}}))

	// Get unset temperature returns empty
	result, err := e.Eval("▶SYSTEM TEMPERATURE ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty, got '%s'", result)
	}

	// Set temperature
	_, err = e.Eval("▶SYSTEM\nTEMPERATURE\n0.7\n◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err = e.Eval("▶SYSTEM TEMPERATURE ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "0.7" {
		t.Errorf("expected '0.7', got '%s'", result)
	}
}

func TestSystemInferenceParams(t *testing.T) {
	e := New(WithProvider(&mockConfigurable{model: "m", params: map[string]string{}}))

	params := []struct{ name, value string }{
		{"NUM_CTX", "8192"},
		{"TOP_K", "40"},
		{"TOP_P", "0.9"},
		{"MAX_TOKENS", "1024"},
	}

	for _, p := range params {
		_, err := e.Eval("▶SYSTEM\n" + p.name + "\n" + p.value + "\n◆")
		if err != nil {
			t.Fatalf("failed to set %s: %v", p.name, err)
		}

		result, err := e.Eval("▶SYSTEM " + p.name + " ◆")
		if err != nil {
			t.Fatalf("failed to get %s: %v", p.name, err)
		}
		if result != p.value {
			t.Errorf("SYSTEM %s: expected '%s', got '%s'", p.name, p.value, result)
		}
	}
}

func TestSystemProviderName(t *testing.T) {
	e := New(WithProvider(&mockConfigurable{model: "m", providerName: "MOCK", params: map[string]string{}}))

	result, err := e.Eval("▶SYSTEM PROVIDER ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "MOCK" {
		t.Errorf("expected 'MOCK', got '%s'", result)
	}
}

func TestSystemProviderSwitch(t *testing.T) {
	original := &mockConfigurable{model: "orig-model", providerName: "ORIG", params: map[string]string{"TEMPERATURE": "0.5"}}
	e := New(WithProvider(original))

	// Register a factory for "NEW" provider
	e.RegisterProviderFactory("NEW", func(streamCb StreamCallback) Provider {
		return &mockConfigurable{model: "new-default", providerName: "NEW", params: map[string]string{}}
	})

	// Switch to NEW provider
	_, err := e.Eval("▶SYSTEM\nPROVIDER\nNEW\n◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check new provider name
	result, err := e.Eval("▶SYSTEM PROVIDER ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "NEW" {
		t.Errorf("expected 'NEW', got '%s'", result)
	}

	// Check that inference params were copied
	result, err = e.Eval("▶SYSTEM TEMPERATURE ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "0.5" {
		t.Errorf("expected temperature '0.5' copied to new provider, got '%s'", result)
	}
}

func TestSystemProviderSwitchUnknown(t *testing.T) {
	e := New(WithProvider(&mockConfigurable{model: "m", params: map[string]string{}}))

	result, err := e.Eval("▶SYSTEM\nPROVIDER\nNONEXISTENT\n◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "UNKNOWN_PROVIDER" {
		t.Errorf("expected 'UNKNOWN_PROVIDER', got '%s'", result)
	}
}

func TestSystemUnknownSetting(t *testing.T) {
	e := New()

	result, err := e.Eval("▶SYSTEM BOGUS ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "UNKNOWN_SETTING" {
		t.Errorf("expected 'UNKNOWN_SETTING', got '%s'", result)
	}
}

func TestSystemWithNilProvider(t *testing.T) {
	e := New() // no provider set

	// MODEL with nil provider should return empty
	result, err := e.Eval("▶SYSTEM MODEL ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty with nil provider, got '%s'", result)
	}

	// TEMPERATURE with nil provider should return empty
	result, err = e.Eval("▶SYSTEM TEMPERATURE ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty with nil provider, got '%s'", result)
	}

	// PROVIDER with nil provider should return empty
	result, err = e.Eval("▶SYSTEM PROVIDER ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty with nil provider, got '%s'", result)
	}
}

func TestSystemPersistModeStillWorks(t *testing.T) {
	e := New()

	result, err := e.Eval("▶SYSTEM PERSIST_MODE ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ON_DEMAND" {
		t.Errorf("expected 'ON_DEMAND', got '%s'", result)
	}

	_, err = e.Eval("▶SYSTEM\nPERSIST_MODE\nNEVER\n◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err = e.Eval("▶SYSTEM PERSIST_MODE ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "NEVER" {
		t.Errorf("expected 'NEVER', got '%s'", result)
	}
}

// mockConfigurable implements both Provider and Configurable for testing.
type mockConfigurable struct {
	model        string
	providerName string
	params       map[string]string
}

func (m *mockConfigurable) Prompt(system, user string) (string, error) {
	return "mock response", nil
}

func (m *mockConfigurable) GetParam(key string) string     { return m.params[key] }
func (m *mockConfigurable) SetParam(key, value string)     { m.params[key] = value }
func (m *mockConfigurable) GetModel() string               { return m.model }
func (m *mockConfigurable) SetModel(model string)          { m.model = model }
func (m *mockConfigurable) ProviderName() string           { return m.providerName }

// =============================================================================
// HISTORY Builtin Tests
// =============================================================================

func TestHistoryCreatesEphemeralExpressions(t *testing.T) {
	s := newMemoryStoreForTest()
	e := New(WithStore(s), WithPersistMode(PersistAlways))

	e.Eval("▽X first ◆")
	e.Eval("▽X second ◆")

	result, err := e.Eval("▶HISTORY X ◆")
	if err != nil {
		t.Fatalf("HISTORY failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 version names, got %d: %v", len(lines), lines)
	}

	// Newest first
	if lines[0] != "_X_2" {
		t.Errorf("expected first line '_X_2', got '%s'", lines[0])
	}
	if lines[1] != "_X_1" {
		t.Errorf("expected second line '_X_1', got '%s'", lines[1])
	}

	// Verify ephemeral expressions exist in namespace
	v1 := e.namespace.Get("_X_1")
	if v1.IsEmpty() {
		t.Error("_X_1 not in namespace")
	}
	v2 := e.namespace.Get("_X_2")
	if v2.IsEmpty() {
		t.Error("_X_2 not in namespace")
	}
}

func TestHistoryReturnsNewestFirst(t *testing.T) {
	s := newMemoryStoreForTest()
	e := New(WithStore(s), WithPersistMode(PersistAlways))

	e.Eval("▽X alpha ◆")
	e.Eval("▽X beta ◆")
	e.Eval("▽X gamma ◆")

	result, err := e.Eval("▶HISTORY X ◆")
	if err != nil {
		t.Fatalf("HISTORY failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 version names, got %d", len(lines))
	}
	if lines[0] != "_X_3" || lines[1] != "_X_2" || lines[2] != "_X_1" {
		t.Errorf("unexpected order: %v", lines)
	}
}

func TestHistoryRespectsLimit(t *testing.T) {
	s := newMemoryStoreForTest()
	e := New(WithStore(s), WithPersistMode(PersistAlways))

	e.Eval("▽X one ◆")
	e.Eval("▽X two ◆")
	e.Eval("▽X three ◆")

	e.historyLimit = 2
	result, err := e.Eval("▶HISTORY X ◆")
	if err != nil {
		t.Fatalf("HISTORY failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 with limit, got %d: %v", len(lines), lines)
	}
}

func TestHistoryNonexistentReturnsEmpty(t *testing.T) {
	s := newMemoryStoreForTest()
	e := New(WithStore(s), WithPersistMode(PersistAlways))

	result, err := e.Eval("▶HISTORY nope ◆")
	if err != nil {
		t.Fatalf("HISTORY failed: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty for nonexistent, got '%s'", result)
	}
}

func TestPersistNoopInAlways(t *testing.T) {
	s := newMemoryStoreForTest()
	e := New(WithStore(s), WithPersistMode(PersistAlways))

	e.Eval("▽X hello ◆")

	// PERSIST should be a no-op in ALWAYS mode
	result, err := e.Eval("▶PERSIST X ◆")
	if err != nil {
		t.Fatalf("PERSIST failed: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty from PERSIST no-op, got '%s'", result)
	}
}

func TestAutoPersistVersionHistory(t *testing.T) {
	s := newMemoryStoreForTest()
	e := New(WithStore(s), WithPersistMode(PersistAlways))

	e.Eval("▽X first ◆")
	e.Eval("▽X second ◆")
	e.Eval("▽X second ◆") // Dedup: should not create version 3

	entries, err := s.GetHistory("X", 0)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 versions (dedup), got %d", len(entries))
	}
}

func TestHistoryRollback(t *testing.T) {
	s := newMemoryStoreForTest()
	e := New(WithStore(s), WithPersistMode(PersistAlways))

	e.Eval("▽X first ◆")
	e.Eval("▽X second ◆")
	e.Eval("▽X third ◆")

	// Call HISTORY to create ephemeral versions
	e.Eval("▶HISTORY X ◆")

	// Execute _X_1 to rollback
	_, err := e.Eval("▶_X_1 ◆")
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}

	// X should now be "first"
	result, err := e.Eval("▲X")
	if err != nil {
		t.Fatalf("retrieve failed: %v", err)
	}
	if result != "first" {
		t.Errorf("expected 'first' after rollback, got '%s'", result)
	}
}

func TestSystemHistoryLimit(t *testing.T) {
	e := New()

	// Default is 0
	result, err := e.Eval("▶SYSTEM HISTORY_LIMIT ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "0" {
		t.Errorf("expected '0', got '%s'", result)
	}

	// Set it
	_, err = e.Eval("▶SYSTEM\nHISTORY_LIMIT\n5\n◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err = e.Eval("▶SYSTEM HISTORY_LIMIT ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "5" {
		t.Errorf("expected '5', got '%s'", result)
	}
}

// newMemoryStoreForTest creates a store.Memory via the store package.
// We use eval.Store interface but the concrete type is store.Memory.
func newMemoryStoreForTest() *memoryStoreWrapper {
	return &memoryStoreWrapper{
		data:     make(map[string]string),
		versions: make(map[string][]versionEntry),
	}
}

type versionEntry struct {
	version int
	value   string
}

// memoryStoreWrapper is a simple in-evaluator-test store that implements
// both eval.Store and store.HistoryStore semantics.
type memoryStoreWrapper struct {
	data     map[string]string
	versions map[string][]versionEntry
}

func (m *memoryStoreWrapper) Get(name string) (expr.Expr, error) {
	if v, ok := m.data[name]; ok {
		return expr.Text{Value: v}, nil
	}
	return nil, nil
}

func (m *memoryStoreWrapper) Put(name string, e expr.Expr) error {
	value := ""
	if e != nil {
		value = e.String()
	}
	// Dedup
	if vv := m.versions[name]; len(vv) > 0 {
		if vv[len(vv)-1].value == value {
			m.data[name] = value
			return nil
		}
	}
	ver := len(m.versions[name]) + 1
	m.versions[name] = append(m.versions[name], versionEntry{version: ver, value: value})
	m.data[name] = value
	return nil
}

func (m *memoryStoreWrapper) Delete(name string) error {
	delete(m.data, name)
	delete(m.versions, name)
	return nil
}

func (m *memoryStoreWrapper) Close() error { return nil }

func (m *memoryStoreWrapper) GetHistory(name string, limit int) ([]store.VersionEntry, error) {
	vv := m.versions[name]
	if len(vv) == 0 {
		return nil, nil
	}
	result := make([]store.VersionEntry, len(vv))
	for i, v := range vv {
		result[len(vv)-1-i] = store.VersionEntry{Version: v.version, Value: v.value}
	}
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}
