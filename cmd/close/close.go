package close

import (
	"github.com/evan-forbes/chip/arango"
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
	return nil
}

func ensureOrderNumber(ctx *cli.Context, sesh *arango.Sesh) (int, error) {
	// is this number a viable order number?
	num := ctx.Int("position")
	if num == 0 {
		// print the current positions
		// ask which order they would like to close
		ctx.Input("which order would you like to close")
	}
	return num, nil
}

func ensureBounds(ctx *cli.Context, sesh *arango.Sesh) {

}
