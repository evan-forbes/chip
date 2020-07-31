package folio

import (
	"context"
	"fmt"
	"testing"

	"github.com/evan-forbes/chip/arango"
)

func TestRender(t *testing.T) {
	sesh, err := arango.NewSesh(context.Background(), "cookie")
	if err != nil {
		t.Error(err)
	}
	bal := &arango.Balance{
		User: "test",
		Balances: map[string]float64{
			"ETH": 30,
			"BTC": 100,
			"FXC": 0.0,
		},
	}
	bal.Clean(nil)
	err = bal.LookupPrices(sesh)
	if err != nil {
		t.Error(err)
	}
	bal.CalcTotal()
	ren := bal.Render()
	fmt.Println(ren)
}
