package limit

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// chip

type Limit struct {
}

func Command(ctx *cli.Context) error {
	// sell asset ticker symbol
	sass := strings.ToUpper(ctx.String("sell"))
	// buy asset ticker symbol
	bass := strings.ToUpper(ctx.String("buy"))
	// amount to sell (overides price if set)
	sam := ctx.Float64("sellamount")
	// amount to buy (overides price if set)
	bam := ctx.Float64("buyamount")
	// price to be executed at (can be used in place of sam and bam)
	price := ctx.Float64("price")

	// ensure assets are valid/present
	err := ensureAssets(ctx, sass, bass)
	if err != nil {
		ctx.Println("are these assets valid?")
		return errors.Wrapf(err, "failure to validate assets: %s and %s: ", sass, bass)
	}

	// amounts are validated at execution

	return nil
}

// ensureAssets validates that the assets described in the limit order are
// indeed actual assets
func ensureAssets(ctx *cli.Context, assets ...string) error {
	return nil
}

// ensurePrice checks for a negative price and determines if the current price
// is above or below specified price
func ensurePrice(ctx *cli.Context, price float64) (above bool, err error) {
	return above, nil
}
