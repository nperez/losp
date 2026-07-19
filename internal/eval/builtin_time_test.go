// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"testing"
	"time"
)

func TestStrftime(t *testing.T) {
	// 2026-07-04 09:05:07.123456 UTC, a Saturday (day 185 of the year)
	ref := time.Date(2026, 7, 4, 9, 5, 7, 123456000, time.UTC)

	tests := []struct {
		format string
		want   string
	}{
		{"%Y-%m-%d", "2026-07-04"},
		{"%H:%M:%S", "09:05:07"},
		{"%y", "26"},
		{"%e", " 4"},
		{"%I %p", "09 AM"},
		{"%a %A", "Sat Saturday"},
		{"%b %B", "Jul July"},
		{"%j", "185"},
		{"%f", "123456"},
		{"%Z", "UTC"},
		{"%z", "+0000"},
		{"%s", "1783155907"},
		{"100%%", "100%"},
		{"no directives", "no directives"},
		{"%Q unknown", "%Q unknown"},
		{"trailing %", "trailing %"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := strftime(ref, tt.format); got != tt.want {
			t.Errorf("strftime(%q) = %q, want %q", tt.format, got, tt.want)
		}
	}
}
