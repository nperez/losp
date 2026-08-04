// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

// losp-wasm is the WASM conformance harness. It has two distinct modes:
//
//	losp-wasm -build     compile cmd/losp to losp.wasm (the build step)
//	losp-wasm [category] run conformance tests through losp.wasm
//
// The run mode requires an existing losp.wasm — build first. This keeps the
// expensive Go->WASM compile separate from test runs.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"nickandperla.net/gigwasm"
)

const (
	colorRed   = "\033[0;31m"
	colorGreen = "\033[0;32m"
	colorReset = "\033[0m"
)

func main() {
	build := flag.Bool("build", false, "compile cmd/losp to the wasm binary and exit")
	wasmPath := flag.String("wasm", "losp.wasm", "path to the losp wasm binary")
	srcPath := flag.String("src", "../../cmd/losp", "path to the losp main package (for -build)")
	conformanceDir := flag.String("conformance", "../../tests/conformance", "path to the conformance test directory")
	flag.Parse()

	if *build {
		if err := buildWASM(*srcPath, *wasmPath); err != nil {
			fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	wasmBytes, err := os.ReadFile(*wasmPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot read %s: %v\nrun `losp-wasm -build` first\n", *wasmPath, err)
		os.Exit(1)
	}

	fmt.Println("Pre-compiling WASM module...")
	compiledModule, err := gigwasm.CompileModule(wasmBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to compile WASM module: %v\n", err)
		os.Exit(1)
	}

	shutdown, err := startFakeHTTPServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start fake HTTP server: %v\n", err)
		os.Exit(1)
	}
	defer shutdown()

	testFiles, err := findTests(*conformanceDir, flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if len(testFiles) == 0 {
		fmt.Fprintln(os.Stderr, "no conformance test files found")
		os.Exit(1)
	}
	fmt.Printf("Found %d conformance tests\n", len(testFiles))

	passed, failed := 0, 0
	for _, testFile := range testFiles {
		relPath, _ := filepath.Rel(*conformanceDir, testFile)
		if runTest(compiledModule, testFile, relPath) {
			passed++
		} else {
			failed++
		}
		// The instance, module, and store are closed explicitly after each
		// run; the per-instance wasmer engine has no Close and is reclaimed
		// only by its finalizer, so nudge the GC to keep engines from piling
		// up (the Go heap alone is too small to ever trigger it).
		runtime.GC()
	}

	fmt.Printf("\nResults: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func buildWASM(srcPath, wasmPath string) error {
	fmt.Println("Compiling losp to WASM...")
	wasmBytes, err := gigwasm.CompileGo(srcPath)
	if err != nil {
		return err
	}
	fmt.Printf("WASM binary: %d bytes, writing to %s\n", len(wasmBytes), wasmPath)
	return os.WriteFile(wasmPath, wasmBytes, 0644)
}

func findTests(conformanceDir, category string) ([]string, error) {
	root := conformanceDir
	if category != "" {
		root = filepath.Join(conformanceDir, category)
	}
	var testFiles []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".losp") {
			testFiles = append(testFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk conformance dir: %w", err)
	}
	return testFiles, nil
}

func runTest(compiledModule *gigwasm.CompiledModule, testFile, testName string) bool {
	content, err := os.ReadFile(testFile)
	if err != nil {
		fmt.Printf("%sFAIL%s %s (read error: %v)\n", colorRed, colorReset, testName, err)
		return false
	}

	expected, input, code := parseDirectives(string(content))

	// Create temp database for isolation
	tmpDB, err := os.CreateTemp("", "losp-wasm-test-*.db")
	if err != nil {
		fmt.Printf("%sFAIL%s %s (temp db error: %v)\n", colorRed, colorReset, testName, err)
		return false
	}
	tmpDBPath := tmpDB.Name()
	tmpDB.Close()
	defer os.Remove(tmpDBPath)

	// Determine args and stdin content
	var args []string
	var stdinContent string

	if input != "" {
		// INPUT mode: pass code via -e, pipe input to stdin
		// For tests that define __startup__, append execution
		eCode := code
		if strings.Contains(code, "__startup__") {
			eCode = code + "\n▶__startup__ ◆"
		}
		args = []string{"losp", "-db", tmpDBPath, "-e", eCode}
		// Add trailing newline to match bash echo behavior
		stdinContent = input + "\n"
	} else {
		// No INPUT: pipe code to stdin
		args = []string{"losp", "-db", tmpDBPath}
		stdinContent = code
	}

	actual, exitCode := runWASMInstance(compiledModule, args, stdinContent)

	// Trim trailing newlines to match bash $() capture behavior
	actual = strings.TrimRight(actual, "\n")

	if actual != expected {
		fmt.Printf("%sFAIL%s %s (exit=%d)\n  Expected: %q\n  Actual:   %q\n",
			colorRed, colorReset, testName, exitCode, expected, actual)
		return false
	}
	fmt.Printf("%sPASS%s %s\n", colorGreen, colorReset, testName)
	return true
}

// parseDirectives extracts EXPECTED lines, optional INPUT, and the losp code
// from a test file. Mirrors the bash test runner's directive parsing.
func parseDirectives(content string) (expected, input, code string) {
	lines := strings.Split(content, "\n")
	var expectedLines []string
	codeStart := 0

	for i, line := range lines {
		if strings.HasPrefix(line, "# EXPECTED: ") {
			expectedLines = append(expectedLines, line[len("# EXPECTED: "):])
		} else if strings.HasPrefix(line, "# EXPECTED:") {
			expectedLines = append(expectedLines, line[len("# EXPECTED:"):])
		} else if strings.HasPrefix(line, "# INPUT: ") {
			input = line[len("# INPUT: "):]
		} else if strings.HasPrefix(line, "# INPUT:") {
			input = line[len("# INPUT:"):]
		} else {
			codeStart = i
			break
		}
	}

	expected = strings.Join(expectedLines, "\n")

	// Expand \n escapes in input (matching bash echo -e behavior)
	if input != "" {
		input = strings.ReplaceAll(input, `\n`, "\n")
		input = strings.ReplaceAll(input, `\t`, "\t")
	}

	// Code is everything after directive lines, with directive lines removed
	// (matching the bash: sed '/^# EXPECTED:/d; /^# INPUT:/d')
	var codeLines []string
	for _, line := range lines[codeStart:] {
		if strings.HasPrefix(line, "# EXPECTED:") || strings.HasPrefix(line, "# INPUT:") {
			continue
		}
		codeLines = append(codeLines, line)
	}
	code = strings.Join(codeLines, "\n")

	return expected, input, code
}

// runWASMInstance runs the WASM binary with the given args and stdin content,
// capturing stdout+stderr. Runs are sequential due to os.Stdin/Stdout redirection.
func runWASMInstance(compiledModule *gigwasm.CompiledModule, args []string, stdinContent string) (string, int) {
	// Save originals
	origStdin, origStdout, origStderr := os.Stdin, os.Stdout, os.Stderr
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	// Set up stdin pipe
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create stdin pipe: %v\n", err)
		return "", -1
	}
	if stdinContent != "" {
		go func() {
			stdinW.Write([]byte(stdinContent))
			stdinW.Close()
		}()
	} else {
		stdinW.Close()
	}
	os.Stdin = stdinR

	// Set up stdout+stderr pipe (both go to same pipe, matching bash 2>&1)
	outR, outW, err := os.Pipe()
	if err != nil {
		os.Stdin = origStdin
		stdinR.Close()
		fmt.Fprintf(os.Stderr, "failed to create output pipe: %v\n", err)
		return "", -1
	}
	os.Stdout = outW
	os.Stderr = outW

	// Drain output in background
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, outR)
		close(done)
	}()

	// Run WASM instance from pre-compiled module (fast deserialization)
	inst, instErr := gigwasm.NewInstanceFromModule(compiledModule,
		gigwasm.WithArgs(args),
		gigwasm.WithImportNamespace(gigwasm.WasmSQLNamespace("sqlite")),
		gigwasm.WithFetch(),
	)

	// Close write end so drain goroutine finishes
	outW.Close()
	stdinR.Close()
	<-done

	exitCode := 0
	if inst != nil {
		exitCode = inst.ExitCode()
		inst.Close()
	}
	if instErr != nil {
		os.Stdout = origStdout
		os.Stderr = origStderr
		fmt.Printf("WASM instance error: %v\n", instErr)
	}

	return buf.String(), exitCode
}

// creaturesCSV is served at /creatures.csv for the 45_pipeline tests. Kept
// byte-identical to CREATURES_CSV in tests/conformance/run_tests.sh.
const creaturesCSV = `name,habitat,trait
Sunfish,the open ocean,drifts near the surface sunning itself
Ibex,steep mountains,climbs near-vertical rock faces
Mole,underground tunnels,digs through soil in total darkness`

// startFakeHTTPServer serves the same deterministic endpoints as the embedded
// server in tests/conformance/run_tests.sh, for the 36_http tests. WASM
// modules reach it through the host's Fetch API (gigwasm WithFetch).
func startFakeHTTPServer() (shutdown func(), err error) {
	port := os.Getenv("LOSP_HTTP_PORT")
	if port == "" {
		port = "8473"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello-from-server")
	})
	mux.HandleFunc("/method", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.Method)
	})
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})
	mux.HandleFunc("/header", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.Header.Get("X-Losp"))
	})
	mux.HandleFunc("/creatures.csv", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, creaturesCSV)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "not-found")
	})

	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		return nil, err
	}

	server := &http.Server{Handler: mux}
	go server.Serve(listener)

	return func() { server.Close() }, nil
}
