package config_test

import (
	"strings"
	"testing"

	"github.com/jaeyeom/godernize/internal/config"
)

func TestParse(t *testing.T) {
	t.Parallel()

	const path = "example.yml"

	tests := []struct {
		name    string
		data    string
		flags   map[string]map[string]string
		wantErr string
	}{
		{
			name:  "version only",
			data:  "version: 1\n",
			flags: map[string]map[string]string{},
		},
		{
			name:  "hex version",
			data:  "version: 0x1\n",
			flags: map[string]map[string]string{},
		},
		{
			name: "scalars and lists",
			data: `
version: 1
pkgvar:
  enabled: true
  count: 0x10
  ratio: 1.5
  allowlist: errors.New,fmt.Errorf
  names:
    - errors.New
    - fmt.Errorf
  bits: [true, 2]
  empty: []
  label: "01"
`,
			flags: map[string]map[string]string{
				"pkgvar": {
					"enabled":   "true",
					"count":     "16",
					"ratio":     "1.5",
					"allowlist": "errors.New,fmt.Errorf",
					"names":     "errors.New,fmt.Errorf",
					"bits":      "true,2",
					"empty":     "",
					"label":     "01",
				},
			},
		},
		{
			name: "null and empty sections",
			data: `
version: 1
pkgvar: null
oserrors: {}
`,
			flags: map[string]map[string]string{
				"pkgvar":   {},
				"oserrors": {},
			},
		},
		{
			name:    "empty file",
			data:    "",
			wantErr: "example.yml: version is required",
		},
		{
			name:    "comments only",
			data:    "# nothing\n",
			wantErr: "example.yml: version is required",
		},
		{
			name:    "missing version",
			data:    "pkgvar:\n  allowlist: errors.New\n",
			wantErr: "example.yml: version is required",
		},
		{
			name:    "string version",
			data:    "version: \"1\"\n",
			wantErr: "example.yml: version must be 1",
		},
		{
			name:    "float version",
			data:    "version: 1.0\n",
			wantErr: "example.yml: version must be 1",
		},
		{
			name:    "wrong version",
			data:    "version: 2\n",
			wantErr: "example.yml: version must be 1",
		},
		{
			name:    "list document",
			data:    "- version\n",
			wantErr: "example.yml: document must be a mapping",
		},
		{
			name:    "multiple documents",
			data:    "version: 1\n---\nversion: 1\n",
			wantErr: "example.yml: multiple documents",
		},
		{
			name:    "duplicate key",
			data:    "version: 1\nversion: 1\n",
			wantErr: "example.yml: duplicate key \"version\"",
		},
		{
			name:    "non-string key",
			data:    "version: 1\n1: {}\n",
			wantErr: "example.yml: key must be a string",
		},
		{
			name: "alias",
			data: `
version: 1
base: &base
  allowlist: errors.New
pkgvar: *base
`,
			wantErr: "example.yml: YAML aliases and merge keys are not supported",
		},
		{
			name: "merge key",
			data: `
version: 1
base: &base
  allowlist: errors.New
pkgvar:
  <<: *base
`,
			wantErr: "example.yml: YAML aliases and merge keys are not supported",
		},
		{
			name: "section must be a mapping",
			data: `
version: 1
pkgvar: true
`,
			wantErr: "example.yml: analyzer \"pkgvar\" must be a mapping",
		},
		{
			name: "nested mapping",
			data: `
version: 1
pkgvar:
  allowlist:
    names:
      - errors.New
`,
			wantErr: "example.yml: analyzer \"pkgvar\" flag \"allowlist\" must be a scalar or a list of scalars",
		},
		{
			name: "nested list",
			data: `
version: 1
pkgvar:
  allowlist:
    - [errors.New]
`,
			wantErr: "example.yml: analyzer \"pkgvar\" flag \"allowlist\" list item must be a scalar",
		},
		{
			name: "comma in list item",
			data: `
version: 1
pkgvar:
  allowlist:
    - errors.New,fmt.Errorf
`,
			wantErr: "example.yml: analyzer \"pkgvar\" flag \"allowlist\" list item \"errors.New,fmt.Errorf\" contains a comma",
		},
		{
			name: "null flag",
			data: `
version: 1
pkgvar:
  allowlist: null
`,
			wantErr: "example.yml: analyzer \"pkgvar\" flag \"allowlist\" has an unsupported value",
		},
		{
			name: "timestamp",
			data: `
version: 1
pkgvar:
  when: 2020-01-01
`,
			wantErr: "example.yml: analyzer \"pkgvar\" flag \"when\" has an unsupported value",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := config.Parse(path, []byte(test.data))
			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("Parse error = nil, want %q", test.wantErr)
				}

				if err.Error() != test.wantErr {
					t.Fatalf("Parse error = %q, want %q", err.Error(), test.wantErr)
				}

				if !settingsZero(got) {
					t.Fatalf("Parse settings = %#v, want zero", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			if got.Path != path {
				t.Errorf("Path = %q, want %q", got.Path, path)
			}

			if !sameFlags(got.Flags, test.flags) {
				t.Errorf("Flags = %#v, want %#v", got.Flags, test.flags)
			}
		})
	}
}

func TestParseSyntaxError(t *testing.T) {
	t.Parallel()

	_, err := config.Parse("example.yml", []byte("version: [\n"))
	if err == nil {
		t.Fatal("Parse error = nil")
	}

	if !strings.HasPrefix(err.Error(), "example.yml:") {
		t.Fatalf("Parse error = %q, want prefix example.yml:", err.Error())
	}
}

func sameFlags(got, want map[string]map[string]string) bool {
	if len(got) != len(want) {
		return false
	}

	for name, wantFlags := range want {
		gotFlags, ok := got[name]
		if !ok || len(gotFlags) != len(wantFlags) {
			return false
		}

		for flagName, wantValue := range wantFlags {
			if gotFlags[flagName] != wantValue {
				return false
			}
		}
	}

	return true
}
