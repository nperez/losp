# losp Code Generation Reference

losp is a streaming expression language using Unicode operators instead of parentheses. Operators consume tokens until the `◆` terminator.

This reference has two parts: **Part 1 — The Language** (operators, builtins, patterns) and **Part 2 — Interpreting Requests** (how to turn a request into losp code).

# Part 1 — The Language

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

**Timing rules:**
- **Immediate** (`△` `▷` `▽`): Fire as encountered during parsing. Result spliced into stream.
- **Deferred** (`▲` `▶` `▼`): Stored as-is. Resolved when executed.
- Inside `▼` bodies: immediate operators fire at DEFINITION time, deferred at EXECUTION time.
- Every `◆` terminates exactly ONE operator. Count your terminators.

## Calling Expressions

`▶` executes exactly ONE expression: `▶Name arguments ◆`. Immediately after the rune comes the NAME of what to execute — a builtin or a user-defined expression, both use the same form. Everything between the name and the terminator is the arguments.

```losp
▼Tidy □_text ▶TRIM ▲_text ◆ ◆
▶Tidy hi ◆
```
Output: `hi` — one `▶` calls Tidy, and inside its body one `▶` calls TRIM.

The name slot can instead hold a single operator; that operator runs first and its RESULT is the name that gets executed (see Dynamic Naming).

## THE ARGUMENT RULE

**Newlines separate text arguments. Spaces do NOT. Operators are natural boundaries.**

This is the most important rule in losp.

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

## Builtins

Builtin names are **ALL CAPS** and case-sensitive.

Single-argument builtins:

| Builtin | Signature | Returns |
|---------|-----------|---------|
| SAY | `▶SAY text ◆` | (prints to console) |
| GENERATE | `▶GENERATE request ◆` | generated losp code |
| AUTHOR | `▶AUTHOR request ◆` | generated losp code, using the namespace as context |
| DESCRIBE | `▶DESCRIBE code ◆` | plain-language description of losp code |
| SURVEY | `▶SURVEY ◆` | documented expressions as `name: info` lines |
| READ | `▶READ [prompt] ◆` | user input line |
| PERSIST | `▶PERSIST name ◆` | (saves to DB) |
| COUNT | `▶COUNT expr ◆` | number of lines |
| RANDOM | `▶RANDOM expr ◆` | one random line |
| UPPER | `▶UPPER text ◆` | uppercased |
| LOWER | `▶LOWER text ◆` | lowercased |
| TRIM | `▶TRIM text ◆` | trimmed |
| HISTORY | `▶HISTORY name ◆` | version names |
| CORPUS | `▶CORPUS name ◆` | handle |
| INDEX | `▶INDEX handle ◆` | EMPTY |
| EMBED | `▶EMBED handle ◆` | EMPTY |
| ASYNC | `▶ASYNC expr-name ◆` | handle |
| AWAIT | `▶AWAIT handle ◆` | result |
| CHECK | `▶CHECK handle ◆` | TRUE/FALSE |
| TICKS | `▶TICKS handle ◆` | ms remaining |
| SLEEP | `▶SLEEP ms ◆` | EMPTY |
| NOW | `▶NOW [format] ◆` | current time (ISO-8601, or strftime-style format like `%Y-%m-%d`) |
| HTTPGET | `▶HTTPGET uri ◆` | HTTP GET response body |
| HTTPDELETE | `▶HTTPDELETE uri ◆` | HTTP DELETE response body |
| TRUE | `▲TRUE` | `TRUE` |
| FALSE | `▲FALSE` | `FALSE` |
| EMPTY | `▲EMPTY` | empty string |

Wrapping a single-argument builtin in a definition takes two terminators — the builtin's, then the definition's:

```losp
▼Tidy □_text ▶TRIM ▲_text ◆ ◆
```

Multi-argument builtins take each plain-text argument on its own line. These signatures are literal:

```losp
▶COMPARE
a
b
◆
```
Returns `TRUE` or `FALSE`.

```losp
▶IF
condition
then
else
◆
```
Returns the selected branch text.

```losp
▶FOREACH
items
body-name
◆
```
Returns concatenated results.

```losp
▶PROMPT
system
user
◆
```
Returns the LLM response.

```losp
▶LOAD
name
default
◆
```
Returns the stored value (default optional).

```losp
▶APPEND
name
content
◆
```
Appends content to the named expression.

```losp
▶EXTRACT
label
source
◆
```
Returns the extracted value.

```losp
▶REVISE
code
instruction
◆
```
Returns revised losp code: rewrites the first argument (a retrieved expression) to follow the plain-language instruction.

```losp
▶SYSTEM
setting
value
◆
```
Sets a setting; with only `setting`, returns its current value.

```losp
▶ADD
handle
name
◆
```
Adds to a corpus. Returns EMPTY.

```losp
▶SEARCH
handle
query
◆
```
Returns matching names.

```losp
▶SIMILAR
handle
query
◆
```
Returns matching names (semantic).

```losp
▶TIMER
ms
expr-name
◆
```
Returns a handle.

