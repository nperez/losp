// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

//go:build js && wasm

package main

import "nickandperla.net/losp/pkg/losp"

func runREPL(runtime *losp.Runtime) {
	// No interactive REPL in WASM mode
}
