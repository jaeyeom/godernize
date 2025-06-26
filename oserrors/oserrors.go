// Package oserrors provides an analyzer to detect deprecated os error checking functions.
package oserrors

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/imports"

	"github.com/jaeyeom/godernize/internal/directive"
)

// Doc describes what this analyzer does.
const Doc = `check for deprecated os error handling patterns

This analyzer reports usage of deprecated os error checking functions and suggests
replacing them with modern errors.Is() patterns:
- os.IsNotExist(err) -> errors.Is(err, fs.ErrNotExist)
- os.IsExist(err) -> errors.Is(err, fs.ErrExist)
- os.IsPermission(err) -> errors.Is(err, fs.ErrPermission)`

const standardTabWidth = 8

// Analyzer is the main analyzer for deprecated os error functions.
//
//nolint:gochecknoglobals // analyzer pattern requires global variable
var Analyzer = &analysis.Analyzer{
	Name:     "oserrors",
	Doc:      Doc,
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer, buildssa.Analyzer},
}

//nolint:gochecknoglobals // analyzer pattern
var deprecatedFunctions = map[string]string{
	"IsNotExist":   "errors.Is(err, fs.ErrNotExist)",
	"IsExist":      "errors.Is(err, fs.ErrExist)",
	"IsPermission": "errors.Is(err, fs.ErrPermission)",
}

type ignoreContext struct {
	file               *ast.File
	hasSpecificIgnores bool
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, fmt.Errorf("failed to get inspector")
	}

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	// Group deprecated calls by ignore context
	contextCallsMap := make(map[ignoreContext][]*deprecatedCall)

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return
		}

		processCallExpr(pass, call, contextCallsMap)
	})

	// Process each context and create appropriate fixes
	for context, calls := range contextCallsMap {
		if len(calls) > 0 {
			// Save the position before creating the fix, as the fix might modify the AST
			firstCallPos := calls[0].call.Pos()

			if context.hasSpecificIgnores {
				// For contexts with specific ignores, report individual diagnostics
				for _, call := range calls {
					pass.Report(analysis.Diagnostic{
						Pos:     call.call.Pos(),
						Message: "os." + call.fun.Sel.Name + " is deprecated, use " + call.replacement + " instead",
					})
				}
			} else {
				// For contexts without specific ignores, create comprehensive fix
				suggestedFix := createComprehensiveFix(pass, context.file, calls)
				if suggestedFix != nil {
					pass.Report(analysis.Diagnostic{
						Pos:            firstCallPos,
						Message:        createSummaryMessage(calls),
						SuggestedFixes: []analysis.SuggestedFix{*suggestedFix},
					})
				}
			}
		}
	}

	return nil, nil //nolint:nilnil // analyzer pattern
}

func processCallExpr(pass *analysis.Pass, call *ast.CallExpr, contextCallsMap map[ignoreContext][]*deprecatedCall) {
	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	if !isOsPackage(pass, fun.X) {
		return
	}

	replacement, deprecated := deprecatedFunctions[fun.Sel.Name]
	if !deprecated {
		return
	}

	// Check for directive to ignore this check
	if shouldIgnore(pass, call, fun.Sel.Name) {
		return
	}

	file := getFileForPos(pass, call.Pos())
	if file == nil {
		return
	}

	// Determine if this call is in a context with specific ignore directives
	hasSpecificIgnores := callHasSpecificIgnoreDirective(pass, call)
	context := ignoreContext{
		file:               file,
		hasSpecificIgnores: hasSpecificIgnores,
	}
	contextCallsMap[context] = append(contextCallsMap[context], &deprecatedCall{
		call:        call,
		fun:         fun,
		replacement: replacement,
	})
}

type deprecatedCall struct {
	call        *ast.CallExpr
	fun         *ast.SelectorExpr
	replacement string
}

