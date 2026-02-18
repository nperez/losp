# Contributing to losp

Contributions are welcome. By submitting a contribution, you agree to the terms
of the [Contributor License Agreement](CLA.md).

## License

All code contributions are licensed under the
[AGPL-3.0-or-later](LICENSE.md). Contributions to the language specification
(PRIMER.md) are licensed under [CC BY-SA 4.0](LICENSE-SPEC.md).

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Ensure all conformance tests pass:
   ```bash
   go generate ./internal/stdlib/
   go build -o ./losp ./cmd/losp
   LOSP_BIN=./losp ./tests/conformance/run_tests.sh
   cd tests/wasm && rm -f losp.wasm && go test -v -count=1 -timeout 600s
   ```
5. Submit a pull request with a `Signed-off-by` line indicating CLA acceptance

## SPDX Headers

All Go source files and shell scripts must include SPDX license headers:

```go
// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (c) 2023-2026 Nicholas R. Perez
```
