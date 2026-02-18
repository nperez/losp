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

Or using [gigwasm](https://nickandperla.net/gigwasm) to compile programmatically:

```go
wasmBytes, err := gigwasm.CompileGo("./cmd/losp")
```

The resulting `losp.wasm` (~11.5 MB) is a standard Go WASM module that expects a host environment implementing:

- The Go JS runtime bridge (`gojs` namespace, as in `wasm_exec.js`)
- Standard I/O via the `fs` global object

### WASM Feature Matrix

| Feature | Native | WASM | Notes |
|---------|--------|------|-------|
| SQLite persistence | Yes | Yes | Host provides SQLite via `wasmsql` namespace |
| stdin/stdout I/O | Yes | Yes | Host bridges to OS streams |
| `-e` flag | Yes | Yes | |
| Pipe input via stdin | Yes | Yes | |
| LLM providers | Yes | Yes | Host provides `net/http` via `WithFetch()` |
| `-f` flag (file loading) | Yes | No | Requires `fs.open` in host |
| Interactive REPL | Yes | No | No terminal support in WASM |

**LLM providers** work in WASM when the host enables the Fetch API (`gigwasm.WithFetch()`), which provides a synchronous `net/http` implementation. All three providers (Ollama, OpenRouter, Anthropic) compile for WASM without build tags.

**File system access** is not implemented in the default host. The `-f` flag and any file I/O operations will not work. Programs should be passed via `-e` or piped through stdin.

### How the wasmsql Database Bridge Works

The WASM build replaces the native SQLite driver (`modernc.org/sqlite`) with gigwasm's `wasmsql` — a 6-function binary protocol that passes SQL operations across the WASM boundary. From losp's perspective, `database/sql` works identically in both builds.

**Guest side** (`gigwasm/wasmsql/driver.go`): A `database/sql/driver` implementation that uses `//go:wasmimport` to call 6 host functions in the `wasmsql` namespace. Imported by losp via `internal/store/sqlite_driver_wasm.go`:

```go
//go:build js && wasm
import _ "nickandperla.net/gigwasm/wasmsql"
const driverName = "wasmsql"
```

**Host side** (`gigwasm.WasmSQLNamespace(driverName)`): Translates the 6 wasmsql calls into `database/sql` operations against any backend driver. Enabled when creating a WASM instance:

```go
inst, _ := gigwasm.NewInstance(wasmBytes,
    gigwasm.WithImportNamespace(gigwasm.WasmSQLNamespace("sqlite")),
)
```

**The 6 wasmsql functions:**

| Function | Description |
|----------|-------------|
| `open(path, pathLen)` | Opens a database connection via `sql.Open(driverName, dsn)` |
| `close(db)` | Closes database and any open result sets |
| `exec(db, sql, sqlLen, params, paramsLen, result, resultLen)` | Non-row statements; returns last insert ID + rows affected |
| `query(db, sql, sqlLen, params, paramsLen, result, resultLen)` | Row-returning queries; returns result handle + column metadata |
| `next(handle, row, rowLen)` | Streams next row from a result handle |
| `close_rows(handle)` | Closes result handle, frees host resources |

Parameters and results use a binary TLV (type-length-value) wire format with little-endian byte order. Supported types: null, int64, float64, text, blob, bool. Buffers resize dynamically — if a result doesn't fit, the host returns the required size and the guest retries with a larger buffer.

### Build Tags

The native/WASM split is handled through Go build tags (`//go:build js && wasm` / `//go:build !(js && wasm)`). Platform-specific files:

| Concern | Native file | WASM file |
|---------|------------|-----------|
| SQLite driver import | `internal/store/sqlite_driver_native.go` | `internal/store/sqlite_driver_wasm.go` |
| REPL | `cmd/losp/repl.go` | `cmd/losp/repl_wasm.go` |

All other code — the interpreter, store, builtins, expression system, providers, and CLI — compiles identically for both targets.

## Running Tests

```bash
# Unit tests
go test ./...

# Conformance suite (native)
go generate ./internal/stdlib/ && go build -o ./losp ./cmd/losp && LOSP_BIN=./losp ./tests/conformance/run_tests.sh

# Conformance suite (WASM via gigwasm)
cd tests/wasm && rm -f losp.wasm && go test -v -count=1 -timeout 600s
```

The WASM conformance test (`tests/wasm/wasm_conformance_test.go`) uses gigwasm to compile losp to WASM, pre-compiles the module for reuse across tests, and runs every `.losp` file in `tests/conformance/` through the WASM runtime with SQLite and Fetch support enabled. The compiled `losp.wasm` is cached on disk — delete it to force recompilation after code changes.

## Docker

See the `Dockerfile` for containerized builds, or use:

```bash
docker build -t losp .
docker run --rm -i losp -db /tmp/losp.db -e '▶SAY hello ◆'
```
