# losp Code Generation Reference

losp is a streaming expression language using Unicode operators instead of parentheses. Operators consume tokens until the `◆` terminator.

## Operators

| Op | Name | Timing | Description |
|----|------|--------|-------------|
| `▼` | Store | Deferred | Define expression (body preserved for later execution) |
| `▽` | ImmStore | Immediate | Evaluate body now, store result |
| `▲` | Retrieve | Deferred | Retrieve stored expression (deferred ops returned as text) |
| `△` | ImmRetrieve | Immediate | Retrieve now, substitute into stream |
| `▶` | Execute | Deferred | Execute expression or builtin (deferred ops evaluated) |
| `▷` | ImmExec | Immediate | Execute now, substitute result |
| `□` | Placeholder | — | Declare argument slot |
| `◯` | Defer | — | Prevent immediate resolution (consumed on use) |
| `◆` | Terminator | — | End ONE operator's scope |

**ASCII shorthand** (for use in prompts where Unicode operators cannot appear):

| ASCII | Unicode | Meaning |
|-------|---------|---------|
| DEF | `▼` | Store (define expression) |
| IDEF | `▽` | ImmStore (evaluate & store) |
| GET | `▲` | Retrieve |
| IGET | `△` | ImmRetrieve |
| RUN | `▶` | Execute |
| IRUN | `▷` | ImmExec |
| ARG | `□` | Placeholder |
| END | `◆` | Terminator |

When a prompt describes a pattern using these ASCII names, output the corresponding Unicode operators.

Example prompt: "DEF _tpl ARG _a ARG _n ARG _m RUN SAY The GET _a GET _n is made of GET _m. END END"
Correct output:
```losp
▼_tpl □_a □_n □_m ▶SAY The ▲_a ▲_n is made of ▲_m. ◆ ◆
```

**Numbered arguments:** `ARG1` through `ARG9` in text become `▲_a` through `▲_i`. The expression gets one `□` per ARGn used.

Example: "Convert to losp expression _tpl: ARG1 called ARG2 about ARG3."
```losp
▼_tpl □_a □_b □_c ▶SAY ▲_a called ▲_b about ▲_c. ◆ ◆
```

**Timing rules:**
- **Immediate** (`△` `▷` `▽`): Fire as encountered during parsing. Result spliced into stream.
- **Deferred** (`▲` `▶` `▼`): Stored as-is. Resolved when executed.
- Inside `▼` bodies: immediate operators fire at DEFINITION time, deferred at EXECUTION time.
- Every `◆` terminates exactly ONE operator. Count your terminators.

## Expression Bodies

The body of an expression IS its output template. When executed, the body is evaluated and the result is returned. Every piece of the desired output — literal text, operators, placeholders — must appear in the body.

```losp
▼F □_a □_b ▲_a meets ▲_b ◆
```

When called with
`▶F
Alice
Bob
◆`, the body evaluates to: `Alice meets Bob`
- `▲_a` → Alice
- `meets` → literal text
- `▲_b` → Bob

If you omit `meets` or `▲_b` from the body, they will NOT appear in the output.

**All whitespace in a body is literal.** Spaces appear in the output exactly as written.
- `[▲x]` → `[value]` (no spaces)
- `[ ▲x ]` → `[ value ]` (spaces in output)

Do not add formatting spaces around operators inside bodies.

## THE ARGUMENT RULE

**Newlines separate text arguments. Spaces do NOT. Operators are natural boundaries.**

This is the most important rule in losp. Violations produce wrong code every time.

```losp
▶BUILTIN hello world ◆
```
This is ONE argument: the text `hello world`.

```losp
▶BUILTIN
hello
world
◆
```
This is TWO arguments: `hello` and `world`.

```losp
▶BUILTIN ▲A ▲B ◆
```
This is TWO arguments: result of `▲A` and result of `▲B`. Operators are already boundaries.

```losp
▶BUILTIN ▲A some text ◆
```
This is TWO arguments: result of `▲A`, then `some text`.

## Builtins

Builtin names are **ALL CAPS** and case-sensitive.

