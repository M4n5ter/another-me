package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type TestSettings struct {
	Host    string `json:"host" toml:"host" yaml:"host"`
	Port    int    `json:"port" toml:"port" yaml:"port"`
	Enabled bool   `json:"enabled" toml:"enabled" yaml:"enabled"`
}

type TestConfig struct {
	Name     string       `json:"name" toml:"name" yaml:"name"`
	Version  string       `json:"version" toml:"version" yaml:"version"`
	Settings TestSettings `json:"settings" toml:"settings" yaml:"settings"`
}

func TestJSONParser_Parse(t *testing.T) {
	parser := NewJSONParser()
	jsonContent := `{
		"name": "TestApp",
		"version": "1.0",
		"settings": {
			"host": "example.com",
			"port": 80,
			"enabled": true
		}
	}`

	var config TestConfig
	err := parser.Parse(strings.NewReader(jsonContent), &config)
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

func TestJSONParser_ParseFile(t *testing.T) {
	parser := NewJSONParser()
	filePath := filepath.Join("testdata", "config.json")

	// Create a dummy config.json for testing
	// content, err := os.ReadFile(filePath)
	// if err != nil {
	// 	t.Fatalf("Failed to read original json file: %v", err)
	// }
	// t.Logf("Original JSON content: %s", string(content))

	var config TestConfig
	err := parser.ParseFile(filePath, &config)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}

	expected := TestConfig{
		Name:    "TestAppJSON",
		Version: "1.0.json",
		Settings: TestSettings{
			Host:    "localhost",
			Port:    8080,
			Enabled: true,
		},
	}

	if !reflect.DeepEqual(config, expected) {
		t.Errorf("ParseFile() got = %v, want %v", config, expected)
	}
}

func TestJSONParser_ParseFile_NonExistent(t *testing.T) {
	parser := NewJSONParser()
	filePath := filepath.Join("testdata", "nonexistent.json")

	var config TestConfig
	err := parser.ParseFile(filePath, &config)
	if err == nil {
		t.Fatalf("ParseFile() expected error for non-existent file, got nil")
	}
}
