package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTOMLParser_Parse(t *testing.T) {
	parser := NewTOMLParser()
	tomlContent := `
name = "TestApp"
version = "1.0"
[settings]
host = "example.com"
port = 80
enabled = true
`
	var config TestConfig
	err := parser.Parse(strings.NewReader(tomlContent), &config)
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

func TestTOMLParser_ParseFile(t *testing.T) {
	parser := NewTOMLParser()
	filePath := filepath.Join("testdata", "config.toml")

	var config TestConfig
	err := parser.ParseFile(filePath, &config)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	expected := TestConfig{
		Name:    "TestAppTOML",
		Version: "1.0.toml",
		Settings: TestSettings{
			Host:    "localhost",
			Port:    8081,
			Enabled: false,
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("ParseFile() got = %v, want %v", config, expected)
	}
}

func TestTOMLParser_ParseFile_NonExistent(t *testing.T) {
	parser := NewTOMLParser()
	filePath := filepath.Join("testdata", "nonexistent.toml")

	var config TestConfig
	err := parser.ParseFile(filePath, &config)
	if err == nil {
		t.Fatalf("ParseFile() expected error for non-existent file, got nil")
	}
}
