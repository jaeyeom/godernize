// Command oserrorsgodernize runs the oserrors analyzer.
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/jaeyeom/godernize/oserrors"
)

func main() {
	singlechecker.Main(oserrors.Analyzer)
}
