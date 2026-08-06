# tfproviderlint-golangci

A [golangci-lint](https://golangci-lint.run/) module plugin exposing [tfproviderlint](https://github.com/bflad/tfproviderlint)'s checks, so they run inside a custom golangci-lint binary — sharing one package-load, one config and one report stream with every other linter — instead of as a separate `tfproviderlint`/`tfproviderlintx` pass over the codebase.

tfproviderlint itself ships no plugin support; this is a thin shim around its exported `passes.AllChecks` (and `xpasses.AllChecks` for the extended `tfproviderlintx` checks), which are standard `golang.org/x/tools/go/analysis` analyzers.

Mirroring upstream's two binaries, one module registers two linters — enable one or the other (`tfproviderlintx` already includes every standard check):

| Linter | Checks | Upstream equivalent |
|---|---|---|
| `tfproviderlint` | AT, R, S, V | `tfproviderlint` |
| `tfproviderlintx` | AT, R, S, V + XAT, XR, XS | `tfproviderlintx` |

## Installation

Add to your `.custom-gcl.yml` (shown alongside [azproviderlint](https://github.com/katbyte/azproviderlint) — plugins combine freely):

```yaml
version: v2.12.2
plugins:
  - module: "github.com/katbyte/tfproviderlint-golangci"
    import: "github.com/katbyte/tfproviderlint-golangci/plugin"
    version: v0.31.0
  - module: "github.com/katbyte/azproviderlint"
    import: "github.com/katbyte/azproviderlint/plugin"
    version: v0.1.0
```

Build the custom binary:

```bash
golangci-lint custom
```

Then enable in `.golangci.yml`:

```yaml
linters:
  enable:
    - tfproviderlint
  settings:
    custom:
      tfproviderlint:
        type: module
```

## Settings

All checks run by default. `enable`/`disable` filter by check name, and `flags` sets per-check flags using the same names as the tfproviderlint CLI (`-AT001.ignored-filename-suffixes` becomes check `AT001`, flag `ignored-filename-suffixes`). Flags are a list rather than dotted map keys because golangci-lint's config loader lowercases map keys and treats dots as nesting. Both linters take the same settings shape:

```yaml
linters:
  settings:
    custom:
      tfproviderlint:
        type: module
        settings:
          enable: [AT001, AT005, AT006, AT007, R001, R002, R003, R004, R006, S001, S002]
          flags:
            - check: AT001
              flag: ignored-filename-suffixes
              value: _data_source_test.go
            - check: R006
              flag: package-aliases
              value: pluginsdk
```

An empty `enable` list means all checks; a check named in `enable`/`disable`/`flags` that does not exist is a configuration error, not silently ignored. X checks are only known to `tfproviderlintx`, matching the upstream binaries.

## Improvements Over the Standalone Binaries

Beyond sharing golangci-lint's single package-load, config exclusions, `//nolint` and `--new-from-rev`:

- Messages are de-duplicated: tfproviderlint embeds the check name in its messages, which golangci-lint would prefix again (`S006: S006: schema ...`); the plugin strips the embedded copy.
- Every check gets a documentation `URL` pointing at its README in the exact wrapped tfproviderlint version (upstream sets none), so editors surfacing golangci diagnostics (e.g. gopls/VS Code) link straight to the check's docs.

## Ignoring Reports

tfproviderlint's own `//lintignore:AT001` comment directives keep working — they are implemented inside the analyzers themselves, so they apply no matter how the checks are run. golangci-lint's `//nolint:tfproviderlint` additionally ignores every check from this plugin on a line.

## Versioning

The wrapped tfproviderlint version is pinned in this module's `go.mod` — that pin is the source of truth for what a given shim version runs.

Tags mirror the wrapped upstream version: shim `v0.31.x` wraps tfproviderlint `v0.31.0`, with the patch digit free for shim-only fixes (upstream patch releases are rare enough that collisions are unlikely; if one ever lands, the shim skips ahead to mirror it). To upgrade the checks, bump tfproviderlint in `go.mod` and tag the matching version.
