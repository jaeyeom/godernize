// Package config reads .godernize.yml from the module that contains the
// working directory and applies it to analyzer flags.
package config

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"golang.org/x/tools/go/analysis"
)

const fileName = ".godernize.yml"

// UnknownAnalyzers controls sections for analyzers this process is not running.
type UnknownAnalyzers int

const (
	// RejectUnknownAnalyzers reports an analyzer name outside the running set.
	RejectUnknownAnalyzers UnknownAnalyzers = iota

	// IgnoreUnknownAnalyzers skips sections for analyzers that are not running.
	IgnoreUnknownAnalyzers
)

// Settings is the flag values read from a .godernize.yml file.
type Settings struct {
	// Path is the file that was read. It is empty when no file applies.
	Path string

	// Flags maps an analyzer name to flag names and their string values.
	Flags map[string]map[string]string
}

// Load reads .godernize.yml beside the nearest go.mod above dir.
// It returns the zero Settings when dir is outside a module or the file is absent.
func Load(dir string) (Settings, error) {
	root, found, err := moduleRoot(dir)
	if err != nil || !found {
		return Settings{}, err
	}

	path := filepath.Join(root, fileName)

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Settings{}, nil
	}

	if err != nil {
		return Settings{}, fmt.Errorf("%s: %w", path, err)
	}

	return Parse(path, data)
}

// ApplyInWorkingDirectory loads .godernize.yml from the working directory and applies it.
func ApplyInWorkingDirectory(analyzers []*analysis.Analyzer, unknown UnknownAnalyzers) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("working directory: %w", err)
	}

	settings, err := Load(dir)
	if err != nil {
		return err
	}

	return Apply(analyzers, settings, unknown)
}

// Apply sets analyzer flags from settings.
//
// Call it before multichecker.Main or singlechecker.Main. Those drivers
// register the same flag.Value and then parse argv, so a command-line flag
// overrides the file. Help text shows the file value as the effective default.
func Apply(analyzers []*analysis.Analyzer, settings Settings, unknown UnknownAnalyzers) error {
	byName, err := analyzerMap(analyzers)
	if err != nil {
		return err
	}

	path := settings.Path
	if path == "" {
		path = fileName
	}

	for _, name := range slices.Sorted(maps.Keys(settings.Flags)) {
		analyzer, ok := byName[name]
		if !ok {
			if unknown == IgnoreUnknownAnalyzers {
				continue
			}

			return fmt.Errorf("%s: unknown analyzer %q", path, name)
		}

		if err := applyFlags(path, analyzer, settings.Flags[name]); err != nil {
			return err
		}
	}

	return nil
}

func analyzerMap(analyzers []*analysis.Analyzer) (map[string]*analysis.Analyzer, error) {
	byName := make(map[string]*analysis.Analyzer, len(analyzers))

	for _, analyzer := range analyzers {
		if analyzer == nil {
			return nil, errors.New("config: nil analyzer")
		}

		if _, exists := byName[analyzer.Name]; exists {
			return nil, fmt.Errorf("config: duplicate analyzer %q", analyzer.Name)
		}

		byName[analyzer.Name] = analyzer
	}

	return byName, nil
}

func applyFlags(path string, analyzer *analysis.Analyzer, flags map[string]string) error {
	for _, name := range slices.Sorted(maps.Keys(flags)) {
		if analyzer.Flags.Lookup(name) == nil {
			return fmt.Errorf("%s: analyzer %q has no flag %q", path, analyzer.Name, name)
		}

		if err := analyzer.Flags.Set(name, flags[name]); err != nil {
			return fmt.Errorf("%s: analyzer %q flag %q: %w", path, analyzer.Name, name, err)
		}
	}

	return nil
}

func moduleRoot(dir string) (string, bool, error) {
	current, err := filepath.Abs(dir)
	if err != nil {
		return "", false, fmt.Errorf("working directory: %w", err)
	}

	for {
		modPath := filepath.Join(current, "go.mod")

		info, err := os.Stat(modPath)
		if err == nil && info.Mode().IsRegular() {
			return current, true, nil
		}

		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("%s: %w", modPath, err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false, nil
		}

		current = parent
	}
}
