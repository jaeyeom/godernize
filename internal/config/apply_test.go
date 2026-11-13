package config_test

import (
	"flag"
	"io"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/jaeyeom/godernize/internal/config"
)

func TestApplySetsListedFlags(t *testing.T) {
	t.Parallel()

	analyzer := &analysis.Analyzer{Name: "pkgvar"}
	analyzer.Flags.String("allowlist", "builtin", "allowlist")
	analyzer.Flags.String("other", "keep", "other")

	settings := config.Settings{
		Path: "example.yml",
		Flags: map[string]map[string]string{
			"pkgvar": {"allowlist": ""},
		},
	}
	if err := config.Apply([]*analysis.Analyzer{analyzer}, settings, config.RejectUnknownAnalyzers); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if got := analyzer.Flags.Lookup("allowlist").Value.String(); got != "" {
		t.Fatalf("allowlist = %q, want empty", got)
	}

	if got := analyzer.Flags.Lookup("other").Value.String(); got != "keep" {
		t.Fatalf("other = %q, want keep", got)
	}
}

func TestApplyZeroSettingsKeepsDefaults(t *testing.T) {
	t.Parallel()

	analyzer := &analysis.Analyzer{Name: "pkgvar"}
	analyzer.Flags.String("allowlist", "builtin", "allowlist")

	err := config.Apply([]*analysis.Analyzer{analyzer}, config.Settings{}, config.RejectUnknownAnalyzers)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if got := analyzer.Flags.Lookup("allowlist").Value.String(); got != "builtin" {
		t.Fatalf("allowlist = %q, want builtin", got)
	}
}

func TestApplyEmptyPathUsesFileName(t *testing.T) {
	t.Parallel()

	analyzer := &analysis.Analyzer{Name: "pkgvar"}
	settings := config.Settings{Flags: map[string]map[string]string{"other": {}}}

	err := config.Apply([]*analysis.Analyzer{analyzer}, settings, config.RejectUnknownAnalyzers)
	want := `.godernize.yml: unknown analyzer "other"`
	if err == nil || err.Error() != want {
		t.Fatalf("Apply error = %v, want %q", err, want)
	}
}

func TestApplyUnknownAnalyzer(t *testing.T) {
	t.Parallel()

	analyzer := &analysis.Analyzer{Name: "pkgvar"}
	analyzer.Flags.String("allowlist", "builtin", "allowlist")
	settings := config.Settings{
		Path: "example.yml",
		Flags: map[string]map[string]string{
			"zeta":  {},
			"alpha": {},
		},
	}

	err := config.Apply([]*analysis.Analyzer{analyzer}, settings, config.RejectUnknownAnalyzers)
	want := `example.yml: unknown analyzer "alpha"`
	if err == nil || err.Error() != want {
		t.Fatalf("Apply error = %v, want %q", err, want)
	}

	err = config.Apply([]*analysis.Analyzer{analyzer}, settings, config.IgnoreUnknownAnalyzers)
	if err != nil {
		t.Fatalf("Apply ignore: %v", err)
	}

	if got := analyzer.Flags.Lookup("allowlist").Value.String(); got != "builtin" {
		t.Fatalf("allowlist = %q, want builtin", got)
	}
}

func TestApplyUnknownFlag(t *testing.T) {
	t.Parallel()

	analyzer := &analysis.Analyzer{Name: "pkgvar"}
	analyzer.Flags.String("allowlist", "builtin", "allowlist")
	settings := config.Settings{
		Path: "example.yml",
		Flags: map[string]map[string]string{
			"pkgvar": {"b": "1", "a": "1"},
			"other":  {"skip": "1"},
		},
	}

	err := config.Apply([]*analysis.Analyzer{analyzer}, settings, config.IgnoreUnknownAnalyzers)
	want := `example.yml: analyzer "pkgvar" has no flag "a"`
	if err == nil || err.Error() != want {
		t.Fatalf("Apply error = %v, want %q", err, want)
	}
}

func TestApplyBoolFlag(t *testing.T) {
	t.Parallel()

	analyzer := &analysis.Analyzer{Name: "pkgvar"}
	analyzer.Flags.Bool("enabled", false, "enabled")

	settings := config.Settings{
		Path:  "example.yml",
		Flags: map[string]map[string]string{"pkgvar": {"enabled": "true"}},
	}
	if err := config.Apply([]*analysis.Analyzer{analyzer}, settings, config.RejectUnknownAnalyzers); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if got := analyzer.Flags.Lookup("enabled").Value.String(); got != "true" {
		t.Fatalf("enabled = %q, want true", got)
	}

	bad := config.Settings{
		Path:  "example.yml",
		Flags: map[string]map[string]string{"pkgvar": {"enabled": "maybe"}},
	}

	err := config.Apply([]*analysis.Analyzer{analyzer}, bad, config.RejectUnknownAnalyzers)
	prefix := `example.yml: analyzer "pkgvar" flag "enabled": `
	if err == nil || !strings.HasPrefix(err.Error(), prefix) {
		t.Fatalf("Apply error = %v, want prefix %q", err, prefix)
	}
}

func TestApplyRejectsNilAndDuplicateAnalyzers(t *testing.T) {
	t.Parallel()

	err := config.Apply([]*analysis.Analyzer{nil}, config.Settings{}, config.RejectUnknownAnalyzers)
	if err == nil || err.Error() != "config: nil analyzer" {
		t.Fatalf("Apply error = %v, want nil analyzer", err)
	}

	first := &analysis.Analyzer{Name: "pkgvar"}
	second := &analysis.Analyzer{Name: "pkgvar"}

	err = config.Apply([]*analysis.Analyzer{first, second}, config.Settings{}, config.RejectUnknownAnalyzers)
	if err == nil || err.Error() != `config: duplicate analyzer "pkgvar"` {
		t.Fatalf("Apply error = %v, want duplicate analyzer", err)
	}
}

func TestCommandLineOverridesFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagName string
		args     []string
	}{
		{name: "multichecker name", flagName: "pkgvar.allowlist", args: []string{"-pkgvar.allowlist=from-cli"}},
		{name: "singlechecker name", flagName: "allowlist", args: []string{"-allowlist=from-cli"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			analyzer := &analysis.Analyzer{Name: "pkgvar"}
			analyzer.Flags.String("allowlist", "builtin", "allowlist")
			settings := config.Settings{
				Path:  "example.yml",
				Flags: map[string]map[string]string{"pkgvar": {"allowlist": "from-file"}},
			}
			if err := config.Apply([]*analysis.Analyzer{analyzer}, settings, config.RejectUnknownAnalyzers); err != nil {
				t.Fatalf("Apply: %v", err)
			}

			parseDriverFlag(t, analyzer, test.flagName, []string{"./..."})
			if got := analyzer.Flags.Lookup("allowlist").Value.String(); got != "from-file" {
				t.Fatalf("allowlist = %q, want from-file", got)
			}

			parseDriverFlag(t, analyzer, test.flagName, test.args)
			if got := analyzer.Flags.Lookup("allowlist").Value.String(); got != "from-cli" {
				t.Fatalf("allowlist = %q, want from-cli", got)
			}
		})
	}
}

func parseDriverFlag(t *testing.T, analyzer *analysis.Analyzer, name string, args []string) {
	t.Helper()

	set := flag.NewFlagSet("driver", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.Var(analyzer.Flags.Lookup("allowlist").Value, name, "allowlist")

	if err := set.Parse(args); err != nil {
		t.Fatalf("parse: %v", err)
	}
}
