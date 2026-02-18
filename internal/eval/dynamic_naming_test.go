// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"testing"
)

func TestDynamicNamingWithImmRetrieve(t *testing.T) {
	e := New()

	// Set the name variable
	e.Eval("▽fieldName X ◆")

	// Use dynamic naming: ▽△fieldName stores to X
	_, err := e.Eval("▽△fieldName hello ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve X to verify it was set
	result, err := e.Eval("▲X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestDynamicNamingDeferredStore(t *testing.T) {
	e := New()

	// Set the name variable
	e.Eval("▽fieldName Y ◆")

	// Use dynamic naming with deferred store
	_, err := e.Eval("▼△fieldName world ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve Y to verify it was set
	result, err := e.Eval("▲Y")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "world" {
		t.Errorf("expected 'world', got '%s'", result)
	}
}

func TestDynamicNamingWithMultipleFields(t *testing.T) {
	e := New()

	// Simulate iterating over fields
	fields := []string{"Name", "Age", "City"}
	values := []string{"Alice", "30", "NYC"}

	for i, field := range fields {
		// Set which field we're storing to
		e.Eval("▽currentField " + field + " ◆")
		// Store the value dynamically
		e.Eval("▽△currentField " + values[i] + " ◆")
	}

	// Verify all fields were set
	result, _ := e.Eval("▲Name")
	if result != "Alice" {
		t.Errorf("expected 'Alice', got '%s'", result)
	}

	result, _ = e.Eval("▲Age")
	if result != "30" {
		t.Errorf("expected '30', got '%s'", result)
	}

	result, _ = e.Eval("▲City")
	if result != "NYC" {
		t.Errorf("expected 'NYC', got '%s'", result)
	}
}

func TestDynamicNamingWithImmExecute(t *testing.T) {
	e := New()

	// Define an expression that returns a field name
	e.Eval("▼GetFieldName fieldResult ◆")

	// Use dynamic naming with immediate execute
	_, err := e.Eval("▽▷GetFieldName ◆ dynamic_value ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve fieldResult to verify it was set
	result, err := e.Eval("▲fieldResult")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "dynamic_value" {
		t.Errorf("expected 'dynamic_value', got '%s'", result)
	}
}

func TestDynamicNamingEmptyValue(t *testing.T) {
	e := New()

	// Set the name variable to empty
	e.Eval("▽fieldName ◆")

	// Dynamic naming with empty name should still work (though name will be empty)
	_, err := e.Eval("▽△fieldName test ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// An empty name should store to "" key
	result, err := e.Eval("▲")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("expected 'test', got '%s'", result)
	}
}

func TestRegularNamingStillWorks(t *testing.T) {
	e := New()

	// Regular naming should still work
	_, err := e.Eval("▽RegularName value ◆")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := e.Eval("▲RegularName")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "value" {
		t.Errorf("expected 'value', got '%s'", result)
	}
}

// NOTE: Tests for dynamic naming with placeholders and iteration patterns were removed.
// These patterns relied on non-ephemeral body semantics where ▽ would fire on every
// execution. With ephemeral semantics, ▽ fires ONCE and is consumed from the body.
// The correct pattern for repeated dynamic stores is to use top-level ▽ calls
// (not inside expression bodies), as shown in TestDynamicNamingWithMultipleFields.

// =============================================================================
// Dynamic Retrieval Tests
// =============================================================================

// TestDynamicRetrievalBasic tests basic indirect variable access with ▲▲
func TestDynamicRetrievalBasic(t *testing.T) {
	e := New()

	// Set up: ActualVar contains "hello", varRef contains "ActualVar"
	e.Eval("▽ActualVar hello ◆")
	e.Eval("▽varRef ActualVar ◆")

	// Direct retrieval works as expected
	result, err := e.Eval("▲ActualVar")
	if err != nil {
		t.Fatalf("direct retrieval failed: %v", err)
	}
	if result != "hello" {
		t.Errorf("direct: expected 'hello', got %q", result)
	}

	// Dynamic retrieval: ▲▲varRef should get value of ActualVar
	result, err = e.Eval("▲▲varRef")
	if err != nil {
		t.Fatalf("dynamic retrieval failed: %v", err)
	}
	if result != "hello" {
		t.Errorf("dynamic ▲▲: expected 'hello', got %q", result)
	}
}

// TestDynamicRetrievalImmediate tests immediate dynamic retrieval with △△
func TestDynamicRetrievalImmediate(t *testing.T) {
	e := New()

	e.Eval("▽Target world ◆")
	e.Eval("▽pointer Target ◆")

	// △△pointer should immediately resolve to "world"
	result, err := e.Eval("△△pointer")
	if err != nil {
		t.Fatalf("immediate dynamic retrieval failed: %v", err)
	}
	if result != "world" {
		t.Errorf("△△: expected 'world', got %q", result)
	}
}

// TestDynamicRetrievalMixed tests mixed immediate/deferred combinations
func TestDynamicRetrievalMixed(t *testing.T) {
	e := New()

	e.Eval("▽Data value123 ◆")
	e.Eval("▽ref Data ◆")

	// ▲△ref: deferred retrieval with immediate name resolution
	result, err := e.Eval("▲△ref")
	if err != nil {
		t.Fatalf("▲△ failed: %v", err)
	}
	if result != "value123" {
		t.Errorf("▲△: expected 'value123', got %q", result)
	}

	// △▲ref: immediate retrieval with deferred name resolution
	// (both behave the same at top level since we're evaluating immediately)
	result, err = e.Eval("△▲ref")
	if err != nil {
		t.Fatalf("△▲ failed: %v", err)
	}
	if result != "value123" {
		t.Errorf("△▲: expected 'value123', got %q", result)
	}
}

// TestDynamicRetrievalInTemplate tests dynamic retrieval within a template
func TestDynamicRetrievalInTemplate(t *testing.T) {
	e := New()

	// Template that retrieves a value indirectly
	e.Eval(`▼GetIndirect
		□varname
		▲▲varname
	◆`)

	// Set up data
	e.Eval("▽Item_torch_Location 001 ◆")
	e.Eval("▽Item_potion_Location 002 ◆")

	// Use the template to get values indirectly
	result, err := e.Eval("▶GetIndirect Item_torch_Location ◆")
	if err != nil {
		t.Fatalf("GetIndirect torch failed: %v", err)
	}
	if result != "001" {
		t.Errorf("torch location: expected '001', got %q", result)
	}

	result, err = e.Eval("▶GetIndirect Item_potion_Location ◆")
	if err != nil {
		t.Fatalf("GetIndirect potion failed: %v", err)
	}
	if result != "002" {
		t.Errorf("potion location: expected '002', got %q", result)
	}
}

// TestDynamicRetrievalWithConstructedName tests building a variable name and retrieving it
// With ephemeral semantics, ▽ inside expression bodies fires once and is consumed.
// This test demonstrates the correct pattern: use top-level ▽ for dynamic naming.
func TestDynamicRetrievalWithConstructedName(t *testing.T) {
	e := New()

	// Set up item data with prefixed names
	e.Eval("▽Item_sword_Name Excalibur ◆")
	e.Eval("▽Item_sword_Location armory ◆")
	e.Eval("▽Item_shield_Name Aegis ◆")
	e.Eval("▽Item_shield_Location vault ◆")

	// Helper variable for underscore
	e.Eval("▽_us _ ◆")

	// CORRECT PATTERN: Construct the variable name at top level, then retrieve
	// First: construct the name
	e.Eval("▽_gif_item sword ◆")
	e.Eval("▽_gif_field Name ◆")
	e.Eval("▽_gif_varname Item_▲_gif_item△_us▲_gif_field ◆")
	// Now retrieve it
	result, _ := e.Eval("▲▲_gif_varname")
	if result != "Excalibur" {
		t.Errorf("sword Name: expected 'Excalibur', got %q", result)
	}

	// Repeat for different values
	e.Eval("▽_gif_item sword ◆")
	e.Eval("▽_gif_field Location ◆")
	e.Eval("▽_gif_varname Item_▲_gif_item△_us▲_gif_field ◆")
	result, _ = e.Eval("▲▲_gif_varname")
	if result != "armory" {
		t.Errorf("sword Location: expected 'armory', got %q", result)
	}

	e.Eval("▽_gif_item shield ◆")
	e.Eval("▽_gif_field Name ◆")
	e.Eval("▽_gif_varname Item_▲_gif_item△_us▲_gif_field ◆")
	result, _ = e.Eval("▲▲_gif_varname")
	if result != "Aegis" {
		t.Errorf("shield Name: expected 'Aegis', got %q", result)
	}
}

// TestDynamicRetrievalWithExecute tests dynamic retrieval using execute for name
func TestDynamicRetrievalWithExecute(t *testing.T) {
	e := New()

	// Template that returns a variable name
	e.Eval("▼GetVarName TargetVar ◆")

	// Set up the target variable
	e.Eval("▽TargetVar executed_value ◆")

	// ▲▷GetVarName ◆ should retrieve TargetVar's value
	result, err := e.Eval("▲▷GetVarName ◆")
	if err != nil {
		t.Fatalf("▲▷ failed: %v", err)
	}
	if result != "executed_value" {
		t.Errorf("▲▷: expected 'executed_value', got %q", result)
	}
}

// TestDynamicRetrievalPreservesRegularBehavior ensures regular retrieval still works
func TestDynamicRetrievalPreservesRegularBehavior(t *testing.T) {
	e := New()

	e.Eval("▽SimpleVar simple_value ◆")

	// Regular retrieval should work exactly as before
	result, err := e.Eval("▲SimpleVar")
	if err != nil {
		t.Fatalf("regular retrieval failed: %v", err)
	}
	if result != "simple_value" {
		t.Errorf("regular: expected 'simple_value', got %q", result)
	}

	// Immediate retrieval should also work
	result, err = e.Eval("△SimpleVar")
	if err != nil {
		t.Fatalf("immediate retrieval failed: %v", err)
	}
	if result != "simple_value" {
		t.Errorf("immediate: expected 'simple_value', got %q", result)
	}
}

// TestDynamicRetrievalNonexistentVariable tests behavior when the referenced variable doesn't exist
func TestDynamicRetrievalNonexistentVariable(t *testing.T) {
	e := New()

	// varRef points to a variable that doesn't exist
	e.Eval("▽varRef NonexistentVar ◆")

	// Dynamic retrieval should return empty (losp's behavior for missing variables)
	result, err := e.Eval("▲▲varRef")
	if err != nil {
		t.Fatalf("retrieval of nonexistent var failed: %v", err)
	}
	if result != "" {
		t.Errorf("nonexistent: expected empty string, got %q", result)
	}
}

// TestDynamicRetrievalChained tests chained indirection
func TestDynamicRetrievalChained(t *testing.T) {
	e := New()

	// Set up a chain: ref2 -> ref1 -> FinalValue
	e.Eval("▽FinalValue chained_result ◆")
	e.Eval("▽ref1 FinalValue ◆")
	e.Eval("▽ref2 ref1 ◆")

	// Single indirection: ▲ref1 -> "FinalValue"
	result, _ := e.Eval("▲ref1")
	if result != "FinalValue" {
		t.Errorf("single indirect: expected 'FinalValue', got %q", result)
	}

	// Double indirection: ▲▲ref1 -> value of FinalValue -> "chained_result"
	result, _ = e.Eval("▲▲ref1")
	if result != "chained_result" {
		t.Errorf("double indirect via ref1: expected 'chained_result', got %q", result)
	}

	// Double indirection from ref2: ▲▲ref2 -> value of ref1 -> "FinalValue"
	result, _ = e.Eval("▲▲ref2")
	if result != "FinalValue" {
		t.Errorf("double indirect via ref2: expected 'FinalValue', got %q", result)
	}

	// For triple indirection, use nested retrieval explicitly
	// Store the intermediate name, then retrieve through it
	e.Eval("▽intermediate ▲ref2 ◆")      // intermediate = "ref1"
	e.Eval("▽intermediate2 ▲▲ref2 ◆")    // intermediate2 = "FinalValue"
	result, _ = e.Eval("▲▲intermediate2") // -> "chained_result"
	if result != "chained_result" {
		t.Errorf("explicit triple: expected 'chained_result', got %q", result)
	}
}
