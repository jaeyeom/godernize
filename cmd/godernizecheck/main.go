// Command godernizecheck runs the godernize analyzer.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/jaeyeom/godernize/oserrors"
)

func main() {
	multichecker.Main(
		oserrors.Analyzer,
	)
}
