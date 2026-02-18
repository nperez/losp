// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez

package stdlib

import _ "embed"

//go:generate cp ../../PRIMER.md .
//go:generate cp ../../PRIMER_COMPACT.md .
//go:generate cp ../../PROMPTING_LOSP.md .

//go:embed PRIMER.md
var Primer string

//go:embed PRIMER_COMPACT.md
var PrimerCompact string

//go:embed PROMPTING_LOSP.md
var PromptingLosp string
