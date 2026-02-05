package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
	"nickandperla.net/losp/pkg/losp"
)

// Alt+key mappings: Alt+key sends ESC (0x1b) followed by the key byte
var altKeyMappings = map[byte]string{
	'>': "▶", // Alt+> (Alt+Shift+.) - Execute
	'<': "▷", // Alt+< (Alt+Shift+,) - ImmExec
	'v': "▼", // Alt+v - Store
	'V': "▽", // Alt+V (Alt+Shift+v) - ImmStore
	'^': "▲", // Alt+^ (Alt+Shift+6) - Retrieve
	'A': "△", // Alt+A (Alt+Shift+a) - ImmRetrieve
	'o': "◯", // Alt+o - Defer
	'*': "◆", // Alt+* (Alt+Shift+8) - Terminator
	'[': "□", // Alt+[ - Placeholder
}

func printBanner() {
	fmt.Println("losp REPL (Ctrl+D to exit)")
	fmt.Println()
	fmt.Println("Operators (use Alt+key):")
	fmt.Println("  Alt+v → ▼ (store)       Alt+V → ▽ (imm store)")
	fmt.Println("  Alt+^ → ▲ (retrieve)    Alt+A → △ (imm retrieve)")
	fmt.Println("  Alt+> → ▶ (execute)     Alt+< → ▷ (imm execute)")
	fmt.Println("  Alt+o → ◯ (defer)       Alt+* → ◆ (terminator)")
	fmt.Println("  Alt+[ → □ (placeholder)")
	fmt.Println()
}

func runREPL(runtime *losp.Runtime) {
	// Try to load and execute __startup__ if it exists in the database
	runtime.Eval("▶LOAD __startup__ ◆")
	_, err := runtime.Eval("▶__startup__ ◆")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in __startup__: %v\n", err)
	}

	printBanner()

	// Check if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Not a TTY, fall back to basic mode
		runBasicREPL(runtime)
		return
	}

	runRawREPL(runtime)
}

// runBasicREPL handles non-TTY input (piped input)
func runBasicREPL(runtime *losp.Runtime) {
	reader := bufio.NewReader(os.Stdin)
	var multiline strings.Builder
	inMultiline := false

	for {
		if inMultiline {
			fmt.Print("... ")
		} else {
			fmt.Print(">>> ")
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println()
			return
		}

		line = strings.TrimRight(line, "\r\n")

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

// runRawREPL handles TTY input with Alt+key support
func runRawREPL(runtime *losp.Runtime) {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set raw mode: %v\n", err)
		runBasicREPL(runtime)
		return
	}
	defer term.Restore(fd, oldState)

	// Set custom input reader for READ builtin that works in raw mode
	runtime.SetInputReader(func(prompt string) (string, error) {
		if prompt != "" {
			// Print prompt with proper line ending for raw mode
			prompt = strings.ReplaceAll(prompt, "\n", "\r\n")
			fmt.Print(prompt)
		}
		line, eof := readLineRaw(fd)
		if eof {
			return "", io.EOF
		}
		return line + "\n", nil
	})

	var multiline strings.Builder
	inMultiline := false

	for {
		if inMultiline {
			fmt.Print("... ")
		} else {
			fmt.Print(">>> ")
		}

		line, eof := readLineRaw(fd)
		if eof {
			fmt.Print("\r\n")
			return
		}

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
			fmt.Printf("Error: %v\r\n", err)
			continue
		}

		if result != "" {
			// Replace newlines with \r\n for raw mode display
			result = strings.ReplaceAll(result, "\n", "\r\n")
			fmt.Println(result)
		}
	}
}

