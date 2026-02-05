# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## ⚠️ AFTER CONTEXT COMPACTION ⚠️

**ALWAYS re-read PRIMER.md completely after every context compaction.** The compaction summary cannot capture all the nuances of losp semantics. Re-reading PRIMER.md ensures you have the authoritative specification fresh in context.

---

## ⚠️ THE FUNDAMENTAL RULE OF losp ⚠️

**WHENEVER THE STREAM IS PARSED, IMMEDIATE OPERATORS FIRE.**

This is absolute. No exceptions except `◯`.

- Defining an expression? **IMMEDIATE OPERATORS FIRE.**
- Executing an expression? The body is parsed. **IMMEDIATE OPERATORS FIRE.**
- Retrieving an expression? The body is parsed. **IMMEDIATE OPERATORS FIRE.**
- Loading from database? The body is parsed. **IMMEDIATE OPERATORS FIRE.**

**EVERY TIME the stream is read, immediate operators (`△`, `▷`, `▽`) execute.**

## ⚠️⚠️⚠️ NEVER USE IMMEDIATE OPERATORS TO ACCESS PLACEHOLDERS ⚠️⚠️⚠️

**THIS IS THE #1 MISTAKE. DO NOT MAKE IT.**

Execution order: LOAD → PARSE → POPULATE → EXECUTE

**Immediate operators fire at PARSE. Placeholders are bound at POPULATE. PARSE happens BEFORE POPULATE.**

```losp
# WRONG - ▽ fires at PARSE, before □arg is bound!
▼Broken
    □arg
    ▽result ▶PROMPT ▲arg ◆ ◆   # ▲arg is EMPTY here!
◆

# WRONG - △ fires at PARSE, before □arg is bound!
▼AlsoBroken
    □arg
    △arg                        # EMPTY!
◆

# CORRECT - use a helper that receives the value as an argument
▼Working
    □arg
    ▶Helper ▶PROMPT ▲arg ◆ ◆   # ▶PROMPT is deferred, executes at EXECUTE time
◆
```

**To capture a computed result that uses placeholder values, pass it to a helper function as an argument.** The helper receives the already-evaluated result via its own placeholder.

## ⚠️ BODIES ARE EPHEMERAL ⚠️

**When immediate operators fire, they are CONSUMED. The stored body is UPDATED.**

```losp
▼Expr ◯▽X hello ◆ ◆◆

▶Expr ◆    # Body "▽X hello ◆" is parsed, ▽ fires, body becomes EMPTY
▶Expr ◆    # Body is empty, nothing happens
```

**Expressions are NOT automatically repeatable.** Each `◯` allows ONE level of deferral:

```losp
▼Expr ◯◯▽X hello ◆ ◆◆◆

▶Expr ◆    # Outer ◯ consumed → body is now "◯▽X hello ◆ ◆"
▶Expr ◆    # Inner ◯ consumed → body is now "▽X hello ◆", ▽ fires, body becomes EMPTY
▶Expr ◆    # Body is empty
```

**To make an expression truly repeatable, use only deferred operators in the body:**

```losp
▼Repeatable ▶COMPARE ▲X hello ◆ ◆

▶Repeatable ◆    # ▶COMPARE executes, returns result
▶Repeatable ◆    # ▶COMPARE executes again, body unchanged
```

Deferred operators (`▲`, `▶`, `▼`) are NOT consumed - they execute each time but remain in the body.

---

## Project Overview

losp is a streaming template language using Unicode operators instead of parentheses. It is designed for LLM metacognition workflows where templates accumulate context and invoke language models.

**losp has no comment syntax.** All text is meaningful. The `#` character has no special meaning—it's just text like any other character and will appear in output.

## Build and Test Commands

```bash
go generate ./internal/stdlib/ && go build ./...   # Generate embedded files and build
go test ./...            # Run all tests
go test ./... -v         # Verbose test output
go test -run TestName    # Run specific test
go run ./cmd/losp        # Run the CLI
go run ./cmd/losp -f examples/simulation.losp -db simulation.db  # Run a losp file
go generate ./internal/stdlib/ && go build -o ./losp ./cmd/losp && LOSP_BIN=./losp ./tests/conformance/run_tests.sh  # Run conformance tests
```

