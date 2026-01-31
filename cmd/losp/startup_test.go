package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestStartupExecutedAfterFile verifies that __startup__ is executed after evaluating a file
func TestStartupExecutedAfterFile(t *testing.T) {
	// Create a temp directory for our test
	tmpDir, err := os.MkdirTemp("", "losp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file that defines __startup__
	testFile := filepath.Join(tmpDir, "test.losp")
	testContent := `▼__startup__
    ▶SAY STARTUP_EXECUTED ◆
◆`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "losp"), "./")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build losp: %v\n%s", err, out)
	}

	// Run the CLI with the test file
	dbPath := filepath.Join(tmpDir, "test.db")
	runCmd := exec.Command(filepath.Join(tmpDir, "losp"), "-f", testFile, "-db", dbPath, "-no-prompt")
	output, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run losp: %v\n%s", err, output)
	}

	// Verify __startup__ was executed
	if !strings.Contains(string(output), "STARTUP_EXECUTED") {
		t.Errorf("expected output to contain 'STARTUP_EXECUTED', got: %s", output)
	}
}

// TestStartupLoadedFromDatabase verifies that __startup__ is loaded from DB when no file is provided
func TestStartupLoadedFromDatabase(t *testing.T) {
	// Create a temp directory for our test
	tmpDir, err := os.MkdirTemp("", "losp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "losp"), "./")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build losp: %v\n%s", err, out)
	}

	dbPath := filepath.Join(tmpDir, "test.db")

	// First, create a file that defines and persists __startup__
	testFile := filepath.Join(tmpDir, "setup.losp")
	testContent := `▼__startup__
    ▶SAY DB_STARTUP_EXECUTED ◆
◆
▶PERSIST __startup__ ◆`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run to persist __startup__ to database
	runCmd := exec.Command(filepath.Join(tmpDir, "losp"), "-f", testFile, "-db", dbPath, "-no-prompt")
	if out, err := runCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to run setup: %v\n%s", err, out)
	}

	// Now run with just the database (no file) - should load and execute __startup__
	runCmd2 := exec.Command(filepath.Join(tmpDir, "losp"), "-db", dbPath, "-no-prompt")
	output, err := runCmd2.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run from db: %v\n%s", err, output)
	}

	// Verify __startup__ was loaded from DB and executed
	if !strings.Contains(string(output), "DB_STARTUP_EXECUTED") {
		t.Errorf("expected output to contain 'DB_STARTUP_EXECUTED', got: %s", output)
	}
}

// TestCompileWorkflow verifies the full compile-then-run workflow
func TestCompileWorkflow(t *testing.T) {
	// Create a temp directory for our test
	tmpDir, err := os.MkdirTemp("", "losp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "losp"), "./")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build losp: %v\n%s", err, out)
	}

	dbPath := filepath.Join(tmpDir, "compiled.db")

	// Create a program file with functions and __startup__
	// Note: __startup__ must explicitly LOAD any functions it needs from the database
	testFile := filepath.Join(tmpDir, "program.losp")
	testContent := `▼Greeting Hello from compiled program! ◆

▼__startup__
    ▶LOAD Greeting ◆
    ▶SAY ▲Greeting ◆
◆`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Compile: run with -compile flag (persists all definitions)
	compileCmd := exec.Command(filepath.Join(tmpDir, "losp"), "-compile", "-f", testFile, "-db", dbPath, "-no-prompt")
	if out, err := compileCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to compile: %v\n%s", err, out)
	}

	// Run from database only
	runCmd := exec.Command(filepath.Join(tmpDir, "losp"), "-db", dbPath, "-no-prompt")
	output, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run from db: %v\n%s", err, output)
	}

	// Verify the program ran correctly
	if !strings.Contains(string(output), "Hello from compiled program!") {
		t.Errorf("expected output to contain 'Hello from compiled program!', got: %s", output)
	}
}

// TestEmptyStartupEntersREPL verifies that empty __startup__ falls through to REPL
// (We can't fully test REPL interaction, but we can verify it doesn't crash)
func TestEmptyStartupNoFile(t *testing.T) {
	// Create a temp directory for our test
	tmpDir, err := os.MkdirTemp("", "losp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "losp"), "./")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build losp: %v\n%s", err, out)
	}

	dbPath := filepath.Join(tmpDir, "empty.db")

	// Run with empty database - should try to enter REPL
	// We'll provide stdin to simulate immediate exit
	runCmd := exec.Command(filepath.Join(tmpDir, "losp"), "-db", dbPath, "-no-prompt")
	runCmd.Stdin = strings.NewReader("") // Empty input triggers EOF
	output, _ := runCmd.CombinedOutput()

	// Should see REPL prompt or empty output (not a crash)
	// The REPL prints "losp REPL" on start
	outputStr := string(output)
	if strings.Contains(outputStr, "panic") || strings.Contains(outputStr, "fatal") {
		t.Errorf("unexpected crash: %s", output)
	}
}

// TestCompileModeDoesNotRunStartup verifies that -compile flag does NOT run __startup__
func TestCompileModeDoesNotRunStartup(t *testing.T) {
	// Create a temp directory for our test
	tmpDir, err := os.MkdirTemp("", "losp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "losp"), "./")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build losp: %v\n%s", err, out)
	}

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create a test file with __startup__ that would print something
	testFile := filepath.Join(tmpDir, "test.losp")
	testContent := `▼__startup__
    ▶SAY STARTUP_SHOULD_NOT_RUN ◆
◆`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Run with -compile flag - should NOT execute __startup__
	runCmd := exec.Command(filepath.Join(tmpDir, "losp"), "-compile", "-f", testFile, "-db", dbPath, "-no-prompt")
	output, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run losp: %v\n%s", err, output)
	}

	// Verify __startup__ was NOT executed
	if strings.Contains(string(output), "STARTUP_SHOULD_NOT_RUN") {
		t.Errorf("compile mode should NOT run __startup__, but got output: %s", output)
	}
}

// TestPipedInputRunsStartup verifies that piped input also runs __startup__
func TestPipedInputRunsStartup(t *testing.T) {
	// Create a temp directory for our test
	tmpDir, err := os.MkdirTemp("", "losp-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build the CLI first
	cmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "losp"), "./")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build losp: %v\n%s", err, out)
	}

	dbPath := filepath.Join(tmpDir, "test.db")

	// Pipe in a program that defines __startup__
	program := `▼__startup__
    ▶SAY PIPED_STARTUP_RAN ◆
◆`

	runCmd := exec.Command(filepath.Join(tmpDir, "losp"), "-db", dbPath, "-no-prompt")
	runCmd.Stdin = strings.NewReader(program)
	output, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run piped: %v\n%s", err, output)
	}

	// Verify __startup__ was executed
	if !strings.Contains(string(output), "PIPED_STARTUP_RAN") {
		t.Errorf("expected output to contain 'PIPED_STARTUP_RAN', got: %s", output)
	}
}