// readLineRaw reads a line in raw mode with Alt+key support
// Returns the line and whether EOF was encountered
func readLineRaw(fd int) (string, bool) {
	var line []rune
	cursor := 0 // Position in line (for arrow key navigation)
	buf := make([]byte, 1)

	// Helper to redraw line from cursor position
	redrawFromCursor := func() {
		// Clear from cursor to end of line
		fmt.Print("\x1b[K")
		// Print remaining characters
		for i := cursor; i < len(line); i++ {
			fmt.Print(string(line[i]))
		}
		// Move cursor back to position
		if cursor < len(line) {
			fmt.Printf("\x1b[%dD", len(line)-cursor)
		}
	}

	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return string(line), true
		}

		b := buf[0]

		switch b {
		case 0x04: // Ctrl+D
			if len(line) == 0 {
				return "", true
			}
			// Delete character at cursor (like Delete key)
			if cursor < len(line) {
				line = append(line[:cursor], line[cursor+1:]...)
				redrawFromCursor()
			}

		case 0x03: // Ctrl+C
			fmt.Print("^C\r\n")
			return "", false

		case 0x0d, 0x0a: // Enter (CR or LF)
			fmt.Print("\r\n")
			return string(line), false

		case 0x7f, 0x08: // Backspace (DEL or BS)
			if cursor > 0 {
				cursor--
				line = append(line[:cursor], line[cursor+1:]...)
				fmt.Print("\b") // Move cursor back
				redrawFromCursor()
			}

		case 0x1b: // ESC - could be Alt+key or arrow key sequence
			// Read next byte with timeout
			nextBuf := make([]byte, 1)
			n, err := os.Stdin.Read(nextBuf)
			if err != nil || n == 0 {
				continue
			}

			if nextBuf[0] == '[' {
				// Arrow key sequence: ESC [ A/B/C/D
				arrowBuf := make([]byte, 1)
				n, err = os.Stdin.Read(arrowBuf)
				if err != nil || n == 0 {
					continue
				}

				switch arrowBuf[0] {
				case 'A': // Up arrow - ignore for now (could add history)
				case 'B': // Down arrow - ignore for now
				case 'C': // Right arrow
					if cursor < len(line) {
						cursor++
						fmt.Print("\x1b[C")
					}
				case 'D': // Left arrow
					if cursor > 0 {
						cursor--
						fmt.Print("\x1b[D")
					}
				case '3': // Delete key: ESC [ 3 ~
					delBuf := make([]byte, 1)
					os.Stdin.Read(delBuf)
					if delBuf[0] == '~' && cursor < len(line) {
						line = append(line[:cursor], line[cursor+1:]...)
						redrawFromCursor()
					}
				}
			} else {
				// Alt+key: ESC followed by key byte
				if op, ok := altKeyMappings[nextBuf[0]]; ok {
					// Insert operator at cursor position
					runes := []rune(op)
					newLine := make([]rune, 0, len(line)+len(runes))
					newLine = append(newLine, line[:cursor]...)
					newLine = append(newLine, runes...)
					newLine = append(newLine, line[cursor:]...)
					line = newLine
					cursor += len(runes)
					// Print operator and redraw rest of line
					fmt.Print(op)
					if cursor < len(line) {
						redrawFromCursor()
					}
				}
			}

		case 0x01: // Ctrl+A - beginning of line
			if cursor > 0 {
				fmt.Printf("\x1b[%dD", cursor)
				cursor = 0
			}

		case 0x05: // Ctrl+E - end of line
			if cursor < len(line) {
				fmt.Printf("\x1b[%dC", len(line)-cursor)
				cursor = len(line)
			}

		case 0x0b: // Ctrl+K - kill to end of line
			if cursor < len(line) {
				line = line[:cursor]
				fmt.Print("\x1b[K")
			}

		case 0x15: // Ctrl+U - kill to beginning of line
			if cursor > 0 {
				fmt.Printf("\x1b[%dD", cursor)
				line = line[cursor:]
				cursor = 0
				redrawFromCursor()
			}

		default:
			if b >= 0x20 && b < 0x7f {
				// Printable ASCII character
				r := rune(b)
				newLine := make([]rune, 0, len(line)+1)
				newLine = append(newLine, line[:cursor]...)
				newLine = append(newLine, r)
				newLine = append(newLine, line[cursor:]...)
				line = newLine
				cursor++
				fmt.Print(string(r))
				if cursor < len(line) {
					redrawFromCursor()
				}
			} else if b >= 0x80 {
				// UTF-8 multi-byte sequence - read remaining bytes
				var utfBuf []byte
				utfBuf = append(utfBuf, b)

				// Determine how many more bytes to read
				numBytes := 0
				if b&0xE0 == 0xC0 {
					numBytes = 1
				} else if b&0xF0 == 0xE0 {
					numBytes = 2
				} else if b&0xF8 == 0xF0 {
					numBytes = 3
				}

				for i := 0; i < numBytes; i++ {
					n, err := os.Stdin.Read(buf)
					if err != nil || n == 0 {
						break
					}
					utfBuf = append(utfBuf, buf[0])
				}

				r := []rune(string(utfBuf))[0]
				newLine := make([]rune, 0, len(line)+1)
				newLine = append(newLine, line[:cursor]...)
				newLine = append(newLine, r)
				newLine = append(newLine, line[cursor:]...)
				line = newLine
				cursor++
				fmt.Print(string(r))
				if cursor < len(line) {
					redrawFromCursor()
				}
			}
		}
	}
}
