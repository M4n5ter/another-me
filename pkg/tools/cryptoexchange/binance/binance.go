package binancetool

import (
	"github.com/adshao/go-binance/v2"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// NewBinance 创建一个新的 Binance 客户端实例
func NewBinance(apiKey, secretKey string, proxy Option[string]) *binance.Client {
	var client *binance.Client
	if proxy.IsSome() {
		client = binance.NewProxiedClient(apiKey, secretKey, proxy.Unwrap())
	} else {
		client = binance.NewClient(apiKey, secretKey)
	}

	client.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0"

	return client
}
