package folio

import (
	"fmt"
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
	if ctx.Bool("all") {
		return showAll(ctx, sesh)
	}
	// detect the user
	user, valid := detectUser(ctx)
	if !valid {
		ctx.Println("no user detected")
		return nil
	}
	ren, err := getStringFolio(sesh, user)
	// send to user
	ctx.Println(ren)
	return posts.Posts(ctx)
}

// show all combines each user's total and positions
func showAll(ctx *cli.Context, sesh *arango.Sesh) error {
	// fetch all users
	users, err := arango.AllUsers(sesh)
	if err != nil {
		return err
	}
	fmt.Println(users)
	for _, u := range users {
		folRend, err := getStringFolio(sesh, u)
		if err != nil {
			return err
		}
		// fetch the positions for that user
		pos, err := posts.Open(sesh, u)
		if err != nil {
			return errors.Wrap(err, "failure to fetch posts")
		}
		if len(pos) == 0 {
			ctx.Println(folRend)
			continue
		}
		// render
		posRend, err := posts.Render(sesh, pos)
		if err != nil {
			return errors.Wrap(err, "failure to render positions")
		}
		fullRender := fmt.Sprintf("%s%s", folRend, posRend)
		ctx.Println(fullRender)
	}
	return nil
}

func getStringFolio(sesh *arango.Sesh, user string) (string, error) {
	const errMsg = "failure to get portfolio for user"
	bal, err := arango.LatestBalance(sesh, user)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	// clean the local copy of the balance
	bal.Clean(nil)
	// lookup prices for the balance
	err = bal.LookupPrices(sesh)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	// calculate the total
	bal.CalcTotal()
	// render the balance
	ren := bal.Render()
	return ren, nil
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
