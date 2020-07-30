package trade

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/evan-forbes/chip/arango"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// Flags returns the flags needed for the trade cli sub command
func Flags() []cli.Flag {
	return []cli.Flag{
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
		&cli.StringFlag{
			Name:    "collateral",
			Aliases: []string{"c"},
			Value:   "",
			Usage:   "specify the asset to be used as collateral in a leveraged position",
		},
		&cli.Float64Flag{
			Name:    "sellamount",
			Aliases: []string{"sam"},
			Value:   0,
			Usage:   "specify the amount to be bought (use -1 to sell all",
		},
		&cli.Float64Flag{
			Name:    "price",
			Aliases: []string{"p"},
			Value:   0,
			Usage:   "specify the price of execution",
		},
		&cli.IntFlag{
			Name:    "leverage",
			Aliases: []string{"l"},
			Value:   1,
			Usage:   "amount of leverage (default of 1)",
		},
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Value:   true,
			Usage:   "sets sell amount (-sam) to your current balance of the selling asset",
		},
	}
}

// Trade issues a leveraged position/limit order paid out in the selling asset
func Trade(long, levered bool) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// connect to the db
		sesh, err := arango.NewSesh(ctx.Context, "cookie")
		if err != nil {
			return err
		}
		user, has := detectUser(ctx)
		if !has {
			return errors.New("failure to set limit order: no user detected")
		}
		// sell asset ticker symbol
		sass := strings.ToUpper(ctx.String("sell"))
		// buy asset ticker symbol
		bass := strings.ToUpper(ctx.String("buy"))
		// collateral asset ticker symbol
		cass := strings.ToUpper(ctx.String("collateral"))
		// amount to sell (overides price if set)
		sam := ctx.Float64("sellamount")
		// amount of leverage to apply
		lever := abs(ctx.Int("leverage"))

		// checks if this order is a limit order or not
		isLim, price := ensureLimit(ctx)

		// make sure that an apropriate amount of leverage is being used
		lever = ensureLeverage(ctx, lever, levered)

		// ensure assets are valid/present
		valid, err := ensureAssets(ctx, sesh, sass, bass, cass)
		if err != nil {
			return errors.Wrapf(err, "failure to validate assets: %s and %s: ", sass, bass)
		}
		// exit if invalid assets
		if !valid {
			return nil
		}
		if cass == "" {
			cass = sass
		}
		// make sure the user has enough to sell
		valid, sam, err = ensureSell(ctx, sesh, user, cass, sam)
		if err != nil {
			return errors.Wrapf(err, "failure to validate assets: %s and %s: ", sass, bass)
		}
		// exit if user doesn't have the funds
		if !valid {
			return nil
		}
		var buyAm float64
		if price > 0 {
			buyAm = sam / price
		}
		limit := Limit{
			Sell:       sass,
			Buy:        bass,
			Collat:     cass,
			User:       user,
			SellAmount: sam,
			CollAmount: sam,
			BuyAmount:  buyAm,
			Price:      price,
			CreateTime: time.Now().Round(time.Second),
			Leverage:   lever,
			Long:       long,
		}
		if isLim {
			err = limit.Insert(sesh)
		} else {
			err = limit.InsertMarket(sesh)
		}
		if err != nil {
			return errors.Wrap(err, "failure to insert limit order")
		}
		const succMsg = `meat bag, your order has been successfully submitted, I will notify you if it gets executed.\nsee !chip help if you seek further action.
		`
		ctx.Println(succMsg)
		return nil
	}
}

// ensureLimit checks for a limit and a price, asking the user for further
// clarification if one is provided and not the other
func ensureLimit(ctx *cli.Context) (isLim bool, price float64) {
	price = ctx.Float64("price")
	if price > 0 {
		isLim = true
	}
	return isLim, price
}

// detectUser attempts to identify the user based on the context
func detectUser(ctx *cli.Context) (string, bool) {
	var user string
	switch {
	case ctx.Slug == nil:
		user = os.Getenv("CHIP_USERNAME")
	case ctx.Slug != nil:
		user = ctx.Slug.User
	}
	if user == "" {
		return "", false
	}
	return user, true
}

// ensureAssets validates that the assets described in the limit order are
// indeed actual assets
func ensureAssets(ctx *cli.Context, sesh *arango.Sesh, assets ...string) (bool, error) {
	const query = `
	for s in fulltext(stamps, "symbol", "%s")
		limit 1
		return s.market_cap > 0
	`
	for _, asset := range assets {
		if asset == "" {
			continue
		}
		var exists bool
		err := sesh.Execute(fmt.Sprintf(query, asset), &exists)
		if err != nil {
			ctx.Println(fmt.Sprintf("According to my books, asset %s does not exist. Please try again.", asset))
			return false, nil
		}
		if !exists {
			ctx.Println(fmt.Sprintf("According to my books, asset %s does not exist. Please try again.", asset))
			return false, nil
		}
	}
	return true, nil
}

// ensureSell checks to make sure that the user has enough funds
func ensureSell(ctx *cli.Context, sesh *arango.Sesh, user, asset string, amount float64) (valid bool, amm float64, err error) {
	var bal arango.Balance
	err = sesh.Execute(fmt.Sprintf(arango.LatestBalanceQ, user), &bal)
	if err != nil {
		return false, 0, err
	}
	currBal, has := bal.Balances[asset]
	if !has || currBal < amount {
		ctx.Println(fmt.Sprintf("beloved meat bag, you do not have enough %s to sell", asset))
		ctx.Println(fmt.Sprintf("current balance: %f.3", currBal))
		return false, 0, nil
	}
	// if there was no sell amount, ask for one
	if amount == 0 {
		input, err := ctx.Input(
			fmt.Sprintf(
				`my meat bag friend, I didn't see a sell amount (flag -sam)
				how much %s would you like to sell? 
				You currently have %f.3 %s
				please only enter a number
				use -1 to sell all`,
				asset, currBal, asset,
			))
		if err != nil {
			return false, 0, errors.Wrap(err, "failure to validate selling asset amount")
		}
		// set amount to the inputed amount if a number is passed
		amount, err = strconv.ParseFloat(input, 64)
		if err != nil {
			return false, amount, errors.Wrap(err, "failure to validate selling asset amount")
		}
		// try again with the newly entered amount
		return ensureSell(ctx, sesh, user, asset, amount)
	}
	if amount < 0 {
		amount = currBal
	}
	return true, amount, nil
}

func ensureLeverage(ctx *cli.Context, lever int, leveraged bool) int {
	if !leveraged {
		return 0
	}
	if lever > 5 {
		ctx.Println("oh cute meat bag, one must walk before one can run. using the max of 5x leverage")
		lever = 5
	}
	return lever
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
