//go:build js && wasm

package store

import _ "nickandperla.net/gigwasm/wasmsql"

const driverName = "wasmsql"
