// Package ctxnil provides an analyzer to detect nil comparisons with context.Context.
package ctxnil

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/jaeyeom/godernize/internal/directive"
)

const (
	trueValue  = "true"
	falseValue = "false"
)

// Doc describes what this analyzer does.
const Doc = `check for nil comparisons with context.Context

This analyzer reports nil comparisons with context.Context values and suggests
removing them since contexts should never be nil. It performs expression
simplification to handle complex boolean expressions and control flow.`

// Analyzer is the main analyzer for context nil comparisons.
//
//nolint:gochecknoglobals // analyzer pattern requires global variable
var Analyzer = &analysis.Analyzer{
	Name:     "ctxnil",
	Doc:      Doc,
	URL:      "https://pkg.go.dev/github.com/jaeyeom/godernize/ctxnil",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

//nolint:nilnil // analyzer pattern
func run(pass *analysis.Pass) (any, error) {
	if pass == nil {
		return nil, nil
	}

	inspect, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok || inspect == nil {
		return nil, nil // Don't fail, just skip analysis
	}

	nodeFilter := []ast.Node{
		(*ast.IfStmt)(nil),
		(*ast.BinaryExpr)(nil),
	}

	fileMap := buildFileMap(pass)
	processedExprs := make(map[ast.Expr]bool) // Track processed expressions to avoid duplicates

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		pos := pass.Fset.Position(n.Pos())
		filename := pos.Filename
		file := fileMap[filename]

		switch node := n.(type) {
		case *ast.IfStmt:
			if diagnostic := diagnoseIfStmt(pass, file, node); diagnostic != nil {
				pass.Report(*diagnostic)
				// Mark the condition as processed to avoid duplicate reports
				if node.Cond != nil {
					markProcessedExpr(node.Cond, processedExprs)
				}
			}
		case *ast.BinaryExpr:
			// Only process if not already handled by an if statement
			if !processedExprs[node] {
				if diagnostic := diagnoseBinaryExpr(pass, file, node); diagnostic != nil {
					pass.Report(*diagnostic)
				}
			}
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

// markProcessedExpr recursively marks an expression and its sub-expressions as processed.
func markProcessedExpr(expr ast.Expr, processed map[ast.Expr]bool) {
	if expr == nil {
		return
	}

	processed[expr] = true

	switch exprType := expr.(type) {
	case *ast.BinaryExpr:
		markProcessedExpr(exprType.X, processed)
		markProcessedExpr(exprType.Y, processed)
	case *ast.UnaryExpr:
		markProcessedExpr(exprType.X, processed)
	case *ast.ParenExpr:
		markProcessedExpr(exprType.X, processed)
	}
}

func diagnoseBinaryExpr(pass *analysis.Pass, file *ast.File, expr *ast.BinaryExpr) *analysis.Diagnostic {
	if expr == nil {
		return nil
	}

	// Check if this is a nil comparison with context
	ctxSide, nilSide, isEqual := analyzeContextNilComparison(pass, expr)
	if ctxSide == nil || nilSide == nil {
		return nil // Not a context nil comparison
	}

	if shouldIgnore(file, expr, "ctxnil") {
		return nil
	}

	// Determine replacement value
	replacement := falseValue
	if !isEqual {
		replacement = trueValue
	}

	message := fmt.Sprintf("context should never be nil, replace '%s' with '%s'",
		formatExpr(expr), replacement)

	return &analysis.Diagnostic{
		Pos:     expr.Pos(),
		Message: message,
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: fmt.Sprintf("Replace with %s", replacement),
			TextEdits: []analysis.TextEdit{{
				Pos:     expr.Pos(),
				End:     expr.End(),
				NewText: []byte(replacement),
			}},
		}},
	}
}

func diagnoseIfStmt(pass *analysis.Pass, file *ast.File, stmt *ast.IfStmt) *analysis.Diagnostic {
	if stmt == nil || stmt.Cond == nil {
		return nil
	}

	if shouldIgnore(file, stmt, "ctxnil") {
		return nil
	}

	// Check if condition contains context nil comparisons
	replacement := buildReplacementCondition(pass, stmt.Cond)
	if replacement == nil {
		return nil // No context nil comparisons found
	}

	// Generate appropriate fix based on replacement
	return createConditionFix(stmt, replacement)
}

// analyzeContextNilComparison checks if this binary expression compares context with nil.
func analyzeContextNilComparison(pass *analysis.Pass, expr *ast.BinaryExpr) (ctxSide, nilSide ast.Expr, isEqual bool) {
	if expr.Op != token.EQL && expr.Op != token.NEQ {
		return nil, nil, false
	}

	// Check if one side is context and other is nil
	leftIsCtx := isContextType(pass, expr.X)
	rightIsCtx := isContextType(pass, expr.Y)
	leftIsNil := isNilIdent(expr.X)
	rightIsNil := isNilIdent(expr.Y)

	if leftIsCtx && rightIsNil {
		return expr.X, expr.Y, expr.Op == token.EQL
	}

	if rightIsCtx && leftIsNil {
		return expr.Y, expr.X, expr.Op == token.EQL
	}

	return nil, nil, false
}

// isContextType checks if the expression has context.Context type.
func isContextType(pass *analysis.Pass, expr ast.Expr) bool {
	if expr == nil {
		return false
	}

	typ := pass.TypesInfo.TypeOf(expr)
	if typ == nil {
		return false
	}

	// Check if type is context.Context
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}

	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}

	return obj.Pkg().Path() == "context" && obj.Name() == "Context"
}

