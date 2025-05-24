package config

import (
	"fmt"
	"io"
	"os"

	"github.com/BurntSushi/toml"
)

// TOMLParser implements the Parser interface for TOML configuration files.
type TOMLParser struct{}

// NewTOMLParser creates a new TOMLParser.
func NewTOMLParser() *TOMLParser {
	return &TOMLParser{}
}

// Parse reads TOML configuration data from an io.Reader and unmarshals it.
func (p *TOMLParser) Parse(r io.Reader, v any) error {
	_, err := toml.NewDecoder(r).Decode(v)
	return fmt.Errorf("toml.Decode: %w", err)
}

// ParseFile reads TOML configuration data from a file and unmarshals it.
func (p *TOMLParser) ParseFile(filePath string, v any) error {
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
