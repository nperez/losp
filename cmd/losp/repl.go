package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"nickandperla.net/losp/pkg/losp"
)

func runREPL(runtime *losp.Runtime) {
	// Try to load and execute __startup__ if it exists in the database
	// LOAD will define it in namespace if it exists, then we execute it
	// If it doesn't exist, these are no-ops
	runtime.Eval("▶LOAD __startup__ ◆")
	_, err := runtime.Eval("▶__startup__ ◆")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in __startup__: %v\n", err)
	}

	reader := bufio.NewReader(os.Stdin)
	var multiline strings.Builder
	inMultiline := false

	fmt.Println("losp REPL (Ctrl+D to exit)")
	fmt.Println()

	for {
		if inMultiline {
			fmt.Print("... ")
		} else {
			fmt.Print(">>> ")
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			// EOF
			fmt.Println()
			return
		}

		line = strings.TrimRight(line, "\r\n")

		// Check for multi-line continuation
		if strings.HasSuffix(line, "\\") {
			multiline.WriteString(strings.TrimSuffix(line, "\\"))
			multiline.WriteString("\n")
			inMultiline = true
			continue
		}

		var input string
		if inMultiline {
			multiline.WriteString(line)
			input = multiline.String()
			multiline.Reset()
			inMultiline = false
		} else {
			input = line
		}

		if strings.TrimSpace(input) == "" {
			continue
		}

		result, err := runtime.Eval(input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if result != "" {
			fmt.Println(result)
		}
	}
}