// isNilIdent checks if expression is the nil identifier.
func isNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)

	return ok && ident != nil && ident.Name == "nil"
}

// ReplacementCondition represents a condition replacement.
type ReplacementCondition struct {
	NewCondition string
	IsLiteral    bool // true if the result is a literal true/false
	Message      string
}

// buildReplacementCondition recursively builds a replacement for conditions containing context nil comparisons.
func buildReplacementCondition(pass *analysis.Pass, expr ast.Expr) *ReplacementCondition {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		return handleBinaryExpr(pass, e)
	case *ast.ParenExpr:
		inner := buildReplacementCondition(pass, e.X)
		if inner == nil {
			return nil
		}

		return &ReplacementCondition{
			NewCondition: "(" + inner.NewCondition + ")",
			IsLiteral:    inner.IsLiteral,
			Message:      inner.Message,
		}
	}

	return nil
}

// handleBinaryExpr handles binary expressions (==, !=, &&, ||).
func handleBinaryExpr(pass *analysis.Pass, expr *ast.BinaryExpr) *ReplacementCondition {
	// Check if this is a direct context nil comparison
	if ctxSide, nilSide, isEqual := analyzeContextNilComparison(pass, expr); ctxSide != nil && nilSide != nil {
		replacement := falseValue
		if !isEqual {
			replacement = trueValue
		}

		return &ReplacementCondition{
			NewCondition: replacement,
			IsLiteral:    true,
			Message:      fmt.Sprintf("context nil comparison '%s' is always %s", formatExpr(expr), replacement),
		}
	}

	// Handle logical operators
	if expr.Op == token.LAND || expr.Op == token.LOR {
		return handleLogicalExpr(pass, expr)
	}

	return nil
}

// handleLogicalExpr handles && and || expressions.
func handleLogicalExpr(pass *analysis.Pass, expr *ast.BinaryExpr) *ReplacementCondition {
	leftReplacement := buildReplacementCondition(pass, expr.X)
	rightReplacement := buildReplacementCondition(pass, expr.Y)

	// If neither side contains context comparisons, we can't help
	if leftReplacement == nil && rightReplacement == nil {
		return nil
	}

	leftExpr := formatExpr(expr.X)
	rightExpr := formatExpr(expr.Y)

	if leftReplacement != nil {
		leftExpr = leftReplacement.NewCondition
	}

	if rightReplacement != nil {
		rightExpr = rightReplacement.NewCondition
	}

	// Now simplify the logical expression
	if expr.Op == token.LAND {
		return simplifyAndExpr(leftExpr, rightExpr, leftReplacement, rightReplacement)
	}

	return simplifyOrExpr(leftExpr, rightExpr, leftReplacement, rightReplacement)
}

// simplifyAndExpr simplifies && expressions.
func simplifyAndExpr(leftExpr, rightExpr string, leftRep, rightRep *ReplacementCondition) *ReplacementCondition {
	// Check for short-circuit cases first
	if result := checkAndShortCircuit(leftExpr, rightExpr, leftRep, rightRep); result != nil {
		return result
	}

	// Check for simplification cases
	if result := checkAndSimplification(leftExpr, rightExpr, leftRep, rightRep); result != nil {
		return result
	}

	// General case where simplification occurred
	return buildAndReplacement(leftExpr, rightExpr, leftRep, rightRep)
}

// checkAndShortCircuit checks for short-circuit cases in && expressions.
func checkAndShortCircuit(leftExpr, rightExpr string, leftRep, rightRep *ReplacementCondition) *ReplacementCondition {
	// false && X -> false (short circuit)
	if leftRep != nil && leftRep.IsLiteral && leftExpr == falseValue {
		return &ReplacementCondition{
			NewCondition: falseValue,
			IsLiteral:    true,
			Message:      "condition is always false (left side is false)",
		}
	}
	// X && false -> false (short circuit)
	if rightRep != nil && rightRep.IsLiteral && rightExpr == falseValue {
		return &ReplacementCondition{
			NewCondition: falseValue,
			IsLiteral:    true,
			Message:      "condition is always false (right side is false)",
		}
	}

	return nil
}

