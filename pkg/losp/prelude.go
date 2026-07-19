// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

// Package losp provides the losp runtime.
package losp

// DefaultPrelude contains the standard library expressions that are
// automatically loaded unless -no-stdlib is specified.
const DefaultPrelude = `
▼__startup__ ◆
▼HTTPGET □_httpget_uri ▶HTTP GET ▲_httpget_uri ◆ ◆
▼HTTPPOST □_httppost_uri □_httppost_data ▶HTTP POST ▲_httppost_uri ▲EMPTY▲_httppost_data ◆ ◆
▼HTTPPUT □_httpput_uri □_httpput_data ▶HTTP PUT ▲_httpput_uri ▲EMPTY▲_httpput_data ◆ ◆
▼HTTPDELETE □_httpdelete_uri ▶HTTP DELETE ▲_httpdelete_uri ◆ ◆
`
