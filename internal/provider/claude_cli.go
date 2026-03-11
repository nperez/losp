// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package provider

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ClaudeCLI is a provider that invokes the claude CLI as a fully detached process.
// It works around Claude Code's nested-session detection by writing a temp script,
// launching via nohup setsid, and polling for a done-marker file.
type ClaudeCLI struct {
	Model    string
	Timeout  time.Duration
	StreamCb StreamCallback
	params   map[string]string
}

// ClaudeCLIOption configures the ClaudeCLI provider.
type ClaudeCLIOption func(*ClaudeCLI)

// WithClaudeCLIModel sets the model name.
func WithClaudeCLIModel(model string) ClaudeCLIOption {
	return func(c *ClaudeCLI) { c.Model = model }
}

// WithClaudeCLITimeout sets the request timeout.
func WithClaudeCLITimeout(timeout time.Duration) ClaudeCLIOption {
	return func(c *ClaudeCLI) { c.Timeout = timeout }
}

// WithClaudeCLIStreamCallback sets the streaming callback.
func WithClaudeCLIStreamCallback(cb StreamCallback) ClaudeCLIOption {
	return func(c *ClaudeCLI) { c.StreamCb = cb }
}

// NewClaudeCLI creates a new ClaudeCLI provider.
func NewClaudeCLI(opts ...ClaudeCLIOption) *ClaudeCLI {
	c := &ClaudeCLI{
		Model:   "haiku",
		Timeout: 5 * time.Minute,
		params:  make(map[string]string),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// GetParam returns an inference parameter value.
func (c *ClaudeCLI) GetParam(key string) string { return c.params[key] }

// SetParam sets an inference parameter value.
func (c *ClaudeCLI) SetParam(key, value string) { c.params[key] = value }

// GetModel returns the current model name.
func (c *ClaudeCLI) GetModel() string { return c.Model }

// SetModel sets the model name.
func (c *ClaudeCLI) SetModel(model string) { c.Model = model }

// ProviderName returns "CLAUDE_CLI".
func (c *ClaudeCLI) ProviderName() string { return "CLAUDE_CLI" }

// Prompt sends a prompt to the claude CLI and returns the response.
// It fully detaches the claude process from the parent's process tree to avoid
// Claude Code's nested-session detection.
func (c *ClaudeCLI) Prompt(system, user string) (string, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude CLI not found in PATH: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "losp-claude-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	promptFile := filepath.Join(tmpDir, "prompt.txt")
	outputFile := filepath.Join(tmpDir, "output.txt")
	doneFile := filepath.Join(tmpDir, "done")
	errFile := filepath.Join(tmpDir, "error.txt")
	scriptFile := filepath.Join(tmpDir, "run.sh")

	if err := os.WriteFile(promptFile, []byte(user), 0600); err != nil {
		return "", fmt.Errorf("failed to write prompt file: %w", err)
	}

	var scriptBuilder strings.Builder
	scriptBuilder.WriteString("#!/bin/sh\n")
	scriptBuilder.WriteString("CLAUDECODE= MAX_THINKING_TOKENS=0 ")
	scriptBuilder.WriteString(fmt.Sprintf("%s -p ", claudePath))
	scriptBuilder.WriteString("--output-format text ")
	scriptBuilder.WriteString(fmt.Sprintf("--model %s ", shellQuote(c.Model)))
	scriptBuilder.WriteString("--max-turns 1 ")
	scriptBuilder.WriteString("--tools '' ")
	scriptBuilder.WriteString("--disable-slash-commands ")
	scriptBuilder.WriteString("--setting-sources '' ")
	scriptBuilder.WriteString(fmt.Sprintf("--system-prompt %s ", shellQuote(system)))
	scriptBuilder.WriteString(fmt.Sprintf("< %s > %s 2>%s\n", shellQuote(promptFile), shellQuote(outputFile), shellQuote(errFile)))
	scriptBuilder.WriteString(fmt.Sprintf("touch %s\n", shellQuote(doneFile)))

	if err := os.WriteFile(scriptFile, []byte(scriptBuilder.String()), 0700); err != nil {
		return "", fmt.Errorf("failed to write script file: %w", err)
	}

	// Launch fully detached: nohup setsid <script> &
	launch := exec.Command("nohup", "setsid", scriptFile)
	launch.Dir = tmpDir
	// Clear environment to prevent nested-session detection
	launch.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}
	// Add API key if set
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		launch.Env = append(launch.Env, "ANTHROPIC_API_KEY="+key)
	}

	if err := launch.Start(); err != nil {
		return "", fmt.Errorf("failed to launch claude CLI: %w", err)
	}

	// Don't block on the launcher — let it detach
	go launch.Wait()

	// Poll for done marker
	deadline := time.Now().Add(c.Timeout)
	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("claude CLI timed out after %v", c.Timeout)
		}

		if _, err := os.Stat(doneFile); err == nil {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	// Check for errors
	if errBytes, err := os.ReadFile(errFile); err == nil && len(errBytes) > 0 {
		errStr := strings.TrimSpace(string(errBytes))
		if errStr != "" {
			// stderr may contain warnings that aren't fatal — only error if output is empty
			if _, statErr := os.Stat(outputFile); statErr != nil {
				return "", fmt.Errorf("claude CLI error: %s", errStr)
			}
		}
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read claude CLI output: %w", err)
	}

	result := strings.TrimSpace(string(output))

	// Stream the result if callback is set (not true streaming, but delivers the output)
	if c.StreamCb != nil && result != "" {
		c.StreamCb(result)
	}

	return result, nil
}

// shellQuote wraps a string in single quotes, escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
