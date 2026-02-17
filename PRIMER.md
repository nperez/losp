# losp Programmer's Primer

A concise reference for programmers familiar with Lisp and FORTH languages.

## Orientation

losp is a streaming expression language using Unicode operators instead of parentheses. Where Lisp uses `(operator arg1 arg2)`, losp uses `▶operator arg1\narg2 ◆`. Operators consume tokens until the `◆` terminator—no nested parens, no balancing. Designed for LLM metacognition workflows where expressions accumulate context and invoke language models.

losp is interpreted in a single streaming pass. There is no separate parse phase—operators are resolved as they are encountered in the input stream.

**losp has no comment syntax.** All text is meaningful. The `#` character has no special meaning—it's just text like any other character. To annotate code, use stored expressions that are never executed, or keep documentation external to losp files.

---

## Operator Reference

| Op | Unicode | Name | Timing | Description |
|----|---------|------|--------|-------------|
| `▼` | U+25BC | Store | Execution | Store expression body (deferred) |
| `▽` | U+25BD | ImmStore | Parse | Evaluate body now, store result |
| `▲` | U+25B2 | Retrieve | Execution | Retrieve stored expression |
| `△` | U+25B3 | ImmRetrieve | Parse | Retrieve now, substitute into stream |
| `▶` | U+25B6 | Execute | Execution | Execute named expression or builtin |
| `▷` | U+25B7 | ImmExec | Parse | Execute now, substitute result into stream |
| `□` | U+25A1 | Placeholder | — | Declare argument slot (binds to global) |
| `◯` | U+25EF | Defer | — | Prevent parse-time resolution |
| `◆` | U+25C6 | Terminator | — | End current operator's scope |

---

## Core Concept: Parse-Time vs Execution-Time

This is losp's central distinction. Every operator has a timing:

**Parse-time (Immediate)**: Resolved as encountered in the input stream. The result is substituted directly into the stream before parsing continues.

**Execution-time (Deferred)**: Stored as-is. Resolved later when retrieved or executed.

The operators form symmetric pairs:

| Parse-Time | Execution-Time | Operation |
|------------|----------------|-----------|
| `△` | `▲` | Retrieve |
| `▷` | `▶` | Execute |
| `▽` | `▼` | Store |

### Parse-Time Examples

```losp
▽X
    first
◆
▽Snapshot △X ◆    # △X resolves NOW to "first", stored in Snapshot
▽X
    second
◆
▲Snapshot         # → "first" (captured at parse time)
▲X                # → "second" (current value)
```

```losp
▷PROMPT You are helpful. What is 2+2? ◆    # LLM called during parsing
# The response is substituted into the stream here
```

### Execution-Time Examples

```losp
▼Expression
    Current value: ▲X
◆
▽X first ◆
▶Expression ◆       # → "Current value: first"
▽X second ◆
▶Expression ◆       # → "Current value: second"
```

The `▲X` inside Expression is not resolved until `▶Expression ◆` executes.

### The Defer Operator

`◯` prevents parse-time resolution. It's analogous to Lisp's quote:

```losp
▽Expression ◯△X ◆ ◆   # Stores the expression △X itself, not its value
▽X first ◆
▲Expression         # NOW △X resolves → "first"
▽X second ◆
▲Expression         # NOW △X resolves → "second", but ▲Expression still has "first" since retrieving the expression parses it and immediate operators are consumed each each parse.
```

Without `◯`, the `△X` would resolve at parse time and the expression would always return whatever X was when the line was parsed.

### When to Use Immediate Operators

Deferred operators (`▲`, `▶`, `▼`) are the default choice—they create expressions that evaluate fresh each time. Immediate operators (`△`, `▷`, `▽`) serve specific purposes where parse-time evaluation is essential.

**Think of immediate operators like Lisp macros**: they run at "compile time" (parse time) and their results are spliced into the program before execution continues.

### Retrieve vs Execute

Both `▲` and `▶` **parse** the body (immediate operators fire). The difference is what happens to deferred operators:

- **`▲Name`** (Retrieve): Parses the body, returns deferred operators **as text** (unevaluated)
- **`▶Name ◆`** (Execute): Parses the body, then **evaluates** deferred operators

```losp
▼_expr 
    ▶COMPARE 
        hello 
        hello 
    ◆
◆

▲_expr        # Parses body → "▶COMPARE hello\nhello\n◆" (deferred op returned as text)
▶_expr ◆      # Parses body, executes deferred ops → "TRUE"
```

**Immediate operators fire during both retrieve and execute:**

```losp
▼_withImmediate 
    ◯▷COMPARE 
        hello
        hello 
    ◆◆
◆

▲_withImmediate   # ◯ was consumed at definition; now ▷COMPARE fires → "TRUE"
▲_withImmediate   # Body is now empty (▷ was consumed)
```

**Deferred operators only fire during execute:**

```losp
▼_withDeferred ▶COMPARE hello hello ◆ ◆

▲_withDeferred    # → "▶COMPARE hello hello ◆" (preserved as text)
▲_withDeferred    # → "▶COMPARE hello hello ◆" (still preserved, can retrieve again)
▶_withDeferred ◆  # → "TRUE" (now it executes)
▶_withDeferred ◆  # → "TRUE" (repeatable - deferred ops not consumed)
```

**Common pattern:** Store an expression with `▼`, then execute it with `▶` to get the result:

```losp
▼Sim_GenerateCharacter
    ▼_prompt ▶PROMPT Generate a character... ◆ ◆
    ▶_prompt ◆    # Execute to get the LLM response
◆
```

#### Snapshot Pattern

Capture a value before subsequent code changes it:

```losp
▽X first ◆
▽Snapshot △X ◆    # △X resolves NOW, Snapshot = "first"
▽X second ◆
▲Snapshot         # → "first" (frozen at parse time)
▲X                # → "second" (current value)
```

Use this when you need to preserve state across modifications—timestamps, initial values, configuration at startup.

#### Compile-Time Computation

Do expensive work once at load time instead of every execution:

```losp
▽ConfigPath /etc/app/config.json ◆
▽Config ▷ProcessConfig △ConfigPath ◆ ◆  # happens ONCE at parse time
▼ShowConfig ▲Config ◆                   # Uses cached result, no disk I/O
```

