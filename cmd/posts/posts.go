package posts

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"text/tabwriter"

	"github.com/evan-forbes/chip/arango"
	"github.com/evan-forbes/chip/cmd/trade"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func Posts(ctx *cli.Context) error {
	// detected user
	user, valid := DetectUser(ctx)
	if !valid {
		ctx.Println("no user detected, set CHIP_USERNAME")
	}
	// fetch open positions
	sesh, err := arango.NewSesh(ctx.Context, "cookie")
	if err != nil {
		return errors.Wrap(err, "failure to fetch open positions")
	}
	pos, err := Open(sesh, user)
	if err != nil {
		return errors.Wrap(err, "failure to fetch posts")
	}
	if len(pos) == 0 {
		ctx.Println("no open positions")
		return nil
	}
	// render
	ren, err := Render(sesh, pos)
	if err != nil {
		return errors.Wrap(err, "failure to render positions")
	}
	ctx.Println(ren)
	return nil
}

func Open(sesh *arango.Sesh, user string) ([]*trade.Position, error) {
	const query = `
	let out = (
		for p in positions
			filter p.alive == true
			filter p.user == "%s"
			sort p._key desc
			return p
	)
	return out
	`
	var pos []*trade.Position
	err := sesh.Execute(fmt.Sprintf(query, user), &pos)
	if err != nil {
		return nil, errors.Wrap(err, "failure to fetch user's open positions")
	}
	return pos, nil
}

// detectUser attempts to identify the user based on the context
func DetectUser(ctx *cli.Context) (string, bool) {
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

// Render returns a formatted string that descibes the state of the balance
func Render(sesh *arango.Sesh, posts []*trade.Position) (string, error) {
	const templ = `
	{{ range $i, $p := .}}
	- {{$i}} )	${{with $cv := $p.CurrValue}}{{printf "%.3f" $cv}}{{end}}	{{$p.Leverage}}x	{{$p.Dir}}	{{$p.Buy}}	{{$p.Sell}}	Size: {{$p.CollAmount}} {{$p.Collat}}{{end}}
	`
	for _, p := range posts {
		p.SetDir()
		posVal, err := p.Value(sesh)
		if err != nil {
			return "", errors.Wrap(err, "failure to calc position value")
		}
		p.CurrValue = posVal.Value
	}
	var buf bytes.Buffer
	twr := tabwriter.NewWriter(&buf, 1, 4, 8, ' ', 0)
	// make and execute the template
	t := template.Must(template.New("positions").Parse(templ))
	err := t.Execute(twr, posts)
	if err != nil {
		fmt.Println("error in template exec:", err)
	}

	err = twr.Flush()
	if err != nil {
		fmt.Println("failure to render balance", err)
	}
	return buf.String(), nil
}
