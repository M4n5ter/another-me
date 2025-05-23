package binancetool

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/adshao/go-binance/v2"
	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// ListTickerPricesTool 获取交易对价格
type ListTickerPricesTool struct {
	service *binance.ListPricesService
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

// ListTickerPricesArgs 获取交易对价格参数
type ListTickerPricesArgs struct {
	Symbols []string `json:"symbols"`
}

// ListTickerPricesResult 获取交易对价格结果
type ListTickerPricesResult struct {
	SymbolPrices []*binance.SymbolPrice `json:"symbol_prices"`
}

// NewListTickerPricesTool 创建获取交易对价格工具
func NewListTickerPricesTool(i18nMgr *i18n.Manager, binanceTool *binance.Client) *ListTickerPricesTool {
	service := binanceTool.NewListPricesService()
	return &ListTickerPricesTool{service: service, logger: slog.Default().WithGroup("list_ticker_prices_tool"), i18nMgr: i18nMgr}
}

var _ toolcore.Tool = (*ListTickerPricesTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *ListTickerPricesTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.cryptoexchange.binance.list_ticker_prices.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.cryptoexchange.binance.list_ticker_prices.name", nil)
	}

	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "symbols", toolcore.ParamTypeArray, true, nil, "tool.cryptoexchange.binance.list_ticker_prices.arg.symbols", nil),
	}

	return toolcore.ToolSchema{
		Name:             "list_ticker_prices",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *ListTickerPricesTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args ListTickerPricesArgs
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("failed to unmarshal input JSON: %w", err)
	}

	symbolPrices, err := t.service.Symbols(args.Symbols).Do(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list ticker prices: %w", err)
	}

	resultJSON, err := json.MarshalToString(symbolPrices)
	if err != nil {
		return "", fmt.Errorf("failed to marshal symbol prices: %w", err)
	}

	return resultJSON, nil
}

func (t *ListTickerPricesTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	symbolPricesDesc := map[string]string{
		"en": "The symbol prices",
		"zh": "交易对价格",
	}

	return []toolcore.ParameterDefinition{
		{
			Name:        "symbol_prices",
			Type:        toolcore.ParamTypeArray,
			Description: symbolPricesDesc,
			Required:    true,
		},
	}
}
