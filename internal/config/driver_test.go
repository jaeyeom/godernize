package config_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDrivers(t *testing.T) {
	t.Parallel()

	bins := buildDrivers(t)

	cases := []struct {
		name     string
		yml      string
		driver   string
		fail     bool
		contains string
	}{
		{name: "version only check", yml: "version: 1\n", driver: "godernizecheck"},
		{name: "version only oserrors", yml: "version: 1\n", driver: "oserrorsgodernize"},
		{name: "version only ctxnil", yml: "version: 1\n", driver: "ctxnilgodernize"},
		{name: "missing file check", driver: "godernizecheck"},
		{name: "missing file oserrors", driver: "oserrorsgodernize"},
		{name: "missing file ctxnil", driver: "ctxnilgodernize"},
		{
			name:     "check rejects unknown analyzer",
			yml:      "version: 1\npkgvar:\n  allowlist:\n    - errors.New\n",
			driver:   "godernizecheck",
			fail:     true,
			contains: `unknown analyzer "pkgvar"`,
		},
		{
			name:   "oserrors ignores other analyzer",
			yml:    "version: 1\npkgvar:\n  allowlist:\n    - errors.New\n",
			driver: "oserrorsgodernize",
		},
		{
			name:   "ctxnil ignores other analyzer",
			yml:    "version: 1\npkgvar:\n  allowlist:\n    - errors.New\n",
			driver: "ctxnilgodernize",
		},
		{
			name:     "check rejects unknown flag",
			yml:      "version: 1\noserrors:\n  nope: true\n",
			driver:   "godernizecheck",
			fail:     true,
			contains: `analyzer "oserrors" has no flag "nope"`,
		},
		{
			name:     "oserrors rejects own flag",
			yml:      "version: 1\noserrors:\n  nope: true\n",
			driver:   "oserrorsgodernize",
			fail:     true,
			contains: `analyzer "oserrors" has no flag "nope"`,
		},
		{
			name:   "ctxnil ignores oserrors flag",
			yml:    "version: 1\noserrors:\n  nope: true\n",
			driver: "ctxnilgodernize",
		},
		{
			name:     "ctxnil rejects own flag",
			yml:      "version: 1\nctxnil:\n  nope: true\n",
			driver:   "ctxnilgodernize",
			fail:     true,
			contains: `analyzer "ctxnil" has no flag "nope"`,
		},
		{
			name:     "structural error fails check",
			yml:      "version: 1\npkgvar:\n  allowlist:\n    nested: true\n",
			driver:   "godernizecheck",
			fail:     true,
			contains: `must be a scalar or a list of scalars`,
		},
		{
			name:     "structural error fails oserrors",
			yml:      "version: 1\npkgvar:\n  allowlist:\n    nested: true\n",
			driver:   "oserrorsgodernize",
			fail:     true,
			contains: `must be a scalar or a list of scalars`,
		},
		{
			name:     "structural error fails ctxnil",
			yml:      "version: 1\npkgvar:\n  allowlist:\n    nested: true\n",
			driver:   "ctxnilgodernize",
			fail:     true,
			contains: `must be a scalar or a list of scalars`,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			writeModule(t, dir, test.yml)
			writeFile(t, filepath.Join(dir, "p.go"), "package p\n")
			expectRun(t, bins[test.driver], dir, test.fail, test.contains)
		})
	}
}

func buildDrivers(t *testing.T) map[string]string {
	t.Helper()

	root := repoRoot(t)
	outDir := t.TempDir()
	packages := map[string]string{
		"godernizecheck":    "./cmd/godernizecheck",
		"oserrorsgodernize": "./oserrors/cmd/oserrorsgodernize",
		"ctxnilgodernize":   "./ctxnil/cmd/ctxnilgodernize",
	}
	bins := make(map[string]string, len(packages))

	for name, pkg := range packages {
		out := filepath.Join(outDir, name)
		cmd := exec.Command("go", "build", "-o", out, pkg)
		cmd.Dir = root
		cmd.Env = toolEnv()

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("build %s: %v\n%s", name, err, output)
		}

		bins[name] = out
	}

	return bins
}

func expectRun(t *testing.T, bin, dir string, fail bool, contains string) {
	t.Helper()

	cmd := exec.Command(bin, "./...")
	cmd.Dir = dir
	cmd.Env = toolEnv()

	output, err := cmd.CombinedOutput()
	code := 0

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run: %v\n%s", err, output)
		}

		code = exitErr.ExitCode()
	}

	if fail && code == 0 {
		t.Fatalf("exit 0, want failure\n%s", output)
	}

	if !fail && code != 0 {
		t.Fatalf("exit %d, want 0\n%s", code, output)
	}

	if contains != "" && !strings.Contains(string(output), contains) {
		t.Fatalf("output %q does not contain %q", output, contains)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		info, statErr := os.Stat(filepath.Join(dir, "go.mod"))
		if statErr == nil && info.Mode().IsRegular() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}

		dir = parent
	}
}

func toolEnv() []string {
	base := os.Environ()
	env := make([]string, 0, len(base)+2)

	for _, entry := range base {
		if strings.HasPrefix(entry, "GOPACKAGESDRIVER=") || strings.HasPrefix(entry, "GOWORK=") {
			continue
		}

		env = append(env, entry)
	}

	return append(env, "GOPACKAGESDRIVER=", "GOWORK=off")
}
