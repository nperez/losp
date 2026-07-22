// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

// Package losp provides the losp runtime.
package losp

import _ "embed"

// DefaultPrelude contains the standard library expressions that are
// automatically loaded unless -no-stdlib is specified. The source lives in
// prelude.losp (the startup hook, the HTTP verb wrappers, and the agent-loop
// library) and is embedded here.
//
//go:embed prelude.losp
var DefaultPrelude string
