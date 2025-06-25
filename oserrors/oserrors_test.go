package oserrors_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/jaeyeom/godernize/oserrors"
)

func TestAnalyzer(t *testing.T) {
	t.Parallel()

	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, oserrors.Analyzer, "a")
}
