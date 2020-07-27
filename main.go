package main

import (
	"log"
	"os"

	"github.com/evan-forbes/chip/cmd/trade"
	"github.com/urfave/cli/v2"
)

// chip limit -b eth -s usdc -sam -bam

// go long on ETH with 5x leverage and using 1000 USDC as a position size
// chip market long -b ETH -s USDC -sam 1000 -l 5

// open a short position if the price gets
// chip limit short -b ETH -s USDC -sam 5000

// chip limit

// chip market -b eth -s USDC -a 1000 -l 5

// > You currently have 1000 USDC. Please enter in only the amount you would like sell or the word "all"
// > 500
// >

// chip short limit -b ETH -s BTC -sam 1 -p 400 -l 5

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
			Name:  "short",
			Usage: "opens a bearish position",
			Flags: trade.Flags(),
			// Action: trade.Short,s
		},
		{
			Name:  "long",
			Usage: "opens a bullish position",
			Flags: trade.Flags(),
			// Action: trade.Short,s
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
		{
			Name:  "short",
			Usage: "opens a bearish position",
			// Flags: tradeFlags,
			// Action: trade.Short,s
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
