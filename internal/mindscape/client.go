package mindscape

import (
	"fmt"
	"net/http"
	"time"

	"github.com/m4n5ter/another-me/internal/mindscape/memory"
	"github.com/m4n5ter/another-me/internal/mindscape/sentinel"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// Config 是 mindscape 的配置
type Config struct {
	Host string `json:"host" toml:"host" yaml:"host"`
	Port int    `json:"port" toml:"port" yaml:"port"`
	TLS  bool   `json:"tls" toml:"tls" yaml:"tls"`
}

// Client 是 mindscape 的客户端
type Client struct {
	Sentinel *sentinel.Sentinel
	Memory   *memory.Memory
}

// NewClient 创建一个 mindscape 客户端
func NewClient(config Config) *Client {
	httpClient := Some(http.Client{
		Timeout: 10 * time.Second,
	})

	scheme := "http"
	if config.TLS {
		scheme = "https"
	}

	baseURL := fmt.Sprintf("%s://%s:%d", scheme, config.Host, config.Port)

	return &Client{
		Sentinel: sentinel.NewSentinel(httpClient, baseURL),
		Memory:   memory.NewMemory(httpClient, baseURL),
	}
}
