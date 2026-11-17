package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jaeyeom/godernize/internal/config"
)

func settingsZero(got config.Settings) bool {
	return got.Path == "" && len(got.Flags) == 0
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func writeModule(t *testing.T, dir, yml string) {
	t.Helper()

	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/fixture\n\ngo 1.26.0\n")

	if yml != "" {
		writeFile(t, filepath.Join(dir, ".godernize.yml"), yml)
	}
}