**Embedded files:** `PRIMER_COMPACT.md` and `PROMPTING_LOSP.md` live at the repo root. `go generate ./internal/stdlib/` copies them into `internal/stdlib/` for `go:embed`. You must run `go generate` before building if either file has changed. The copies in `internal/stdlib/` are gitignored.

**Conformance Tests:** The losp conformance tests are `.losp` files in `./tests/conformance/`. They are NOT Go tests. Build the binary first with `go build -o ./losp ./cmd/losp`, then run with `LOSP_BIN=./losp ./tests/conformance/run_tests.sh`.

**IMPORTANT: `go build` only compiles Go code.** It does NOT validate losp syntax or run losp files. To test a losp file, you must actually run it with the CLI.

## Language Specification

**PRIMER.md is the authoritative source for losp language semantics.** When implementing operators or builtins, consult PRIMER.md for correct behavior.

**CRITICAL: Never trust the implementation over PRIMER.md.** If you observe behavior in the Go code that conflicts with what PRIMER.md specifies, ASK THE USER what to do. The implementation may have bugs; PRIMER.md defines the intended semantics. Do not use implementation details to "explain" behavior—verify against PRIMER.md first.

### Key Concepts

**Two Evaluation Times:**
- Parse-time (Immediate): `△` `▷` `▽` - resolved as encountered in input stream
- Execution-time (Deferred): `▲` `▶` `▼` - stored as-is, resolved when executed

## CRITICAL: Immediate Operator Evaluation Rule

**WHENEVER THE STREAM IS PARSED, IMMEDIATE OPERATORS FIRE.**

**IMMEDIATE OPERATORS ALWAYS EVALUATE IMMEDIATELY WHEN ENCOUNTERED IN THE INPUT STREAM.**

This is absolute. There is NO exception except the `◯` defer operator.

Parsing happens:
- When code is first loaded/evaluated
- When an expression is executed via `▶`
- When an expression is retrieved via `▲`
- When LOAD retrieves from the database

**Every parse fires immediate operators. Every single time.**

**Inside `▼` bodies, immediate operators STILL execute immediately:**
```losp
▽X first ◆
▼Template △X ◆    # △X resolves NOW to "first" - body becomes "first"
▽X second ◆
▶Template ◆       # Returns "first", NOT "second"
```

**Contrast with deferred:**
```losp
▽X first ◆
▼Template ▲X ◆    # ▲X is preserved literally in body
▽X second ◆
▶Template ◆       # Returns "second" - ▲X resolves at execution
```

**The `◯` defer operator is the ONLY way to prevent immediate evaluation:**
```losp
▽X first ◆
▽Snapshot ◯△X ◆ ◆  # ◯ needs its own ◆, then ▽ needs its own ◆
▽X second ◆
▲Snapshot          # NOW △X resolves to "second"
```

**IMPORTANT: `◯` requires its own terminating `◆`.** You cannot share terminators between operators. Every operator MUST have its own `◆`.

## CRITICAL: ◯ IS ALWAYS CONSUMED

**The `◯` defer operator is ALWAYS consumed when encountered.** It is NEVER preserved in stored bodies. This is intentional and enables single-trigger effects on load.

When `◯` is processed:
1. The `◯` rune itself is consumed (not stored)
2. Its content (until the matching `◆`) is collected
3. Immediate operators inside are NOT fired (deferDepth > 0)
4. Only the content is stored/returned (without the `◯` wrapper)

**Example:**
```losp
▽Snapshot ◯△X ◆ ◆
# ◯ is consumed, Snapshot contains "△X" (not "◯△X ◆")
# When ▲Snapshot is later evaluated, △X fires at THAT time
```

**DO NOT attempt to preserve ◯ in any scanning or parsing function.** The consumption of ◯ is fundamental to losp's single-trigger semantics. If you find yourself wanting to "preserve" ◯, you are misunderstanding the problem - ask the user for clarification instead.

**Common mistake:** Thinking that `▼` "protects" or "quotes" its body from immediate evaluation. IT DOES NOT. The body is collected as a stream, and immediate operators fire as they are encountered during that collection.

**Implications for `▽` inside `▼`:**
```losp
▼Outer
    ▽inner value ◆   # This executes NOW during Outer's definition!
◆
```
The `▽inner value ◆` sets the global `inner` at PARSE TIME when `Outer` is being defined, NOT at runtime when `▶Outer ◆` is called.

