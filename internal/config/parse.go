package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	fileVersion = 1
	versionKey  = "version"

	// decimalBase is the base passed to strconv.FormatInt.
	decimalBase = 10

	// floatFormatPrecision tells strconv.FormatFloat to use the smallest number of digits.
	floatFormatPrecision = -1

	// floatBitSize is the precision passed to strconv.FormatFloat.
	floatBitSize = 64
)

// Parse decodes a .godernize.yml document.
// The path is included in errors and stored on the returned Settings.
func Parse(path string, data []byte) (Settings, error) {
	flags, err := parseDocument(path, data)
	if err != nil {
		return Settings{}, err
	}

	return Settings{Path: path, Flags: flags}, nil
}

func parseDocument(path string, data []byte) (map[string]map[string]string, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	var root yaml.Node

	err := decoder.Decode(&root)
	if errors.Is(err, io.EOF) {
		return nil, versionRequired(path)
	}

	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	if err := oneDocument(decoder, path); err != nil {
		return nil, err
	}

	return parseRoot(path, &root)
}

func oneDocument(decoder *yaml.Decoder, path string) error {
	var extra yaml.Node

	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	return fmt.Errorf("%s: multiple documents", path)
}

func parseRoot(path string, root *yaml.Node) (map[string]map[string]string, error) {
	if root.Kind != yaml.DocumentNode || len(root.Content) != 1 {
		return nil, notMapping(path)
	}

	body := root.Content[0]
	if isAlias(body) {
		return nil, aliasErr(path)
	}

	if isNull(body) {
		return nil, versionRequired(path)
	}

	if body.Kind != yaml.MappingNode {
		return nil, notMapping(path)
	}

	return parseBody(path, body)
}

func parseBody(path string, body *yaml.Node) (map[string]map[string]string, error) {
	pairs, err := mappingPairs(path, body)
	if err != nil {
		return nil, err
	}

	flags := make(map[string]map[string]string)
	sawVersion := false

	for _, item := range pairs {
		if item.key == "<<" || isAlias(item.value) {
			return nil, aliasErr(path)
		}

		if item.key == versionKey {
			sawVersion = true

			if err := checkVersion(path, item.value); err != nil {
				return nil, err
			}

			continue
		}

		section, err := parseSection(path, item.key, item.value)
		if err != nil {
			return nil, err
		}

		flags[item.key] = section
	}

	if !sawVersion {
		return nil, versionRequired(path)
	}

	return flags, nil
}

func checkVersion(path string, node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!int" {
		return fmt.Errorf("%s: version must be 1", path)
	}

	var number int64
	if err := node.Decode(&number); err != nil || number != fileVersion {
		return fmt.Errorf("%s: version must be 1", path)
	}

	return nil
}

type pair struct {
	key   string
	value *yaml.Node
}

func mappingPairs(path string, node *yaml.Node) ([]pair, error) {
	if isAlias(node) {
		return nil, aliasErr(path)
	}

	if node.Kind != yaml.MappingNode || len(node.Content)%2 != 0 {
		return nil, notMapping(path)
	}

	seen := make(map[string]struct{}, len(node.Content)/2)
	pairs := make([]pair, 0, len(node.Content)/2)

	for index := 0; index < len(node.Content); index += 2 {
		key, err := mappingKey(path, node.Content[index])
		if err != nil {
			return nil, err
		}

		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%s: duplicate key %q", path, key)
		}

		seen[key] = struct{}{}

		pairs = append(pairs, pair{key: key, value: node.Content[index+1]})
	}

	return pairs, nil
}

func mappingKey(path string, node *yaml.Node) (string, error) {
	if isAlias(node) || node.Tag == "!!merge" || node.Value == "<<" {
		return "", aliasErr(path)
	}

	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return "", fmt.Errorf("%s: key must be a string", path)
	}

	return node.Value, nil
}

func parseSection(path, name string, node *yaml.Node) (map[string]string, error) {
	if isAlias(node) {
		return nil, aliasErr(path)
	}

	if isNull(node) {
		return map[string]string{}, nil
	}

	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: analyzer %q must be a mapping", path, name)
	}

	pairs, err := mappingPairs(path, node)
	if err != nil {
		return nil, err
	}

	flags := make(map[string]string, len(pairs))

	for _, item := range pairs {
		text, err := parseFlag(path, name, item.key, item.value)
		if err != nil {
			return nil, err
		}

		flags[item.key] = text
	}

	return flags, nil
}

func parseFlag(path, analyzer, name string, node *yaml.Node) (string, error) {
	if isAlias(node) {
		return "", aliasErr(path)
	}

	if text, ok := scalarText(node); ok {
		return text, nil
	}

	if node.Kind == yaml.SequenceNode {
		return parseList(path, analyzer, name, node)
	}

	if node.Kind == yaml.ScalarNode {
		return "", fmt.Errorf("%s: analyzer %q flag %q has an unsupported value", path, analyzer, name)
	}

	return "", fmt.Errorf("%s: analyzer %q flag %q must be a scalar or a list of scalars", path, analyzer, name)
}

func parseList(path, analyzer, name string, node *yaml.Node) (string, error) {
	parts := make([]string, 0, len(node.Content))

	for _, item := range node.Content {
		if isAlias(item) {
			return "", aliasErr(path)
		}

		text, ok := scalarText(item)
		if !ok {
			return "", fmt.Errorf("%s: analyzer %q flag %q list item must be a scalar", path, analyzer, name)
		}

		if strings.Contains(text, ",") {
			return "", fmt.Errorf(
				"%s: analyzer %q flag %q list item %q contains a comma",
				path,
				analyzer,
				name,
				text,
			)
		}

		parts = append(parts, text)
	}

	return strings.Join(parts, ","), nil
}

func scalarText(node *yaml.Node) (string, bool) {
	if node == nil || node.Kind != yaml.ScalarNode {
		return "", false
	}

	switch node.Tag {
	case "!!str":
		return node.Value, true
	case "!!bool":
		return boolText(node)
	case "!!int":
		return intText(node)
	case "!!float":
		return floatText(node)
	default:
		return "", false
	}
}

func boolText(node *yaml.Node) (string, bool) {
	var value bool
	if err := node.Decode(&value); err != nil {
		return "", false
	}

	return strconv.FormatBool(value), true
}

func intText(node *yaml.Node) (string, bool) {
	var value int64
	if err := node.Decode(&value); err != nil {
		return "", false
	}

	return strconv.FormatInt(value, decimalBase), true
}

func floatText(node *yaml.Node) (string, bool) {
	var value float64
	if err := node.Decode(&value); err != nil {
		return "", false
	}

	return strconv.FormatFloat(value, 'f', floatFormatPrecision, floatBitSize), true
}

func isAlias(node *yaml.Node) bool {
	return node != nil && (node.Kind == yaml.AliasNode || node.Alias != nil)
}

func isNull(node *yaml.Node) bool {
	return node != nil && node.Kind == yaml.ScalarNode && node.Tag == "!!null"
}

func versionRequired(path string) error {
	return fmt.Errorf("%s: version is required", path)
}

func notMapping(path string) error {
	return fmt.Errorf("%s: document must be a mapping", path)
}

func aliasErr(path string) error {
	return fmt.Errorf("%s: YAML aliases and merge keys are not supported", path)
}