```losp
▶HTTPPOST
http://host/items
the request body
◆
```
HTTP POST of the second argument to the uri; returns the response body. HTTPPUT works the same way.

```losp
▶HTTP
method
uri
▲HeadersName ▲DataName ◆
```
Full HTTP request; returns the response body. headers and data are optional. They MUST be retrieved (`▲Name`) from stored expressions, or `▲EMPTY` for an unused position — NEVER literal text at the call site, because newlines would split them into separate arguments. Headers are one `Key: Value` per line:

```losp
▼_Headers X-Auth: token123
Content-Type: application/json ◆
▼_Data {"k": "v"} ◆
▶HTTP POST
http://host/path
▲_Headers ▲_Data ◆
```

Operator arguments need no newlines — `▶COMPARE ▲X ▲Y ◆` is two arguments on one line, because operators are boundaries by themselves.

## IF and COMPARE

IF takes exactly 3 arguments: condition, then-branch, else-branch. COMPARE takes exactly 2 arguments and returns `TRUE` or `FALSE`.

**Pattern 1: COMPARE with operator args inline, IF text branches on separate lines**
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
Output: `no`

## Dynamic Naming (computed names)

Store and execute operators support dynamic naming: the name slot can be an operator instead of literal text. The operator runs first and its RESULT becomes the name.

```losp
▼FieldName X ◆
▼▲FieldName hello ◆
▲X
```
Output: `hello` — `▲FieldName` resolves to `X`, so the store writes to `X`.

```losp
▼ExecDynamic □name ▶▲name ◆ ◆
▶ExecDynamic MyExpression ◆
```
`▶▲name ◆` executes whatever expression the bound placeholder names.

### Conditional Execution (only run selected branch)

Dynamic naming with IF avoids eager evaluation of both branches — pass expression NAMES as the branches:
```losp
▼DoA result-A ◆
▼DoB result-B ◆

▶▶IF TRUE
DoA
DoB
◆ ◆
```
IF returns the selected branch text (`DoA` or `DoB`); that result is the name the outer `▶` executes. Only the selected expression runs.

Dynamic naming is only for computing names. An ordinary call to a defined expression is a single `▶`:
```losp
▼Main ▶Transform hello ◆ ◆
```

## Retrieve vs Execute

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
▼Tidy □_text ▶TRIM ▲_text ◆ ◆
```

- Terminator after `▲_text`: closes `▶TRIM`
- Final terminator: closes `▼Tidy`

Two operators opened, two terminators — the code ends with `◆ ◆`.

```losp
▼Check □_val ▶IF ▶COMPARE ▲_val target ◆
yes
no
◆ ◆
```

- Terminator after `target`: closes `▶COMPARE`
- Terminator after `no`: closes `▶IF`
- Final terminator: closes `▼Check`

The scope-opening operators `▼` `▽` `▶` `▷` `◯` each take exactly one `◆`. The name operators `▲` `△` `□` take just a name and no terminator. A missing `◆` leaves an operator unclosed; an extra `◆` closes an enclosing scope too early. Trailing `◆` is an error. When every opened operator has its terminator, the code is complete.

Top-level statements each close themselves — there is no outer block to close:
```losp
▽List first ◆
▶APPEND
List
second item
◆
```
Two operators, two `◆`: `▽List` gets the first, `▶APPEND` gets the second. The code is complete after `▶APPEND`'s terminator.

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

Placeholders are read with deferred retrieve (`▲name`), which resolves after the argument is bound.

### Two Placeholders (text and operators interleave freely on one line)
```losp
▼Introduce □_who □_to ▲_who meets ▲_to ◆
▶Introduce
Alice
Bob
◆
```
Output: `Alice meets Bob`

### A Program of Several Definitions

Each definition carries all of its own terminators before the next begins:

```losp
▼Shrink □_t ▶LOWER ▲_t ◆ ◆
▼Report ▶Shrink LOUD ◆ ◆
▶Report ◆
```
Output: `loud` — Shrink ends with `◆ ◆` (one for `▶LOWER`, one for `▼Shrink`), and Report ends with `◆ ◆` (one for `▶Shrink`, one for `▼Report`).

### Expression with IF
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

### HTTP Requests

```losp
▶SAY ▶HTTPGET http://host/status ◆ ◆
```
Output: the response body.

```losp
▶SAY ▶HTTPPOST
http://host/echo
the request body text
◆ ◆
```
Output: the response body. Two `◆`: one for `▶HTTPPOST`, one for `▶SAY`.

### Executing Generated Code

GENERATE returns code as text. To splice generated code into an expression body, use `▷` (ImmExec, immediate): `▷GENERATE` fires at parse time and splices the result into the body.

```losp
▼_run ▷GENERATE Create code that outputs hello world ◆ ◆
▶_run ◆
```

**With code after the splice:**
```losp
▼Maker ▷GENERATE define an expression named _val with body test ◆ ▲_val ◆
```
Two `◆`: one for `▷GENERATE`, one for `▼Maker`. `▲_val` takes none.

**GENERATE's argument is a plain-text request, not code.** If the request itself contains ASCII operator names (a nested GENERATE request), leave them as plain text — the nested GENERATE converts them at runtime:
```losp
▼Meta ▷GENERATE DEF _msg hello world END ◆ ▶SAY ▲_msg ◆ ◆
```
Here `DEF _msg hello world END` stays as text inside the request.

AUTHOR and REVISE also return code as text and splice the same way as GENERATE above.

## Critical Rules

1. **Placeholders are read with deferred retrieve.** `▼Func □arg ▲arg ◆` — `▲arg` resolves after the argument is bound.
2. **Every operator gets its own `◆`.** `▼Outer ▼Inner value ◆ ◆` — Inner gets one, Outer gets one.
3. **IF's text branches each go on their own line.**
4. **Inside expression bodies, deferred operators** (`▲` `▶` `▼`) give runtime behavior; immediate operators fire at definition time.
5. **losp has no comments.** `#` is just text.