## Execution Order and Placeholder Timing

When an expression executes, there are four phases:

1. **LOAD** - body text is retrieved from the namespace
2. **PARSE** - immediate operators fire (parse-time effects)
3. **POPULATE** - placeholders are bound to arguments
4. **EXECUTE** - deferred expressions run

**The critical insight:** Immediate operators in step 2 fire BEFORE placeholders are bound in step 3.

**You cannot capture placeholder values with immediate operators:**

```losp
▼Broken
    □arg
    ◯▽result △arg ◆ ◆   # △arg fires at PARSE, before arg is bound!
◆
```

**This is the fundamental constraint of losp's execution model:**
- Immediate operators fire at PARSE (before POPULATE)
- Deferred operators resolve at EXECUTE (after POPULATE)
- To use placeholder values, you MUST use deferred retrieval (`▲`) for indirection

**Correct pattern for referencing placeholder values:**

```losp
▼GoodFunc
    □arg
    The value is: ▲arg   # ▲ resolves at EXECUTE, after arg is bound
◆
```

## Operator Timing Inside Expression Definitions

**Immediate vs deferred is about WHEN evaluation happens, not correctness.**

Inside a `▼` expression definition:
- **Immediate** (`▽`, `△`, `▷`): Fires at parse time when the expression is DEFINED → value baked into body
- **Deferred** (`▼`, `▲`, `▶`): Preserved in body → fires when the expression is EXECUTED

**Choose based on your intent:**

| You want... | Use | Why |
|-------------|-----|-----|
| Current runtime state | Deferred (`▶COMPARE`) | Evaluates when expression runs |
| Constant baked at load time | Immediate (`▷COMPUTE`) | Evaluates once at definition |
| Metaprogramming / code gen | Immediate | Result spliced into body |

**Common mistake:** Using `▷COMPARE` inside an expression expecting runtime comparison. The comparison fires at definition time and the result (TRUE/FALSE) is baked in.

```losp
# WRONG for runtime state checking:
▼CheckMode
    ▶IF ▷COMPARE ▲Mode active ◆ ... ◆   # Compares at DEFINITION time!
◆

# CORRECT for runtime state checking:
▼CheckMode
    ▶IF ▶COMPARE ▲Mode active ◆ ... ◆   # Compares at EXECUTION time
◆
```

**The PRIMER.md examples using `▷COMPARE` work at top-level because parse time IS execution time there. Inside expression definitions, parse time is definition time.**

## Retrieve vs Execute

Both `▲` and `▶` **parse** the body (immediate operators fire). The difference:

- **`▲Name`** (Retrieve): Returns deferred operators **as text**
- **`▶Name ◆`** (Execute): **Evaluates** deferred operators

```losp
▼_expr ▶COMPARE hello hello ◆ ◆

▲_expr        # → "▶COMPARE hello hello ◆" (text, unevaluated)
▶_expr ◆      # → "TRUE" (deferred ops fire)
```

Deferred operators are **repeatable** (not consumed), while immediate operators inside `◯` are **consumed** on first parse.

**IF statements inside functions:** Use `▶IF` (deferred) for the IF itself so its output becomes part of the function's result. COMPARE arguments use `▲` to reference placeholder values:

```losp
▼ConditionalFunc
    □val
    ▶IF ▷COMPARE ▲val target ◆
        matched
        not matched
    ◆
◆
```

Here `▷COMPARE` (immediate) executes at parse time during function execution, but `▲val` inside it is deferred and resolves to the bound placeholder value.

**IF returns text, not executed results.** IF returns the selected branch as text. To execute the selected branch, use dynamic execution with `▶▶IF ...`:

```losp
▼_ThenBranch
    ▶SAY Setting up... ◆
    ▼State ready ◆
◆
▼_ElseBranch ◆

▶▶IF ▶COMPARE ▲Mode new ◆
    _ThenBranch
    _ElseBranch
◆ ◆
```

The inner IF returns the name ("_ThenBranch" or "_ElseBranch"), then the outer `▶` executes only the selected expression. This avoids eager evaluation of both branches.

**Arguments are expressions.** Newlines separate TEXT arguments. Operators are already expression boundaries:

```losp
▶COMPARE ▲A ▲B ◆           # Correct: two operator arguments
▶COMPARE hello world ◆     # WRONG: one text argument "hello world"
▶COMPARE
hello
world
◆                          # Correct: two text arguments
```

