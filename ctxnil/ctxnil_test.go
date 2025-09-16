package ctxnil_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/jaeyeom/godernize/ctxnil"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, ctxnil.Analyzer, "a")
}
