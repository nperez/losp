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

## Expressions

An expression is a name and a body of text. Names and bodies are what a losp program is made of. The body is a template, and executing the expression evaluates that template and gives its output: text stands for itself, operators give their results.

This expression defines `Foo` with a template of plain text:

```losp
▼Foo hello ◆
```

Executing it with `▶Foo ◆` gives `hello`, since a template of plain text and its output are the same.

This one defines `Foo` with a template that holds an operator:

```losp
▼Foo ▶UPPER hello ◆ ◆
```

Executing it gives `HELLO`, its output. Retrieving it with `▲Foo` gives `▶UPPER hello ◆`, the template itself, exactly as written. The template is kept as written until the moment it is executed.

`▶Foo ◆` asks for the output and `▲Foo` asks for the template, so a name is executed wherever its value is wanted.

That gives two ways to keep something under a name, and each has its own way back:

- `▼` keeps a template, and `▶Name ◆` runs it to get the value. Written into a line of text, `▶Name ◆` puts the value there.
- `APPEND` keeps the value itself, since arguments are evaluated before they arrive, and `▲Name` reads that value back.

The same definition is easier to read with each operator on a line of its own, its arguments indented beneath it, and its terminator lined up under it:

```losp
▼Foo
    ▶UPPER
        hello
    ◆
◆
```

That layout holds however deep the body goes, and every `◆` sits directly under the operator it closes:

```losp
▼Field
    ▶GRAB
        0
        ▶SPLIT ▲Row ◆
    ◆
◆
```

`▼Field` opens three operators and closes three. Executing it gives the first field of `Row`.

Assuming both `Foo` expressions were in the same context, each definition replaces the body `Foo` had before it, so `Foo` ends up holding only the last one.

**A template is kept, an argument is evaluated.** `▼Count ▶COUNT ▲Rows ◆ ◆` keeps the counting in the template, to be done each time `Count` is executed. Handing those same operators to something as an argument evaluates them first, so what arrives is the output, the number itself:

```losp
▶APPEND
    Count
    ▶COUNT ▲Rows ◆
◆
```

`▶APPEND` receives `5`. Every argument works this way, whatever receives it.

`APPEND` adds to the end of a body, so an expression that is appended to repeatedly accumulates everything it was given. Redefining an expression replaces it, and that is how such an accumulator is cleared: `▼Count ◆` puts it back to an empty body, and the appends that follow build it up again from there. An expression that gathers its value fresh on every call is cleared first, then appended to.

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

```losp
▶BUILTIN one ▲A two ◆
```
This is THREE arguments: `one`, the result of `▲A`, then `two`.

An expression body is a text template: executing it keeps the literal words exactly as written and interpolates each operator's result among them. A sentence that has values inside it is therefore written as its own expression, and executing that name fills a single argument position with the finished sentence.

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

## Calling Expressions

`▶` executes exactly ONE expression: `▶Name arguments ◆`. Immediately after the rune comes the NAME of what to execute — a builtin or a user-defined expression, both use the same form. Everything between the name and the terminator is the arguments.

```losp
▼Tidy □_text
    ▶TRIM ▲_text ◆
◆
▶Tidy hi ◆
```
Output: `hi` — one `▶` calls Tidy, and inside its body one `▶` calls TRIM.

The name slot can instead hold a single operator, for when the name has to be worked out at that moment; that operator runs first and its RESULT is the name that gets executed (see Dynamic Naming).

## Builtins

Builtin names are **ALL CAPS** and case-sensitive, and a builtin does its work when it is executed, as in `▶TRIM some text ◆`. An expression the program defines keeps the mixed-case name it was given and is executed the same way, as in `▶Tidy some text ◆`. Each name is written exactly as it is spelled where it is defined. The three values `TRUE`, `FALSE` and `EMPTY` are the exception — they hold a value rather than doing work, so they are retrieved as `▲TRUE`, `▲FALSE`, `▲EMPTY`, and take no terminator.

Single-argument builtins:

