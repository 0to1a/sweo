# License Audit Report

**Project:** github.com/0to1a/sweo
**Date:** 2026-04-04
**Go version:** 1.25.0

## Summary

- **Total dependencies:** 13 (4 direct, 9 indirect)
- ✅ **MIT-compatible:** 13
- ⚠️ **Flagged:** 0

## Dependency License Table

### Direct Dependencies

| Module | Version | License | MIT-Compatible |
|---|---|---|---|
| github.com/fatih/color | v1.19.0 | MIT | ✅ Yes |
| github.com/spf13/cobra | v1.10.2 | Apache-2.0 | ✅ Yes |
| github.com/stretchr/testify | v1.11.1 | MIT | ✅ Yes |
| gopkg.in/yaml.v3 | v3.0.1 | MIT | ✅ Yes |

### Indirect (Transitive) Dependencies

| Module | Version | License | MIT-Compatible |
|---|---|---|---|
| github.com/creack/pty | v1.1.24 | MIT | ✅ Yes |
| github.com/davecgh/go-spew | v1.1.1 | ISC | ✅ Yes |
| github.com/gorilla/websocket | v1.5.3 | MIT | ✅ Yes (BSD-2-Clause-like) |
| github.com/inconshreveable/mousetrap | v1.1.0 | Apache-2.0 | ✅ Yes |
| github.com/mattn/go-colorable | v0.1.14 | MIT | ✅ Yes |
| github.com/mattn/go-isatty | v0.0.20 | MIT | ✅ Yes |
| github.com/pmezard/go-difflib | v1.0.0 | MIT | ✅ Yes |
| github.com/spf13/pflag | v1.0.9 | MIT (BSD-3-Clause style) | ✅ Yes |
| golang.org/x/sys | v0.42.0 | BSD-3-Clause | ✅ Yes |
| gopkg.in/check.v1 | v0.0.0-20161208181325-20d25e280405 | BSD-2-Clause | ✅ Yes |

## Flagged Dependencies

None.

## Methodology

- License detection performed by inspecting the `LICENSE`, `LICENSE.md`, `LICENSE.txt`, or `COPYING` file at the root of each module in the Go module cache.
- License text matched against known SPDX license patterns (MIT, Apache-2.0, BSD-2-Clause, BSD-3-Clause, ISC).
- Standard library modules excluded from audit.
- Transitive-only dependencies (not present in module cache: `cpuguy83/go-md2man/v2`, `russross/blackfriday/v2`, `stretchr/objx`, `go.yaml.in/yaml/v3`) are build/doc toolchain dependencies of cobra and not compiled into the final binary.

## Compatible License Types Found

| License | Count |
|---|---|
| MIT | 8 |
| Apache-2.0 | 2 |
| BSD-2-Clause | 1 |
| BSD-3-Clause | 1 |
| ISC | 1 |

## Verdict: ✅ Safe to publish as MIT

All dependencies use licenses compatible with the MIT license. No GPL, LGPL, AGPL, SSPL, or unknown licenses detected.
