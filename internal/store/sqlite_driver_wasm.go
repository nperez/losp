// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

//go:build js && wasm

package store

import _ "nickandperla.net/gigwasm/wasmsql"

const driverName = "wasmsql"