The `▷ProcessConfig` executes during parsing. Every subsequent `▶ShowConfig ◆` uses the cached value without re-processing.

#### Execution Order and Placeholder Timing

Understanding when placeholders are bound is critical. When an expression executes:

1. **LOAD** - body text is retrieved
2. **PARSE** - immediate operators fire (parse-time effects)
3. **POPULATE** - placeholders are bound to arguments
4. **EXECUTE** - deferred expressions run

**The critical insight:** Immediate operators in step 2 fire BEFORE placeholders are bound in step 3.

**You cannot capture placeholder values directly**:

```losp
▼Broken
    □arg
    ◯▽result △arg ◆ ◆   # △arg fires at PARSE, before arg is bound!
◆
```

**This is the fundamental constraint of losp's execution model:**
- Immediate operators fire at PARSE (before POPULATE)
- Deferred operators resolve at EXECUTE (after POPULATE)
- There is no way to "capture" a placeholder value at execute time.

#### When NOT to Use Immediate Operators

**Default to deferred operators.** Use immediate only when you specifically need:

- Snapshot semantics (value frozen at definition time)
- Parse-time computation (work done once at load)
- Code generation (metaprogramming)

**Never use immediate operators inside expression bodies to access placeholder values**—they fire at PARSE time before placeholders are bound.

#### Operator Timing Inside Expression Definitions

The timing rules apply recursively. Inside a `▼` expression definition:

- Immediate operators fire when the **outer expression is defined**
- Deferred operators fire when the **outer expression is executed**

This means examples using `▷COMPARE` at top-level do NOT translate directly into expression bodies:

```losp
# At top-level: ▷COMPARE fires during script execution (correct)
▶IF ▷COMPARE ▲Status active ◆ ... ◆

# Inside expression: ▷COMPARE fires during DEFINITION (probably wrong!)
▼MyExpr
    ▶IF ▷COMPARE ▲Status active ◆ ... ◆
◆
# The comparison result is baked in when MyExpr is defined,
# not when ▶MyExpr ◆ is executed.

# For runtime comparison inside an expression:
▼MyExpr
    ▶IF ▶COMPARE ▲Status active ◆ ... ◆
◆
```

**Rule of thumb:** If you want to check current state at execution time, use deferred operators. If you want to bake in a value at definition time (metaprogramming), use immediate operators.

#### Summary: Immediate vs Deferred

| Goal | Operator | Pattern |
|------|----------|---------|
| Expression evaluates fresh each execute | `▲` | `▼Tmpl ▲X ◆` |
| Freeze value at definition time | `△` | `▽Snap △X ◆` |
| Compute once at load time | `▷` | `▽Val ▷FUNC ◆ ◆` |
| Generate code dynamically | `▷` | `▷FOREACH ... ◆` |
| Reference placeholder values | `▲` | `▼Tmpl ▲arg ◆` (indirection, not capture) |

---

## Global Namespace: the dictionary

losp has a single flat namespace, a dictionary. All stores write to it. All retrieves read from it. There is no scope chain, no lexical binding, no closures.

```losp
▽X
    hello
◆
▼SetX
    ▼X
        world
    ◆
◆
▶SetX ◆
▲X            # → "world" — X was modified globally
```

This is intentional. losp is designed for accumulator patterns where state is shared and modified across expressions.

### Dynamic Naming

Store operators support dynamic naming—the name to store under can be computed at runtime:

```losp
▼FieldName X ◆
▼▲FieldName hello ◆   # ▲FieldName resolves to "X", stores "hello" to X
▲X                     # → "hello"
```

This enables iteration patterns and programmatic variable management:

```losp
▼StoreField
    □sf_name □sf_value
    ▼▲sf_name ▲sf_value ◆
◆

▶StoreField
    MyVar
    test
◆
▶MyVar ◆               # → "test" (execute to resolve ▲sf_value)
```

Both `△` (immediate) and `▲` (deferred) work for dynamic naming inside stored expressions because the body is stored as text and evaluated after argument binding.

Note: `▲MyVar` would return `▲sf_value` as text. Use `▶MyVar ◆` to execute and get the actual value.

### Dynamic Execute

Execute operators also support dynamic naming—the expression to execute can be computed at runtime:

```losp
▼ExecDynamic
    □name
    ▶▲name ◆
◆

▶ExecDynamic MyExpression ◆   # Executes whatever expression is named "MyExpression"
```

This is particularly useful with IF to avoid eager branch evaluation:

```losp
▼ShowDebug ▶SAY Debug info ◆ ◆
▼DoNothing ◆

▼MaybeDebug
    ▶ExecDynamic ▶IF ▶COMPARE ▲DebugMode TRUE ◆
        ShowDebug
        DoNothing
    ◆ ◆
◆
```

IF returns the selected branch's value. Since arguments are evaluated during parsing, using `▶Expr ◆` would execute BOTH branches before IF even runs. By passing text expressions (the names) instead, only the selected name is later executed by `▶▲name ◆`.

This can be condensed into a compact pattern for branch execution:

```losp
▼ReturnResult
    ▶_retry_result ◆
◆

▼DoRetry
    ▶SAY [Retrying prompt...] ◆
    ▶PROMPT ▶▲_retry_pname ◆ ◆
◆

▼RetryCheck
    □_src_result □_src_pname
    ▼_retry_result ▲_src_result ◆
    ▼_retry_pname ▲_src_pname ◆
    ▶▶IF ▶COMPARE ▲_src_result ▲EMPTY ◆
        DoRetry
        ReturnResult
    ◆ ◆
◆

▶RetryCheck ▶PROMPT Some prompt that might return EMPTY ◆
```

---

## Placeholder Arguments

`□` declares a parameter slot. When an expression is executed with arguments, each argument is stored into the corresponding placeholder's global variable:

```losp
▼Greet
    □name
    Hello, ▲name!
◆
▶Greet Alice ◆    # Stores "Alice" into global `name`, returns "Hello, Alice!"
▲name             # → "Alice" (still in global namespace)
```

Arguments bind positionally. Use newlines to separate multiple string expression arguments:

```losp
▼Swap
    □a □b
    First: ▲b, Second: ▲a
◆
▶Swap
    X
    Y
◆                 # a="X", b="Y" → "First: Y, Second: X"
```