// checkAndSimplification checks for simplification cases in && expressions.
func checkAndSimplification(leftExpr, rightExpr string, leftRep, rightRep *ReplacementCondition) *ReplacementCondition {
	// true && X -> X
	if leftRep != nil && leftRep.IsLiteral && leftExpr == trueValue {
		isRightLiteral := isLiteralExpr(rightExpr, rightRep)

		return &ReplacementCondition{
			NewCondition: rightExpr,
			IsLiteral:    isRightLiteral,
			Message:      fmt.Sprintf("simplify to '%s' (left side is always true)", rightExpr),
		}
	}
	// X && true -> X
	if rightExpr == trueValue {
		isLeftLiteral := isLiteralExpr(leftExpr, leftRep)

		return &ReplacementCondition{
			NewCondition: leftExpr,
			IsLiteral:    isLeftLiteral,
			Message:      fmt.Sprintf("simplify to '%s' (right side is always true)", leftExpr),
		}
	}

	return nil
}

// buildAndReplacement builds the replacement for general && cases.
func buildAndReplacement(leftExpr, rightExpr string, leftRep, rightRep *ReplacementCondition) *ReplacementCondition {
	if leftRep != nil || rightRep != nil {
		newCondition := leftExpr + " && " + rightExpr

		return &ReplacementCondition{
			NewCondition: newCondition,
			IsLiteral:    false,
			Message:      fmt.Sprintf("simplify to '%s'", newCondition),
		}
	}

	return nil
}

// isLiteralExpr checks if an expression is a literal value.
func isLiteralExpr(expr string, rep *ReplacementCondition) bool {
	return rep != nil && rep.IsLiteral || expr == trueValue || expr == falseValue
}

// simplifyOrExpr simplifies || expressions.
func simplifyOrExpr(leftExpr, rightExpr string, leftRep, rightRep *ReplacementCondition) *ReplacementCondition {
	// Check for short-circuit cases first
	if result := checkOrShortCircuit(leftExpr, rightExpr, leftRep, rightRep); result != nil {
		return result
	}

	// Check for simplification cases
	if result := checkOrSimplification(leftExpr, rightExpr, leftRep, rightRep); result != nil {
		return result
	}

	// General case where simplification occurred
	return buildOrReplacement(leftExpr, rightExpr, leftRep, rightRep)
}

// checkOrShortCircuit checks for short-circuit cases in || expressions.
func checkOrShortCircuit(leftExpr, rightExpr string, leftRep, rightRep *ReplacementCondition) *ReplacementCondition {
	// true || X -> true (short circuit)
	if leftRep != nil && leftRep.IsLiteral && leftExpr == trueValue {
		return &ReplacementCondition{
			NewCondition: trueValue,
			IsLiteral:    true,
			Message:      "condition is always true (left side is true)",
		}
	}
	// X || true -> true (short circuit)
	if rightRep != nil && rightRep.IsLiteral && rightExpr == trueValue {
		return &ReplacementCondition{
			NewCondition: trueValue,
			IsLiteral:    true,
			Message:      "condition is always true (right side is true)",
		}
	}

	return nil
}

// checkOrSimplification checks for simplification cases in || expressions.
func checkOrSimplification(leftExpr, rightExpr string, leftRep, rightRep *ReplacementCondition) *ReplacementCondition {
	// false || X -> X
	if leftRep != nil && leftRep.IsLiteral && leftExpr == falseValue {
		isRightLiteral := isLiteralExpr(rightExpr, rightRep)

		return &ReplacementCondition{
			NewCondition: rightExpr,
			IsLiteral:    isRightLiteral,
			Message:      fmt.Sprintf("simplify to '%s' (left side is always false)", rightExpr),
		}
	}
	// X || false -> X
	if rightExpr == falseValue {
		isLeftLiteral := isLiteralExpr(leftExpr, leftRep)

		return &ReplacementCondition{
			NewCondition: leftExpr,
			IsLiteral:    isLeftLiteral,
			Message:      fmt.Sprintf("simplify to '%s' (right side is always false)", leftExpr),
		}
	}

	return nil
}

// buildOrReplacement builds the replacement for general || cases.
func buildOrReplacement(leftExpr, rightExpr string, leftRep, rightRep *ReplacementCondition) *ReplacementCondition {
	if leftRep != nil || rightRep != nil {
		newCondition := leftExpr + " || " + rightExpr

		return &ReplacementCondition{
			NewCondition: newCondition,
			IsLiteral:    false,
			Message:      fmt.Sprintf("simplify to '%s'", newCondition),
		}
	}

	return nil
}

