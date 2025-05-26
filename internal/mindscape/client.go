package mindscape

import (
	"fmt"
	"net/http"
	"time"

	"github.com/m4n5ter/another-me/internal/mindscape/sentinel"
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
}

// NewClient 创建一个 mindscape 客户端
func NewClient(config Config) *Client {
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	scheme := "http"
	if config.TLS {
		scheme = "https"
	}

	return &Client{Sentinel: sentinel.NewSentinel(httpClient, fmt.Sprintf("%s://%s:%d", scheme, config.Host, config.Port))}
}
