package limit

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

// Limit describes an order that could be executed by chip
type Limit struct {
	Key        string    `json:"_key,omitempty"`
	Sell       string    `json:"sell"`
	Buy        string    `json:"buy"`
	User       string    `json:"user"`
	BuyAmount  float64   `json:"buy_amount,omitempty"`
	SellAmount float64   `json:"sell_amount,omitempty"`
	Price      float64   `json:"price,omitempty"`
	ChanID     string    `json:"channel_id"`
	Timestamp  time.Time `json:"timestamp"`
	Leverage   int       `json:"leverage"`
}

func (l *Limit) Insert(sesh *arango.Sesh) error {
	return sesh.CreateDoc("limits", l)
}

func (l *Limit) InsertMarket(sesh *arango.Sesh) error {
	return sesh.CreateDoc("pending", l)
}

func Command(ctx *cli.Context) (err error) {
	// connect to the db
	sesh, err := arango.NewSesh(ctx.Context, "cookie")
	if err != nil {
		return err
	}
	user, has := detectUser(ctx)
	if !has {
		return errors.New("failure to set limit order: no user detected")
	}
	var chanID string
	if ctx.Slug == nil {
		chanID = "local"
	} else {
		chanID = ctx.Slug.ChanID
	}
	// sell asset ticker symbol
	sass := strings.ToUpper(ctx.String("sell"))
	// buy asset ticker symbol
	bass := strings.ToUpper(ctx.String("buy"))
	// amount to sell (overides price if set)
	sam := ctx.Float64("sellamount")
	// amount to buy (overides price if set)
	bam := ctx.Float64("buyamount")

	// ensure assets are valid/present
	valid, err := ensureAssets(ctx, sesh, sass, bass)
	if err != nil {
		return errors.Wrapf(err, "failure to validate assets: %s and %s: ", sass, bass)
	}
	// exit if invalid assets
	if !valid {
		return nil
	}
	// make sure the user has enough to sell
	valid, sam, err = ensureSell(ctx, sesh, user, sass, sam)
	if err != nil {
		return errors.Wrapf(err, "failure to validate assets: %s and %s: ", sass, bass)
	}
	// exit if user doesn't have the funds
	if !valid {
		return nil
	}
	// make sure there is a set buy amount
	valid, bam, err = ensureBuy(ctx, bass, bam)
	if err != nil {
		return errors.Wrap(err, "failure to set buy amount:")
	}
	if !valid {
		return nil
	}
	limit := Limit{
		Sell:       sass,
		Buy:        bass,
		SellAmount: sam,
		BuyAmount:  bam,
		User:       user,
		Price:      sam / bam,
		ChanID:     chanID,
		Timestamp:  time.Now().Round(time.Second),
		Leverage:   1,
	}

	return limit.Insert(sesh)
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
		var exists bool
		err := sesh.Execute(fmt.Sprintf(query, asset), &exists)
		if err != nil {
			return false, errors.Wrap(err, "failure to validate asset amount:")
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
	err = sesh.Execute(fmt.Sprintf(arango.LatestBalance, user), &bal)
	if err != nil {
		return false, 0, err
	}
	currBal, has := bal.Balances[asset]
	if !has || currBal < amount {
		ctx.Println(fmt.Sprintf("you don't have enough"))
		return false, 0, nil
	}
	// if there was no sell amount, ask for one
	if amount <= 0 {
		input, err := ctx.Input(
			fmt.Sprintf(
				`I didn't see a sell amount (flag -sam)
				how much %s would you like to sell? 
				You currently have %f.3 %s
				please only enter a number`,
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
	}
	return true, amount, nil
}

// ensureBuy checks to see that user has specified a buy amount, if not, it
// double checks
func ensureBuy(ctx *cli.Context, asset string, amount float64) (valid bool, bam float64, err error) {
	// if there was no sell amount, ask for one
	if amount <= 0 {
		input, err := ctx.Input(
			fmt.Sprintf(
				`I didn't see a buy amount (flag -bam)
				how much %s would you like to buy?
				please only enter a number`,
				asset,
			))
		if err != nil {
			return false, 0, errors.Wrap(err, "failure to validate buying asset amount")
		}
		// set amount to the inputed amount if a number is passed
		amount, err = strconv.ParseFloat(input, 64)
		if err != nil {
			return false, amount, errors.Wrap(err, "failure to parse float while validating buying asset amount")
		}
	}
	return true, amount, nil
}