| Builtin | Signature | Returns |
|---------|-----------|---------|
| SAY | `▶SAY text ◆` | (prints to console) |
| GENERATE | `▶GENERATE request ◆` | generated losp code |
| DESCRIBE | `▶DESCRIBE name ◆` | the name on the first line, then a plain-language description |
| SURVEY | `▶SURVEY ◆` | documented expressions as `name: info` lines |
| RELEVANT | `▶RELEVANT request ◆` | names from SURVEY worth knowing about, one per line |
| BRIEF | `▶BRIEF names ◆` | those names expanded to `name: info` lines |
| PARSE | `▶PARSE name ◆` | TRUE if the named expression is well-formed |
| READ | `▶READ [prompt] ◆` | user input line |
| PERSIST | `▶PERSIST name ◆` | (saves to DB) |
| COUNT | `▶COUNT expr ◆` | number of lines |
| RANDOM | `▶RANDOM expr ◆` | one random line |
| UPPER | `▶UPPER text ◆` | uppercased |
| LOWER | `▶LOWER text ◆` | lowercased |
| TRIM | `▶TRIM text ◆` | trimmed |
| SPLIT | `▶SPLIT line ◆` | the pieces of that one line, split on SPLIT_CHAR, one to a line |
| GRAB | `▶GRAB index list ◆` | item at index, 0-based, -1 is last |
| FIRST | `▶FIRST list ◆` | first item |
| LAST | `▶LAST list ◆` | last item |
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
▼Tidy □_text
    ▶TRIM ▲_text ◆
◆
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

An argument position can hold an operator instead of a line of text. The operator is written in the position it fills and carries its own terminator:

```losp
▼Ask □_text ▶PROMPT
summarise this
▶TRIM ▲_text ◆
◆ ◆
```
`▶TRIM ▲_text ◆` fills the second argument position. Three operators are opened, so three terminators close them.

The words for a model are a text template like any other body, so they are kept under a name of their own and handed over by executing that name:

```losp
▼AskAbout □_row
    ▼_AskOwner
        ▶FIRST
            ▶SPLIT ▲_row ◆
        ◆
    ◆
    ▼_AskState
        ▶LAST
            ▶SPLIT ▲_row ◆
        ◆
    ◆
    ▼_AskWords
        the ticket held by ▶_AskOwner ◆ is ▶_AskState ◆ today
    ◆
    ▶PROMPT
        you answer in one sentence
        ▶_AskWords ◆
    ◆
◆
```

`▶AskAbout ana,open ◆` asks about `the ticket held by ana is open today`. `_AskWords` interpolates both values into one line, and `▶_AskWords ◆` is one operator, so PROMPT receives that whole line as its second argument.

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
▶SLICE
1
▲EMPTY
▲Rows
◆
```
A list is a piece of text of several lines, one item to each line, so several lines are already a list. SLICE returns the items of a list from the start position up to the end position, counting from 0. `▲EMPTY` as the end position runs to the last item, so this returns every line of `Rows` after the first.

```losp
▶AUTHOR
name
request
◆
```
Writes losp code for the request, defining it under the given name. Returns that code as text.

```losp
▶REVISE
name
instruction
◆
```
Returns revised losp code: rewrites the named expression to follow the plain-language instruction.

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

Operator arguments need no newlines — `▶COMPARE ▲X ▲Y ◆` is two arguments on one line, because operators are boundaries by themselves. An argument that is itself computed fills its position the same way. Give each operator a line of its own and indent what it contains; the terminators then line up with the operators they close:

```losp
▼Same □_line
    ▶COMPARE
        ▶FIRST
            ▶SPLIT ▲_line ◆
        ◆
        ▶LAST
            ▶SPLIT ▲_line ◆
        ◆
    ◆
◆
```

`▶COMPARE` gets two arguments, each one an operator that closes itself. Five operators are opened, so five terminators close them. An expression of your own takes its arguments the same way.

## IF and COMPARE

IF takes exactly 3 arguments: condition, then-branch, else-branch. COMPARE takes exactly 2 arguments and returns `TRUE` or `FALSE`.

**Pattern 1: each of IF's three arguments on its own line**
```losp
▶IF
    ▶COMPARE ▲X target ◆
    matched
    not matched
◆
```
Three args: `▶COMPARE` result (operator), `matched` (line), `not matched` (line). A branch of one word takes a line of its own exactly as a branch of several words does.

**Pattern 2: Inside an expression with placeholder**
```losp
▼Check □_val
    ▶IF
        ▶COMPARE ▲_val target ◆
        matched
        not matched
    ◆
