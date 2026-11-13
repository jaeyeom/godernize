package config_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/jaeyeom/godernize/internal/config"
)

func TestLoadFromSubdirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeModule(t, root, "version: 1\npkgvar:\n  allowlist:\n    - errors.New\n")

	got, err := config.Load(filepath.Join(root, "sub", "pkg"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := map[string]map[string]string{
		"pkgvar": {"allowlist": "errors.New"},
	}
	if got.Path != filepath.Join(root, ".godernize.yml") || !sameFlags(got.Flags, want) {
		t.Fatalf("Load = %#v, want path beside %s and flags %#v", got, root, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeModule(t, root, "")

	got, err := config.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !settingsZero(got) {
		t.Fatalf("Load = %#v, want zero", got)
	}
}

func TestLoadOutsideModule(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".godernize.yml"), "version: 1\n")

	got, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !settingsZero(got) {
		t.Fatalf("Load = %#v, want zero", got)
	}
}

func TestLoadNearerModuleHidesOuterFile(t *testing.T) {
	t.Parallel()

	outer := t.TempDir()
	writeModule(t, outer, "version: 1\npkgvar:\n  allowlist: outer\n")

	inner := filepath.Join(outer, "inner")
	writeModule(t, inner, "")

	got, err := config.Load(filepath.Join(inner, "sub"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !settingsZero(got) {
		t.Fatalf("Load = %#v, want zero", got)
	}
}

func TestLoadNearerModuleWins(t *testing.T) {
	t.Parallel()

	outer := t.TempDir()
	writeModule(t, outer, "version: 1\npkgvar:\n  allowlist: outer\n")

	inner := filepath.Join(outer, "inner")
	writeModule(t, inner, "version: 1\npkgvar:\n  allowlist: inner\n")

	got, err := config.Load(inner)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := map[string]map[string]string{
		"pkgvar": {"allowlist": "inner"},
	}
	if got.Path != filepath.Join(inner, ".godernize.yml") || !sameFlags(got.Flags, want) {
		t.Fatalf("Load = %#v, want inner flags %#v", got, want)
	}
}

func TestLoadInvalidFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeModule(t, root, "pkgvar: {}\n")

	_, err := config.Load(root)
	if err == nil {
		t.Fatal("Load error = nil")
	}

	want := filepath.Join(root, ".godernize.yml") + ": version is required"
	if err.Error() != want {
		t.Fatalf("Load error = %q, want %q", err.Error(), want)
	}
}

func TestApplyInWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	writeModule(t, root, "version: 1\npkgvar:\n  allowlist: from-file\n")

	sub := filepath.Join(root, "sub")
	writeFile(t, filepath.Join(sub, "keep.txt"), "ok\n")
	t.Chdir(sub)

	analyzer := &analysis.Analyzer{Name: "pkgvar"}
	analyzer.Flags.String("allowlist", "builtin", "allowlist")

	err := config.ApplyInWorkingDirectory(
		[]*analysis.Analyzer{analyzer},
		config.RejectUnknownAnalyzers,
	)
	if err != nil {
		t.Fatalf("ApplyInWorkingDirectory: %v", err)
	}

	got := analyzer.Flags.Lookup("allowlist").Value.String()
	if got != "from-file" {
		t.Fatalf("allowlist = %q, want from-file", got)
	}
}
