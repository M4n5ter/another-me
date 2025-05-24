package config

import (
	"fmt"
	"io"
	"os"

	"github.com/BurntSushi/toml"
)

// TOMLParser 实现 TOML 配置文件的解析器
type TOMLParser struct{}

// NewTOMLParser 创建新的 TOMLParser
func NewTOMLParser() *TOMLParser {
	return &TOMLParser{}
}

// Parse 从 io.Reader 读取 TOML 配置数据并反序列化到 'v' 接口
func (p *TOMLParser) Parse(r io.Reader, v any) error {
	if _, err := toml.NewDecoder(r).Decode(v); err != nil {
		return fmt.Errorf("toml.Decode: %w", err)
	}
	return nil
}

// ParseFile 从文件路径读取 TOML 配置数据并反序列化到 'v' 接口
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