---

## Argument Parsing

**THE FUNDAMENTAL RULE: All arguments are expressions. Whitespace within a line does NOT split arguments. Only newlines and operator boundaries separate arguments.**

Arguments are parsed as expressions. The rules are:

1. **Text on a single line is one argument** — whitespace within a line does NOT split arguments
2. **Newlines separate arguments** — each line of text becomes a separate argument
3. **Operators are argument boundaries** — each operator result is one argument

**Key insight:** Newlines are only needed to separate TEXT. Operators are already expression boundaries.

```losp
▶COMPARE ▲A ▲B ◆           # Correct: two operator arguments
▶COMPARE hello world ◆     # WRONG: one text argument "hello world"
▶COMPARE
hello
world
◆                          # Correct: two text arguments
```

```losp
▶IF TRUE ▲Yes ▲No ◆
```
This has 3 arguments:
- `TRUE` — text before first operator
- Result of `▲Yes` — operator
- Result of `▲No` — operator

```losp
▶PROMPT
    You are a helpful assistant.
    What is 2+2?
◆
```
This has 2 arguments (two lines of text):
- `You are a helpful assistant.`
- `What is 2+2?`

**Multi-word values from operators are preserved:**

```losp
▼UserInput ▶READ ◆ ◆    # User types "Hello, how are you?"
▶Echo ▶UserInput ◆ ◆    # ▶UserInput ◆ executes, result is ONE argument
```

This is essential for passing user input, LLM responses, and other multi-word content to expressions without it being split apart.

### Clobbering

Because placeholders write to globals, nested executes can clobber:

```losp
▼Outer
    □x
    ▶Inner one ◆
    ▲x
◆
▼Inner
    □x
    ▲x
◆
▶Outer two ◆      # Inner sets x="one", so Outer's ▲x returns "one"
```

This is predictable (depth-first execution order) and confined to the engine instance when operating in in-memory mode. Use distinct placeholder names if you need to avoid collision.

---

## Builtins

**Builtin names are case-sensitive and ALL CAPS.** `▶SAY` invokes the builtin; `▶say`, `▶Say`, etc. look up user-defined expressions. This means user expressions can use any casing without colliding with builtins.

### Control Flow

**IF**: `▶IF condition then-expr else-expr ◆`

Evaluates condition. If result equals `TRUE`, evaluates then-expr; otherwise evaluates else-expr.

```losp
▶IF ▶COMPARE ▲State new ◆
    Setting up...
    Already initialized
◆
```

**One expression per branch.** IF takes exactly 3 arguments: condition, then-expression, else-expression. Each branch is ONE expression. Indentation is for human readability only—losp sees a flat stream of operators and arguments. To execute multiple statements in a branch, wrap them in a stored expression:

```losp
▼DoSetup
    ▶SAY Setting up... ◆
    ▼Initialized TRUE ◆
◆
▼DoNothing ◆

▶▶IF ▶COMPARE ▲State new ◆
    DoSetup
    DoNothing
◆
```

**COMPARE**: `▶COMPARE ▲a ▲b ◆` → `TRUE` or `FALSE` (string equality)

**Mixed-timing pattern**: Use `▷COMPARE` (immediate) inside `▶IF` (deferred) when the comparison can be resolved at parse time:

```losp
▶IF ▷COMPARE ▲State new ◆
    Setting up...
    Already initialized
◆
```

The `▷COMPARE` fires during IF's argument parsing, returning TRUE or FALSE immediately. This is useful when comparing against constants or values that won't change during execution.

**FOREACH**: `▶FOREACH items-expr body-name ◆`

Two expression arguments. The first evaluates to text containing expressions (one per line or operator boundary); these are re-parsed, and each result is passed as the first argument to the expression named by the second argument:

```losp
▼ShowItem
    □item
    ▶SAY - ▲item ◆
◆

▼Items
    apple
    banana
    cherry
◆

▶SAY == Items! == ◆

▶FOREACH
    ▲Items
    ShowItem
◆

# Output
== Items! ==
- apple
- banana
- cherry
```

The body name is evaluated, so dynamic dispatch works:

```losp
▼BodyRef ShowItem ◆
▶FOREACH ▲Items ▲BodyRef ◆
```

### LLM Interaction

**PROMPT**: `▶PROMPT system-prompt user-prompt ◆`

Sends to LLM with the given system and user prompts. Returns the response.

```losp
▼Response ▶PROMPT
    You are a helpful assistant.
    What is the capital of France?
◆ ◆
▲Response    # → "Paris"
```

For a simple prompt without a system message, use an empty first argument:

```losp
▶PROMPT

    What is 2+2?
◆
```

### Code Generation

**GENERATE**: `▶GENERATE request ◆`

LLM code generation of losp. You can express losp in plain language and have it generated. Returns the generated code as text.

GENERATE returns code — it does not execute it. To execute generated code, splice it into an expression body using `▷` (immediate execute) during a `▼` (store) definition:

```losp
▼_run ▷GENERATE Create a function that outputs hello world ◆ ◆
▶_run ◆
```

How this works:
1. `▼_run` begins collecting its body
2. `▷GENERATE` fires immediately during body collection — the generated code text is spliced into the body
3. The generated code (e.g., `▶SAY hello world ◆`) becomes the stored body of `_run`
4. `▶_run ◆` executes the body — deferred operators fire, producing output

If no LLM provider is configured, GENERATE returns EMPTY. If the request is empty, GENERATE returns EMPTY.

### I/O

**SAY**: `▶SAY text... ◆` → outputs text and any number of expressions

**READ**: `▶READ [prompt] ◆` → reads line from user input

```losp
▼UserInput ▶READ Enter your name: ◆ ◆
▶SAY Hello, ▲UserInput ◆
```

### Persistence

**PERSIST**: `▶PERSIST name ◆` → saves current value to backing store (disk, sqlite, blob storage, etc.)

**LOAD**: `▶LOAD name [default] ◆` → retrieves from backing store

```losp
▶LOAD History ◆      # Load previous session (if exists)
▼History
    ▲History
    New entry
◆
▶PERSIST History ◆   # Save for next session
```

