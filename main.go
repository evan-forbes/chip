package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

// chip order -b ETH -s USDC -sp 200
// chip order -b ETH -s USDC -bp .005
// chip limit -b eth -s usdc -sam -bam

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
			Name:    "buyamount",
			Aliases: []string{"bam"},
			Value:   0,
			Usage:   "specify the amount to be bought",
		},
		&cli.Float64Flag{
			Name:    "sellamount",
			Aliases: []string{"sam"},
			Value:   0,
			Usage:   "specify the amount to be bought",
		},
		&cli.Float64Flag{
			Name:    "price",
			Aliases: []string{"p"},
			Value:   0,
			Usage:   "specify the price in USD at which to execute the order.",
		},
	}

	// subcommands
	app.Commands = []*cli.Command{
		{
			Name:  "limit",
			Usage: "issue a limit order between two assets",
			Flags: limitFlags,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
