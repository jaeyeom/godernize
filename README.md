[![Go Reference](https://pkg.go.dev/badge/github.com/jaeyeom/godernize.svg)](https://pkg.go.dev/github.com/jaeyeom/godernize)

# godernize

`godernize` is a linter designed to modernize deprecated Go patterns, helping developers update their code to use current best practices. By detecting deprecated function usage and suggesting modern alternatives, `godernize` ensures your Go code remains up-to-date and maintainable.

It consists of several analyzers:
1. `oserrors`: Detects deprecated os error checking functions and suggests replacing them with modern errors.Is() patterns.

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
- Add necessary imports (`errors`, `fs`)
- Remove unused `os` import if no longer needed
- Properly organize imports using `goimports`

### Ignoring checks

You can ignore specific checks using Go comments with the `//godernize:ignore` directive:

```go
//godernize:ignore oserrors
if os.IsNotExist(err) {
    // This check will be ignored
}

//godernize:ignore IsNotExist
if os.IsNotExist(err) {
    // This specific function check will be ignored
}
```

The directive can be placed:
- Above the function containing the deprecated call
- In a comment block before the specific line
- In the function's documentation comment