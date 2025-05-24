package config

import (
	"reflect"
	"testing"
)

func TestNewParser(t *testing.T) {
	tests := []struct {
		filePath   string
		wantType   any
		wantErr    bool
		expectedID string // To differentiate in test names
	}{
		{"config.json", &JSONParser{}, false, "JSON"},
		{"config.toml", &TOMLParser{}, false, "TOML"},
		{"config.yaml", &YAMLParser{}, false, "YAML"},
		{"config.yml", &YAMLParser{}, false, "YML"},
		{"config.txt", nil, true, "Unsupported"},
		{".json", &JSONParser{}, false, "ExtOnlyJSON"},
		{"nodir/config.json", &JSONParser{}, false, "NoDirJSON"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedID, func(t *testing.T) {
			parser, err := NewParser(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && reflect.TypeOf(parser) != reflect.TypeOf(tt.wantType) {
				t.Errorf("NewParser() got type %T, want type %T", parser, tt.wantType)
			}
		})
	}
}
