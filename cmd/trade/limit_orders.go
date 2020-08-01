package trade

import (
	"fmt"
	"log"
	"time"

	"github.com/evan-forbes/chip/arango"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2/disc"
)

// CheckLimits fetches all limit orders of a given grouping, either pending
// (market orders) or limits (limit orders)
func CheckLimits(srv *disc.Server, sesh *arango.Sesh) error {
	const query = `
	let out = (
		for l in limits
			return l
	)
	return out
	`
	errMsg := "failure check limits"
	var limits []Limit
	err := sesh.Execute(query, &limits)
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

func ExecuteMarketOrders(srv *disc.Server, sesh *arango.Sesh) error {
	const query = `
	let out = (
		for l in pending
			return l
	)
	return out
	`
	errMsg := "failure execute market orders"
	// connect to the db
	var limits []Limit
	err := sesh.Execute(query, &limits)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	for _, lim := range limits {
		err := lim.Execute(srv, sesh)
		if err != nil {
			return errors.Wrap(err, errMsg)
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
	CollAmount float64   `json:"coll_amount"`         // amount of collateral
	Price      float64   `json:"price"`               // buy amount / sell amount
	CreateTime time.Time `json:"create_time"`         // time when order was submitted to chip
	ExecTime   time.Time `json:"exec_time,omitempty"` // time when the order was executed
	Leverage   int       `json:"leverage"`
	Long       bool      `json:"long"`
	liqPrice   float64   // price at which position is worthless
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
		return errors.Wrap(err, "failure to find user during limit order execution")
	}

	// check that the user has enough collateral or amount to sell
	if l.Collat != "" {
		collBal, has := bal.Balances[l.Collat]
		if collBal < l.SellAmount || !has {
			errMsg := fmt.Sprintf("meat bag, failed to execute your limit order %s: you do not have enough %s", l.Key, l.Collat)
			srv.Message(id, errMsg)
			// remove the limit order
			return sesh.RemoveDoc("limits", l.Key)
		}
	} else {
		// check the user's sell balance
		sellBal, has := bal.Balances[l.Sell]
		if sellBal < l.SellAmount || !has {
			errMsg := fmt.Sprintf("meat bag, failed to execute your limit order: you do not have enough %s", l.Sell)
			srv.Message(id, errMsg)
			// remove the limit order
			return sesh.RemoveDoc("limits", l.Key)
		}
		l.Collat = l.Sell
	}
	switch {
	// limit should be executed at market
	case l.Price > 0 && l.Leverage == 0:
		err = l.executeTrade(sesh, bal)
	// limit order should be executed at market prices
	case l.Price == 0 && l.Leverage > 0:
		err = l.executeMarketLevered(sesh, bal)
	// limit order is not levered
	case l.Price == 0 && l.Leverage == 0:
		err = l.executeMarketTrade(sesh, bal)
	// limit order is levered
	case l.Price > 0 && l.Leverage > 0:
		err = l.executeLevered(sesh, bal)
	}
	if err != nil {
		return errors.Wrap(err, "failure to execute limit order")
	}

	// set the time of execution
	l.ExecTime = time.Now().Round(time.Second)

	// create a new balance entry using the updated balance
	bal.Timestamp = time.Now().Round(time.Second)
	err = sesh.CreateDoc("balances", bal)
	if err != nil {
		return errors.Wrap(err, "failure to execute limit order")
	}

	// notify the user and exit
	if l.Leverage != 0 {
		return srv.Message(id, l.renderLevered())
	}
	return srv.Message(id, l.renderTrade())
}

// executeTrade alters a users balances according to limit order. It assumes the
// order is ready to be executed and is valid. Uses the buy price in the limit,
// not the current buy price
func (l *Limit) executeTrade(sesh *arango.Sesh, bal *arango.Balance) error {
	// check that there is enough asset to sell
	sellPrice, err := arango.FetchLatestPrice(sesh, l.Sell)
	if err != nil {
		return err
	}
	buyPrice := l.Price
	sellCost := sellPrice * l.SellAmount
	l.BuyAmount = sellCost / buyPrice

	// adjust balances
	bal.Balances[l.Sell] = bal.Balances[l.Sell] - l.SellAmount
	bal.Balances[l.Buy] = bal.Balances[l.Buy] + l.BuyAmount

	// add the limit to trades
	// set the time of execution
	l.ExecTime = time.Now().Round(time.Second)

	err = sesh.CreateDoc("trades", l)
	if err != nil {
		fmt.Printf("could not insert: %+v", l)
		return errors.Wrap(err, "failure to insert limit trade")
	}

	err = sesh.RemoveDoc("limits", l.Key)
	if err != nil {
		fmt.Println("could not remove old limit", l)
		return errors.Wrap(err, "failure to remove old limit")
	}

	return nil
}

// executeMarketTrade alters a users balances according to limit order. It assumes the
// order is ready to be executed and is valid. Uses the buy price in the limit,
// not the current buy price
func (l *Limit) executeMarketTrade(sesh *arango.Sesh, bal *arango.Balance) error {
	// check that there is enough asset to sell
	sellPrice, err := arango.FetchLatestPrice(sesh, l.Sell)
	if err != nil {
		return err
	}
	buyPrice, err := arango.FetchLatestPrice(sesh, l.Buy)
	if err != nil {
		return err
	}
	sellCost := sellPrice * l.SellAmount
	l.BuyAmount = sellCost / buyPrice
	l.Price = buyPrice / sellPrice

	// adjust balances
	bal.Balances[l.Sell] = bal.Balances[l.Sell] - l.SellAmount
	bal.Balances[l.Buy] = bal.Balances[l.Buy] + l.BuyAmount

	// set the time of execution
	l.ExecTime = time.Now().Round(time.Second)

	err = sesh.CreateDoc("trades", l)
	if err != nil {
		log.Println(errors.Wrap(err, "failure to insert executed trade"))
	}

	// remove the old limit order
	err = sesh.RemoveDoc("pending", l.Key)
	if err != nil {
		return errors.Wrap(err, "failure to remove executed limit order")
	}

	return err
}

func (l *Limit) executeLevered(sesh *arango.Sesh, bal *arango.Balance) error {
	// check that there is enough asset to sell
	// sellPrice, err := arango.FetchLatestPrice(sesh, l.Sell)
	// if err != nil {
	// 	return err
	// }
	// buyPrice, err := arango.FetchLatestPrice(sesh, l.Buy)
	// if err != nil {
	// 	return err
	// }
	l.BuyAmount = l.SellAmount / l.Price
	l.Price = l.BuyAmount / l.SellAmount
	bal.Balances[l.Collat] = bal.Balances[l.Collat] - l.CollAmount
	// add position to positions using current price
	//
	post := &Position{
		Limit: *l,
		Start: time.Now().Round(time.Second),
		Alive: true,
	}

	lp := post.LiquidationPrice()
	l.liqPrice = lp
	post.LiqPrice = lp

	err := sesh.CreateDoc("positions", post)
	if err != nil {
		return errors.Wrap(err, "failure to insert limit postion")
	}

	// remove the old limit order
	err = sesh.RemoveDoc("limits", l.Key)
	if err != nil {
		return errors.Wrap(err, "failure to remove executed limit order")
	}
	return nil
}

func (l *Limit) executeMarketLevered(sesh *arango.Sesh, bal *arango.Balance) error {
	// check that there is enough asset to sell
	sellPrice, err := arango.FetchLatestPrice(sesh, l.Sell)
	if err != nil {
		return err
	}
	buyPrice, err := arango.FetchLatestPrice(sesh, l.Buy)
	if err != nil {
		return err
	}
	l.BuyAmount = (sellPrice * l.SellAmount) / buyPrice
	l.Price = buyPrice / sellPrice
	bal.Balances[l.Collat] = bal.Balances[l.Collat] - l.CollAmount
	// add position to positions using current price
	//
	post := &Position{
		Limit: *l,
		Start: time.Now().Round(time.Second),
		Alive: true,
	}
	lp := post.LiquidationPrice()
	l.liqPrice = lp
	post.LiqPrice = lp
	err = sesh.CreateDoc("positions", post)
	if err != nil {
		return errors.Wrap(err, "failure to insert market post")
	}
	// remove the old limit order
	err = sesh.RemoveDoc("pending", l.Key)
	if err != nil {
		return errors.Wrap(err, "failure to remove executed market limit order")
	}

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
		"limit order has been executed: bought %.3f %s using %.3f %s",
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
		"position has been opended: %d x %s on %s relative to %s using %.2f %s as collateral. Liquidation at %.3f %s/%s",
		l.Leverage,
		dir,
		l.Buy,
		l.Sell,
		l.CollAmount,
		l.Collat,
		l.liqPrice,
		l.Buy,
		l.Sell,
	)
}
