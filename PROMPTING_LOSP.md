# losp Prompting Guide

How to prompt for correct losp code generation.

## Core Principles

1. **Give exact templates, not descriptions.** Show the code pattern to copy.
2. **Show wrong patterns explicitly.** Mark them as WRONG so the model avoids them.
3. **Use neutral language.** Avoid "Hello", "Greet", "Thanks" - they trigger formatting.
4. **End with output format.** Last line: "Output ONLY losp code. No markdown."
5. **Build piecemeal.** Generate in phases, test each part, include prior code in context.

## Prompt Template

```
[PRIMER_COMPACT.md content]

---

[Task description with exact patterns]

Output ONLY losp code. No markdown. No explanation.
```

## Critical Patterns

### COMPARE with literal text
```
WRONG: ▶COMPARE a b ◆

CORRECT:
▶COMPARE
a
b
◆
```

### IF with COMPARE
```
▶IF
▶COMPARE
▲Value
expected
◆
then-result
else-result
◆
```

### OR logic (check multiple conditions)
```
▼IsEitherBad
    ▶IF
    ▶COMPARE
    ▲First
    bad
    ◆
    TRUE
    ▶IF
    ▶COMPARE
    ▲Second
    bad
    ◆
    TRUE
    FALSE
    ◆
    ◆
◆
```

### Capture and use PROMPT result
```
▼ProcessAction
    □_input
    ▼_result ▶PROMPT
You are a narrator.
▲_input
    ◆ ◆
    ▶SAY ▶_result ◆ ◆
◆
```

### Capture READ input
```
▼HandleInput □_in ▶SAY Got: ▲_in ◆ ◆
▶HandleInput ▶READ > ◆ ◆
```

### Dynamic execution (execute only selected branch)
```
▼DoA ▶SAY A ◆ result-A ◆
▼DoB ▶SAY B ◆ result-B ◆

▶▶IF ▶COMPARE ▲Mode A ◆
DoA
DoB
◆ ◆
```

## Piecemeal Generation

For complex applications, generate in phases:

**Phase 1: State**
```
Write losp code for state initialization:
1. App_Init: Set App_Day to 1, App_Status to active
2. App_ShowStatus: Display state with SAY
```

**Phase 2: Logic (include Phase 1)**
```
Existing code:
▼App_Init ▽App_Day 1 ◆ ▽App_Status active ◆ ◆
▼App_ShowStatus ▶SAY Day: ▲App_Day Status: ▲App_Status ◆ ◆

Now add:
1. App_Check: Return TRUE if App_Status equals done
2. App_Loop: Show status, get input, check, loop or exit
```

**Phase 3: LLM (include pattern)**
```
EXACT pattern to copy:
▼Example
    ▼_result ▶PROMPT
    system message
    user message
    ◆ ◆
    ▶SAY ▶_result ◆ ◆
◆

Write App_Process using this pattern...
```

## Common Fixes

| Problem | Fix |
|---------|-----|
| `▶COMPARE a b ◆` on one line | Use newlines between args |
| `Day: [▲App_Day]` with brackets | Re-prompt: "no brackets around operators" |
| `▶SAY ▶PROMPT ...` without storing | Provide "store then execute" pattern |
| Nested `▼Outer ▼Inner ...` | Add: "use flat structure, all functions at top level" |
| Markdown ``` fences | End prompt with: "No markdown" |
| "Hello, World!" with punctuation | Use neutral: "Value: X" not "Hello X" |

## Task Decomposition

Break large tasks into focused pieces:

1. **State**: Variables, initialization, display
2. **Flow**: Loops, conditionals, recursion
3. **I/O**: READ input, SAY output
4. **LLM**: PROMPT integration with stored results
5. **Persistence**: LOAD/PERSIST for state

Generate each piece, test it, then include in context for the next piece.