| Builtin | Signature | Returns |
|---------|-----------|---------|
| SAY | `▶SAY text... ◆` | (outputs text) |
| COMPARE | `▶COMPARE val1 val2 ◆` | `TRUE` or `FALSE` |
| IF | `▶IF condition then else ◆` | selected branch text |
| FOREACH | `▶FOREACH items body-name ◆` | concatenated results |
| PROMPT | `▶PROMPT system user ◆` | LLM response |
| GENERATE | `▶GENERATE request ◆` | generated losp code |
| READ | `▶READ [prompt] ◆` | user input line |
| PERSIST | `▶PERSIST name ◆` | (saves to DB) |
| LOAD | `▶LOAD name [default] ◆` | stored value |
| COUNT | `▶COUNT expr ◆` | number of lines |
| RANDOM | `▶RANDOM expr ◆` | one random line |
| APPEND | `▶APPEND name content ◆` | (appends to expression) |
| EXTRACT | `▶EXTRACT label source ◆` | extracted value |
| UPPER | `▶UPPER text ◆` | uppercased |
| LOWER | `▶LOWER text ◆` | lowercased |
| TRIM | `▶TRIM text ◆` | trimmed |
| SYSTEM | `▶SYSTEM setting [value] ◆` | current value or EMPTY |
| HISTORY | `▶HISTORY name ◆` | version names |
| CORPUS | `▶CORPUS name ◆` | handle |
| ADD | `▶ADD handle name ◆` | EMPTY |
| INDEX | `▶INDEX handle ◆` | EMPTY |
| SEARCH | `▶SEARCH handle query ◆` | matching names |
| EMBED | `▶EMBED handle ◆` | EMPTY |
| SIMILAR | `▶SIMILAR handle query ◆` | matching names |
| ASYNC | `▶ASYNC expr-name ◆` | handle |
| AWAIT | `▶AWAIT handle ◆` | result |
| CHECK | `▶CHECK handle ◆` | TRUE/FALSE |
| TIMER | `▶TIMER ms expr-name ◆` | handle |
| TICKS | `▶TICKS handle ◆` | ms remaining |
| SLEEP | `▶SLEEP ms ◆` | EMPTY |
| TRUE | `▲TRUE` | `TRUE` |
| FALSE | `▲FALSE` | `FALSE` |
| EMPTY | `▲EMPTY` | empty string |

## IF and COMPARE

IF takes exactly 3 arguments: condition, then-branch, else-branch.

COMPARE takes exactly 2 arguments and returns `TRUE` or `FALSE`.

**When COMPARE arguments are operators, they can be inline:**
```losp
▶COMPARE ▲X ▲Y ◆
```
Two arguments (operator boundaries).

**When COMPARE arguments are plain text, they MUST be on separate lines:**
```losp
▶COMPARE
hello
hello
◆
```
Returns: `TRUE`

**`▶COMPARE hello hello ◆` is WRONG** — that is ONE argument `hello hello` compared to nothing.

### IF with COMPARE — the correct patterns

**Pattern 1: COMPARE with operator args, IF branches on separate lines**
```losp
▶IF ▶COMPARE ▲X target ◆
matched
not matched
◆
```
Three args: `▶COMPARE` result (operator), `matched` (line), `not matched` (line).

**Pattern 2: Inside an expression with placeholder**
```losp
▼Check □_val ▶IF ▶COMPARE ▲_val target ◆
matched
not matched
◆ ◆
▶Check target ◆
```
Output: `matched`

**Pattern 3: COMPARE with two text literals**
```losp
▶IF
▶COMPARE
a
b
◆
yes
no
◆
```

**WRONG — branches on same line:**
```losp
▶IF ▶COMPARE ▲X target ◆ yes no ◆
```
`yes no` is ONE argument. IF sees condition + one arg, no else branch.

**WRONG — then and else on same line:**
```losp
▶IF ▶COMPARE ▲_val BAR ◆ correct incorrect ◆
```
`correct incorrect` is ONE argument. Must be:
```losp
▶IF ▶COMPARE ▲_val BAR ◆
correct
incorrect
◆
```

## Patterns

### Store and Retrieve
```losp
▽X hello ◆
▲X
```
Output: `hello`

### Expression with Placeholder
```losp
▼Greet □name Hello, ▲name! ◆
▶Greet Alice ◆
```
Output: `Hello, Alice!`