◆
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

### Recording a result under a computed name

`▼` stores a body, so a retrieval returns that body verbatim. To record a computed *value*, use `APPEND`: it evaluates its content and stores the resulting text, its name argument may be an operator, and it creates the expression if it does not exist.

```losp
▼StoreNote □_sn_name □_sn_text
    ▶APPEND ▲_sn_name ▲_sn_text ◆
◆
▶StoreNote
Sunfish
a fish that drifts in the ocean
◆
▲Sunfish
```
Output: `a fish that drifts in the ocean`

Each name holds its own text, so `PERSIST`, `ADD`, `SEARCH` and `SIMILAR` all read the value. Use `▼` to define an expression to be run or recalled; use `APPEND` to record a value.

### Holding a Value for Later in the Same Body

`APPEND` records a computed value under a name, and `▲` reads it back further along the body. The name belongs to this expression alone, since one namespace holds them all.

The name is emptied first, because `APPEND` adds to whatever is already there and an expression called once per item of a list starts each call from empty.

```losp
▼File □_line
    ▼_FileTag ◆
    ▶APPEND
        _FileTag
        ▶GRAB
            0
            ▶SPLIT ▲_line ◆
        ◆
    ◆
    tag ▲_FileTag is filed
◆
```

`▶File a,b ◆` outputs `tag a is filed`. `▶APPEND` stores the first field under `_FileTag` and returns EMPTY, so what the body returns is the text that follows it. Four operators are opened, four terminators close them. Indenting an argument does not change it: the leading space belongs to the layout, not to the value.

### Clearing an Expression

`APPEND` adds to the end of a body, so appending twice gathers both pieces:

```losp
▼Notes ◆
▶APPEND
    Notes
    first line
◆
▶APPEND
    Notes
    second line
◆
```

`▲Notes` now gives `first line` and `second line`, one per line. `▼Notes ◆` at the top put an empty body in place of whatever `Notes` held before, so the gathering starts from nothing. Writing `▼Notes ◆` again empties it.

An expression that gathers a value afresh every time it is executed is cleared this way at the start, before the appends that fill it.

### Conditional Execution (only run selected branch)

A double execute runs whatever the inner operator names. It is worth seeing in two steps first. These branches are ordinary expressions, and `_Choice` picks between their names:

```losp
▼_Loud
    ▶UPPER ▲_word ◆
◆

▼_Quiet
    ▶LOWER ▲_word ◆
◆

▼_Choice □_word
    ▶IF
        ▶COMPARE ▲_word shout ◆
        _Loud
        _Quiet
    ◆
◆
```

Executing `_Choice` with `shout` gives the text `_Loud`, a name. Keeping that answer and executing it runs the branch:

```losp
▼Say □_word
    ▼_SayPick
        ▶_Choice ▲_word ◆
    ◆
    ▶▶_SayPick
    ◆
    ◆
◆
```

The double execute is two operators, so it takes two terminators, one under the other: the first closes the inner `▶_SayPick`, which gives `_Loud`, and the second closes the outer `▶`, which executes that name.

`▶Say shout ◆` gives `SHOUT` and `▶Say hush ◆` gives `hush`, and only the chosen branch runs. A double execute belongs where the inner answer is the name of an expression defined in the program, the way `_Choice` answers with `_Loud`. Where the inner answer is a value, a single `▶` gives that value. This holds when the value is itself somebody's name or a word that reads like one: what makes `_Choice` a double execute is that `_Loud` is an expression defined above it, not that the answer is called a name.

Dynamic naming is only for computing names. An ordinary call to a defined expression is a single `▶`, and so is each name executed to fill one of its arguments:

```losp
▼Main
    ▶Transform
        hello
    ◆
◆
```

```losp
▼Record □_row
    ▼_RecordTag
        ▶GRAB
            0
            ▶SPLIT ▲_row ◆
        ◆
    ◆
    ▶Transform
        ▶_RecordTag ◆
    ◆
◆
```

`▶Transform` opens one scope and takes one terminator. `▶_RecordTag ◆` opens and closes its own.

A body gives back what it produces, so a body that finishes by executing a name gives back the value that name holds:

```losp
▼Ticket □_owner
    ▼_TicketOwner
        ▶TRIM ▲_owner ◆
    ◆
    ▶_TicketOwner ◆
◆
```

`▶Ticket ana ◆` gives `ana`. `_TicketOwner` holds a person's name, which is a value like any other, so one `▶` gives that value back and one `◆` closes it. The closing line reads the same as it would in any other position.

## Retrieve vs Execute

Given these two definitions:

```losp
▼Row
    a,b,c
◆

▼_Field
    ▶GRAB
        0
        ▶SPLIT ▲Row ◆
    ◆
◆
```

Executing `_Field` gives its output:

```losp
▶_Field ◆
```

Output:

```losp
a
```

Retrieving `_Field` gives its template:

```losp
▲_Field
```

Output:

```losp
▶GRAB
    0
    ▶SPLIT ▲Row ◆
◆
```

A retrieved template stays as text wherever it lands. This body puts the template into the line:

```losp
▼LineA
    the field is ▲_Field
◆
```

Output of executing `LineA`:

```losp
the field is ▶GRAB
    0
    ▶SPLIT ▲Row ◆
◆
```

This body puts the value into the line, by executing the name where the value belongs:

```losp
▼LineB
    the field is ▶_Field ◆
◆
```

Output of executing `LineB`:

```losp
the field is a
```

Wherever a value belongs, in a line of text or in an argument to something else, the name that holds the work is executed.

## Terminator Counting

Count one `◆` per operator. Read inside-out:

```losp
▼Tidy □_text
    ▶TRIM ▲_text ◆
◆
```

- Terminator after `▲_text`: closes `▶TRIM`
- Final terminator: closes `▼Tidy`

Two operators opened, two terminators — the code ends with `◆ ◆`.

An expression that takes no arguments is executed the same way, and its terminator comes straight after its name:

```losp
▼Total
    ▶COUNT ▲Rows ◆
◆

▼Report
    there are ▶Total ◆ rows in all
◆
```

`▼Report` opens two operators and closes two: `▶Total ◆` is one of them, sitting in the middle of a line of text. The same execute fills an argument the same way:

```losp
▼Pairs
    ▶COMPARE
        ▶Total ◆
        ▶Total ◆
    ◆
◆
```

Three operators opened, three terminators. Each `▶Total ◆` carries its own.

One operator inside another adds one more of each. Giving each operator its own line puts every terminator directly under the operator it closes:

```losp
▼Neat □_text
    ▶UPPER
        ▶TRIM ▲_text ◆
    ◆
◆
```

- Terminator after `▲_text`: closes `▶TRIM`
- Next terminator: closes `▶UPPER`
- Final terminator: closes `▼Neat`

Three operators opened, three terminators. A definition whose body holds operators is written the same way however deep it goes:

```losp
▼Tag □_row
    ▼_TagField
        ▶GRAB
            0
            ▶SPLIT ▲_row ◆
        ◆
    ◆
    tag is ▶_TagField ◆
◆
```

Five operators opened, five terminators: `▶SPLIT`, then `▶GRAB`, then `▼_TagField`, then `▶_TagField`, then `▼Tag`. `□_row` takes none. `▶Tag a,b ◆` gives `tag is a`: `▼_TagField` keeps the template, and `▶_TagField ◆` runs it where the field belongs in the line.

```losp
▼Check □_val
    ▶IF
        ▶COMPARE ▲_val target ◆
        yes
        no
    ◆
◆
```

- Terminator after `target`: closes `▶COMPARE`
- Terminator after `no`: closes `▶IF`
- Final terminator: closes `▼Check`

The scope-opening operators `▼` `▽` `▶` `▷` `◯` each take exactly one `◆`. The name operators `▲` `△` `□` take just a name and no terminator. A missing `◆` leaves an operator unclosed; an extra `◆` inside an enclosing scope closes it too early. When every opened operator has its terminator, the code is complete.

**Tally the terminators before answering.** Count the scope-opening operators in the code — every `▼` `▽` `▶` `▷` `◯` — and count the `◆`. The two counts match exactly. The `▲` `△` `□` operators add nothing to either count. The final `◆` closes the outermost operator, and the code ends there.

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
▼Shrink □_t
    ▶LOWER ▲_t ◆