# Part 2 — Interpreting Requests

How to turn a request into losp code.

## ASCII Shorthand

Requests may describe the shape of losp code using ASCII names instead of Unicode operators:

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

When a request describes a pattern using these ASCII names, output the corresponding Unicode operators. Each ASCII name maps to exactly ONE operator rune.

Example request: "DEF _tpl ARG _a ARG _n ARG _m The GET _a GET _n is made of GET _m. END"
Correct output:
```losp
▼_tpl □_a □_n □_m The ▲_a ▲_n is made of ▲_m. ◆
```

`RUN` works the same for user-defined expressions as for builtins.

Example request: "DEF Main RUN Transform hello END END"
Correct output:
```losp
▼Main ▶Transform hello ◆ ◆
```

**Numbered arguments:** `ARG1` through `ARG9` in text become `▲_a` through `▲_i`. The expression gets one `□` per ARGn used.

Example request: "Convert to losp expression _tpl: ARG1 called ARG2 about ARG3."
```losp
▼_tpl □_a □_b □_c ▲_a called ▲_b about ▲_c. ◆
```

## Names

**Copy names EXACTLY as written, including case.** `DEF FOO` produces `▼FOO`. `DEF Check` produces `▼Check`. Keep the case of every requested name.

## Multi-Argument Phrases

**When a builtin takes multiple arguments and the request lists them as adjacent words, each argument goes on its OWN line.** IF's then/else branches, COMPARE's two values, and APPEND's name/content are separate arguments — newlines separate text arguments in losp.

Example request: "DEF Check ARG _val RUN IF RUN COMPARE GET _val YES END yep nope END END"
Correct output:
```losp
▼Check □_val ▶IF ▶COMPARE ▲_val YES ◆
yep
nope
◆ ◆
```
`yep` and `nope` are IF's two branch arguments: one per line.

Example request: "an expression named Notify that posts the text weekly summary ready to http://host/notify"
Correct output:
```losp
▼Notify ▶HTTPPOST
http://host/notify
weekly summary ready
◆ ◆
```
The uri and the data are HTTPPOST's two arguments: one per line.

## Spacing and Quoting

**Preserve the request's spacing exactly.** Body whitespace is literal output: characters adjacent to `GET` in the request stay adjacent to `▲` in the code.

Example request: "DEF Wrap ARG _item [GET _item] END"
Correct output:
```losp
▼Wrap □_item [▲_item] ◆
```
The brackets hug `▲_item` because they hug `GET _item` in the request.

**Quoted text in requests is literal.** The quotes are delimiters, not content: reproduce exactly the characters between them — no quotes, no added spacing. A request to join two arguments with ' greets ' produces the body `▲_a greets ▲_b`; the spaces around greets come from inside the quotes.

The same holds when quoted text meets an operator: the operator follows the quoted text directly, so the quoted characters are the only characters between them.

**A quoted phrase that ends with a space already carries that one space — it IS the whole separator. Write the operator immediately after it, and add none of your own.** Concatenating a trailing-space quote with an argument yields exactly one space between them.

Example request: "an expression named Label that returns the concatenation of 'Page ' and the argument"
Correct output:
```losp
▼Label □_n Page ▲_n ◆
```
`▶Label 7 ◆` outputs `Page 7`. The quote `'Page '` supplies the single trailing space, and `▲_n` follows it directly.

Example request: "an expression named Price that returns the concatenation of 'Cost: ' and the argument"
Correct output:
```losp
▼Price □_x Cost: ▲_x ◆
```
`▶Price 5 ◆` outputs `Cost: 5`. The single space before `▲_x` is the trailing space inside the quotes.

## Return vs Print

**An expression RETURNS its body's result.** When a request says the expression returns, builds, or concatenates a value, write the text and operators directly in the body — the body IS the return value. SAY prints to the console; use it only when the request asks to print or output.

## Output Rules

Output ONLY the requested losp code. No markdown code fences. No explanation text.

**The code is complete when every opened operator has its `◆`.** A definition ends with its own terminator, after the terminators of everything inside it.

**Output only what was requested.** If asked to write an expression named Foo, output ONLY the `▼Foo ... ◆` definition — no calls, no demonstrations, no sample data.
