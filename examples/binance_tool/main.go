package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	binancetool "github.com/m4n5ter/another-me/pkg/tools/cryptoexchange/binance"
)

func main() {
	binanceClient := binancetool.NewBinance("", "", Some("socks5://127.0.0.1:55535"))
	listTickerPricesTool := binancetool.NewListTickerPricesTool(i18n.GlobalManager, binanceClient)

	symbolPrices, err := listTickerPricesTool.Call(context.Background(), `{"symbols": ["BTCUSDT", "ETHUSDT", "SOLUSDT", "XRPUSDT", "DOGEUSDT", "ADAUSDT", "DOTUSDT", "LINKUSDT", "BCHUSDT", "LTCUSDT", "XLMUSDT", "XMRUSDT"]}`)
	if err != nil {
		slog.Error("failed to list ticker prices", "error", err)
		return
	}

	fmt.Println(symbolPrices)
}
