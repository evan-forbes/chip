package close

import (
	"fmt"
	"strconv"

	"github.com/evan-forbes/chip/arango"
	"github.com/evan-forbes/chip/cmd/posts"
	"github.com/evan-forbes/chip/cmd/trade"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const UsageText = `
// I want to select a position and close it
!chip close
- 1 ) $49167.416   5x  short  DAI  USDC  Size: 50000 USDC
- 2 ) $1881.335    5x  short  ETH  USDC  Size: 2000 USDC
// I input  just the number '1' and position 1 gets closed

OR

// close position 1 if it reaches $100,000 in value
!chip close -p 1 -u 100000

// close position 1 if it reaches $25,000 in value
!chip close -p 1 -l 25000
`

// Flags returns the flags for the close command
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "position",
			Aliases: []string{"p"},
			Value:   0,
			Usage:   "select which position to close",
		},
		&cli.Float64Flag{
			Name:    "upper",
			Aliases: []string{"u"},
			Value:   0,
			Usage:   "set the upper value in USD in which the position should close",
		},
		&cli.Float64Flag{
			Name:    "lower",
			Aliases: []string{"l"},
			Value:   0,
			Usage:   "set the lower value in USD in which the position should close",
		},
	}
}

func Close(ctx *cli.Context) error {
	// detected user
	user, valid := posts.DetectUser(ctx)
	if !valid {
		ctx.Println("no user detected, set CHIP_USERNAME")
	}
	// fetch open positions
	sesh, err := arango.NewSesh(ctx.Context, "cookie")
	if err != nil {
		return errors.Wrap(err, "failure to fetch open positions")
	}
	pos, err := posts.Open(sesh, user)
	if err != nil {
		return errors.Wrap(err, "failure to fetch posts")
	}
	if len(pos) == 0 {
		ctx.Println("beloved meat bag, you do not have any open positions")
		return nil
	}

	p, err := ensureInput(ctx, sesh, pos)
	if err != nil {
		return errors.Wrap(err, "failure to close order")
	}
	if p == nil {
		return nil
	}
	update, err := ensureUpLow(ctx, sesh, p)
	if err != nil {
		return errors.Wrap(err, "failure to update high or low limit on position")
	}
	if update {
		ctx.Println("position updated")
		return nil
	}
	// close the position
	err = p.Close(sesh, false)
	if err != nil {
		ctx.Println("could not close position!", err)
		return errors.Wrap(err, "failure to close position")
	}
	ctx.Println("position has been closed")
	return nil
}

func ensureInput(ctx *cli.Context, sesh *arango.Sesh, pos []*trade.Position) (*trade.Position, error) {
	p := ctx.Int("position")
	if p > 0 && p <= len(pos) {
		return pos[p-1], nil
	}
	// render
	ren, err := posts.Render(sesh, pos)
	if err != nil {
		return nil, errors.Wrap(err, "failure to render positions")
	}
	// show the render and ask for input
	ctx.Println(ren)
	rawinput, err := ctx.Input("please select a position (enter a number)")
	if err != nil {
		return nil, errors.Wrap(err, "failure to close position: no input")
	}
	// check input
	input, err := strconv.ParseInt(rawinput, 10, 64)
	if err != nil {
		ctx.Println(fmt.Sprintf("aborting: could not parse input: %s, please enter a number next time", rawinput))
		return nil, nil
	}
	p = int(input)
	if p > 0 && p <= len(pos) {
		return pos[p-1], nil
	}
	ctx.Println("aborting: invalid selection, please select a position number")
	return nil, nil
}

func ensureUpLow(ctx *cli.Context, sesh *arango.Sesh, p *trade.Position) (bool, error) {
	up := ctx.Float64("upper")
	low := ctx.Float64("lower")
	if up == 0 && low == 0 {
		return false, nil
	}
	p.CloseCond = &trade.CloseCondition{Upper: up, Lower: low}
	return true, sesh.Update("positions", p.Key, p)
}