LOAD accepts an optional default value. If the key doesn't exist or is empty, the default is used:

```losp
▶LOAD NPC_Trust
    low
◆                    # Sets NPC_Trust to "low" if not in DB
```

Persistence is explicit. Normal global variables exist only for the engine instance lifetime.

In `ALWAYS` mode (`▶SYSTEM PERSIST_MODE ALWAYS ◆`), every store operation auto-persists, and PERSIST is a no-op — the value is already persisted. PERSIST is also a no-op in `NEVER` mode.

Persistence uses append-only versioned storage: every mutation that changes an expression's value appends a new version row. Retrieval always returns the latest version. Use `HISTORY` to query prior versions.

### Data Extraction

**EXTRACT**: `▶EXTRACT label source ◆` → extracts labeled value from structured text

Parses text for `LABEL: value` format and returns the value. Useful for parsing LLM responses without additional API calls:

```losp
▼raw_response ▶PROMPT
    Analyze this and respond with:
    SENTIMENT: positive/negative/neutral
    CONFIDENCE: high/medium/low
    User text to analyze...
◆ ◆

▼sentiment ▶EXTRACT SENTIMENT ▲raw_response ◆ ◆
▼confidence ▶EXTRACT CONFIDENCE ▲raw_response ◆ ◆
```

EXTRACT handles multi-line values (continues until the next label or end of text) and is case-insensitive for label matching.

### String Manipulation

**UPPER**: `▶UPPER expr... ◆` → converts each expression to uppercase

```losp
▶UPPER hello world ◆           # → "HELLO WORLD"
▶UPPER
    first line
    second line
◆                               # → "FIRST LINE\nSECOND LINE"
```

**LOWER**: `▶LOWER expr... ◆` → converts each expression to lowercase

```losp
▶LOWER HELLO WORLD ◆           # → "hello world"
```

**TRIM**: `▶TRIM expr... ◆` → removes leading/trailing whitespace from each expression

```losp
▶TRIM    hello world    ◆      # → "hello world"
▶TRIM
    padded line one
      padded line two
◆                               # → "padded line one\npadded line two"
```

These operate on all expressions passed to them. Results are the mutated expressions. TRIM filters out expressions that become empty after trimming.

Useful for case-insensitive comparison:

```losp
▶IF ▶COMPARE ▶LOWER ▲UserInput ◆ yes ◆
    User said yes
    User said something else
◆
```

### Utilities

**COUNT**: `▶COUNT expr ◆` → counts expressions within the expression

**RANDOM**: `▶RANDOM expr ◆` → returns one random expression from the evaluated list

```losp
▼Colors
    red
    green
    blue
◆
▶RANDOM ▲Colors ◆    # → one of "red", "green", or "blue"
```

Returns EMPTY if the input is empty.

**APPEND**: Appends an expression to another expression. First argument is an expression with the name of another expression or a string of the name. Second argument is an expression to append:

```losp
▶APPEND
    ListName
    new content to append
◆
```

**EMPTY**: `▲EMPTY` → Special empty expression useful for empty testing

### Async Primitives

losp supports concurrent execution through async forking. This is primarily useful for launching parallel LLM calls via PROMPT/GENERATE.

**ASYNC**: `▶ASYNC expression-name ◆` → returns a handle (e.g. `_async_1`)

Forks execution of a named expression in a new goroutine. The forked evaluator gets a **cloned namespace** (snapshot at fork time, writes are isolated) but **shares** the persistence store and LLM provider. SAY is silenced and READ returns EMPTY in forked evaluators.

```losp
▼SlowCall ▶PROMPT
    You are an expert.
    Analyze this data...
◆ ◆

▽handle ▶ASYNC SlowCall ◆ ◆
▽result ▶AWAIT ▲handle ◆ ◆
▶SAY ▲result ◆
```

ASYNC takes only the expression name (no arguments). To pass data, set namespace values before forking — the clone captures them:

```losp
▽context some relevant data ◆
▽handle ▶ASYNC UseContext ◆ ◆
```

Returns EMPTY if the expression doesn't exist.

**AWAIT**: `▶AWAIT handle ◆` → blocks until completion, returns result text

Blocks the current evaluator until the async handle completes. Returns the result of the forked expression, or EMPTY on error. Double-AWAIT on the same handle is safe and returns the cached result immediately.

```losp
▽h1 ▶ASYNC Call1 ◆ ◆
▽h2 ▶ASYNC Call2 ◆ ◆
▽r1 ▶AWAIT ▲h1 ◆ ◆
▽r2 ▶AWAIT ▲h2 ◆ ◆
```

Returns EMPTY if the handle is unknown.

**CHECK**: `▶CHECK handle ◆` → `TRUE` or `FALSE`

Non-blocking completion check. Returns TRUE if the async operation has finished, FALSE otherwise. Returns FALSE for unknown handles.

```losp
▽h ▶ASYNC SlowOp ◆ ◆
▶IF ▶CHECK ▲h ◆
    Done!
    Still working...
◆
```

**TIMER**: `▶TIMER ms expression-name ◆` → returns a handle

Delayed fire-once execution. The expression runs after the specified milliseconds. A 0ms timer fires immediately (effectively an ASYNC).

```losp
▼Cleanup ▶PERSIST State ◆ ◆

▽t ▶TIMER
5000
Cleanup
◆ ◆
```

**TICKS**: `▶TICKS handle ◆` → milliseconds remaining (text)

Query a timer's countdown. Returns 0 for non-timer handles (promises) and for completed timers.

```losp
▽t ▶TIMER
5000
Cleanup
◆ ◆
▶SAY ▶TICKS ▲t ◆ ms remaining ◆
```

**SLEEP**: `▶SLEEP ms ◆` → EMPTY

Blocks the current evaluator for the specified duration in milliseconds.

```losp
▶SLEEP 1000 ◆
▶SAY One second later ◆
```

All handles are unified — AWAIT, CHECK, and TICKS work on both ASYNC and TIMER handles.

### Runtime Configuration

**SYSTEM**: `▶SYSTEM setting [value] ◆`

Query or change runtime settings. With one argument, returns the current value. With two arguments, sets a new value.

