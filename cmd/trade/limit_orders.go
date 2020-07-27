package trade

import (
	"context"
	"time"

	"github.com/evan-forbes/chip/arango"
	"github.com/pkg/errors"
)

func CheckTradeLimits(ctx context.Context) error {
	const query = `
	let out = (
		for l in limits
			filter l.leverage == 0
			return l
	)
	return out
	`
	// connect to the db
	sesh, err := arango.NewSesh(ctx, "cookie")
	if err != nil {
		return errors.Wrap(err, "failure check limits")
	}
	var limits []Limit
	err = sesh.Execute(query, &limits)
	if err != nil {
		return errors.Wrap(err, "failure to check limits")
	}
	for _, lim := range limits {

	}
	return nil
}

func CheckLeveredLimits(ctx context.Context) error {
	const query = `
	let out = (
		for l in limits
			filter l.leverage > 0
			return l
	)
	return out
	`
	// connect to the db
	sesh, err := arango.NewSesh(ctx, "cookie")
	if err != nil {
		return errors.Wrap(err, "failure check levered limits")
	}
	var limits []Limit
	err = sesh.Execute(query, &limits)
	if err != nil {
		return errors.Wrap(err, "failure to check limits")
	}
	return nil
}

// Limit describes an order that could be executed by chip
type Limit struct {
	Key        string    `json:"_key,omitempty"`
	Sell       string    `json:"sell"`
	Buy        string    `json:"buy"`
	User       string    `json:"user"`
	BuyAmount  float64   `json:"buy_amount,omitempty"`
	SellAmount float64   `json:"sell_amount,omitempty"`
	Price      float64   `json:"price,omitempty"` // buy amount / sell amount
	CreateTime time.Time `json:"create_time"`
	ExecTime   time.Time `json:"exec_time"`
	Leverage   int       `json:"leverage"`
	Long       bool      `json:"long"`
}

// Insert adds the limit to the database for potential execution
func (l *Limit) Insert(sesh *arango.Sesh) error {
	return sesh.CreateDoc("limits", l)
}

// InsertMarket adds the limit to database to be executed upon the next price
// update
func (l *Limit) InsertMarket(sesh *arango.Sesh) error {
	return sesh.CreateDoc("pending", l)
}

// Execute assumes the limit order is valid and changes the user's balance
// accordingly, being followed by deleting the limit order from the database
func (l *Limit) Execute(sesh *arango.Sesh) error {
	// get the user's balance
	// check to see that they have the proper amount to sell
	// alter the balance
	// update the balance
	// remove the old limit order
	return nil
}

func (l *Limit) ExecuteMarket(sesh *arango.Sesh) error {
	return nil
}

// IsValid checks to see if the limit is valid
func (l *Limit) IsValid(sesh *arango.Sesh) (bool, error) {
	// lookup the price of the assets
	currPrice, err := arango.FetchLatestPrice(sesh, l.Sell)
	if err != nil {
		return false, errors.Wrap(err, "could not check limit validity")
	}
	switch {
		case l.long
	}
}
