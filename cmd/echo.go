package cmd

import (
	"fmt"
	"log"

	"github.com/urfave/cli/v2"
)

// Echo reads the response from the user and writes it back.
// fullfills. cli.ActionFunc
func Echo(ctx *cli.Context) error {
	fmt.Println("running echo")
	_, err := ctx.Write([]byte("ready for input"))
	if err != nil {
		log.Println(err)
	}
	input, err := ctx.Input("yes? ")
	if err != nil {
		log.Println("failure to ask for input", err)
	}
	ctx.Println(input)
	return nil
}
