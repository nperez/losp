# Building losp

losp builds as a standard Go binary and can also be cross-compiled to WebAssembly for use with a WASM host runtime.

## Native Build

```bash
go build -o losp ./cmd/losp/
```

This produces a fully-featured binary with:

- Interactive REPL (with Unicode operator shortcuts via Alt+key)
- LLM providers (Ollama, OpenRouter, Anthropic)
- SQLite persistence via `modernc.org/sqlite`
- File loading (`-f`)
- All CLI flags

### Requirements

- Go 1.24+
- No CGo dependencies (pure Go SQLite driver)

## WebAssembly Build

losp can be compiled to WebAssembly targeting the Go `js/wasm` platform:

```bash
GOOS=js GOARCH=wasm go build -o losp.wasm ./cmd/losp/
```

The resulting `losp.wasm` (~3.5 MB) is a standard Go WASM module that expects a host environment implementing:

- The Go JS runtime bridge (`gojs` namespace, as in `wasm_exec.js`)
- A `sqlite3` import namespace (19 functions) for database access
- Standard I/O via the `fs` global object

### WASM Limitations

The WASM build excludes features that depend on capabilities the host must provide:

| Feature | Native | WASM | Notes |
|---------|--------|------|-------|
| SQLite persistence | Yes | Yes | Host provides SQLite via import namespace |
| stdin/stdout I/O | Yes | Yes | Host bridges to OS streams |
| `-e` flag | Yes | Yes | |
| Pipe input via stdin | Yes | Yes | |
| `-f` flag (file loading) | Yes | No | Requires `fs.open` in host |
| Interactive REPL | Yes | No | No terminal support in WASM |
| LLM providers | Yes | No | See below |
| `net/http` | Yes | No | See below |

**LLM providers** and **`net/http`** are not available in the default WASM build. It is up to the WASM host to provide network access if needed. A host could implement HTTP support through additional import namespaces, making LLM providers available to the guest module.

**File system access** is not implemented in the default host. The `-f` flag and any file I/O operations will not work. Programs should be passed via `-e` or piped through stdin.

### How the SQLite Bridge Works

The WASM build replaces the native SQLite driver with a thin shim (`wasmsql/driver.go`) that uses `//go:wasmimport` to call 19 host functions in the `sqlite3` namespace (open, prepare, step, column_text, etc.). The host side (`sqlitehost.go`) translates these into `database/sql` calls against a real SQLite database. From losp's perspective, `database/sql` works identically in both builds.

### Build Tags

The native/WASM split is handled entirely through Go build tags (`//go:build js && wasm` / `//go:build !(js && wasm)`). The following files have platform-specific variants:

| Concern | Native file | WASM file |
|---------|------------|-----------|
| SQLite driver import | `internal/store/sqlite_driver_native.go` | `internal/store/sqlite_driver_wasm.go` |
| LLM provider config | `cmd/losp/provider_native.go` | `cmd/losp/provider_wasm.go` |
| REPL | `cmd/losp/repl.go` | `cmd/losp/repl_wasm.go` |
| Provider options | `pkg/losp/options_native.go` | _(none needed)_ |
| Provider packages | `internal/provider/*.go` | _(excluded by build tag)_ |

All shared logic (the interpreter, store, builtins, expression system) compiles identically for both targets.

## Running Tests

```bash
# Unit tests
go test ./...

# Conformance suite (native)
LOSP_BIN=./losp ./tests/conformance/run_tests.sh

# Conformance suite (WASM) - 117/123 pass
# 2 failures: -f flag tests (no filesystem)
# 4 failures: GENERATE tests (no LLM provider)
LOSP_BIN="./wasm-losp -wasm losp.wasm" ./tests/conformance/run_tests.sh
```

## Docker

See the `Dockerfile` for containerized builds, or use:

```bash
docker build -t losp .
docker run --rm -i losp -db /tmp/losp.db -e '▶SAY hello ◆'
```