func isOsPackage(pass *analysis.Pass, expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		if obj := pass.TypesInfo.Uses[ident]; obj != nil {
			if pkg, ok := obj.(*types.PkgName); ok {
				return pkg.Imported().Path() == "os"
			}
		}
	}

	return false
}

func createSummaryMessage(calls []*deprecatedCall) string {
	if len(calls) == 1 {
		return "os." + calls[0].fun.Sel.Name + " is deprecated, use " + calls[0].replacement + " instead"
	}

	return "Replace multiple deprecated os error functions with modern errors.Is() patterns"
}

func createComprehensiveFix(pass *analysis.Pass, file *ast.File, calls []*deprecatedCall) *analysis.SuggestedFix {
	// Create a copy of the file for modification
	fileCopy := copyASTFile(file)

	// Replace all deprecated function calls in the copied AST
	for _, call := range calls {
		if !replaceFunctionCall(fileCopy, call.call, call.replacement) {
			return nil // Failed to replace
		}
	}

	// Check if os import will become unused after all fixes
	if willAllOsUsageBeReplaced(pass, file, calls) {
		removeUnusedImport(fileCopy, "os")
	}

	// Use goimports to properly organize imports and add missing ones
	newContent, err := formatFileWithImports(pass, fileCopy)
	if err != nil {
		return nil
	}

	// Create a single TextEdit that replaces the entire file
	return &analysis.SuggestedFix{
		Message: "Replace deprecated os error functions and organize imports",
		TextEdits: []analysis.TextEdit{{
			Pos:     file.Pos(),
			End:     file.End(),
			NewText: newContent,
		}},
	}
}

func getFileForPos(pass *analysis.Pass, pos token.Pos) *ast.File {
	for _, file := range pass.Files {
		if file.Pos() <= pos && pos <= file.End() {
			return file
		}
	}

	return nil
}

func copyASTFile(file *ast.File) *ast.File {
	// Create a deep copy of the AST file
	// This is a simplified copy - in practice, you might want a more robust deep copy
	newFile := &ast.File{
		Doc:        file.Doc,
		Package:    file.Package,
		Name:       file.Name,
		Decls:      make([]ast.Decl, len(file.Decls)),
		Scope:      file.Scope,
		Imports:    make([]*ast.ImportSpec, len(file.Imports)),
		Unresolved: file.Unresolved,
		Comments:   file.Comments,
	}

	copy(newFile.Decls, file.Decls)
	copy(newFile.Imports, file.Imports)

	return newFile
}

func replaceFunctionCall(file *ast.File, originalCall *ast.CallExpr, _ string) bool {
	found := false

	ast.Inspect(file, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if call.Pos() == originalCall.Pos() && call.End() == originalCall.End() {
				// Replace the call with a simple identifier for the replacement
				// This is a simplified approach - parsing the replacement properly would be better
				if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
					// For our specific case, we know the replacement pattern
					// Replace os.IsXxx(err) with errors.Is(err, fs.ErrXxx)
					newCall := &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   &ast.Ident{Name: "errors"},
							Sel: &ast.Ident{Name: "Is"},
						},
						Args: []ast.Expr{
							call.Args[0], // Keep the original err argument
							createFsErrorExpr(fun.Sel.Name),
						},
					}
					*call = *newCall
					found = true

					return false
				}
			}
		}

		return true
	})

	return found
}

func createFsErrorExpr(funcName string) ast.Expr {
	var errName string

	switch funcName {
	case "IsNotExist":
		errName = "ErrNotExist"
	case "IsExist":
		errName = "ErrExist"
	case "IsPermission":
		errName = "ErrPermission"
	default:
		errName = "ErrNotExist" // fallback
	}

	return &ast.SelectorExpr{
		X:   &ast.Ident{Name: "fs"},
		Sel: &ast.Ident{Name: errName},
	}
}

