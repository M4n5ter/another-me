package config

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// Parser 定义配置解析器
type Parser interface {
	// Parse 从 io.Reader 读取配置数据并反序列化到 'v' 接口
	Parse(r io.Reader, v any) error
	// ParseFile 从文件路径读取配置数据并反序列化到 'v' 接口
	ParseFile(filePath string, v any) error
}

// NewParser 根据文件扩展名创建新的解析器
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
