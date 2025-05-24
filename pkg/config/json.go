package config

import (
	"fmt"
	"io"
	"os"

	json "github.com/json-iterator/go"
)

// JSONParser implements the Parser interface for JSON configuration files.
type JSONParser struct{}

// NewJSONParser creates a new JSONParser.
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse reads JSON configuration data from an io.Reader and unmarshals it.
func (p *JSONParser) Parse(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	return fmt.Errorf("json.Decode: %w", decoder.Decode(v))
}

// ParseFile reads JSON configuration data from a file and unmarshals it.
func (p *JSONParser) ParseFile(filePath string, v any) error {
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
