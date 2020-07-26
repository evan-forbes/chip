package limit

import (
	"strings"
	"time"

	"github.com/evan-forbes/chip/arango"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func MarketCommand(ctx *cli.Context) error {
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

	limit := Limit{
		Sell:       sass,
		Buy:        bass,
		SellAmount: sam,
		User:       user,
		ChanID:     chanID,
		Timestamp:  time.Now().Round(time.Second),
		Leverage:   1,
	}

	return limit.InsertMarket(sesh)
}
