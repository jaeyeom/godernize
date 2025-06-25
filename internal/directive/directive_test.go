package directive_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/jaeyeom/godernize/internal/directive"
)

type parseIgnoreTestCase struct {
	comment  string
	expected *directive.Ignore
}

func TestParseIgnore(t *testing.T) {
	t.Parallel()

	tests := []parseIgnoreTestCase{
		{"//godernize:ignore", &directive.Ignore{}},
		{"//godernize:ignore=oserrors", &directive.Ignore{Names: []string{"oserrors"}}},
		{"//godernize:ignore=IsNotExist", &directive.Ignore{Names: []string{"IsNotExist"}}},
		{"//godernize:ignore=oserrors,IsNotExist", &directive.Ignore{Names: []string{"oserrors", "IsNotExist"}}},
		{"// some other comment", nil},
	}

	for _, test := range tests {
		t.Run(test.comment, func(t *testing.T) {
			t.Parallel()

			commentGroup := parseCommentFromSource(t, test.comment)
			result := directive.ParseIgnore(commentGroup)
			assertIgnoreResult(t, test.expected, result, test.comment)
		})
	}
}

func parseCommentFromSource(t *testing.T, comment string) *ast.CommentGroup {
	t.Helper()

	src := "package test\n\n" + comment + "\nfunc f() {}"

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(file.Comments) > 0 {
		return file.Comments[0]
	}

	return nil
}

func assertIgnoreResult(t *testing.T, expected *directive.Ignore, result *directive.Ignore, comment string) {
	t.Helper()

	if expected == nil && result != nil {
		t.Errorf("Expected nil for %q, got %+v", comment, result)

		return
	}

	if expected != nil && result == nil {
		t.Errorf("Expected %+v for %q, got nil", expected, comment)

		return
	}

	if expected != nil && result != nil {
		assertNamesMatch(t, expected.Names, result.Names, comment)
	}
}

func assertNamesMatch(t *testing.T, expectedNames, resultNames []string, comment string) {
	t.Helper()

	if len(expectedNames) != len(resultNames) {
		t.Errorf("Expected %d names, got %d for %q", len(expectedNames), len(resultNames), comment)

		return
	}

	for i, name := range expectedNames {
		if resultNames[i] != name {
			t.Errorf("Expected name %q at index %d, got %q for %q", name, i, resultNames[i], comment)
		}
	}
}
