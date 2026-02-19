# losp

This is losp, a streaming programming language specifically designed for LLM structured metacognition. It is influenced by Lisp, FORTH, brainfuck, Perl but with a healthy dose of novelty.

## wtf?

What the fuck does all of that nonsense above even mean? Let's start with the problem that is being solved. LLMs do not have infinite context. In fact, the vast majority of the work of prompting an LLM is managing that context. And programmatically doing that with trad programming languages is very verbose and you spend a lot of time writing a bunch of code to glue together some kind of template system (or use a trad server-side rendering text templating from the early web days). What's worse is that if you want to do some kind of metaprompting or workflow that requires you to ouroborus the LLM output or make decisions based on it, you're now writing even more of that glue code to solidify that workflow. And it becomes rigid and difficult to mutate. And while you're probably using another LLM to manage this glue code, you're still dealing with the underlying abstraction to effect the orchestration.  

So what if we designed a very specific DSL for doing all of this? What would that language look like? It would need to allow you dynamically create templates and smoothly incorporate LLM calls as a fundamental component of the language. It would allow you to build larger abstractions on top of a small set of operators and semantics. It would allow you to nest LLM calls, flow control on their output, keep state, and express really complicated ideas and shape context naturally. 

Finally, it would be specified sufficiently to feed it to an LLM and to achieve one-shot complete competence in authoring in the language.

## metacognition?

Okay, yeah, sure, you're saying. You made a weird template language for prompting. Big deal. What is this metacognition bullshit? Well, I have a hypothesis that I am specifically testing with this language. Can we model internal mental processes sufficiently to really "embue" an LLM with "soul"? Like, could we basically take the idea from that Pixar movie, Inside-Out and make a very convincing automaton that would evolve? And where would we start? 

Introspection. Let me give an example. Imagine we're making a little LLM game about a robot and the robot has two attributes of state: battery, and physical condition. And this little robot is traveling through a post-apocalyptic landscape trying to survive. The player describes that actions it wants the robot to do in the face of encounters. If this robot encounters a group of raiders, for example, and it has full power and pristine physical condition, odds will be good it can escape or fight and survive. You can play this little game by hand with an LLM actually. And if you vary the attributes on the robot and ask it about the probability if it can successfully fight off the raiders or run away it will change its answer. So we can interrogate the state of the game with an LLM and derive useful outcomes from that. So all we need to do is get the state into the prompt. And of course, if something damages the robot or the robot's solar panels break, the state of their attributes will need to change. And you can directly interrogate the LLM about that too. And this is when it clicks: What is really playing pretend all about? Some agreed upon context and "rules" that you need to mentally process to stay consistent and continue the play. This is all acting is too. So can we make the LLMs play pretend in a structured way with state they control?

## How the fuck do you get a LLM to play pretend?

An engine that drives the loop of crystallized thoughts is enough, really. You start with modeling the basic flow and very heavily leverage the LLM to introspect, consider, synthesize, and mutate its own state. And once you realize that for just the little robot game above to have any fidelity to it, you need to make quite a few (nested) prompts and loops and give it a way to affect state, so tools or whatever. Doing this in a trad language is incredibly cumbersome. And this is where losp comes in.

## What is losp?

It is not quite lisp, and definitely an anagram of slop. Those of you in the future reading this after the AIs got good enough to fool even the most vigilant human, we went through a period of low quality, low effort AI content generation that everyone rightfully called slop. So I thought it was cute name. I originally started this idea from initial experiments from the early Llama days playing that little robot game by hand and it graduated to trying to get smol LLMs to write better prose with a clunky system of "perspective" system prompts and rotating them so that characters would be narrated consistently according to their character over time. 

losp is the answer to this problem of context management for prompting LLMs in a loosely structured way. 

## Why didn't you put the quick reference syntax at the top? ##

Because this isn't AI slop and I wanted you to understand a bit of the journey before I melt your brain with this weird ass syntax that I swear is going to make sense after I explain it. So here it is. Along with strings (newline terminated), this is it. 

### Operator Reference

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

Ask your closest LLM for keybindings to type these. 

So the first thing you'll notice (I hope) is that the triangles are little arrows: down, right, and up. And that there are two versions of them. Then there is the special placeholder operator (the empty square) and the defer operators (empty circle), and finally the diamond as the terminator for operator expressions. These symbols were the result of staring at all the various emojis and trying to pick stuff really out of distribution of the LLM. Why? Masochism. It literally made this so much harder because the LLMs kept struggling to produce examples in this made up language. But no, I wanted to break away from trad langs.

