package oserrors_test

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"gotest.tools/v3/golden"

	"github.com/jaeyeom/godernize/oserrors"
)

type testCase struct {
	name        string
	inputFile   string
	goldenFile  string
	description string
}

func TestAutoFix(t *testing.T) {
	t.Parallel()

	testCases := []testCase{
		{
			name:        "single_deprecated",
			inputFile:   "single_deprecated.input.txt",
			goldenFile:  "single_deprecated.golden.txt",
			description: "Single deprecated function with unused import removal",
		},
		{
			name:        "multiple_deprecated",
			inputFile:   "multiple_deprecated.input.txt",
			goldenFile:  "multiple_deprecated.golden.txt",
			description: "Multiple deprecated functions in same file",
		},
		{
			name:        "keep_os_import",
			inputFile:   "keep_os_import.input.txt",
			goldenFile:  "keep_os_import.golden.txt",
			description: "Keep os import when still used elsewhere",
		},
		{
			name:        "import_grouping",
			inputFile:   "import_grouping.input.txt",
			goldenFile:  "import_grouping.golden.txt",
			description: "Verify proper import grouping with goimports",
		},
		{
			name:        "no_changes",
			inputFile:   "no_changes.input.txt",
			goldenFile:  "no_changes.golden.txt",
			description: "No changes needed for already modern code",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Read input file content
			inputPath := filepath.Join("testdata", testCase.inputFile)
			// #nosec G304 - test file paths are hardcoded and safe
			inputContent, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("Failed to read input file %s: %v", inputPath, err)
			}

			// Apply the analyzer's auto-fix functionality
			fixedContent, err := applyAutoFix(string(inputContent), testCase.inputFile)
			if err != nil {
				t.Fatalf("Failed to apply auto-fix: %v", err)
			}

			// Compare with golden file
			golden.Assert(t, fixedContent, testCase.goldenFile)

			// Verify that the result compiles
			if err := verifyGoSyntax(fixedContent); err != nil {
				t.Errorf("Fixed code has syntax errors: %v", err)
			}
		})
	}
}

// applyAutoFix runs the oserrors analyzer with fix mode enabled and returns the fixed content.
func applyAutoFix(source, filename string) (string, error) {
	// Create a temporary file set and parse the source
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, source, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse file: %w", err)
	}

	// Create inspector for the file
	inspectorInst := inspector.New([]*ast.File{file})

	// Track the final source content
	finalSource := source

	// Create a mock pass for the analyzer
	pass := &analysis.Pass{
		Analyzer: oserrors.Analyzer,
		Fset:     fset,
		Files:    []*ast.File{file},
		TypesInfo: &types.Info{
			Uses: make(map[*ast.Ident]types.Object),
		},
		ResultOf: map[*analysis.Analyzer]interface{}{
			inspect.Analyzer: inspectorInst,
		},
		Report: func(d analysis.Diagnostic) {
			// Apply the suggested fix if available
			if len(d.SuggestedFixes) > 0 {
				// For our test, we'll apply the first suggested fix
				fix := d.SuggestedFixes[0]
				for _, edit := range fix.TextEdits {
					// Apply the text edit to our source
					if edit.Pos == file.Pos() && edit.End == file.End() {
						// This is a full file replacement
						finalSource = string(edit.NewText)
					}
				}
			}
		},
	}

	// Create a basic types info for the os package identification
	pkg := types.NewPackage("os", "os")

	// Walk the AST to find os package references and populate TypesInfo
	ast.Inspect(file, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && ident.Name == "os" {
			if ident.Obj == nil { // This is likely a package reference
				pkgName := types.NewPkgName(ident.Pos(), pkg, "os", pkg)
				pass.TypesInfo.Uses[ident] = pkgName
			}
		}

		return true
	})

	// Run the analyzer
	_, err = oserrors.Analyzer.Run(pass)
	if err != nil {
		return "", fmt.Errorf("analyzer run failed: %w", err)
	}

	return finalSource, nil
}

// verifyGoSyntax checks if the given Go source code is syntactically correct.
func verifyGoSyntax(source string) error {
	fset := token.NewFileSet()

	_, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("syntax error: %w", err)
	}

	// Also try to format it to ensure it's well-formed
	_, err = format.Source([]byte(source))
	if err != nil {
		return fmt.Errorf("format error: %w", err)
	}

	return nil
}
