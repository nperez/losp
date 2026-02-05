# losp Quick Reference

A streaming template language using Unicode operators.

## Operators

| Op | Name | Timing | Description |
|----|------|--------|-------------|
| `▼` | Store | Deferred | Store expression (definition preserved) |
| `▽` | ImmStore | Immediate | Evaluate now, store result |
| `▲` | Retrieve | Deferred | Retrieve stored expression |
| `△` | ImmRetrieve | Immediate | Retrieve now, substitute into stream |
| `▶` | Execute | Deferred | Execute expression or builtin |
| `▷` | ImmExec | Immediate | Execute now, substitute result |
| `□` | Placeholder | — | Declare argument slot |
| `◯` | Defer | — | Prevent immediate resolution (consumed when used) |
| `◆` | Terminator | — | End operator scope |

## The Rules

1. **Immediate operators fire when parsed.** `△`, `▷`, `▽` resolve as encountered.
2. **Deferred operators fire when executed.** `▲`, `▶`, `▼` resolve later.
3. **Every `◆` terminates ONE operator.** Count your terminators.
4. **◯ defers until next parse, then is consumed.** Not preserved in bodies.
5. **Global namespace.** All variables share one flat dictionary.

## Execution Order

```
LOAD → PARSE → POPULATE → EXECUTE
         ↑           ↑
   imm ops fire   placeholders bound
```

**Critical:** Immediate operators fire BEFORE placeholders are bound.

## Arguments

**Newlines separate text arguments. Operators are already boundaries.**

| Code | Args |
|------|------|
| `▶FUNC hello world ◆` | 1: "hello world" |
| `▶FUNC`<br>`hello`<br>`world`<br>`◆` | 2: "hello", "world" |
| `▶FUNC ▲A ▲B ◆` | 2: result of ▲A, result of ▲B |
| `▶COMPARE ▲X yes ◆` | 2: result of ▲X, "yes" (operator + text) |

**CRITICAL: For COMPARE with two literal strings, MUST use newlines:**
```losp
▶COMPARE
a
a
◆
```
Returns: TRUE

```losp
▶COMPARE
a
b
◆
```
Returns: FALSE

**`▶COMPARE a a ◆` is WRONG** - it compares "a a" with nothing!

## Builtins

| Builtin | Signature | Returns |
|---------|-----------|---------|
| COMPARE | `▶COMPARE val1 val2 ◆` | TRUE or FALSE |
| IF | `▶IF cond then else ◆` | selected branch (cond is TRUE/FALSE text) |
| FOREACH | `▶FOREACH ▲items ▲body ◆` | concatenated results |
| PROMPT | `▶PROMPT system user ◆` | LLM response |
| GENERATE | `▶GENERATE request ◆` | generated losp code (text) |
| SAY | `▶SAY text... ◆` | (outputs text) |
| READ | `▶READ [prompt] ◆` | user input line |
| PERSIST | `▶PERSIST name ◆` | (saves to storage) |
| LOAD | `▶LOAD name [default] ◆` | stored value |
| COUNT | `▶COUNT expr ◆` | number of lines |
| APPEND | `▶APPEND name content ◆` | (appends to expr) |
| EXTRACT | `▶EXTRACT label source ◆` | extracted value |
| UPPER | `▶UPPER text ◆` | uppercased text |
| LOWER | `▶LOWER text ◆` | lowercased text |
| TRIM | `▶TRIM text ◆` | trimmed text |
| TRUE | `▲TRUE` | "TRUE" |
| FALSE | `▲FALSE` | "FALSE" |
| EMPTY | `▲EMPTY` | "" |

## Patterns

### Store and Retrieve
```losp
▽X hello ◆
▲X
```
Output: `hello`

### Function with Placeholder
```losp
▼Greet □name Hello, ▲name! ◆
▶Greet Alice ◆
```
Output: `Hello, Alice!`