◆
▼Report
    ▶Shrink LOUD ◆
◆
▶Report ◆
```
Output: `loud` — Shrink ends with `◆ ◆` (one for `▶LOWER`, one for `▼Shrink`), and Report ends with `◆ ◆` (one for `▶Shrink`, one for `▼Report`).

A larger program is laid out the same way: one operator to a line, its arguments indented beneath it, and its terminator on a line of its own lining up with it. Here `Digest` walks a list of `code,label` lines, `Entry` handles one line, `Blurb` asks the model, and `Store` records a value under a name and gives that name back:

```losp
▼Blurb □_topic
    ▶PROMPT
        write one short line about this
        ▲_topic
    ◆
◆

▼Store □_key □_text
    ▶APPEND ▲_key ▲_text ◆
    ▲_key
◆

▼Entry □_row
    ▼_EntryCode
        ▶GRAB
            0
            ▶SPLIT ▲_row ◆
        ◆
    ◆
    ▼_EntryTopic
        the row for ▶_EntryCode ◆
    ◆
    ▼_EntryText
        ▶Blurb
            ▶_EntryTopic ◆
        ◆
    ◆
    ▶Store
        ▶_EntryCode ◆
        ▶_EntryText ◆
    ◆
◆

▼Clean
    ▶TRIM ▲Rows ◆
◆

▼Recent
    ▶SLICE
        1
        ▲EMPTY
        ▶Clean ◆
    ◆
◆

▼Digest
    ▶FOREACH
        ▶Recent ◆
        Entry
    ◆
◆
```

`Entry` keeps the work for each part of the row under a name of its own, then executes those names where their values belong. `_EntryTopic` builds a line of text with `▶_EntryCode ◆` inside it, and `Blurb` receives that whole line as one argument, because `▶_EntryTopic ◆` is one operator and one argument. Every operator that opens a scope has one terminator directly below it at its own indentation, so each `◆` matches the operator it closes, and each definition carries all of its own terminators before the next begins.

The two sizes of text are worth seeing side by side here. `Rows` is several lines, so it is already a list. One row is a single line, so `Entry` uses `▶SPLIT` to reach the fields inside that line.

The two ways of reaching a list sit side by side as well. `Rows` holds its lines as written, so `▲Rows` puts them into `Clean`. `Clean` and `Recent` work their lines out, so `▶Clean ◆` and `▶Recent ◆` pass them on. A name that holds work is executed in the list position exactly as it is anywhere else.

`Clean` gives back several lines, and several lines are a list, so `Recent` holds a list of its own without splitting anything. `▶SPLIT` appears only inside `Entry`, where one line is opened up into its fields.

An expression takes computed arguments the same way a builtin does. Given an expression named `Join` that takes two arguments:

```losp
▼Pair □_line
    ▶Join
        ▶FIRST
            ▶SPLIT ▲_line ◆
        ◆
        ▶LAST
            ▶SPLIT ▲_line ◆
        ◆
    ◆
◆
```

Each argument is an operator that closes itself, so `▶Join` receives the first field and the last field. Five operators are opened, five terminators close them.

### Expression with IF
```losp
▼IsTarget □_val
    ▶IF
        ▶COMPARE ▲_val target ◆
        yes
        no
    ◆
◆
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

**An expression that makes a request** takes the varying part as a placeholder and keeps its headers in a definition of their own, because `▶HTTP` reads headers and data from retrievals:

```losp
▼_Key X-Auth: token123 ◆
▼Send □_note ▶HTTP PUT
http://host/notes
▲_Key ▲_note ◆ ◆
```
Three `◆`: one for `▼_Key`, one for `▶HTTP`, one for `▼Send`. The method and uri are lines of `▶HTTP`'s arguments; the headers and data positions are filled by retrievals, and `▲EMPTY` fills a position that carries nothing.

### Names and Scope

Every name lives in one namespace shared by the whole program. There is no inner or outer: a definition written anywhere is reachable everywhere by any expression, from the moment it is made. A placeholder is a name in that same namespace, bound when its expression is called, so one expression can read another's placeholder:

