package config

import (
	"fmt"
	"io"
	"os"

	json "github.com/json-iterator/go"
)

// JSONParser 实现 JSON 配置文件的解析器
type JSONParser struct{}

// NewJSONParser 创建新的 JSONParser
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// Parse 从 io.Reader 读取 JSON 配置数据并反序列化到 'v' 接口
func (p *JSONParser) Parse(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("json.Decode: %w", err)
	}
	return nil
}

// ParseFile 从文件路径读取 JSON 配置数据并反序列化到 'v' 接口
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