### Two Placeholders (text and operators interleave freely on one line)
```losp
▼Introduce □_who □_to ▲_who meets ▲_to ◆
▶Introduce
Alice
Bob
◆
```
Output: `Alice meets Bob`

### Expression with IF
Remember: IF branches MUST be on separate lines.
```losp
▼IsTarget □_val ▶IF ▶COMPARE ▲_val target ◆
yes
no
◆ ◆
▶IsTarget target ◆
```
Output: `yes`

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
ShowItem
◆
```
Output: `[a]\n[b]\n[c]`

### RANDOM (pick one from a list)
```losp
▼Colors
red
green
blue
◆
▶RANDOM ▲Colors ◆
```
Output: one of `red`, `green`, or `blue` (random each time).

`▶RANDOM` takes one argument: an expression whose lines are the items to pick from. Use `▲` to retrieve the list. Returns EMPTY if empty.

**Multiple RANDOM picks in one expression:**
```losp
▼Colors red
green
blue
◆
▼Animals cat
dog
bird
◆
▼Sentence The ▶RANDOM ▲Colors ◆ ▶RANDOM ▲Animals ◆ runs fast. ◆
▶Sentence ◆
```
Output: `The green cat runs fast.` (random each time). Each `▶RANDOM ▲list ◆` is a separate operator with its own `◆`.

### APPEND (arguments on separate lines)
```losp
▽List first ◆
▶APPEND
List
second item
◆
```

### Executing Generated Code
GENERATE returns code as text. To splice generated code into an expression body, use `▷` (ImmExec, immediate) — NOT `▶` (Execute, deferred). `▷GENERATE` fires at parse time and splices the result into the body. `▶GENERATE` would defer execution and NOT splice.

```losp
▼_run ▷GENERATE Create code that outputs hello world ◆ ◆
▶_run ◆
```

**With code after the splice:**
```losp
▼Maker ▷GENERATE define an expression named _val with body test ◆ ▶SAY ▲_val ◆ ◆
```
Three `◆`: one for `▷GENERATE`, one for `▶SAY`, one for `▼Maker`.

### Conditional Execution (only run selected branch)
```losp
▼DoA ▶SAY A ran ◆ result-A ◆
▼DoB ▶SAY B ran ◆ result-B ◆

▶▶IF TRUE
DoA
DoB
◆ ◆
```
IF returns `DoA` or `DoB`, outer `▶` executes only the selected one.

### Retrieve vs Execute
```losp
▼Expr ▶COMPARE hello hello ◆ ◆
▲Expr
```
Output: `▶COMPARE hello hello ◆` (text, unevaluated)

```losp
▶Expr ◆
```
Output: `TRUE` (evaluated)

## Terminator Counting

Count one `◆` per operator. Read inside-out:

```losp
▼Check □_val ▶IF ▶COMPARE ▲_val target ◆
yes
no
◆ ◆
```

- Terminator after `target`: closes `▶COMPARE`
- Terminator after `no`: closes `▶IF`
- Final terminator: closes `▼Check`

Each operator opened must have exactly one `◆`. Missing one leaves an operator unclosed. An extra `◆` closes the wrong thing.

## Critical Rules

1. **Placeholders use deferred retrieve.** `▼Func □arg ▲arg ◆` is correct. `▼Func □arg △arg ◆` is WRONG (△ fires before arg is bound).
2. **Every operator needs its own `◆`.** `▼Outer ▼Inner value ◆ ◆` — Inner gets one, Outer gets one.
3. **IF branches MUST be separate expressions.** Use newlines for text branches. Never `then else` on one line.
4. **Inside expression bodies, use deferred operators** (`▲` `▶` `▼`) for runtime behavior. Immediate operators fire at definition time.
5. **losp has no comments.** `#` is just text.

## Output Rules

Output ONLY the requested losp code. No markdown code fences. No explanation text.

**Do NOT add test or demo code.** If asked to write an expression named Foo, output ONLY the `▼Foo ... ◆` definition. Do not add `▶Foo ◆` calls, `▶SAY` demonstrations, sample data, or any other code beyond what was requested.
