// Test cases for ctxnil analyzer
//
// Current analyzer capabilities:
// ✅ Direct context nil comparisons: ctx == nil, ctx != nil
// ✅ Simple boolean expressions: ctx != nil && ready
// ✅ If statements with context comparisons
// ✅ Binary expressions in assignments and function calls
// ✅ Context comparisons in switch cases
// ✅ Ignore directives: //godernize:ignore=ctxnil
//
// Limitations:
// ⚠️  Complex nested expressions may not simplify optimally
// ⚠️  Unary expressions (!(ctx == nil)) detect inner binary expressions
// ⚠️  Some parenthesized expressions produce different formatting

package a

import "context"

func testBasic(ctx context.Context) {
	// Basic nil comparisons
	if ctx == nil { // want "condition is always false, remove entire if statement"
		panic("ctx is nil")
	}

	if ctx != nil { // want "condition is always true"
		doSomething()
	}

	if nil == ctx { // want "condition is always false, remove entire if statement"
		return
	}

	// Complex boolean expressions with literal
	if ctx != nil && true { // want "condition is always true"
		doSomething()
	}

	if ctx == nil || false { // want "condition is always false, remove entire if statement"
		return
	}

	// If-else statements
	if ctx == nil { // want "condition is always false, then clause is unreachable"
		panic("unreachable")
	} else {
		doSomething()
	}

	if ctx != nil { // want "condition is always true, else clause is unreachable"
		doSomething()
	} else {
		panic("unreachable")
	}
}

//godernize:ignore=ctxnil
func ignoredFunction(ctx context.Context) {
	if ctx == nil { // This should be ignored
		return
	}
}

// Core functionality test cases
func testWithLogicalOperators(ctx context.Context) {
	var ready bool

	// Simple boolean expressions - these work perfectly
	if ctx != nil && ready { // want "simplify to 'ready' \\(left side is always true\\)"
		doSomething()
	}

	if ctx == nil || ready { // want "simplify to 'ready' \\(left side is always false\\)"
		doSomething()
	}

	if ready && ctx != nil { // want "simplify to 'ready' \\(right side is always true\\)"
		doSomething()
	}

	if ready || ctx == nil { // want "simplify to 'ready' \\(right side is always false\\)"
		doSomething()
	}
}

// Test different context variable names
func testDifferentContextNames(backgroundCtx context.Context, requestCtx context.Context) {
	var ready bool

	if backgroundCtx == nil { // want "condition is always false, remove entire if statement"
		return
	}

	if requestCtx != nil && ready { // want "simplify to 'ready' \\(left side is always true\\)"
		doSomething()
	}
}

// Test binary expressions outside if statements
func testBinaryExpressions(ctx context.Context) {
	var result bool

	// These should be detected as binary expressions
	result = ctx == nil // want "context should never be nil, replace 'ctx == nil' with 'false'"
	result = ctx != nil // want "context should never be nil, replace 'ctx != nil' with 'true'"
	result = nil == ctx // want "context should never be nil, replace 'nil == ctx' with 'false'"

	_ = result

	// In function calls
	doSomethingWithBool(ctx == nil) // want "context should never be nil, replace 'ctx == nil' with 'false'"
	doSomethingWithBool(ctx != nil) // want "context should never be nil, replace 'ctx != nil' with 'true'"
}

// Test ignore functionality
//
//godernize:ignore=ctxnil
func testIgnored(ctx context.Context) {
	var ready bool

	// All of these should be ignored
	if ctx == nil {
		return
	}

	if ctx != nil && ready {
		doSomething()
	}
}

func doSomething() {
	// implementation
}

func doSomethingWithBool(b bool) {
	// implementation
}
