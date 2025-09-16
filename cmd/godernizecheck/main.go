// Command godernizecheck runs the godernize analyzer.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/jaeyeom/godernize/ctxnil"
	"github.com/jaeyeom/godernize/oserrors"
)

func main() {
	multichecker.Main(
		ctxnil.Analyzer,
		oserrors.Analyzer,
	)
}
