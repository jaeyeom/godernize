# godernize

Go static analyzers that modernize deprecated patterns. Each analyzer is a standalone package; `cmd/godernizecheck` bundles them via `multichecker`.

## Validation

Run all three before committing — CI runs `go test` and `golangci-lint` only, not `gofumpt`:

```sh
make test lint check-format
```

Use `make format` to fix formatting. CI tests against `oldstable` and `stable` Go; lint uses `oldstable`.

## Architecture

| Package | Role |
|---|---|
| `oserrors/`, `ctxnil/` | Top-level analyzer packages — each exports a package-level `Analyzer` var |
| `internal/directive` | Shared `//godernize:ignore` parser; reuse it, do not duplicate |
| `cmd/godernizecheck` | `multichecker` entrypoint — register new analyzers here |
| `<analyzer>/cmd/*godernize` | Standalone `singlechecker` binary for one analyzer |

**Adding a new analyzer:**

1. Create a top-level package with an `Analyzer` variable (`Name`, `Doc`, `URL`, `Run`, `Requires`).
2. Depend on `inspect.Analyzer` only — do not add `buildssa`; this repo intentionally avoids it for nogo/Bazel compatibility.
3. Wire ignore checks through `internal/directive` (see existing `shouldIgnore` helpers in `oserrors` and `ctxnil`).
4. Register in `cmd/godernizecheck/main.go` and add a `singlechecker` binary under `<analyzer>/cmd/`.

## Testing

- Tests live in external `<pkg>_test` packages (`testpackage` linter enforces this).
- Use `golang.org/x/tools/go/analysis/analysistest` with `testdata/src/<scenario>/` directories.
- Diagnostic assertions: inline `// want "escaped regex message"` on the flagged line.
- Suggested-fix assertions: `analysistest.RunWithSuggestedFixes` with sibling `.golden` files under `testdata/src/autofix/`.
- Analyzer globals need `//nolint:gochecknoglobals`; `Run` returning `(any, error)` needs `//nolint:nilnil`.

## Ignore directives

`//godernize:ignore[=name1,name2]` — parsed by `internal/directive`:

| Form | Effect |
|---|---|
| `//godernize:ignore` | Ignore all analyzers for the enclosing scope |
| `//godernize:ignore=oserrors` | Ignore by analyzer name |
| `//godernize:ignore=IsNotExist` | Ignore a specific `oserrors` function name |

Placement: function doc comment, or a line comment ending within **200 bytes** before the diagnosed node.

## Gotchas

- **oserrors fixes are text-only.** `SuggestedFix` replaces the call expression but does not add `errors`/`io/fs` imports or prune unused `os` imports. Golden files in `oserrors/testdata/src/autofix/` reflect this — do not expect import rewriting until implemented.
- **ctxnil type matching is strict.** Only `context.Context` from package `context` is matched; custom context interfaces or wrappers are not.
- **ctxnil if-statement fixes use placeholder formatting.** `formatStmt` returns stub text (`{ /* statements */ }`), not full `go/format` output — expanding fix quality requires improving those helpers.
- **Duplicate ignore helpers.** `shouldIgnore` / `shouldIgnoreInFunction` / `shouldIgnoreFromComment` are copied per analyzer today; follow the existing pattern when adding analyzers until shared helpers are extracted.