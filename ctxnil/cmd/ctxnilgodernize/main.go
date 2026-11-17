// Command ctxnilgodernize runs the ctxnil analyzer.
package main

import (
	"log"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/jaeyeom/godernize/ctxnil"
	"github.com/jaeyeom/godernize/internal/config"
)

func main() {
	analyzers := []*analysis.Analyzer{ctxnil.Analyzer}
	if err := config.ApplyInWorkingDirectory(analyzers, config.IgnoreUnknownAnalyzers); err != nil {
		log.SetFlags(0)
		log.SetPrefix(filepath.Base(os.Args[0]) + ": ")
		log.Fatal(err)
	}

	singlechecker.Main(ctxnil.Analyzer)
}
