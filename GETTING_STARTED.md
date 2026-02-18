# Getting Started

This guide gets you from zero to typing in the losp REPL. For the philosophy and "why," see [README.md](README.md). For the full language spec, see [PRIMER.md](PRIMER.md).

## Prerequisites

**Native build:**
- Go 1.24+

**Docker:**
- Docker

**For LLM features** (PROMPT, GENERATE builtins):
- [Ollama](https://ollama.ai) running locally — optional for basic exploration

## Quick Start — Native

```bash
git clone https://github.com/nicholasgasior/losp.git
cd losp
go generate ./internal/stdlib/ && go build -o losp ./cmd/losp/
```

Launch the REPL:

```bash
./losp
```

You'll see:

```
losp REPL (Ctrl+D to exit)

Operators (use Alt+key):
  Alt+v → ▼ (store)       Alt+V → ▽ (imm store)
  Alt+^ → ▲ (retrieve)    Alt+A → △ (imm retrieve)
  Alt+> → ▶ (execute)     Alt+< → ▷ (imm execute)
  Alt+o → ◯ (defer)       Alt+* → ◆ (terminator)
  Alt+[ → □ (placeholder)
```

The Alt+key bindings let you type operators without copy-pasting Unicode. The REPL creates `losp.db` in the current directory for persistence.

## Quick Start — Docker

```bash
docker build -t losp .
```

Launch the REPL:

```bash
docker run --rm -it losp
```

If you want to use Ollama running on the host:

```bash
docker run --rm -it --network=host losp
```

The database is ephemeral inside the container. To persist it, mount a volume:

```bash
docker run --rm -it -v "$PWD/data:/app" losp
```

## Your First Session

Open the REPL (`./losp` or `docker run --rm -it losp`) and follow along. The `>` below is the REPL prompt — don't type it.

### 1. Output text

```
> ▶SAY Hello, World! ◆
Hello, World!
```

`▶SAY` is a builtin that prints its argument. `◆` terminates the operator.

### 2. Store and retrieve

```
> ▼Greeting Hello from losp! ◆
> ▲Greeting
Hello from losp!
```

`▼` stores an expression under a name. `▲` retrieves it.

### 3. Functions with placeholders

```
> ▼Greet □name Hello, ▲name! ◆
> ▶Greet Alice ◆
Hello, Alice!
```

`□name` declares a placeholder. When you execute `▶Greet Alice ◆`, "Alice" binds to `name`, and `▲name` resolves to it.

### 4. Multiline input

End a line with `\` to continue on the next line:

```
> ▼Greet \
    □name \
    Hello, ▲name! \
◆
> ▶Greet Bob ◆
Hello, Bob!
```

### 5. Conditionals

```
> ▶IF ▶COMPARE ▲Greeting Hello from losp! ◆ \
    matched \
    not-matched \
◆
matched
```

`▶COMPARE` tests equality and returns TRUE or FALSE. `▶IF` takes a condition then two branches — each on its own line (remember: newlines separate arguments, spaces don't).

## Running Files

```bash
./losp -f examples/hello.losp
```

```bash
./losp -f examples/composition.losp
```

The `examples/` directory has more to explore:

| File | What it shows |
|------|---------------|
| `hello.losp` | Minimal one-liner |
| `composition.losp` | Functions and placeholders |
| `conditional.losp` | IF/COMPARE branching |
| `loop.losp` | FOREACH iteration |
| `chatbot.losp` | Stateful LLM conversation (needs Ollama) |
| `wasteland.losp` | Full text adventure game (needs Ollama) |

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-e` | | Evaluate a losp string inline |
| `-f` | | Execute a losp file |
| `-db` | `losp.db` | SQLite database path |
| `-provider` | | LLM provider: `ollama` or `openrouter` |
| `-model` | | LLM model name |
| `-stream` | `false` | Enable streaming output |
| `-ollama` | `http://localhost:11434` | Ollama API URL |
| `-persist-mode` | `on_demand` | Persistence: `on_demand`, `always`, or `never` |
| `-compile` | `false` | Run program then persist all definitions |

Examples:

```bash
# Evaluate inline
./losp -e '▶SAY hello ◆'

# Run a file with a specific database
./losp -f app.losp -db myapp.db

# Use Ollama with a specific model
./losp -f chatbot.losp -provider ollama -model llama3.2
```

## Next Steps

- **[PRIMER.md](PRIMER.md)** — Full language specification with all operators, builtins, and semantics
- **[examples/chatbot.losp](examples/chatbot.losp)** — Stateful LLM conversation loop
- **[examples/wasteland.losp](examples/wasteland.losp)** — A post-apocalyptic text adventure built entirely in losp
