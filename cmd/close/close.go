package close

import (
	"fmt"
	"strconv"

	"github.com/evan-forbes/chip/arango"
	"github.com/evan-forbes/chip/cmd/posts"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// Flags returns the flags for the close command
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:    "position",
			Aliases: []string{"p"},
			Value:   0,
			Usage:   "select which position to close",
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
	// render
	ren, err := posts.Render(sesh, pos)
	if err != nil {
		return errors.Wrap(err, "failure to render positions")
	}
	// show the render and ask for input
	ctx.Println(ren)
	input, err := ctx.Input("please enter the position number you would like to close")
	if err != nil {
		return errors.Wrap(err, "failure to close position: no input")
	}
	// check input
	num, valid, err := ensureOrderNumber(len(pos), input)
	if err != nil {
		ctx.Println(fmt.Sprintf("%s is not a valid input", input))
		return nil
	}
	if !valid {
		ctx.Println(fmt.Sprintf("%d is too high or too low use one of the position numbers listed", num))
		return nil
	}
	// close the position
	err = pos[num].Close(sesh, false)
	if err != nil {
		ctx.Println("could not close position!", err)
		return errors.Wrap(err, "failure to close position")
	}
	ctx.Println("position has been closed")
	return nil
}

func ensureOrderNumber(l int, in string) (int, bool, error) {
	input, err := strconv.ParseInt(in, 10, 64)
	if err != nil {
		return 0, false, err
	}
	out := int(input)
	if out >= l || out < 0 {
		return out, false, nil
	}
	return out, true, nil
}
