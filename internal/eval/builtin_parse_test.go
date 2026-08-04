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

		// The evaluator passes over a terminator that closes nothing: both of
		// these define what they say they define, so PARSE accepts them rather
		// than being stricter than the language it checks.
		"terminator closing nothing at the end": "▼A □_b ▶TRIM ▲_b ◆ ◆ ◆",
		"terminator closing nothing mid stream": "▼A\nbody ◆\n◆\n▼B second ◆",
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
		"unclosed definition": {
			"▼A □_b ▶TRIM ▲_b ◆",
			"DEF on line 1 is missing its END terminator",
		},
		"two unclosed operators": {
			"▼A ▶IF ▶COMPARE x y ◆",
			"2 operators are missing their END terminator: RUN on line 1, DEF on line 1",
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

// DESCRIBE puts this name on its first line, so callers can address the name and
// the description positionally. It has to hold for every shape of input,
// including code that defines nothing at all.
func TestFirstDefinedName(t *testing.T) {
	cases := map[string]struct{ src, want string }{
		"plain definition":          {"▼DOUBLE □d ▲d ▲d ◆", "DOUBLE"},
		"immediate definition":      {"▽Greeting hello ◆", "Greeting"},
		"leading text":              {"here is the code\n▼Wrap □w [▲w] ◆", "Wrap"},
		"space before name":         {"▼ Wrap □w ▲w ◆", "Wrap"},
		"first of several":          {"▼One a ◆\n▼Two b ◆", "One"},
		"nested definition":         {"▼Outer ▼Inner x ◆ ◆", "Outer"},
		"underscores and digits":    {"▼_au_step2 x ◆", "_au_step2"},
		"execute is no definition":  {"▶SAY hello ◆", SentinelAnonymous},
		"retrieve is no definition": {"▲greet", SentinelAnonymous},
		"dynamic name":              {"▼▶_ag_slot ▲i ▲f ◆ ◆", SentinelAnonymous},
		"no name at all":            {"▼ ◆", SentinelAnonymous},
		"empty source":              {"", SentinelAnonymous},
		"prose only":                {"this text defines nothing", SentinelAnonymous},
	}
	for name, tc := range cases {
		if got := FirstDefinedName(tc.src); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

// An empty definition is well-formed, so nothing downstream can tell it from
// working code: it parses, installs, and then does nothing wherever it is
// called. GENERATE watches for it to catch a repair that reached validity by
// discarding the code it was asked to correct.
func TestDefinesOnlyEmptyBody(t *testing.T) {
	cases := map[string]struct {
		src  string
		want bool
	}{
		"empty definition":        {"▼ProcessCreatureLine◆", true},
		"empty with space":        {"▼Name ◆", true},
		"empty immediate":         {"▽Name ◆", true},
		"body of one operator":    {"▼Tidy □_t ▶TRIM ▲_t ◆ ◆", false},
		"body of plain text":      {"▽X hello ◆", false},
		"empty then another":      {"▼A ◆ ▼B x ◆", false},
		"no definition at all":    {"▶SAY hello ◆", false},
		"prose only":              {"this text defines nothing", false},
		"empty source":            {"", false},
		"unterminated definition": {"▼Name", false},
	}
	for name, tc := range cases {
		if got := DefinesOnlyEmptyBody(tc.src); got != tc.want {
			t.Errorf("%s: got %v, want %v", name, got, tc.want)
		}
	}
}
