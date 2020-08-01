package folio

import (
	"os"

	"github.com/evan-forbes/chip/arango"
	"github.com/evan-forbes/chip/cmd/posts"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// Flags returns the flags needed for the trade cli sub command
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Value:   false,
			Usage:   "see everyone's portfolio",
		},
	}
}

// TODO: include postions

func Folio(ctx *cli.Context) error {
	const errMsg = "failure to display portfolio"
	// fetch the users current balance
	sesh, err := arango.NewSesh(ctx.Context, "cookie")
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	// detect the user
	user, valid := detectUser(ctx)
	if !valid {
		ctx.Println("no user detected")
		return nil
	}
	bal, err := arango.LatestBalance(sesh, user)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	// clean the local copy of the balance
	bal.Clean(nil)
	// lookup prices for the balance
	err = bal.LookupPrices(sesh)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	// calculate the total
	bal.CalcTotal()
	// render the balance
	ren := bal.Render()
	// send to user
	ctx.Println(ren)
	return posts.Posts(ctx)
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
