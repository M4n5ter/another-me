package internal

import (
	"github.com/m4n5ter/another-me/internal/mindscape"
)

// Config 总配置
type Config struct {
	Host      string           `json:"host" toml:"host" yaml:"host"`
	Port      int              `json:"port" toml:"port" yaml:"port"`
	Mindscape mindscape.Config `json:"mindscape" toml:"mindscape" yaml:"mindscape"`
}
