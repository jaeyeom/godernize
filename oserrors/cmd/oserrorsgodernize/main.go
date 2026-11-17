// Command oserrorsgodernize runs the oserrors analyzer.
package main

import (
	"log"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/jaeyeom/godernize/internal/config"
	"github.com/jaeyeom/godernize/oserrors"
)

func main() {
	analyzers := []*analysis.Analyzer{oserrors.Analyzer}
	if err := config.ApplyInWorkingDirectory(analyzers, config.IgnoreUnknownAnalyzers); err != nil {
		log.SetFlags(0)
		log.SetPrefix(filepath.Base(os.Args[0]) + ": ")
		log.Fatal(err)
	}

	singlechecker.Main(oserrors.Analyzer)
}
