package begin

import (
	"fmt"
	"os"
	"time"

	"github.com/evan-forbes/chip/arango"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func Begin(ctx *cli.Context) error {
	// detect user
	user, chanid, exists := detectUser(ctx)
	if !exists {
		ctx.Println("could not detech user")
		return errors.New("could not detect user")
	}
	if chanid == "local" {
		ctx.Println("please begin using your discord dms")
		return nil
	}
	// see if they already exist
	sesh, err := arango.NewSesh(ctx.Context, "cookie")
	if err != nil {
		return errors.Wrap(err, "failure to begin")
	}
	const query = `
	let out = (
		for u in users
			filter u._key == "%s"
			return u
	)
	return length(out)
	`
	var already int
	err = sesh.Execute(fmt.Sprintf(query, user), &already)
	if err != nil {
		return errors.Wrap(err, "failure to begin")
	}
	if already > 0 {
		ctx.Println("you have already begun your journey with chip")
		return nil
	}
	// register new user
	u := User{
		Name:     user,
		ChanID:   chanid,
		JoinTime: time.Now().Round(time.Second),
	}
	err = sesh.CreateDoc("users", u)
	if err != nil {
		return errors.Wrap(err, "failure to begin")
	}
	ctx.Println(":partying_face: CONGRADULATIONS, MY NEW MEAT BAG FRIEND! You may now commence trading. I will send any personal notifications to this channel. see what I can do with !chip help, and to get specific help with a subcommand, try !chip help sub-command-name-here")
	return nil
}

type User struct {
	Name     string    `json:"_key"`
	ChanID   string    `json:"channel_id"`
	JoinTime time.Time `json:"join_time"`
}

// detectUser attempts to identify the user based on the context
func detectUser(ctx *cli.Context) (string, string, bool) {
	var user string
	var id string
	switch {
	case ctx.Slug == nil:
		user = os.Getenv("CHIP_USERNAME")
		id = "local"
	case ctx.Slug != nil:
		user = ctx.Slug.User
		id = ctx.Slug.ChanID
	}
	if user == "" {
		return "", "", false
	}
	return user, id, true
}