For placeholders, the same rules apply:

```losp
▶ThreeArgs one two three ◆    # WRONG: a="one two three", b="", c=""

▶ThreeArgs
one
two
three
◆                              # Correct: a="one", b="two", c="three"

▶Func ▲A ▲B ▲C ◆              # Correct: three operator arguments
```

**Operators (Unicode):**
| Op | Name | Timing | Description |
|----|------|--------|-------------|
| `▼` U+25BC | Store | Execution | Store expression (deferred) |
| `▽` U+25BD | ImmStore | Parse | Evaluate now, store result |
| `▲` U+25B2 | Retrieve | Execution | Retrieve stored expression |
| `△` U+25B3 | ImmRetrieve | Parse | Retrieve now, substitute into stream |
| `▶` U+25B6 | Execute | Execution | Execute named expression or builtin |
| `▷` U+25B7 | ImmExec | Parse | Execute now, substitute result |
| `□` U+25A1 | Placeholder | — | Declare argument slot (binds to global) |
| `◯` U+25EF | Defer | — | Prevent parse-time resolution |
| `◆` U+25C6 | Terminator | — | End current operator's scope |

**Global Namespace:** All variables share a single flat namespace. Placeholders write to globals, which can cause clobbering in nested calls.

**Builtins:** IF, COMPARE, FOREACH, PROMPT, SAY, READ, PERSIST, LOAD, COUNT, APPEND, EXTRACT, SYSTEM, UPPER, LOWER, TRIM, TRUE, FALSE, EMPTY, GENERATE

## GENERATE and Executing Generated Code

**GENERATE returns losp code as text. It does NOT execute the generated code.** The returned text is inert — operators in it do not fire until the text is parsed by the evaluator.

**To execute generated code, splice it into an expression body with `▷` (immediate execute), then execute the expression:**

```losp
▼_run ▷GENERATE produce losp code that outputs hello world ◆ ◆
▶_run ◆
```

How this works:
1. `▼_run` starts collecting its body via `evalBodyForDeferredStore`
2. `▷GENERATE` fires immediately during body collection — the generated code text is spliced into the body
3. The generated code (e.g., `▶SAY hello world ◆`) becomes the stored body of `_run`
4. `▶_run ◆` executes: the body is parsed, deferred operators fire, output is produced

**Why `▶▶GENERATE` does NOT work:** The inner `▶GENERATE` returns a multi-line code block as text. The outer `▶` tries to use that entire text blob as an expression name to look up in the namespace — which doesn't exist. The `▶▶` (double execute) pattern only works when the inner expression returns a short name (like `▶▶IF` returning `_ThenBranch`).

## Deliverables

1. **Library** - Programmatic API for embedding losp
2. **CLI** - Standalone executable with flags: `-e`, `-f`, `-db`, `-provider`, `-model`, `-stream`, `-no-prompt`, `-no-stdlib`, `-ollama`, `-persist-mode`, `-compile`
3. **REPL** - Interactive mode when invoked without arguments

## Architecture Notes

- Single-pass streaming interpretation (no separate parse phase)
- SQLite persistence for PERSIST/LOAD builtins
- LLM providers: Ollama (local) and OpenRouter (API)
- Thread-safe runtime for concurrent evaluations

## Exploration and Documentation

**When exploring the codebase, write findings to a document.** Don't just gather information mentally—create a written artifact that captures:

1. **What was explored** - files read, patterns found, code paths traced
2. **Key findings** - important discoveries, existing patterns, constraints
3. **Open questions** - ambiguities that need clarification
4. **Recommendations** - how findings inform the task at hand

Place exploration documents in a sensible location (e.g., alongside the plan file, or in a docs/ directory if appropriate).

**Update CLAUDE.md with key findings.** After completing exploration, add relevant discoveries to this file so future sessions can reference documented knowledge instead of re-exploring. Over time, CLAUDE.md should grow to capture project-specific patterns, gotchas, and architectural decisions—reducing the need for repeated exploration.

## Module Structure: Flat vs Nested

**Prefer flat function definitions at top level.** When functions are nested inside a container `▼`, the defer operator `◯` is consumed at each nesting level, requiring additional `◯` prefixes ("escaping hell"):

