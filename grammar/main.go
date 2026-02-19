// losp-check: Syntax checker for .losp files.
//
// Generated from Losp.g4 via ANTLR4.  Validates structural syntax:
// matched operator/terminator pairs, proper nesting, and well-formed
// operator constructs.
//
// Usage:
//
//	losp-check FILE [FILE...]
//	losp-check --dir DIR
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"losp-grammar/parser"
)

// errorCollector gathers syntax errors from the ANTLR4 parser.
type errorCollector struct {
	antlr.DefaultErrorListener
	errors []string
}

func (c *errorCollector) SyntaxError(
	_ antlr.Recognizer,
	_ interface{},
	line, column int,
	msg string,
	_ antlr.RecognitionException,
) {
	c.errors = append(c.errors, fmt.Sprintf("line %d:%d: %s", line, column, msg))
}

// checkResult holds the outcome of checking a single file.
type checkResult struct {
	path         string
	errors       []string
	expectsError bool
}

// checkFile parses a .losp file and returns syntax errors.
// It strips conformance-test directives (# EXPECTED:, # INPUT:) before parsing.
// If any # EXPECTED: line contains "Error:", the file is marked as expecting errors.
func checkFile(path string) checkResult {
	content, err := os.ReadFile(path)
	if err != nil {
		return checkResult{
			path:   path,
			errors: []string{fmt.Sprintf("read error: %v", err)},
		}
	}

	lines := strings.Split(string(content), "\n")
	var lospLines []string
	expectsError := false

	for _, line := range lines {
		if strings.HasPrefix(line, "# EXPECTED:") {
			rest := strings.TrimPrefix(line, "# EXPECTED:")
			rest = strings.TrimSpace(rest)
			if strings.HasPrefix(rest, "Error:") || strings.HasPrefix(rest, "Error ") {
				expectsError = true
			}
			continue
		}
		if strings.HasPrefix(line, "# INPUT:") {
			continue
		}
		lospLines = append(lospLines, line)
	}

	lospContent := strings.Join(lospLines, "\n")

	// Create ANTLR4 lexer + parser with custom error collection.
	input := antlr.NewInputStream(lospContent)
	lexer := parser.NewLospLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	p := parser.NewLospParser(stream)

	collector := &errorCollector{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(collector)
	p.RemoveErrorListeners()
	p.AddErrorListener(collector)

	// Parse.  Errors are collected by the listener.
	p.Program()

	return checkResult{
		path:         path,
		errors:       collector.errors,
		expectsError: expectsError,
	}
}

// findLospFiles recursively finds all .losp files under dir.
func findLospFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".losp") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: losp-check [--dir DIR] FILE [FILE...]")
		os.Exit(1)
	}

	var files []string
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--dir" {
			if i+1 >= len(os.Args) {
				fmt.Fprintln(os.Stderr, "Error: --dir requires an argument")
				os.Exit(1)
			}
			i++
			found, err := findLospFiles(os.Args[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error scanning directory %s: %v\n", os.Args[i], err)
				os.Exit(1)
			}
			files = append(files, found...)
		} else {
			files = append(files, os.Args[i])
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "No .losp files found")
		os.Exit(1)
	}

	passed := 0
	failed := 0
	expectedErr := 0

	for _, f := range files {
		result := checkFile(f)
		hasErrors := len(result.errors) > 0

		if result.expectsError {
			// Error test: finding errors is expected, not finding them is OK too
			// (grammar may be more permissive than runtime for semantic errors).
			expectedErr++
			if hasErrors {
				fmt.Printf("OK   %s (expected error, found %d)\n", f, len(result.errors))
			} else {
				fmt.Printf("OK   %s (expected error, grammar accepted)\n", f)
			}
		} else if hasErrors {
			failed++
			fmt.Printf("FAIL %s\n", f)
			for _, e := range result.errors {
				fmt.Printf("     %s\n", e)
			}
		} else {
			passed++
			fmt.Printf("OK   %s\n", f)
		}
	}

	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Passed:          %d\n", passed)
	fmt.Printf("Expected errors: %d\n", expectedErr)
	fmt.Printf("Failed:          %d\n", failed)
	fmt.Printf("Total:           %d\n", len(files))

	if failed > 0 {
		os.Exit(1)
	}
}
