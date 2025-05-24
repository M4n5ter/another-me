package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestYAMLParser_Parse(t *testing.T) {
	parser := NewYAMLParser()
	yamlContent := `
name: TestApp
version: "1.0"
settings:
  host: example.com
  port: 80
  enabled: true
`
	var config TestConfig
	err := parser.Parse(strings.NewReader(yamlContent), &config)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	expected := TestConfig{
		Name:    "TestApp",
		Version: "1.0",
		Settings: TestSettings{
			Host:    "example.com",
			Port:    80,
			Enabled: true,
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("Parse() got = %v, want %v", config, expected)
	}
}

func TestYAMLParser_ParseFile(t *testing.T) {
	parser := NewYAMLParser()
	filePath := filepath.Join("testdata", "config.yaml")

	var config TestConfig
	err := parser.ParseFile(filePath, &config)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	expected := TestConfig{
		Name:    "TestAppYAML",
		Version: "1.0.yaml",
		Settings: TestSettings{
			Host:    "localhost",
			Port:    8082,
			Enabled: true,
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("ParseFile() got = %v, want %v", config, expected)
	}
}

func TestYAMLParser_ParseFile_NonExistent(t *testing.T) {
	parser := NewYAMLParser()
	filePath := filepath.Join("testdata", "nonexistent.yaml")

	var config TestConfig
	err := parser.ParseFile(filePath, &config)
	if err == nil {
		t.Fatalf("ParseFile() expected error for non-existent file, got nil")
	}
}