```losp
# PROBLEMATIC - nested structure
▼Module
    ▼InnerFunc
        ◯▽x ▷COMPUTE ◆ ◆◆    # ◯ consumed when Module is defined!
    ◆                         # At InnerFunc call time, ▽ fires immediately
◆

# PREFERRED - flat structure
▼InnerFunc
    ◯▽x ▷COMPUTE ◆ ◆◆        # ◯ consumed when InnerFunc is called
◆
▼Module
    ▶InnerFunc ◆
◆
```

When writing new losp modules, prefer the flat pattern.

## Capturing Execution Results

Execution results (from `▶READ`, `▶PROMPT`, etc.) must flow through function arguments and placeholders:

```losp
▼ProcessInput
    □input
    ▶UseInput ▲input ◆
    ▶UseAgain ▲input ◆
◆

▶ProcessInput ▶READ prompt ◆ ◆
```

The `▶READ` executes during argument parsing. The result is bound to the `input` placeholder. Then `▲input` retrieves it for multiple uses.

For dynamic storage under a computed name:

```losp
▼StoreValue □name □value ▼▲name ▲value ◆ ◆

▶StoreValue
    MyResult
    ▶PROMPT system user ◆
◆
▶MyResult ◆    # Execute to get the stored value
```

Use `▶MyResult ◆` (execute) to resolve the stored `▲value`, not `▲MyResult` (retrieve).

## Database Schema

losp uses SQLite for persistence. The schema is:

```sql
CREATE TABLE expressions (
    name TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

The `expressions` table stores persisted expressions. The `value` column contains the full expression definition (e.g., `▼Name body ◆`) or raw text values.

**Useful queries for debugging:**

```bash
# List all persisted expressions
sqlite3 app.db "SELECT name FROM expressions ORDER BY name"

# Check specific value (truncated)
sqlite3 app.db "SELECT name, substr(value, 1, 100) FROM expressions WHERE name = 'MyVar'"

# Check for whitespace issues (shows raw bytes)
sqlite3 app.db "SELECT name, length(value), quote(value) FROM expressions WHERE name = 'MyVar'"

# Find expressions by prefix
sqlite3 app.db "SELECT name, substr(value, 1, 80) FROM expressions WHERE name LIKE 'Sim_Char_%'"
```

## Testing and Debugging losp Applications

When debugging losp applications, follow these strategies:

### Isolate Components First

Before assuming application bugs, test builtins and patterns in isolation:

```bash
# Test APPEND works
./losp -e '▼List ◆ ▶APPEND List item ◆ ▶SAY ▶List ◆ ◆'

# Test argument passing
./losp -e '▼F □x ▶SAY Got: ▲x ◆ ◆ ▶F hello ◆'
```

### Trace Argument Flow

Add SAY debug output at each layer when values disappear in nested calls:

```losp
▼Outer □_o_in ▶SAY [Outer: ▲_o_in] ◆ ▶Middle ▲_o_in ◆ ◆
▼Middle □_m_in ▶SAY [Middle: ▲_m_in] ◆ ▶Inner ▲_m_in ◆ ◆
▼Inner □_i_in ▶SAY [Inner: ▲_i_in] ◆ ◆
```

### Inspect Database State

Verify what was actually persisted (see queries above).

### Check for Placeholder Clobbering

If values vanish in nested calls, look for conflicting placeholder names:

```losp
# BAD: both use □input - Inner clobbers Outer's value
▼Outer □input ▶Inner x ◆ ▲input ◆
▼Inner □input ◆

# GOOD: prefixed names prevent clobbering
▼Outer □_o_input ▶Inner x ◆ ▲_o_input ◆
▼Inner □_i_input ◆
```

### Watch for Clear-Then-Append Patterns

This pattern wipes data if new content is empty:

```losp
▼SetValue □_val
    ▼Target ◆              # CLEARS Target first!
    ▶APPEND Target ▲_val ◆ # If _val is empty, Target is now empty
◆
```

### Automated Testing with Piped Input

For interactive apps, pipe input for repeatable tests:

```bash
echo -e 'line1\nline2\nline3' | ./losp -f app.losp -db test.db
```

### Verify LLM Response Format

When EXTRACT returns empty, check if the LLM response is malformed:

```losp
▶SAY [Raw: ▲_response] ◆
▶SAY [Extracted: ▶EXTRACT FIELD ▲_response ◆] ◆
```