| Setting | Description |
|---------|-------------|
| `MODEL` | LLM model name |
| `PROVIDER` | LLM provider (OLLAMA, OPENROUTER, ANTHROPIC) |
| `PERSIST_MODE` | Persistence behavior (ON_DEMAND, ALWAYS, NEVER) |
| `TEMPERATURE` | Sampling temperature |
| `NUM_CTX` | Context window size (Ollama) |
| `TOP_K` | Top-k sampling |
| `TOP_P` | Top-p / nucleus sampling |
| `MAX_TOKENS` | Max response tokens |
| `EMBED_MODEL` | Embedding model (Ollama default: `qwen3-embedding:0.6b`) |
| `SEARCH_LIMIT` | Max results from SEARCH/SIMILAR (default 10) |
| `HISTORY_LIMIT` | Max versions returned by HISTORY (default 0 = all) |

```losp
▶SAY Current model: ▶SYSTEM MODEL ◆ ◆

▶SYSTEM
    MODEL
    qwen3:4b
◆
▶SYSTEM
    TEMPERATURE
    0.3
◆
▶PROMPT Be concise. What is 2+2? ◆

▶SYSTEM
    PROVIDER
    ANTHROPIC
◆
▶SYSTEM
    MODEL
    claude-sonnet-4-20250514
◆
▶SYSTEM
    TEMPERATURE
    0.9
◆
▶PROMPT Be creative. Write a haiku. ◆
```

Switching providers with `SYSTEM PROVIDER` creates a new provider instance and copies inference parameters (TEMPERATURE, TOP_K, etc.) from the old provider. The MODEL is not copied — each provider starts with its default model.

Unknown settings return `UNKNOWN_SETTING`. Unknown provider names return `UNKNOWN_PROVIDER`. If no provider is configured, MODEL/TEMPERATURE/etc. return EMPTY.

### Corpus and Search

**CORPUS**: `▶CORPUS name ◆` → returns a handle (e.g. `_corpus_1`)

Creates or loads a named corpus — a persistent collection of expressions that can be indexed for full-text search and vector similarity search. If a corpus with the given name already exists in the database, it is loaded with its membership and indexes intact. Returns a handle ID for use with ADD, INDEX, SEARCH, EMBED, and SIMILAR.

```losp
▽c ▶CORPUS characters ◆ ◆
▶ADD ▲c CharName ◆
▶ADD ▲c CharBio ◆
```

CORPUS is idempotent — calling it multiple times with the same name returns a handle to the same corpus.

**ADD**: `▶ADD handle expr-name ◆` → `EMPTY`

Adds a named expression to a corpus. The expression must exist in the namespace. Both the membership and the expression's current value are recorded.

```losp
▶ADD ▲c Sim_Char_Name ◆
▶ADD ▲c Sim_Char_Bio ◆
```

**INDEX**: `▶INDEX handle ◆` → `EMPTY`

Builds or rebuilds the full-text search (FTS5) index for a corpus. Indexes the current value of each member expression. Call again after adding new members or updating expression values.

```losp
▶INDEX ▲c ◆
```

**SEARCH**: `▶SEARCH handle query ◆` → matching expression names (newline-separated)

Full-text search within a corpus. Returns the names of matching expressions, ordered by relevance. Max results controlled by `SYSTEM SEARCH_LIMIT` (default 10).

```losp
▶SEARCH ▲c warrior ◆
```

**EMBED**: `▶EMBED handle ◆` → `EMPTY`

Generates vector embeddings for all un-embedded members of a corpus using the active LLM provider. Embeddings are persisted in the database. Incremental — only processes members that don't already have embeddings. Also builds the HNSW vector index.

```losp
▶EMBED ▲c ◆
```

**SIMILAR**: `▶SIMILAR handle query ◆` → matching expression names (newline-separated)

Vector similarity search within a corpus. Embeds the query text, then finds the nearest neighbors in the HNSW index. Returns expression names ordered by similarity. Max results controlled by `SYSTEM SEARCH_LIMIT` (default 10).

```losp
▶SIMILAR ▲c brave hero who fights dragons ◆
```

EMBED must have been called on the corpus first.

### Version History

**HISTORY**: `▶HISTORY name ◆` → versioned expression names (newline-separated, newest first)

Queries the version history of a persisted expression. All persisted expressions have history — every write to the database that changes the value appends a new version. In `PERSIST_MODE ALWAYS`, versions accumulate automatically on every store operation. In `EXPLICIT` mode, each `▶PERSIST name ◆` call that changes the value adds a new version. Duplicate consecutive values are not stored.

HISTORY creates ephemeral named expressions in the namespace (e.g., `_X_1`, `_X_2`, `_X_3`) — one per version. Each is a deferred store that, when executed, redefines the original expression to that version's value (rollback).

```losp
▶SYSTEM
    PERSIST_MODE
    ALWAYS
◆
▽X first value ◆
▽X second value ◆
▽X third value ◆

▶SAY ▶HISTORY X ◆ ◆
# Prints:
# _X_3
# _X_2
# _X_1
```

**Rollback** — execute a version expression to restore an earlier value:

```losp
▽_h ▶HISTORY X ◆ ◆
▶_X_1 ◆           # X is now "first value" again
▶SAY ▲X ◆         # Prints: first value
```

**Introspect** — retrieve a version expression to see its definition without rollback:

```losp
▲_X_2              # Returns the ▼X second value ◆ definition text
```

Control the number of versions returned with `SYSTEM HISTORY_LIMIT`:

```losp
▶SYSTEM
    HISTORY_LIMIT
    5
◆
▶HISTORY X ◆      # Returns at most 5 most recent versions
```

HISTORY returns EMPTY if the expression has no version history or doesn't exist.

HISTORY expressions work with CORPUS for semantic search over version history:

```losp
▽_h ▶HISTORY X ◆ ◆
▽c ▶CORPUS versions ◆ ◆
▶ADD ▲c _X_1 ◆
▶ADD ▲c _X_2 ◆
▶ADD ▲c _X_3 ◆
▶INDEX ▲c ◆
▶SEARCH ▲c keyword ◆   # Find which version mentions "keyword"
```

---

## Gotchas

### Immediate Operators Execute During Parsing

```losp
▷PROMPT You are a bot. Tell me a joke ◆
# The LLM is called RIGHT HERE during parsing
# The joke appears in the parse stream at this position
```

