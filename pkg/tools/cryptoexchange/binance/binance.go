package binancetool

import (
	"context"
	"fmt"
	"log/slog"

	ccxt "github.com/ccxt/ccxt/go/v4"
	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// BinanceTool 用于与 Binance 交易所交互
type BinanceTool struct {
	apiKey    Option[string]
	apiSecret Option[string]
	client    ccxt.Binance
	i18nMgr   *i18n.Manager
	logger    *slog.Logger
}

// NewBinance 创建一个新的 BinanceTool 实例
func NewBinance(i18nMgr *i18n.Manager, apiKey, apiSecret, proxy Option[string]) *BinanceTool {
	config := make(map[string]any, 8)

	if apiKey.IsSome() {
		config["apiKey"] = apiKey.Unwrap()
	}

	if apiSecret.IsSome() {
		config["secret"] = apiSecret.Unwrap()
	}

	if proxy.IsSome() {
		config["proxy"] = proxy.Unwrap()
	}

	config["timeout"] = 30000
	config["enableRateLimit"] = true
	config["userAgent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0"

	return &BinanceTool{apiKey: apiKey, apiSecret: apiSecret, client: ccxt.NewBinance(config), i18nMgr: i18nMgr, logger: slog.Default().WithGroup("binance_tool")}
}

// FetchMarkets 获取市场信息
func (t *BinanceTool) FetchMarkets(ctx context.Context, symbols []string) (string, error) {
	params := make(map[string]any, 1)
	symbolsJSON, err := json.MarshalToString(symbols)
	if err != nil {
		return "", fmt.Errorf("failed to marshal symbols: %w", err)
	}
	params["symbols"] = symbolsJSON

	markets, err := t.client.FetchMarkets(params)
	if err != nil {
		return "", fmt.Errorf("failed to fetch markets: %w", err)
	}

	marketsJSON, err := json.MarshalToString(common.SanitizeValue(markets))
	if err != nil {
		return "", fmt.Errorf("failed to marshal markets: %w", err)
	}
	return marketsJSON, nil
}
