package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// YAMLParser implements the Parser interface for YAML configuration files.
type YAMLParser struct{}

// NewYAMLParser creates a new YAMLParser.
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{}
}

// Parse reads YAML configuration data from an io.Reader and unmarshals it.
func (p *YAMLParser) Parse(r io.Reader, v any) error {
	decoder := yaml.NewDecoder(r)
	return fmt.Errorf("yaml.Decode: %w", decoder.Decode(v))
}

// ParseFile reads YAML configuration data from a file and unmarshals it.
func (p *YAMLParser) ParseFile(filePath string, v any) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("os.Open: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("close file error: %v\n", err)
		}
	}()
	return p.Parse(file, v)
}
