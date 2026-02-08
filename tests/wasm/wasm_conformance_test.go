package wasm_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nickandperla.net/gigwasm"
)

var compiledModule *gigwasm.CompiledModule

func TestMain(m *testing.M) {
	fmt.Println("Compiling losp to WASM...")
	wasmBytes, err := gigwasm.CompileGo("../../cmd/losp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compile losp to WASM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("WASM binary: %d bytes\n", len(wasmBytes))

	fmt.Println("Pre-compiling WASM module...")
	compiledModule, err = gigwasm.CompileModule(wasmBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to compile WASM module: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Module compiled, running tests...")

	os.Exit(m.Run())
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

func TestWASMConformance(t *testing.T) {
	if compiledModule == nil {
		t.Fatal("WASM module not compiled")
	}

	conformanceDir := "../../tests/conformance"
	absDir, err := filepath.Abs(conformanceDir)
	if err != nil {
		t.Fatalf("Failed to resolve conformance dir: %v", err)
	}

	var testFiles []string
	err = filepath.WalkDir(absDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".losp") {
			testFiles = append(testFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk conformance dir: %v", err)
	}

	if len(testFiles) == 0 {
		t.Fatal("No conformance test files found")
	}

	t.Logf("Found %d conformance tests", len(testFiles))

	for _, testFile := range testFiles {
		relPath, _ := filepath.Rel(absDir, testFile)
		testName := strings.TrimSuffix(relPath, ".losp")
		testName = strings.ReplaceAll(testName, string(filepath.Separator), "/")

		t.Run(testName, func(t *testing.T) {
			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			expected, input, code := parseDirectives(string(content))

			// Create temp database for isolation
			tmpDB, err := os.CreateTemp("", "losp-wasm-test-*.db")
			if err != nil {
				t.Fatalf("Failed to create temp db: %v", err)
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

			actual, exitCode := runWASMInstance(t, args, stdinContent)

			// Trim trailing newlines to match bash $() capture behavior
			actual = strings.TrimRight(actual, "\n")

			if actual != expected {
				t.Errorf("Output mismatch (exit=%d)\n  Expected: %q\n  Actual:   %q", exitCode, expected, actual)
			}
		})
	}
}

// runWASMInstance runs the WASM binary with the given args and stdin content,
// capturing stdout+stderr. Tests run sequentially due to os.Stdin/Stdout redirection.
func runWASMInstance(t *testing.T, args []string, stdinContent string) (string, int) {
	t.Helper()

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
		t.Fatalf("Failed to create stdin pipe: %v", err)
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
		t.Fatalf("Failed to create output pipe: %v", err)
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
		gigwasm.WithImportNamespace(gigwasm.SQLiteNamespace()),
		gigwasm.WithFetch(),
	)

	// Close write end so drain goroutine finishes
	outW.Close()
	stdinR.Close()
	<-done

	exitCode := 0
	if inst != nil {
		exitCode = inst.ExitCode()
	}
	if instErr != nil {
		// Restore stdout so we can log
		os.Stdout = origStdout
		os.Stderr = origStderr
		t.Logf("WASM instance error: %v", instErr)
	}

	return buf.String(), exitCode
}
