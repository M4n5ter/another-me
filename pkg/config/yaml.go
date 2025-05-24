package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// YAMLParser 实现 YAML 配置文件的解析器
type YAMLParser struct{}

// NewYAMLParser 创建新的 YAMLParser
func NewYAMLParser() *YAMLParser {
	return &YAMLParser{}
}

// Parse 从 io.Reader 读取 YAML 配置数据并反序列化到 'v' 接口
func (p *YAMLParser) Parse(r io.Reader, v any) error {
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("yaml.Decode: %w", err)
	}
	return nil
}

// ParseFile 从文件路径读取 YAML 配置数据并反序列化到 'v' 接口
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
