//go:build js && wasm

package main

import "nickandperla.net/losp/pkg/losp"

func runREPL(runtime *losp.Runtime) {
	// No interactive REPL in WASM mode
}
