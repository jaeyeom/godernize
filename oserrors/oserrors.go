// Package oserrors provides an analyzer to detect deprecated os error checking functions.
package oserrors

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"path/filepath"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/jaeyeom/godernize/internal/directive"
)

// Doc describes what this analyzer does.
const Doc = `check for deprecated os error handling patterns

This analyzer reports usage of deprecated os error checking functions and suggests
replacing them with modern errors.Is() patterns:
- os.IsNotExist(err) -> errors.Is(err, fs.ErrNotExist)
- os.IsExist(err) -> errors.Is(err, fs.ErrExist)
- os.IsPermission(err) -> errors.Is(err, fs.ErrPermission)`

// Analyzer is the main analyzer for deprecated os error functions.
//
//nolint:gochecknoglobals // analyzer pattern requires global variable
var Analyzer = newAnalyzer(map[string]string{
	"IsNotExist":   "ErrNotExist",
	"IsExist":      "ErrExist",
	"IsPermission": "ErrPermission",
})

func newAnalyzer(osFuncsToFsErr map[string]string) *analysis.Analyzer {
	runner := runner{osFuncsToFsErr: osFuncsToFsErr}

	analyzer := &analysis.Analyzer{
		Name:     "oserrors",
		Doc:      Doc,
		URL:      "https://pkg.go.dev/github.com/jaeyeom/godernize/oserrors",
		Run:      runner.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}

	return analyzer
}

type runner struct {
	osFuncsToFsErr map[string]string
}

//nolint:nilnil // analyzer pattern
func (r *runner) run(pass *analysis.Pass) (any, error) {
	if pass == nil {
		return nil, nil
	}

	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok || inspect == nil {
		return nil, nil // Don't fail, just skip analysis
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	fileMap := buildFileMap(pass)

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok || call == nil {
			return
		}

		pos := pass.Fset.Position(call.Pos())
		filename := pos.Filename
		file := fileMap[filename]

		if diagnostic := r.diagnoseCallExpr(file, call); diagnostic != nil {
			pass.Report(*diagnostic)
		}
	})

	return nil, nil
}

func buildFileMap(pass *analysis.Pass) map[string]*ast.File {
	fileMap := make(map[string]*ast.File)

	for _, file := range pass.Files {
		pos := pass.Fset.Position(file.Pos())
		fileMap[pos.Filename] = file
	}

	return fileMap
}

func (r *runner) diagnoseCallExpr(file *ast.File, call *ast.CallExpr) *analysis.Diagnostic {
	if call == nil || call.Fun == nil {
		return nil
	}

	fName, fsErr := r.findMapping(file, call)
	if fsErr == "" {
		return nil // Not a deprecated os function
	}

	if shouldIgnore(file, call, fName) {
		return nil
	}

	return createDiagnostic(file, call, fName, fsErr)
}

func (r *runner) findMapping(file *ast.File, call *ast.CallExpr) (fName, fsErr string) {
	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || fun == nil || fun.X == nil || fun.Sel == nil || !isPkg(file, fun.X, "os") {
		return "", ""
	}

	return fun.Sel.Name, r.osFuncsToFsErr[fun.Sel.Name]
}

func isPkg(file *ast.File, expr ast.Expr, path string) bool {
	if expr == nil {
		return false
	}

	ident, ok := expr.(*ast.Ident)
	if !ok || ident == nil {
		return false
	}

	// Simple approach: check if identifier is "os" and os package is imported
	return ident.Name == importedIdentifier(file, ident, path)
}

func importedIdentifier(file *ast.File, ident *ast.Ident, path string) string {
	if file == nil || ident == nil || !ident.Pos().IsValid() {
		return ""
	}

	// Check if this identifier is in this file
	if !file.Pos().IsValid() || !file.End().IsValid() ||
		ident.Pos() < file.Pos() || ident.Pos() > file.End() {
		return ""
	}

	return findAliasName(file, path)
}

func shouldIgnore(file *ast.File, call *ast.CallExpr, funcName string) bool {
	return shouldIgnoreInFunction(file, call, funcName) || shouldIgnoreFromComment(file, call, funcName)
}

func shouldIgnoreInFunction(file *ast.File, call *ast.CallExpr, funcName string) bool {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if call.Pos() >= funcDecl.Pos() && call.End() <= funcDecl.End() {
			ignore := directive.ParseIgnore(funcDecl.Doc)
			if ignore != nil && (ignore.ShouldIgnore("oserrors") || ignore.ShouldIgnore(funcName)) {
				return true
			}
		}
	}

	return false
}

func shouldIgnoreFromComment(file *ast.File, call *ast.CallExpr, funcName string) bool {
	for _, cg := range file.Comments {
		// Check if comment appears before the call and is reasonably close
		if cg.End() <= call.Pos() && call.Pos()-cg.End() <= 200 {
			ignore := directive.ParseIgnore(cg)
			if ignore != nil && (ignore.ShouldIgnore("oserrors") || ignore.ShouldIgnore(funcName)) {
				return true
			}
		}
	}

	return false
}

func createDiagnostic(file *ast.File, call *ast.CallExpr, fName, fsErr string) *analysis.Diagnostic {
	if call == nil || !call.Pos().IsValid() || !call.End().IsValid() {
		return nil
	}

	// Extract the original argument from the call
	if len(call.Args) != 1 {
		return nil // Only handle single-argument calls
	}

	// Get the argument as text
	argText := formatASTNode(call.Args[0])
	if argText == "" {
		argText = "err" // fallback
	}

	replacementText := buildReplacementText(file, argText, fsErr)
	if replacementText == "" {
		return nil // No valid replacement found
	}

	return &analysis.Diagnostic{
		Pos:     call.Pos(),
		Message: fmt.Sprintf("os.%s is deprecated, use %s instead", fName, replacementText),
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: "Replace with " + replacementText,
			TextEdits: []analysis.TextEdit{{
				Pos:     call.Pos(),
				End:     call.End(),
				NewText: []byte(replacementText),
			}},
		}},
	}
}

func formatASTNode(node ast.Node) string {
	if node == nil {
		return ""
	}

	fset := token.NewFileSet()

	var buf bytes.Buffer

	err := format.Node(&buf, fset, node)
	if err != nil {
		return ""
	}

	return buf.String()
}

func buildReplacementText(file *ast.File, argText, fsErr string) string {
	errorsPackage := findAliasName(file, "errors")
	if errorsPackage == "" {
		errorsPackage = "errors" // new import
	}

	fsPackage := findAliasName(file, "io/fs")
	if fsPackage == "" {
		fsPackage = "fs" // new import
	}

	return fmt.Sprintf("%s.Is(%s, %s.%s)", errorsPackage, argText, fsPackage, fsErr)
}

func findAliasName(file *ast.File, path string) string {
	for _, imp := range file.Imports {
		if imp == nil || imp.Path == nil {
			continue
		}

		if imp.Path.Value == strconv.Quote(path) {
			aliasName := aliasNameOf(imp)
			if aliasName == "" {
				return filepath.Base(path)
			}

			return aliasName
		}
	}

	return ""
}

func aliasNameOf(imp *ast.ImportSpec) string {
	if imp.Name == nil {
		return ""
	}

	return imp.Name.Name
}