This is powerful but can be surprising. You can capture the result with a store:

```losp
▼Joke ▷PROMPT You are a bot. Tell me a joke ◆ ◆
# LLM called during parsing, result stored in Joke
```

### Placeholder Clobbering

All placeholders write to the dictionary:

```losp
▼x important_value ◆
▼Func
    □x
    ...
◆
▶Func something ◆
▲x    # → "something" — the original value is gone
```

Use unique placeholder names to avoid unintended clobbering.

### Nested `▼` and the Defer Operator

losp is a streaming interpreter—operators are processed as they are encountered. This has implications for nested expression definitions.

When you define an expression inside another expression:

```losp
▼Outer
    ▼Inner
        ◯▽result ▶PROMPT Say hi ◆ ◆◆
    ◆
◆
```

The defer operator protects `▽` from immediate evaluation, producing to be stored:

```
Inner's body = "▽result ▶PROMPT Say hi ◆ ◆"
```

But when `▶Outer ◆` runs and `▼Inner` is defined, `Inner`'s body is **re-parsed**. Now there's no `◯` protecting `▽`, so the PROMPT executes during `▶Outer ◆`—not during `▶Inner ◆`.

**The depth of nesting determines when deferred-immediate operators execute:**

| Nesting | When `◯▽` executes |
|---------|-------------------|
| `▼Func` at top level | When `▶Func ◆` is executed |
| `▼Outer` → `▼Inner` | When `▶Outer ◆` is executed |
| `▼A` → `▼B` → `▼C` | When `▶A ◆` is executed |

**To defer through multiple nesting levels**, you need multiple `◯` operators:

```losp
▼Outer
    ▼Inner
        ◯◯▽result ▶PROMPT Say hi ◆ ◆◆◆  # Two defers for two levels
    ◆
◆
```

**Recommendation**: Avoid deeply nested expression definitions. Define expressions at the top level and use a simple initialization expression to set up state:

```losp
# Define at top level
▼Inner
    ◯▽result ▶PROMPT Say hi ◆ ◆◆
◆

▼Outer
    # Just execute Inner, don't define it here
    ▶Inner ◆
◆
```

This keeps the defer semantics predictable and avoids "escaping hell" when combining multiple levels of deferral.

---

## Patterns

### The Chatbot Accumulator

```losp
▼ChatLoop
    ▶ChatLoopWithInput ▶READ You: ◆ ◆
◆

▼ChatLoopWithInput
    □_cli_input
    ▶APPEND History User: ▲_cli_input ◆
    ▼_cli_response ▶PROMPT
        You are a helpful assistant.
        ▲History
    ◆ ◆
    ▶SAY Assistant: ▶_cli_response ◆ ◆
    ▶APPEND History Assistant: ▶_cli_response ◆ ◆
    ▶PERSIST History ◆
    ▶ChatLoop ◆
◆

▶LOAD History ◆
▶ChatLoop ◆
```

**Key patterns used:**
- **Argument passing for I/O results**: `▶READ` executes when passed as an argument, and the result flows through the placeholder `□_cli_input`. This ensures READ runs fresh each iteration.
- **APPEND for accumulation**: Use `▶APPEND` to add to History rather than redefining it with nested retrieves.
- **Execute for stored prompts**: Use `▶_cli_response ◆` to execute and get the PROMPT result.

### Cached LLM Responses

```losp
▼Analysis ▷PROMPT
    You are an analyst.
    Analyze this document...
◆ ◆
▲Analysis    # Returns cached result
▲Analysis    # Same cached result, no API call
```

### Parameterized Expressions

```losp
▼Summarize
    □text □style
    ▶PROMPT
        You summarize text in the requested style.
        Summarize in a ▲style style: ▲text
    ◆
◆
▶Summarize
    Some long document that needs summarizing...
    formal
◆
```

### Capturing Execution Results

To use an execution result (from `▶READ`, `▶PROMPT`, etc.) in multiple places, flow it through a function's placeholder:

```losp
▼ProcessInput
    □input
    ▶CheckMode ▲input ◆
    ▶Respond ▲input ◆
◆

▶ProcessInput ▶READ > ◆ ◆
```

The `▶READ` executes during argument parsing. The result binds to `□input`, then `▲input` retrieves it for each use.

For storing under a dynamic name:

```losp
▼StoreResult □name □value ▼▲name ▲value ◆ ◆

▶StoreResult
    MyVar
    ▶PROMPT system user ◆
◆
▶MyVar ◆    # Execute to get the value
```

---

## Program Organization

losp has a flat, global namespace called the dictionary by design. This section describes conventions for organizing larger programs.

### Module Envelope Pattern

Group related expressions under a module definition that defines everything when executed:

```losp
▼MyApp
    ▼MyApp_Helper ... ◆
    ▼_MyApp_private ... ◆
    ▼MyApp_Main ... ◆
◆

▶MyApp ◆           # Defines all nested expressions
▶MyApp_Main ◆      # Run the program
```