The symbols spoke to me in representing the operations necessary for templating. This is also where the brainfuck influence comes in. Brainfuck only has a handful of ops too and yet you can do somewhat real stuff with it. So down triangles were store. Up triangles load. Right triangles "play" an expression. Empty squares are placeholders. Defer with the empty circle is really the only odd one out. Originally, I chose these much more colorful ones:❗‼️🔚(<--former Terminator)⚛️🔒🔓 (<-- Defer start/end) 🔼⏫🔽⏬

But the LLMs and the myriad of interfaces really hate these emojis and it would fuck up all sorts of display.

### So what does a losp program look like?

```losp
▼ChatLoop
    ▶ChatLoopWithInput ▶READ You: ◆ ◆
◆

▼ChatLoopWithInput
    □_cli_input
    ▶APPEND History 
        User: ▲_cli_input 
    ◆
    ▼_cli_response ▶PROMPT
        You are a helpful assistant.
        ▲History
    ◆ ◆
    ▶SAY Assistant: ▶_cli_response ◆ ◆
    ▶APPEND History 
        Assistant: ▶_cli_response ◆ 
    ◆
    ▶PERSIST History ◆
    ▶ChatLoop ◆
◆

▶LOAD History ◆
▶ChatLoop ◆
```

Not so bad, huh? Its an s-exp-kinda language that is lisp-like, but also very not. Parenthenses drive me nuts. Really hard to read. I wanted something where the starts and stops on expressions were obvious and even if heavily nested, would be readable. 

### So how does this work?

Everything is an "expression". A string is really just another expression that doesn't do anything other than hold concepts. The operators and their arguments are expressions. And everything is parsed in stream. There is no lexical scoping. Everything accesses a global namespace (the dictionary in FORTH speak). Each stored expression has a name (that first word after the operator) and everything is compiled/backed by some kind of store (SQLite in the current implementation). 

### Parse-time vs. Execution-time

There are two different timings on the operators. You'll notice I've not talked about the immediate operators yet. You can think of these like lisp macros or C preprocessor in that at parse time they are immediately evaluated/executed and their results return back to the parse stream. Okay, yeah, that's cool I guess, I hear you say, but here is the novel bit. They are ephemeral. Once they are parsed and evaluated, they are forever gone. With this ability you can build latches, run-once expressions for "constants", metaprogramming and more.

Wait, wut? Yeah, ephemeral. Anytime the stream is parsed, immediate operators fire and replace that part of the stream with their output. And then those expressions are simply consumed, gone. Here is an example:

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

Let's walk through how this is interpreted. `▽X` is encountered, it is an immediate operator that is storing. And `first` is stored under the name `X`. The result of an immediate store operation is the EMPTY expression so nothing ends up in the parse stream, but `X` has been set at parse time. Next, `▽Snapshot` is encountered, and it does the same thing as above, then we get to the expressions to be stored and we notice an immediate load: `△X`! `X` is now returned to the parse stream verbatim which is just `first`. So now `Snapshot` holds `first`. We do one more immediate store to `X` with `second`.

So now `Snapshot` has `first` and `X` has `second`. Finally, we retrieve those expressions and return them verbatim.

But what if we don't want the immediate operators to fire at definition, but instead at execution? You're in luck, because I thought of that:

#### The Defer Operator

`◯` prevents parse-time resolution. It's analogous to Lisp's quote:

```losp
▽Snapshot ◯△X ◆ ◆   # Stores the Snapshot △X itself, not its value
▽X first ◆
▲Snapshot         # NOW △X resolves → `first` (△X fires and is consumed, body becomes "first")
▽X second ◆
▲Snapshot         # Returns `first` — △X was consumed on the previous retrieve, body is now the literal text "first"
```

Without `◯`, the `△X` would resolve at parse time and the expression would always return whatever X was when the line was parsed.

You can nest these! And they are consumed much like the immediate operators on each parse. On each parse (and parse-time happens when retrieving or executing), the immediate operators and defer operators are processed and consumed. 

### ... But why?!

Well, think about it a bit if you really wanted to model thinking. Thoughts are ephemeral and are consumed. And I wanted semantics for enabling setting a starting condition and then the system evolving from that point, never to return.

## What now?

Well, this is the end of the README. If this crazy ass idea resonates with you for building structured metacognition workflows, take a look at [PRIMER.md](PRIMER.md) for the language specification that was used to vibe code this whole damn thing. And also look at [CLAUDE.md](CLAUDE.md) for what additional instructions you need to feed the LLMs in order to get them to "understand" losp and write it cogently.

If you want to make a new implementation of losp, snag the language [PRIMER.md](PRIMER.md) and the conformance tests (stand alone losp programs run with a bash harness that uses whatever binary you want) and let your favorite robots get to work building.

Finally, to just play with the language a little bit and satisfy your curiosity follow the directions in [GETTING_STARTED.md](GETTING_STARTED.md)

