package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/evan-forbes/chip/arango"
	"github.com/evan-forbes/chip/cmd/begin"
	"github.com/evan-forbes/chip/cmd/close"
	"github.com/evan-forbes/chip/cmd/folio"
	"github.com/evan-forbes/chip/cmd/posts"
	"github.com/evan-forbes/chip/cmd/trade"
	"github.com/pkg/errors"
	cron "github.com/robfig/cron/v3"
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
	app.Usage = "paper trade the top 300 crypto currencies"

	// subcommands
	app.Commands = []*cli.Command{
		{
			Name:      "short",
			Usage:     "opens a bearish position/limit order",
			UsageText: trade.ShortUsageText,
			Flags:     trade.Flags(),
			Action:    trade.Trade(false, true),
		},
		{
			Name:      "long",
			Usage:     "opens a bullish position/limit order",
			UsageText: trade.LongUsageText,
			Flags:     trade.Flags(),
			Action:    trade.Trade(true, true),
		},
		{
			Name:      "trade",
			Usage:     "exchange one asset for another",
			UsageText: trade.TradeUsageText,
			Flags:     trade.Flags(),
			Action:    trade.Trade(true, false),
		},
		{
			Name:   "folio",
			Usage:  "look at your current portfolio",
			Action: folio.Folio,
			Flags:  folio.Flags(),
		},
		{
			Name:   "posts",
			Usage:  "look at your open positions",
			Action: posts.Posts,
		},
		{
			Name:      "close",
			Usage:     "end a currently open levered position",
			UsageText: close.UsageText,
			Action:    close.Close,
			Flags:     close.Flags(),
		},

		// {
		// 	Name:  "award",
		// 	Usage: "starts the process of issuing a reward",
		// 	// Flags: "",
		// 	// Action: trade.Short,s
		// },
		// {
		// 	Name:  "post",
		// 	Usage: "shows you your current open positions",
		// 	// Flags: tradeFlags,
		// 	// Action: trade.Short,s
		// },
		// {
		// 	Name:  "close",
		// 	Usage: "ends/describes an ending for an open position, cementing losses or gains",
		// 	// Flags: tradeFlags,
		// 	// Action: trade.Short,s
		// },
		// {
		// 	Name:  "brag",
		// 	Usage: "provides details on a current position",
		// 	// Flags: tradeFlags,
		// 	// Action: trade.Short,s
		// },
		// {
		// 	Name:  "orders",
		// 	Usage: "shows you all of your current limit orders",
		// 	// Flags: tradeFlags,
		// 	// Action: trade.Short,s
		// },
		// {
		// 	Name:  "cancel",
		// 	Usage: "removes a limit order",
		// 	// Flags: tradeFlags,
		// 	// Action: trade.Short,s
		// },
		{
			Name:   "begin",
			Usage:  "start your journey with chip",
			Action: begin.Begin,
		},
	}

	// setup
	if strings.Contains(strings.Join(os.Args, ""), "boot") {
		crn := cron.New()
		crn.AddFunc("*/15 * * * *", func() {
			time.Sleep(time.Second * 30)
			// connect to arango
			sesh, err := arango.NewSesh(context.Background(), "cookie")
			if err != nil {
				log.Println(errors.Wrap(err, "failure to update chip:"))
				return
			}
			// execute market orders
			err = trade.ExecuteMarketOrders(app.Disc, sesh)
			if err != nil {
				log.Println(errors.Wrap(err, "failure to update chip: could not execute market orders"))
				return
			}
			// update any limit orders
			err = trade.CheckLimits(app.Disc, sesh)
			if err != nil {
				log.Println(errors.Wrap(err, "failure to update chip: could not update limit orders"))
				return
			}
			// update all positions
			err = trade.UpdatePositions(app.Disc, sesh)
			if err != nil {
				log.Println(errors.Wrap(err, "failure to update chip: could not update positions"))
				return
			}
		})
		crn.Start()
		defer crn.Stop()
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