### Conditional
IF checks if condition equals "TRUE" (the text). Use COMPARE to produce TRUE/FALSE:
```losp
▶IF
▶COMPARE ▲X yes ◆
matched
not matched
◆
```
Or use TRUE/FALSE directly:
```losp
▶IF
TRUE
yes-branch
no-branch
◆
```

### Return Values
Expressions return their body's final text. No RETURN builtin exists.
```losp
▼Func ▶SAY side effect ◆ returned value ◆
▶Func ◆
```
Output (SAY): `side effect`
Result: `returned value`

### Conditional Execution (execute selected branch only)
```losp
▼DoA ▶SAY A ran ◆ result-A ◆
▼DoB ▶SAY B ran ◆ result-B ◆

▶▶IF TRUE
DoA
DoB
◆ ◆
```
IF returns text "DoA" or "DoB", outer `▶` executes only the selected one.

### FOREACH
```losp
▼ShowItem □item [▲item] ◆
▼Items
a
b
c
◆
▶FOREACH
▲Items
▲ShowItem
◆
```
Output: `[a]\n[b]\n[c]`

### APPEND (note the newlines!)
```losp
▽List first ◆
▶APPEND
List
second item
◆
```

### Executing Generated Code
GENERATE returns code as text, not executed. Splice into an expression body with `▷`:
```losp
▼_run ▷GENERATE Create code that outputs hello world ◆ ◆
▶_run ◆
```
`▷GENERATE` fires during `▼`'s body collection, splicing the generated code into the body. `▶_run ◆` then executes it.

### Immediate vs Deferred Inside Expressions
```losp
▽X first ◆
▼Template △X ◆
▽X second ◆
▶Template ◆
```
Output: `first` (△X resolved at definition time)

```losp
▽X first ◆
▼Template ▲X ◆
▽X second ◆
▶Template ◆
```
Output: `second` (▲X resolved at execution time)

## DO NOT

### Never use immediate ops to access placeholders
```losp
▼Broken □arg △arg ◆
```
**WRONG:** △arg fires at PARSE, before arg is bound. Result: empty.

```losp
▼Working □arg ▲arg ◆
```
**CORRECT:** ▲arg fires at EXECUTE, after arg is bound.

### Never forget terminators
Every operator needs its own `◆`. Count them:
```losp
▼Outer ▼Inner value ◆ ◆
```
Inner gets one ◆, Outer gets one ◆.

### Never put multiple text args on one line (THE #1 MISTAKE)
```losp
▶COMPARE a a ◆
```
**WRONG:** One argument "a a". COMPARE sees this as comparing "a a" to nothing!

```losp
▶COMPARE
a
a
◆
```
**CORRECT:** Two arguments on separate lines. Returns TRUE.

**IF with COMPARE - correct pattern:**
```losp
▶IF
▶COMPARE
a
a
◆
match
nomatch
◆
```

### Never expect ◯ to be preserved
```losp
▽Snap ◯△X ◆ ◆
▲Snap
▲Snap
```
First ▲Snap fires the △X and returns result. Second ▲Snap returns empty (body consumed).

### Never use ▷COMPARE inside functions for runtime checks
```losp
▼CheckMode
    ▶IF ▷COMPARE ▲Mode active ◆ yes no ◆
◆
```
**WRONG:** ▷COMPARE fires when CheckMode is DEFINED, not when executed.

```losp
▼CheckMode
    ▶IF ▶COMPARE ▲Mode active ◆ yes no ◆
◆
```
**CORRECT:** ▶COMPARE fires when CheckMode is EXECUTED.

## Retrieve vs Execute

- `▲Name` — returns body with deferred ops as text
- `▶Name ◆` — returns body with deferred ops evaluated

```losp
▼Expr ▶COMPARE hello hello ◆ ◆
▲Expr
```
Output: `▶COMPARE hello hello ◆` (text)

```losp
▼Expr ▶COMPARE hello hello ◆ ◆
▶Expr ◆
```
Output: `TRUE` (evaluated)

## Output Format

When writing losp code, output ONLY the raw losp code. Never wrap in markdown code fences. Never add explanations.
