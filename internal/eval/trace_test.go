// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"strings"
	"testing"
)

func TestTraceIF(t *testing.T) {
	var output strings.Builder
	e := New(WithOutputWriter(func(text string) error {
		output.WriteString(text)
		return nil
	}))
	
	// Set debug to TRUE
	e.Eval("▽NPC_System_Debug TRUE ◆")
	
	// Test the COMPARE directly
	result, _ := e.Eval("▶COMPARE ▲NPC_System_Debug TRUE ◆")
	t.Logf("COMPARE result: %q", result)
	
	// Test the IF directly
	output.Reset()
	result, err := e.Eval(`▶IF ▷COMPARE ▲NPC_System_Debug TRUE ◆
		yes
		no
	◆`)
	if err != nil {
		t.Fatalf("IF error: %v", err)
	}
	t.Logf("IF result: %q", result)
	
	// Test with stored expressions
	e.Eval("▼Yes yes-branch ◆")
	e.Eval("▼No no-branch ◆")
	
	result, _ = e.Eval(`▶IF ▷COMPARE ▲NPC_System_Debug TRUE ◆
		▶Yes ◆
		▶No ◆
	◆`)
	t.Logf("IF with templates result: %q", result)
}
