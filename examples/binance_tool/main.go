package main

import (
	"context"
	"fmt"
	"log"

	"github.com/m4n5ter/another-me/pkg/i18n"
	binancetool "github.com/m4n5ter/another-me/pkg/tools/cryptoexchange/binance"
)

func main() {
	binanceTool := binancetool.NewBinance(i18n.GlobalManager, nil, nil, nil)
	markets, err := binanceTool.FetchMarkets(context.Background(), []string{"BTCUSDT"})
	if err != nil {
		log.Fatalf("failed to fetch markets: %v", err)
	}
	fmt.Println(markets)
}