// createConditionFix creates a diagnostic with appropriate fix for if statement.
func createConditionFix(stmt *ast.IfStmt, replacement *ReplacementCondition) *analysis.Diagnostic {
	if replacement.IsLiteral {
		// Handle literal true/false cases
		if replacement.NewCondition == trueValue {
			return createTrueConditionFix(stmt)
		}

		return createFalseConditionFix(stmt)
	}

	// Handle non-literal simplifications
	return &analysis.Diagnostic{
		Pos:     stmt.Cond.Pos(),
		Message: replacement.Message,
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: fmt.Sprintf("Replace condition with '%s'", replacement.NewCondition),
			TextEdits: []analysis.TextEdit{{
				Pos:     stmt.Cond.Pos(),
				End:     stmt.Cond.End(),
				NewText: []byte(replacement.NewCondition),
			}},
		}},
	}
}

// createTrueConditionFix handles if statements with always-true conditions.
func createTrueConditionFix(stmt *ast.IfStmt) *analysis.Diagnostic {
	if stmt.Else != nil {
		return &analysis.Diagnostic{
			Pos:     stmt.Pos(),
			Message: "condition is always true, else clause is unreachable",
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: "Replace with then clause",
				TextEdits: []analysis.TextEdit{{
					Pos:     stmt.Pos(),
					End:     stmt.End(),
					NewText: []byte(formatStmt(stmt.Body)),
				}},
			}},
		}
	}

	return &analysis.Diagnostic{
		Pos:     stmt.Pos(),
		Message: "condition is always true",
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: "Replace with then clause",
			TextEdits: []analysis.TextEdit{{
				Pos:     stmt.Pos(),
				End:     stmt.End(),
				NewText: []byte(formatStmt(stmt.Body)),
			}},
		}},
	}
}

// createFalseConditionFix handles if statements with always-false conditions.
func createFalseConditionFix(stmt *ast.IfStmt) *analysis.Diagnostic {
	if stmt.Else != nil {
		return &analysis.Diagnostic{
			Pos:     stmt.Pos(),
			Message: "condition is always false, then clause is unreachable",
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: "Replace with else clause",
				TextEdits: []analysis.TextEdit{{
					Pos:     stmt.Pos(),
					End:     stmt.End(),
					NewText: []byte(formatStmt(stmt.Else)),
				}},
			}},
		}
	}

	return &analysis.Diagnostic{
		Pos:     stmt.Pos(),
		Message: "condition is always false, remove entire if statement",
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: "Remove if statement",
			TextEdits: []analysis.TextEdit{{
				Pos:     stmt.Pos(),
				End:     stmt.End(),
				NewText: []byte(""),
			}},
		}},
	}
}

func formatExpr(expr ast.Expr) string {
	// Simple formatting - in practice you'd use go/format
	switch exprType := expr.(type) {
	case *ast.BinaryExpr:
		return fmt.Sprintf("%s %s %s", formatExpr(exprType.X), exprType.Op.String(), formatExpr(exprType.Y))
	case *ast.Ident:
		return exprType.Name
	default:
		return "expr"
	}
}

func formatStmt(stmt ast.Stmt) string {
	// Simple formatting - in practice you'd use go/format
	switch stmt.(type) {
	case *ast.BlockStmt:
		return "{\n\t// statements\n}"
	case *ast.IfStmt:
		return "if condition { /* ... */ }"
	default:
		return "/* statement */"
	}
}

func shouldIgnore(file *ast.File, node ast.Node, analyzerName string) bool {
	if file == nil {
		return false
	}

	return shouldIgnoreInFunction(file, node, analyzerName) || shouldIgnoreFromComment(file, node, analyzerName)
}

func shouldIgnoreInFunction(file *ast.File, node ast.Node, analyzerName string) bool {
	if file == nil {
		return false
	}

	for _, decl := range file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		if node.Pos() >= funcDecl.Pos() && node.End() <= funcDecl.End() {
			ignore := directive.ParseIgnore(funcDecl.Doc)
			if ignore != nil && ignore.ShouldIgnore(analyzerName) {
				return true
			}
		}
	}

	return false
}

func shouldIgnoreFromComment(file *ast.File, node ast.Node, analyzerName string) bool {
	if file == nil {
		return false
	}

	for _, cg := range file.Comments {
		// Check if comment appears before the node and is reasonably close
		if cg.End() <= node.Pos() && node.Pos()-cg.End() <= 200 {
			ignore := directive.ParseIgnore(cg)
			if ignore != nil && ignore.ShouldIgnore(analyzerName) {
				return true
			}
		}
	}

	return false
}