This pattern:
- Keeps related code together in source files
- Allows selective loading (don't execute the envelope to skip the module)
- Makes dependencies explicit

### Naming Conventions

| Pattern | Purpose | Example |
|---------|---------|---------|
| `Module_Category_Name` | Data variables | `NPC_Identity_Name` |
| `Module_ExpressionName` | Public expressions | `NPC_GenerateResponse` |
| `_Module_helper` | Private/internal helpers | `_NPC_extractField` |
| `_Module_Manifest` | Module key list (system) | `_NPC_Manifest` |
| `fn_localvar` | Expression-local temps | `gen_result`, `loop_input` |

**Underscore prefix (`_`)** indicates internal/system keys not intended for direct use.

**Expression prefixes** prevent clobbering in nested executions:

```losp
▼GenerateResponse
    □gen_input          # Prefixed with gen_
    ▼gen_result ... ◆   # Won't clobber other expressions' variables
◆

▼IntrospectEmotion
    □intro_input        # Prefixed with intro_
    ▼intro_raw ... ◆
◆
```

### Manifest Convention

For bulk operations (load all, persist all), use explicit manifests:

```losp
▼MyModule
    ▼MyModule_Func1 ... ◆
    ▼MyModule_Func2 ... ◆
    ▼_MyModule_Manifest Func1
Func2 ◆
◆
```

The manifest lists key suffixes. Helper expressions can iterate over it:

```losp
▼MyModule_LoadAll
    ▶LOAD MyModule_Func1 ◆
    ▶LOAD MyModule_Func2 ◆
◆

▼MyModule_PersistAll
    ▶PERSIST MyModule_Func1 ◆
    ▶PERSIST MyModule_Func2 ◆
◆
```

This is more explicit than pattern-matching and survives refactoring.

### Placeholder Safety

Placeholders write to globals, creating clobbering risk. Two mitigation strategies:

**Strategy 1: Prefixed placeholders**

```losp
▼SafeFunc
    □sf_arg1 □sf_arg2
    ▶OtherFunc ◆       # Cannot clobber sf_arg1
◆
```

**Strategy 2: Immediate capture**

```losp
▼SafeFunc
    □input
    ▼sf_input ▲input ◆  # Capture to prefixed name immediately
    ▶OtherFunc ◆        # input may be clobbered, but sf_input is safe
◆
```

Use prefixed placeholders for expressions that execute other expressions. Simple leaf expressions can use short names.

### Program Lifecycle

Structure programs for persistence and reload:

```losp
# 1. Define all expressions
▼MyApp
    ▼MyApp_Initialize ... ◆
    ▼MyApp_Main ... ◆
    ▼MyApp_Cleanup ... ◆
◆
▶MyApp ◆

# 2. Load state (with defaults for first run)
▶LOAD MyApp_State
    initial
◆

# 3. Run the program
▶MyApp_Main ◆

# 4. Persist state before exit
▶PERSIST MyApp_State ◆
```

For complex state, separate immutable (identity, configuration) from mutable (dynamic state):

```losp
▼LoadImmutable
    ▶LOAD Config_Name ◆
    ▶LOAD Config_Version ◆
◆

▼LoadMutable
    ▶LOAD State_Counter ◆
    ▶LOAD State_LastRun ◆
◆

▼PersistMutable
    ▶PERSIST State_Counter ◆
    ▶PERSIST State_LastRun ◆
◆
```

---

## Standard Library

losp includes a minimal standard library (prelude) that's automatically loaded unless `-no-stdlib` is specified.

### __startup__

A placeholder expression executed after loading. Programs can override it to define their entry point:

```losp
▼__startup__
    ▶MyApp_Main ◆
◆
```

The default `__startup__` is empty.

### Customizing the Standard Library

The standard library can be overridden by persisting a custom `__stdlib__`:

```losp
▼__stdlib__
    ▼std_MyHelper ... ◆
    ▼std_AnotherHelper ... ◆
◆
▶PERSIST __stdlib__ ◆
```

On subsequent runs, the backing store `__stdlib__` replaces the built-in prelude.

---

## Builtin Return Values

Every builtin returns a value. Understanding what each builtin returns is critical for composing expressions correctly. Builtins that perform side effects (output, storage, persistence) return EMPTY. Builtins that compute or transform data return their result as text.

| Builtin | Returns | Value |
|---------|---------|-------|
| `TRUE` | Text | `"TRUE"` |
| `FALSE` | Text | `"FALSE"` |
| `EMPTY` | Empty | `""` |
| `COMPARE` | Text | `"TRUE"` or `"FALSE"` |
| `IF` | Text | Selected branch text (then or else) |
| `FOREACH` | Text | Joined results of body execution (newline-separated) |
| `SAY` | Empty | Always EMPTY — output is a side effect via the output writer |
| `READ` | Text | User input text, or EMPTY if no input reader |
| `COUNT` | Text | Number of expressions as a string (e.g., `"3"`) |
| `RANDOM` | Text or Empty | One random expression from the list, or EMPTY if input is empty |
| `APPEND` | Empty | Always EMPTY — mutation is a side effect |
| `EXTRACT` | Text or Empty | Extracted field value, or EMPTY if label not found |
| `UPPER` | Text | Uppercased text |
| `LOWER` | Text | Lowercased text |
| `TRIM` | Text or Empty | Trimmed text, or EMPTY if result is blank |
| `PERSIST` | Empty | Always EMPTY — persistence is a side effect |
| `LOAD` | Empty | Always EMPTY — loads into namespace as a side effect |
| `PROMPT` | Text | LLM response text, or EMPTY if no provider |
| `GENERATE` | Text | Generated losp code text, or EMPTY if no provider |
| `SYSTEM` | Text or Empty | Current setting value (getter) or EMPTY (setter) |
| `ASYNC` | Text | Handle ID (e.g., `"_async_1"`), or EMPTY if expression missing |
| `AWAIT` | Text or Empty | Async result text, or EMPTY on error/unknown handle |
| `CHECK` | Text | `"TRUE"` or `"FALSE"` |
| `TIMER` | Text | Handle ID, or EMPTY if expression missing |
| `TICKS` | Text | Milliseconds remaining as string (e.g., `"4500"`) |
| `SLEEP` | Empty | Always EMPTY |
| `CORPUS` | Text | Handle ID (e.g., `"_corpus_1"`) |
| `ADD` | Empty | Always EMPTY |
| `INDEX` | Empty | Always EMPTY |
| `SEARCH` | Text or Empty | Matching expression names (newline-separated), or EMPTY |
| `EMBED` | Empty | Always EMPTY |
| `SIMILAR` | Text or Empty | Matching expression names (newline-separated), or EMPTY |
| `HISTORY` | Text or Empty | Version expression names (newline-separated), or EMPTY |

**Key distinctions:**

- **SAY returns EMPTY, not its argument.** SAY's job is to produce output via the output writer. Its return value is always EMPTY. If you need the text that SAY would output, use the text directly — don't try to capture SAY's return.
- **LOAD returns EMPTY, not the loaded value.** LOAD populates the namespace as a side effect. To access the loaded value, retrieve it by name after LOAD: `▶LOAD X ◆` then `▲X`.
- **APPEND returns EMPTY.** Like SAY, APPEND mutates state as a side effect. The appended-to expression is modified in place.
- **IF returns text, not an executed result.** IF returns the selected branch as a string. To execute the selected branch, use `▶▶IF ...◆ ◆` (dynamic execute).
- **FOREACH returns joined results.** Each body execution produces a result; FOREACH joins them with newlines and returns the combined text.

---

## Quick Reference

| Want to... | Use |
|------------|-----|
| Store expressions | `▼Name body ◆` |
| Store expressions during parsing | `▽Name body ◆` |
| Store with dynamic name | `▼▲NameVar value ◆` |
| Retrieve at execution time | `▲Name` |
| Retrieve now (parse time) | `△Name` |
| Execute at execution time | `▶Name args ◆` (args are expressions) |
| Execute now (parse time) | `▷Name args ◆` (args are expressions) |
| Prevent parse-time resolution | `◯ expr ◆` |
| Declare placeholder | `□paramName` |
| End operator scope | `◆` |
| Check equality | `▶COMPARE ▲a ▲b ◆` → TRUE/FALSE |
| Conditional | `▶IF cond then else ◆` (args are expressions) |
| Iterate over items | `▶FOREACH items-expr body-name ◆` |
| Prompt LLM | `▶PROMPT system user ◆` (args are expressions) |
| Extract labeled field | `▶EXTRACT LABEL ▲source ◆` |
| Convert to uppercase | `▶UPPER expr... ◆` |
| Convert to lowercase | `▶LOWER expr... ◆` |
| Trim whitespace | `▶TRIM expr... ◆` |
| Save to backing store | `▶PERSIST name ◆` |
| Load from backing store | `▶LOAD name ◆` |
| Load with default | `▶LOAD name default ◆` (args are expressions) |
| Pick random expression | `▶RANDOM expr ◆` → one random item |
| Fork async execution | `▶ASYNC expr-name ◆` → handle |
| Wait for async result | `▶AWAIT handle ◆` → result text |
| Check if async done | `▶CHECK handle ◆` → TRUE/FALSE |
| Delayed execution | `▶TIMER ms expr-name ◆` → handle |
| Query timer remaining | `▶TICKS handle ◆` → ms remaining |
| Sleep | `▶SLEEP ms ◆` |
| Query/set runtime config | `▶SYSTEM setting [value] ◆` |
| Create/load corpus | `▶CORPUS name ◆` → handle |
| Add expression to corpus | `▶ADD handle expr-name ◆` |
| Build FTS index | `▶INDEX handle ◆` |
| Full-text search | `▶SEARCH handle query ◆` → names |
| Generate embeddings | `▶EMBED handle ◆` |
| Vector similarity search | `▶SIMILAR handle query ◆` → names |
| Query version history | `▶HISTORY name ◆` → version names |
| Rollback to version | `▶_Name_N ◆` (execute a HISTORY version) |

---

## Testing losp Applications

### Isolate and Test Components

Test builtins and patterns in isolation before debugging complex applications:

```bash
# Test APPEND behavior
./losp -e '▼List ◆
▶APPEND
List
first item
◆
▶SAY Result: ▶List ◆ ◆'

# Test argument passing through helper
./losp -e '▼Helper □arg ▶SAY Got: ▲arg ◆ ◆
▶Helper test value ◆'
```

### Use SAY for Debug Output

Wrap values in SAY to trace execution flow:

```losp
▼ProcessData
    □_pd_input
    ▶SAY [ProcessData received: ▲_pd_input] ◆
    ▶NextStep ▲_pd_input ◆
◆
```

### Test Argument Flow Through Layers

When debugging nested calls, add debug output at each layer:

```losp
▼Outer
    □_o_input
    ▶SAY [Outer got: ▲_o_input] ◆
    ▶Middle ▲_o_input ◆
◆

▼Middle
    □_m_input
    ▶SAY [Middle got: ▲_m_input] ◆
    ▶Inner ▲_m_input ◆
◆

▼Inner
    □_i_input
    ▶SAY [Inner got: ▲_i_input] ◆
◆

▶Outer test value ◆
```

### Inspect Database State

Use sqlite3 to verify what was actually persisted:

```bash
# Check specific values
sqlite3 app.db "SELECT name, substr(value, 1, 100) FROM expressions WHERE name LIKE 'MyApp_%'"

# Check raw bytes for whitespace issues
sqlite3 app.db "SELECT name, length(value), quote(value) FROM expressions WHERE name = 'MyVar'"
```

### Automated Testing with Piped Input

For interactive applications, pipe input for automated testing:

```bash
# Using echo -e for newline-separated inputs
echo -e 'input1\ninput2\ninput3' | ./losp -f app.losp -db test.db

# Using a file for longer test sequences
cat > /tmp/test_input.txt << 'EOF'
line 1
line 2
line 3
EOF
./losp -f app.losp -db test.db < /tmp/test_input.txt
```

### Common Debugging Patterns

**Placeholder clobbering**: If values disappear in nested calls, check for conflicting placeholder names:

```losp
# BAD: both use □input
▼Outer □input ▶Inner something ◆ ▲input ◆
▼Inner □input ... ◆

# GOOD: prefixed names
▼Outer □_o_input ▶Inner something ◆ ▲_o_input ◆
▼Inner □_i_input ... ◆
```

**Empty results from EXTRACT**: Check if the LLM response contains the expected label format:

```losp
▶SAY [Raw response: ▲_raw ◆] ◆
▶SAY [Extracted FIELD: ▶EXTRACT FIELD ▲_raw ◆] ◆
```

**Compaction/clearing bugs**: If data disappears, check if any expression uses `▼Name ◆` (which clears) before `▶APPEND`:

```losp
# This pattern CLEARS then appends - dangerous if new content is empty
▼SetValue
    □_val
    ▼Target ◆           # Clears Target!
    ▶APPEND Target ▲_val ◆
◆
```

---

## Summary

losp has two evaluation times: parse-time (immediate) and execution-time (deferred). Immediate operators (`△`, `▷`, `▽`) resolve as the input stream is read. Deferred operators (`▲`, `▶`, `▼`) resolve when explicitly executed. The `◯` defer operator until the next `◆` prevents parse-time resolution.

All variables live in the dictionary. Placeholders bind arguments to the global variables with those names. There is no lexical scoping.

The language is optimized for stateful LLM workflows where dynamic context accumulates and mutates across turns, not for pure functional computation.
