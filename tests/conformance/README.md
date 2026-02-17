# losp Conformance Tests

Conformance tests validate that a losp implementation correctly handles the language specification defined in PRIMER.md. Any losp implementation (native binary, WASM, embedded runtime) must pass all conformance tests.

## Test Format

Each test is a `.losp` file with directive comments at the top followed by losp code:

```
# EXPECTED: first line of expected output
# EXPECTED: second line of expected output
# INPUT: text provided to stdin for READ calls
▶SAY first line of expected output ◆
▶SAY second line of expected output ◆
```

### Directives

- **`# EXPECTED: <text>`** — One line of expected output. Multiple `# EXPECTED:` lines define multi-line expected output, joined with newlines. The `#` character is not a comment syntax in losp — these lines work because the test runner strips them before piping code to the interpreter.
- **`# INPUT: <text>`** — Text piped to stdin for `▶READ ◆` calls. Supports `\n` escape sequences for multi-line input.

All directives must appear at the top of the file, before any losp code. The test runner reads directives until it encounters a line that is neither `# EXPECTED:` nor `# INPUT:`.

### Output Matching

The test runner captures all stdout from the losp interpreter and compares it exactly (byte-for-byte) against the expected output. Output comes from two sources:

1. **SAY side effects** — `▶SAY text ◆` writes `text\n` to stdout via the output writer
2. **Top-level expression results** — Non-empty results from top-level expressions are echoed to stdout

The test passes if and only if the captured output exactly matches the joined `# EXPECTED:` lines.

### Empty Output

A test with no `# EXPECTED:` directives expects empty output (no stdout at all). This is useful for testing that side-effect-only operations produce no visible output.

## Running Tests

Build the losp binary first, then run the test runner:

```bash
cd losp
go generate ./internal/stdlib/
go build -o ./losp ./cmd/losp
LOSP_BIN=./losp ./tests/conformance/run_tests.sh
```

Run a single category:

```bash
LOSP_BIN=./losp ./tests/conformance/run_tests.sh 32_return_values
```

### Test Isolation

Each test runs with a fresh temporary SQLite database that is deleted after the test completes. Tests do not share state. The database is created via `-db <tmpfile>` so PERSIST/LOAD builtins work.

### Exit Codes

- `0` — All tests passed
- `1` — One or more tests failed

## Test Organization

Tests are organized into numbered category directories:

| Directory | Category |
|-----------|----------|
| `01_store` | Store operator (`▼`) |
| `02_execute` | Execute operator (`▶`) |
| `03_terminator` | Terminator matching (`◆`) |
| `04_placeholder` | Placeholder arguments (`□`) |
| `05_defer` | Defer operator (`◯`) |
| `06_constants` | Constant expressions |
| `07_compare` | COMPARE builtin |
| `08_if` | IF builtin |
| `09_foreach` | FOREACH builtin |
| `10_io` | SAY and READ builtins |
| `11_persist` | PERSIST and LOAD builtins |
| `12_util` | APPEND, COUNT, EXTRACT builtins |
| `13_timing` | ASYNC, AWAIT, TIMER, SLEEP builtins |
| `14_dynamic` | Dynamic naming (`▼▲name`) |
| `15_ephemeral` | Ephemeral expression behavior |
| `16_patterns` | Common losp patterns |
| `17_gotchas` | Edge cases and common mistakes |
| `18_errors` | Error handling |
| `19_retrieve_semantics` | Retrieve vs execute differences |
| `20_capture_result` | Capturing execution results |
| `21_patterns` | Additional patterns |
| `22_read` | READ builtin with INPUT directive |
| `23_string` | UPPER, LOWER, TRIM builtins |
| `24_loadonly` | Load-only mode |
| `25_generate` | GENERATE builtin |
| `26_async` | Async primitives |
| `27_system` | SYSTEM builtin |
| `28_corpus` | CORPUS, ADD, INDEX, SEARCH builtins |
| `29_expr_args` | Expression argument parsing |
| `30_history` | HISTORY builtin |
| `31_dynamic_execute_args` | Dynamic execute with arguments |
| `32_return_values` | Builtin return value verification |
| `33_autoload` | ImmRetrieve inside Store, GENERATE + splice patterns |

## Writing New Tests

1. Create a `.losp` file in the appropriate category directory (or create a new numbered directory)
2. Add `# EXPECTED:` lines at the top with the exact expected output
3. Write the losp code below the directives
4. Run the test to verify it passes

### Conventions

- Test file names describe what is being tested: `say_returns_empty.losp`, `compare_returns_true.losp`
- Each test verifies one specific behavior
- Tests should be self-contained — no dependencies on other test files
- Use `▶COMPARE ... ▶EMPTY ◆ ◆` to verify a builtin returns Empty
- Use stored expressions (`▽A val ◆`) to set up test state

### Implementation Requirements

Any conforming losp implementation must:

1. Process all Unicode operators (`▼▽▲△▶▷□◯◆`) as specified in PRIMER.md
2. Implement all builtins with the correct return values (see PRIMER.md "Builtin Return Values" section)
3. Support the `# EXPECTED:` and `# INPUT:` directive format (stripped before interpretation)
4. Provide an output writer for SAY that writes to stdout
5. Support SQLite-backed persistence for PERSIST/LOAD tests
6. Echo non-empty top-level expression results to stdout
