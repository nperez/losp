// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package eval

import (
	"fmt"
	"strings"
	"time"

	"nickandperla.net/losp/internal/expr"
)

func builtinNow(e *Evaluator, argsRaw string) (expr.Expr, error) {
	// NOW [format]
	// Returns wall-clock time. The optional format argument uses
	// strftime-style directives. Default is ISO-8601 (RFC 3339).
	args, err := e.parseArgs(argsRaw)
	if err != nil {
		return nil, err
	}

	t := time.Now()

	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return expr.Stored{Body: t.Format(time.RFC3339)}, nil
	}

	return expr.Stored{Body: strftime(t, args[0])}, nil
}

// strftime formats t according to a strftime-style format string.
// Unsupported directives are passed through literally (including the %).
func strftime(t time.Time, format string) string {
	var sb strings.Builder
	runes := []rune(format)

	for i := 0; i < len(runes); i++ {
		if runes[i] != '%' || i+1 >= len(runes) {
			sb.WriteRune(runes[i])
			continue
		}
		i++
		switch runes[i] {
		case 'Y':
			fmt.Fprintf(&sb, "%04d", t.Year())
		case 'y':
			fmt.Fprintf(&sb, "%02d", t.Year()%100)
		case 'm':
			fmt.Fprintf(&sb, "%02d", int(t.Month()))
		case 'd':
			fmt.Fprintf(&sb, "%02d", t.Day())
		case 'e':
			fmt.Fprintf(&sb, "%2d", t.Day())
		case 'H':
			fmt.Fprintf(&sb, "%02d", t.Hour())
		case 'I':
			sb.WriteString(t.Format("03"))
		case 'M':
			fmt.Fprintf(&sb, "%02d", t.Minute())
		case 'S':
			fmt.Fprintf(&sb, "%02d", t.Second())
		case 'f':
			fmt.Fprintf(&sb, "%06d", t.Nanosecond()/1000)
		case 'p':
			sb.WriteString(t.Format("PM"))
		case 'a':
			sb.WriteString(t.Format("Mon"))
		case 'A':
			sb.WriteString(t.Format("Monday"))
		case 'b':
			sb.WriteString(t.Format("Jan"))
		case 'B':
			sb.WriteString(t.Format("January"))
		case 'j':
			fmt.Fprintf(&sb, "%03d", t.YearDay())
		case 'Z':
			sb.WriteString(t.Format("MST"))
		case 'z':
			sb.WriteString(t.Format("-0700"))
		case 's':
			fmt.Fprintf(&sb, "%d", t.Unix())
		case '%':
			sb.WriteRune('%')
		default:
			sb.WriteRune('%')
			sb.WriteRune(runes[i])
		}
	}

	return sb.String()
}
