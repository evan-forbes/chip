package trade

import (
	"fmt"
	"testing"
	"time"
)

func TestPositionValue(t *testing.T) {
	lim := Limit{
		Sell:       "USDC",
		Buy:        "ETH",
		Collat:     "USDC",
		User:       "zkFART",
		BuyAmount:  0,
		SellAmount: 100,
		CollAmount: 100,
		Price:      200,
		Leverage:   5,
		Long:       true,
	}
	p := &Position{
		Limit: lim,
		Alive: true,
	}
	// find the percent change of the starting price
	currPrice := 160.0
	collPrice := 1.0
	percChange := (currPrice - p.Price) / p.Price
	dir := 1.0
	if !p.Long {
		dir = -1.0
	}
	delta := percChange * float64(p.Leverage) * dir
	out := PosVal{
		Time:     time.Now().Round(time.Second),
		Value:    (p.CollAmount * collPrice) + (delta * p.CollAmount * collPrice),
		Position: p.Key,
	}
	fmt.Println(out.Value)
}
