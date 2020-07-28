package main

import (
	"log"
	"os"

	"github.com/evan-forbes/chip/cmd/trade"
	"github.com/urfave/cli/v2"
)

// chip trade -b eth -s usdc -sam 1000 -p 300
// chip trade -b usdc -s eth -sam -1 -p .005

// chip long -b eth -s usdc -sam 1000
// chip short -b usdc -s eth -sam 3.4

// chip award

// chip posits
// ... list open positions
// to close a position, just say !chip close 3, or !chip close all
// chip close

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "chip"

	// subcommands
	app.Commands = []*cli.Command{
		{
			Name:   "short",
			Usage:  "opens a bearish position/limit order",
			Flags:  trade.Flags(),
			Action: trade.Trade(false, true),
		},
		{
			Name:   "long",
			Usage:  "opens a bullish position/limit order",
			Flags:  trade.Flags(),
			Action: trade.Trade(true, true),
		},
		{
			Name:   "trade",
			Usage:  "puts in an order to trade assets at a certain price",
			Flags:  trade.Flags(),
			Action: trade.Trade(true, false),
		},
		{
			Name:  "award",
			Usage: "starts the process of issuing a reward",
			// Flags: "",
			// Action: trade.Short,s
		},
		{
			Name:  "post",
			Usage: "shows you your current open positions",
			// Flags: tradeFlags,
			// Action: trade.Short,s
		},
		{
			Name:  "close",
			Usage: "ends an open position, cementing losses or gains",
			// Flags: tradeFlags,
			// Action: trade.Short,s
		},
		{
			Name:  "brag",
			Usage: "provides details on a current position",
			// Flags: tradeFlags,
			// Action: trade.Short,s
		},
		{
			Name:  "orders",
			Usage: "shows you all of your current limit orders",
			// Flags: tradeFlags,
			// Action: trade.Short,s
		},
		{
			Name:  "cancel",
			Usage: "removes a limit order",
			// Flags: tradeFlags,
			// Action: trade.Short,s
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
