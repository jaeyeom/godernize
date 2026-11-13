// Command godernizecheck runs the godernize analyzer.
package main

import (
	"log"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/jaeyeom/godernize/ctxnil"
	"github.com/jaeyeom/godernize/internal/config"
	"github.com/jaeyeom/godernize/oserrors"
)

func main() {
	analyzers := []*analysis.Analyzer{
		ctxnil.Analyzer,
		oserrors.Analyzer,
	}
	if err := config.ApplyInWorkingDirectory(analyzers, config.RejectUnknownAnalyzers); err != nil {
		log.SetFlags(0)
		log.SetPrefix(filepath.Base(os.Args[0]) + ": ")
		log.Fatal(err)
	}

	multichecker.Main(analyzers...)
}
