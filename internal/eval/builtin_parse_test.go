// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"testing"

	"nickandperla.net/losp/internal/token"
)

// Structural errors cannot be written into a .losp conformance file — the file
// itself would have to be malformed — so ValidateSyntax is exercised directly.
func TestValidateSyntaxAcceptsWellFormedCode(t *testing.T) {
	cases := map[string]string{
		"definition with placeholder":  "▼Tidy □_t ▶TRIM ▲_t ◆ ◆",
		"nested execute":               "▼Report ▶Shrink LOUD ◆ ◆",
		"if with compare":              "▼Check □_v ▶IF ▶COMPARE ▲_v target ◆\nyes\nno\n◆ ◆",
		"dynamic execute of if":        "▶▶IF ▶COMPARE ▲Mode A ◆\nDoA\nDoB\n◆ ◆",
		"dynamic name in store":        "▼▲FieldName hello ◆",
		"immediate store":              "▽X hello ◆",
		"defer wrapper":                "▽Snapshot ◯△X ◆ ◆",
		"bare retrieve":                "▲X",
		"plain text":                   "just some text",
		"http call":                    "▼Post ▶HTTP POST\nhttp://host/path\n▲_Headers ▲_Data ◆ ◆",
		"empty body definition":        "▼_ElseBranch ◆",
		"name followed by punctuation": "▼Wrap □_item [▲_item] ◆",
	}
	for name, src := range cases {
		if problem := ValidateSyntax(src); problem != "" {
			t.Errorf("%s: expected valid, got %q", name, problem)
		}
	}
}

func TestValidateSyntaxRejectsStructuralErrors(t *testing.T) {
	cases := map[string]struct {
		src  string
		want string
	}{
		"surplus terminator": {
			"▼A □_b ▶TRIM ▲_b ◆ ◆ ◆",
			"surplus END terminator on line 1 - it closes nothing",
		},
		"surplus terminator on a later line": {
			"▼A\nbody ◆\n◆",
			"surplus END terminator on line 3 - it closes nothing",
		},
		"unclosed definition": {
			"▼A □_b ▶TRIM ▲_b ◆",
			"DEF on line 1 is missing its END terminator",
		},
		"two unclosed operators": {
			"▼A ▶IF ▶COMPARE x y ◆",
			"2 operators are missing their END terminator, the last is RUN on line 1",
		},
		"definition without a name": {
			"▼ ◆",
			"DEF on line 1 has no name",
		},
		"execute without a name": {
			"▼A ▶ ◆ ◆",
			"RUN on line 1 has no name",
		},
		"placeholder without a name": {
			"▼A □ ▲x ◆",
			"ARG on line 1 has no name",
		},
		"retrieve without a name": {
			"▼A ▲ ◆",
			"GET on line 1 has no name",
		},
	}
	for name, tc := range cases {
		if got := ValidateSyntax(tc.src); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

// Reasons are read back with ▲PARSE_REASON, which re-parses them, so a reason
// containing an operator rune would be mangled — or worse, fire.
func TestValidateSyntaxReasonsCarryNoOperators(t *testing.T) {
	malformed := []string{
		"▼A ◆ ◆",
		"▼A ▶TRIM x ◆",
		"▼ ◆",
		"▼A □ ◆",
	}
	for _, src := range malformed {
		problem := ValidateSyntax(src)
		if problem == "" {
			t.Fatalf("%q: expected a problem", src)
		}
		for _, r := range problem {
			if token.IsOperator(r) {
				t.Errorf("%q: reason %q contains operator %c", src, problem, r)
			}
		}
	}
}
