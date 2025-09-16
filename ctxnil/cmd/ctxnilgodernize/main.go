// Command ctxnilgodernize runs the ctxnil analyzer.
package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/jaeyeom/godernize/ctxnil"
)

func main() {
	singlechecker.Main(ctxnil.Analyzer)
}