func willAllOsUsageBeReplaced(pass *analysis.Pass, file *ast.File, calls []*deprecatedCall) bool {
	// Count all os package usages in the file
	osUsages := 0
	deprecatedUsages := 0
	callsToReplace := len(calls)

	ast.Inspect(file, func(n ast.Node) bool {
		countOsUsage(pass, n, &osUsages, &deprecatedUsages)

		return true
	})

	// If all os usages are deprecated functions and we're replacing all of them, the import will be unused
	return osUsages > 0 && osUsages == deprecatedUsages && deprecatedUsages == callsToReplace
}

func countOsUsage(pass *analysis.Pass, n ast.Node, osUsages, deprecatedUsages *int) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return
	}

	fun, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	if !isOsPackage(pass, fun.X) {
		return
	}

	*osUsages++

	if _, isDeprecated := deprecatedFunctions[fun.Sel.Name]; isDeprecated {
		*deprecatedUsages++
	}
}

func removeUnusedImport(file *ast.File, importPath string) {
	for i, imp := range file.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path == importPath {
			// Remove the import from the slice
			file.Imports = append(file.Imports[:i], file.Imports[i+1:]...)
			removeImportFromDeclarations(file, imp)

			return
		}
	}
}

func removeImportFromDeclarations(file *ast.File, imp *ast.ImportSpec) {
	for declIdx, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}

		for k, spec := range genDecl.Specs {
			if spec == imp {
				genDecl.Specs = append(genDecl.Specs[:k], genDecl.Specs[k+1:]...)
				if len(genDecl.Specs) == 0 {
					// Remove the entire import declaration if empty
					file.Decls = append(file.Decls[:declIdx], file.Decls[declIdx+1:]...)
				}

				return
			}
		}
	}
}

func formatFileWithImports(pass *analysis.Pass, file *ast.File) ([]byte, error) {
	// First, format the AST to get the source code
	var buf bytes.Buffer

	tokenFileSet := pass.Fset

	// Create a buffer to write the formatted code
	err := format.Node(&buf, tokenFileSet, file)
	if err != nil {
		return nil, fmt.Errorf("failed to format AST node: %w", err)
	}

	// Use goimports to organize imports properly
	options := &imports.Options{
		Comments:  true,
		TabIndent: true,
		TabWidth:  standardTabWidth,
		Fragment:  false,
	}

	result, err := imports.Process("", buf.Bytes(), options)
	if err != nil {
		return nil, fmt.Errorf("failed to process imports: %w", err)
	}

	return result, nil
}

func callHasSpecificIgnoreDirective(pass *analysis.Pass, call *ast.CallExpr) bool {
	// Check if this call is in a function or context that has specific ignore directives
	// (not just general //godernize:ignore, but specific ones like //godernize:ignore=IsNotExist)
	for _, file := range pass.Files {
		if hasSpecificFunctionIgnore(file, call) {
			return true
		}

		if hasSpecificCommentIgnore(file, call) {
			return true
		}
	}

	return false
}

func hasSpecificFunctionIgnore(file *ast.File, call *ast.CallExpr) bool {
	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if call.Pos() >= funcDecl.Pos() && call.End() <= funcDecl.End() {
			ignore := directive.ParseIgnore(funcDecl.Doc)
			if ignore != nil && ignore.HasSpecificRules() {
				return true
			}
		}
	}

	return false
}

func hasSpecificCommentIgnore(file *ast.File, call *ast.CallExpr) bool {
	for _, cg := range file.Comments {
		if cg.End() <= call.Pos() && call.Pos()-cg.End() <= 200 {
			ignore := directive.ParseIgnore(cg)
			if ignore != nil && ignore.HasSpecificRules() {
				return true
			}
		}
	}

	return false
}

func shouldIgnore(pass *analysis.Pass, call *ast.CallExpr, funcName string) bool {
	// Check all files for comments and function docs
	for _, file := range pass.Files {
		if shouldIgnoreInFunction(file, call, funcName) {
			return true
		}

		if shouldIgnoreFromComment(file, call, funcName) {
			return true
		}
	}

	return false
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
