package main

import (
	"log"
	"os"

	"github.com/evan-forbes/chip/cmd/limit"
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

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true
	app.Name = "chip"

	limitFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "buy",
			Aliases: []string{"b"},
			Value:   "",
			Usage:   "specify the asset to be bought",
		},
		&cli.StringFlag{
			Name:    "sell",
			Aliases: []string{"s"},
			Value:   "",
			Usage:   "specify the asset to be sold",
		},
		&cli.Float64Flag{
			Name:    "sellamount",
			Aliases: []string{"sam"},
			Value:   0,
			Usage:   "specify the amount to be bought",
		},
		&cli.Float64Flag{
			Name:    "buyamount",
			Aliases: []string{"bam"},
			Value:   0,
			Usage:   "specify the amount to be bought",
		},
	}

	synth := []*cli.Command{
		{
			Name: "synth",
		},
	}
	
	// subcommands
	app.Commands = []*cli.Command{
		{
			Name:   "limit",
			Usage:  "issue a limit order between two assets",
			Flags:  limitFlags,
			Action: limit.Command,
		},
		{
			Name: "market",
			Usage: "trade assets at the current price",
			Flags: limitFlags[:3],
			Action: market.Command,
		}
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