```losp
▼_Quiet
    ▶LOWER ▲_word ◆
◆
▼_Loud
    ▶UPPER ▲_word ◆
◆
▼Say □_word
    ▶▶IF
        ▶COMPARE ▲_word shout ◆
        _Loud
        _Quiet
    ◆
    ◆
◆
```

The two terminators before the last one close `▶IF` and then the outer `▶` that executes its answer.

`▶Say shout ◆` gives `SHOUT`, and `▶Say hush ◆` gives `hush`. `_Loud` and `_Quiet` read `▲_word`, the placeholder of `Say`, because one namespace holds every name.

Because one namespace holds them all, each expression gives its placeholders names of its own — `_body` here — so that one call's arguments stay clear of another's.

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

**GENERATE's argument is a plain-text request.** A nested GENERATE carries its own request as words, and that inner call turns those words into code when it fires:
```losp
▼Meta
    ▷GENERATE define an expression named _msg containing hello world ◆
    ▶SAY ▲_msg ◆
◆
```
Everything between `▷GENERATE` and its terminator is the request the inner call receives.

AUTHOR and REVISE also return code as text and splice the same way as GENERATE above.

## Critical Rules

1. **Placeholders are read with deferred retrieve.** `▼Func □arg ▲arg ◆` — `▲arg` resolves after the argument is bound.
2. **Every operator gets its own `◆`.** `▼Outer ▼Inner value ◆ ◆` — Inner gets one, Outer gets one.
3. **IF's text branches each go on their own line.**
4. **Inside expression bodies, deferred operators** (`▲` `▶` `▼`) give runtime behavior; immediate operators fire at definition time.
5. **losp has no comments.** `#` is just text.
6. **Every name that is retrieved or executed is defined.** A `▲Name` or `▶Name ◆` refers to a name the request supplies, a placeholder of the enclosing expression, a builtin, or a definition written in the same answer.
7. **Values stand where they are used.** A body computes each value at the point it appears, so a value wanted in two places is written in both.
8. **The answer is losp code and nothing else.** Plain text belongs in bodies where the request asks for it; symbols, annotations and example results are not part of the code.

# Part 2 — Interpreting Requests

How to turn a request into losp code.

## ASCII Shorthand

A code task can spell out the shape of losp code using ASCII names in place of the operator runes:

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

These names belong to the code task. Each one names exactly one operator, and the code is written with the operators themselves, so every ASCII name in a code task becomes its rune in the code. Text that the code carries, such as the words it hands to a builtin, is part of the code and is written the same way.

Example code task: "DEF _tpl ARG _a ARG _n ARG _m The GET _a GET _n is made of GET _m. END"
Correct output:
```losp
▼_tpl □_a □_n □_m
    The ▲_a ▲_n is made of ▲_m.
◆
```
The words keep their places, and each ASCII name becomes its operator where it stood.

`RUN` works the same for user-defined expressions as for builtins, and reads the same way inside a line of text.

Example code task: "DEF Main RUN Transform hello END END"
Correct output:
```losp
▼Main
    ▶Transform hello ◆
◆
```

Example code task: "DEF Report The tally is RUN Total END for today. END"
Correct output:
```losp
▼Report
    The tally is ▶Total ◆ for today.
◆
```
`▶Total ◆` stands in the middle of the sentence, with its own terminator, and the words on either side of it are kept.

**Numbered arguments:** `ARG1` through `ARG9` in text become `▲_a` through `▲_i`. The expression gets one `□` per ARGn used.

Example code task: "Convert to losp expression _tpl: ARG1 called ARG2 about ARG3."
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
▼Check □_val
    ▶IF
        ▶COMPARE ▲_val YES ◆
        yep
        nope
    ◆
◆
```
`yep` and `nope` are IF's two branch arguments: one per line.

Example request: "an expression named Rank that takes one argument and returns gold when the argument equals first, otherwise silver"
Correct output:
```losp
▼Rank □_r
    ▶IF
        ▶COMPARE ▲_r first ◆
        gold
        silver
    ◆
◆
```
A request that says one word when something holds and another word otherwise is naming two separate arguments, so `gold` and `silver` each take a line of their own, however short they are.

Example request: "an expression named Notify that posts the text weekly summary ready to http://host/notify"
Correct output:
```losp
▼Notify
    ▶HTTPPOST
        http://host/notify
        weekly summary ready
    ◆
◆
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
