package config

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Parser defines the interface for a configuration parser.
type Parser interface {
	// Parse reads configuration data from an io.Reader and unmarshals it into the provided 'v' interface{}.
	Parse(r io.Reader, v any) error
	// ParseFile reads configuration data from the given file path and unmarshals it into the provided 'v' interface{}.
	ParseFile(filePath string, v any) error
}

// NewParser creates a new parser based on the file extension.
func NewParser(filePath string) (Parser, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		return NewJSONParser(), nil
	case ".toml":
		return NewTOMLParser(), nil
	case ".yaml", ".yml":
		return NewYAMLParser(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}
