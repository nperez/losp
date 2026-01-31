// Command losp is the losp interpreter CLI.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"

	"nickandperla.net/losp/pkg/losp"
)

func main() {
	var (
		evalStr     = flag.String("e", "", "Evaluate losp string")
		file        = flag.String("f", "", "Execute losp file")
		dbPath      = flag.String("db", "losp.db", "SQLite database path")
		providerF   = flag.String("provider", "", "LLM provider: ollama or openrouter")
		model       = flag.String("model", "", "LLM model name")
		stream      = flag.Bool("stream", false, "Enable streaming output")
		noPrompt    = flag.Bool("no-prompt", false, "Disable LLM prompts")
		noStdlib    = flag.Bool("no-stdlib", false, "Disable standard library prelude")
		ollamaURL   = flag.String("ollama", "http://localhost:11434", "Ollama API URL")
		persistMode = flag.String("persist-mode", "on_demand", "Persistence mode: on_demand, always, or never")
		compile     = flag.Bool("compile", false, "Compile mode: run program then persist all definitions")
	)

	flag.Parse()

	// Build options
	opts := []losp.Option{
		losp.WithSQLiteStore(*dbPath),
	}

	// Configure provider
	if !*noPrompt {
		switch *providerF {
		case "ollama":
			opts = append(opts, losp.WithOllama(*ollamaURL, *model))
		case "openrouter":
			opts = append(opts, losp.WithOpenRouter(*model))
		case "anthropic":
			opts = append(opts, losp.WithAnthropic(*model))
		case "":
			// Default to ollama if available
			opts = append(opts, losp.WithOllama(*ollamaURL, *model))
		default:
			fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", *providerF)
			os.Exit(1)
		}
	}

	// Configure streaming
	if *stream {
		opts = append(opts, losp.WithStreamCallback(func(token string) {
			fmt.Print(token)
		}))
	}

	// Configure stdlib
	if *noStdlib {
		opts = append(opts, losp.WithNoStdlib())
	}

	// Configure persist mode
	if *compile {
		// Compile mode: automatically persist all definitions
		opts = append(opts, losp.WithPersistMode(losp.PersistAlways))
	} else {
		mode, ok := losp.ParsePersistMode(*persistMode)
		if !ok {
			fmt.Fprintf(os.Stderr, "Unknown persist mode: %s (use on_demand, always, or never)\n", *persistMode)
			os.Exit(1)
		}
		opts = append(opts, losp.WithPersistMode(mode))
	}

	// Configure input reader - create reader ONCE, reuse across all READ calls
	stdinReader := bufio.NewReader(os.Stdin)
	opts = append(opts, losp.WithInputReader(func(prompt string) (string, error) {
		if prompt != "" {
			fmt.Print(prompt)
		}
		return stdinReader.ReadString('\n')
	}))

	runtime := losp.New(opts...)
	defer runtime.Close()

	var result string
	var err error

	// Step 1: Load file if specified (definitions only, no top-level execution)
	if *file != "" {
		err = runtime.LoadFile(*file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading file: %v\n", err)
			os.Exit(1)
		}
	}

	// Step 2: Run -e expression if provided (runs BEFORE __startup__)
	if *evalStr != "" {
		result, err = runtime.Eval(*evalStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if result != "" {
			fmt.Println(result)
		}
	}

	// Step 3: Determine main execution
	switch {
	case *file != "":
		// File was loaded, run __startup__ (unless compile mode)
		if !*compile {
			_, err = runtime.Eval("▶__startup__ ◆")
		}

	case *evalStr != "":
		// -e only (no file), already executed above, nothing more to do
		return

	case !isTerminal(os.Stdin):
		// Piped input (no file specified)
		input, readErr := io.ReadAll(os.Stdin)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", readErr)
			os.Exit(1)
		}
		result, err = runtime.Eval(string(input))
		// In compile mode, just persist and exit - don't run __startup__
		if err == nil && !*compile {
			_, err = runtime.Eval("▶__startup__ ◆")
		}

	default:
		// No file/string specified - load __startup__ from database and run it
		// LOAD retrieves from database into namespace, then we execute it
		runtime.Eval("▶LOAD __startup__ ◆")
		result, err = runtime.Eval("▶__startup__ ◆")
		// If __startup__ is empty/not found, fall through to REPL
		if result == "" && err == nil {
			runREPL(runtime)
			return
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if result != "" {
		fmt.Println(result)
	}
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
