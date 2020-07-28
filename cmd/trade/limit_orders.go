package trade

import (
	"context"
	"fmt"
	"time"

	"github.com/evan-forbes/chip/arango"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2/disc"
)

// CheckLimits fetches all limit orders of a given grouping, either pending
// (market orders) or limits (limit orders)
func CheckLimits(ctx context.Context, srv *disc.Server, group string) error {
	const query = `
	let out = (
		for l in %s
			return l
	)
	return out
	`
	errMsg := "failure check limits"
	// connect to the db
	sesh, err := arango.NewSesh(ctx, "cookie")
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	var limits []Limit
	err = sesh.Execute(fmt.Sprintf(query, group), &limits)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	for _, lim := range limits {
		ready, err := lim.IsReady(sesh)
		if err != nil {
			return errors.Wrap(err, errMsg)
		}
		if ready {
			err := lim.Execute(srv, sesh)
			if err != nil {
				return errors.Wrap(err, errMsg)
			}
		}
	}
	return nil
}

// Limit describes an order that could be executed by chip
type Limit struct {
	Key        string    `json:"_key,omitempty"`
	Sell       string    `json:"sell"`
	Buy        string    `json:"buy"`
	Collat     string    `json:"collateral,omitempty"` // asset that the user locks/loses/gets paid in
	User       string    `json:"user"`
	BuyAmount  float64   `json:"buy_amount"`
	SellAmount float64   `json:"sell_amount"`
	Price      float64   `json:"price"`               // buy amount / sell amount
	LiqPrice   float64   `json:"liquid_price"`        // price at which position is worthless
	CreateTime time.Time `json:"create_time"`         // time when order was submitted to chip
	ExecTime   time.Time `json:"exec_time,omitempty"` // time when the order was executed
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
func (l *Limit) Execute(srv *disc.Server, sesh *arango.Sesh) error {
	// get the user's balance
	bal, err := arango.LatestBalance(sesh, l.User)
	if err != nil {
		return errors.Wrap(err, "could not execute limit order")
	}
	// get the user's channel id to write to
	id, err := arango.UserChanID(sesh, l.User)
	if err != nil {
		return errors.Wrap(err, "failure to execute limit order:")
	}
	switch {
	// limit should be executed at market
	case l.Price == 0 && l.Leverage == 0:
		err = l.executeMarketTrade(srv, sesh, bal, id)
	// limit order should be executed at market prices
	case l.Price == 0 && l.Leverage > 0:
		err = l.executeMarketLevered(srv, sesh, bal, id)
	// limit order is not levered
	case l.Price > 0 && l.Leverage == 0:
		err = l.executeTrade(srv, sesh, bal, id)
	// limit order is levered
	case l.Price > 0 && l.Leverage > 0:
		err = l.executeLevered(srv, sesh, bal, id)
	}

	if err != nil {
		return err
	}

	// remove the old limit order
	arango.RemoveLimit(sesh, l.Key)
	return nil
}

// executeTrade alters a users balances according to limit order. It assumes the
// order is ready to be executed and is valid
func (l *Limit) executeTrade(srv *disc.Server, sesh *arango.Sesh, bal arango.Balance, id string) error {
	srv.Message(id, l.renderTrade())
	return nil
}

func (l *Limit) executeMarketTrade(srv *disc.Server, sesh *arango.Sesh, bal arango.Balance, id string) error {
	srv.Message(id, l.renderTrade())
	return nil
}

func (l *Limit) executeLevered(srv *disc.Server, sesh *arango.Sesh, bal arango.Balance, id string) error {
	srv.Message(id, l.renderLevered())
	return nil
}

func (l *Limit) executeMarketLevered(srv *disc.Server, sesh *arango.Sesh, bal arango.Balance, id string) error {
	srv.Message(id, l.renderLevered())
	return nil
}

// IsReady checks to see if the limit is valid
func (l *Limit) IsReady(sesh *arango.Sesh) (bool, error) {
	// lookup the price of the assets
	sellPrice, err := arango.FetchLatestPrice(sesh, l.Sell)
	if err != nil {
		return false, errors.Wrap(err, "could not check limit validity")
	}
	buyPrice, err := arango.FetchLatestPrice(sesh, l.Buy)
	if err != nil {
		return false, errors.Wrap(err, "could not check limit validity")
	}
	currPrice := buyPrice / sellPrice
	switch {
	case l.Long && currPrice < l.Price:
		return true, nil
	case l.Long && currPrice > l.Price:
		return false, nil
	case !l.Long && currPrice < l.Price:
		return false, nil
	case !l.Long && currPrice > l.Price:
		return true, nil
	}
	return false, nil
}

func (l *Limit) renderTrade() string {
	return fmt.Sprintf(
		"limit order has been executed: bought %f.3 %s using %f.3 %s",
		l.BuyAmount,
		l.Buy,
		l.SellAmount,
		l.Sell,
	)
}

func (l *Limit) renderLevered() string {
	dir := "short"
	if l.Long {
		dir = "long"
	}
	return fmt.Sprintf(
		"position has been opended: %d x %s on %s relative to %s using %s as collateral. Liquidation at %f.3 %s/%s",
		dir,
		l.Buy,
		l.Sell,
		l.Collat,
		l.LiqPrice,
		l.Buy,
		l.Sell,
	)
}
