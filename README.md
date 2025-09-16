[![Go Reference](https://pkg.go.dev/badge/github.com/jaeyeom/godernize.svg)](https://pkg.go.dev/github.com/jaeyeom/godernize)

# godernize

`godernize` is a linter designed to modernize deprecated Go patterns, helping developers update their code to use current best practices. By detecting deprecated function usage and suggesting modern alternatives, `godernize` ensures your Go code remains up-to-date and maintainable.

It consists of several analyzers:
1. `oserrors`: Detects deprecated os error checking functions and suggests replacing them with modern errors.Is() patterns.
2. `ctxnil`: Detects nil comparisons with context.Context and suggests removing them since contexts should never be nil.

## Usage

To install the linter:
```sh
go install github.com/jaeyeom/godernize/cmd/godernizecheck@latest
```

Run the linter:
```sh
godernizecheck ./...
```

## Analyzers

### oserrors

The `oserrors` analyzer reports usage of deprecated os error checking functions and suggests replacing them with modern `errors.Is()` patterns:

- `os.IsNotExist(err)` → `errors.Is(err, fs.ErrNotExist)`
- `os.IsExist(err)` → `errors.Is(err, fs.ErrExist)`
- `os.IsPermission(err)` → `errors.Is(err, fs.ErrPermission)`

The analyzer provides comprehensive fixes that:
- Replace all deprecated function calls in a file
- (Not implemented) Add necessary imports (`errors`, `fs`)
- (Not implemented) Remove unused `os` import if no longer needed
- (Not implemented) Properly organize imports using `goimports`

#### Standalone Usage

You can also use the `oserrors` analyzer independently:

```sh
go install github.com/jaeyeom/godernize/oserrors/cmd/oserrorsgodernize@latest
oserrorsgodernize ./...
```

### ctxnil

The `ctxnil` analyzer reports nil comparisons with `context.Context` values and suggests removing them since contexts should never be nil:

**Direct context comparisons:**
- `if ctx == nil { ... }` → Remove the entire if statement (then clause is unreachable)
- `if ctx != nil { ... }` → Replace with just the then clause (condition is always true)
- `if ctx != nil { ... } else { ... }` → Replace with just the then clause (else is unreachable)

**Boolean expressions with context:**
- `if ctx != nil && ready` → `if ready` (simplify to just the variable)
- `if ctx == nil || critical` → `if critical` (simplify to just the variable)
- `if ctx != nil && true` → Remove if condition (always true)
- `if ctx == nil || false` → Remove entire if statement (always false)

**Standalone expressions:**
- `result = ctx == nil` → `result = false`
- `result = ctx != nil` → `result = true`
- `doSomething(ctx == nil)` → `doSomething(false)`

Since Go's context package guarantees that contexts are never nil (functions like `context.Background()` and `context.TODO()` always return valid contexts), these checks are unnecessary and can indicate a misunderstanding of the context API.

#### Standalone Usage

You can also use the `ctxnil` analyzer independently:

```sh
go install github.com/jaeyeom/godernize/ctxnil/cmd/ctxnilgodernize@latest
ctxnilgodernize ./...
```

### Ignoring checks

You can ignore specific checks using Go comments with the `//godernize:ignore` directive:

```go
//godernize:ignore=oserrors
if os.IsNotExist(err) {
    // This check will be ignored
}

//godernize:ignore=ctxnil
if ctx == nil {
    // This nil check will be ignored
}

//godernize:ignore=IsNotExist
if os.IsNotExist(err) {
    // This specific function check will be ignored
}
```

The directive can be placed:
- Above the function containing the deprecated call
- In a comment block before the specific line
- In the function's documentation comment
