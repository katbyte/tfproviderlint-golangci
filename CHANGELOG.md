*Versions track the wrapped [tfproviderlint](https://github.com/bflad/tfproviderlint) release.*

## v0.31.1 (2026-08-07)

- register a second `tfproviderlintx` linter exposing the extended XAT/XR/XS checks alongside the standard set, mirroring upstream's two binaries
- add lint (golangci-lint), CodeQL, govulncheck, and dependency check workflows
- add dependabot updates for gomod and github-actions

## v0.31.0 (2026-08-06)

- initial release: a golangci-lint module plugin wrapping tfproviderlint v0.31.0's AT/R/S/V checks as a `tfproviderlint` linter
- support `enable`/`disable` settings plus a `flags` list for passing check-specific flags
- strip the embedded check-name prefix from messages so reports don't repeat it
- set per-check doc URLs from build info so links always match the wrapped tfproviderlint version
- add test workflow with a weekly plugin-build canary against golangci-lint
