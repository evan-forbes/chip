package trade

import (
	"fmt"
	"time"

	"github.com/evan-forbes/chip/arango"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2/disc"
)

// UpdatePositions checks for liquidations and updates the historic value of each position
func UpdatePositions(srv *disc.Server, sesh *arango.Sesh) error {
	const query = `
	let out = (
		for p in positions
			filter p.alive == true
			return p
	)
	return out
	`
	var ps []Position
	err := sesh.Execute(query, &ps)
	if err != nil {
		return errors.Wrap(err, "failure to fetch positions")
	}
	for _, p := range ps {
		val, err := p.Value(sesh)
		if err != nil {
			return errors.Wrap(err, "failure to calculate value of position")
		}
		// liquidate position if needed
		if val.Value <= 0 {
			return p.Liquidate(srv, sesh)
		}
		// add the value to the records
		err = sesh.CreateDoc("post_val", val)
		if err != nil {
			return errors.Wrap(err, "failure to add position historical value")
		}
		// check if this position should be closed
		crossed, u, err := p.Check(sesh, val.Value)
		if err != nil {
			return errors.Wrap(err, "failure to update position: could not check for close condidtion")
		}
		if crossed {
			// notify user
			id, err := arango.UserChanID(sesh, p.User)
			if err != nil {
				return errors.Wrap(err, "failure to find user id")
			}
			srv.Message(id, fmt.Sprintf("position %s crossed %s limit, it has been closed", p.Key, u))
		}
	}
	return nil
}

// Position describes all pertinant data for a position
type Position struct {
	Start      time.Time       `json:"start"`
	End        time.Time       `json:"end"`
	Alive      bool            `json:"alive"`
	LiqPrice   float64         `json:"liquidation_price"`
	Liquidated bool            `json:"liquidated"`
	CloseCond  *CloseCondition `json:"close_condition,omitempty"`
	Dir        string
	CurrValue  float64
	Limit
}

// Close ends a position and solidifies gains or losses
func (p *Position) Close(sesh *arango.Sesh, liquidated bool) error {
	p.Alive = false
	p.End = time.Now().Round(time.Second)
	p.Liquidated = liquidated
	pos, err := sesh.GetCol("positions")
	if err != nil {
		return errors.Wrap(err, "failure to close position:")
	}
	_, err = pos.UpdateDocument(sesh.Ctx, p.Key, p)
	if err != nil {
		return errors.Wrap(err, "failure to close position:")
	}
	if liquidated {
		return nil
	}
	// add the leftover/gains to the user's balance
	// calculate the current value
	errMsg := fmt.Sprintf("!!!!!failure to add closed position value to user!!!!!! %s %s", p.User, p.Key)
	val, err := p.Value(sesh)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	// add that to the user's balance
	collPrice, err := arango.FetchLatestPrice(sesh, p.Collat)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	award := val.Value / collPrice
	err = arango.UpdateBalance(sesh, p.User, p.Collat, award)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	return nil
}

// Liquidate closes the user's position and notifies them
func (p *Position) Liquidate(srv *disc.Server, sesh *arango.Sesh) error {
	err := p.Close(sesh, true)
	if err != nil {
		return errors.Wrap(err, "failure to close position")
	}
	// notify user
	id, err := arango.UserChanID(sesh, p.User)
	if err != nil {
		return errors.Wrap(err, "failure to find user id")
	}

	return srv.Message(id, p.liquidationMessage())
}

// Value calculates the current worth of the position in USD
func (p *Position) Value(sesh *arango.Sesh) (PosVal, error) {
	var out PosVal
	// get fresh price data
	buyPrice, err := arango.FetchLatestPrice(sesh, p.Buy)
	if err != nil {
		return out, errors.Wrap(err, "failure to check value of coin")
	}
	sellPrice, err := arango.FetchLatestPrice(sesh, p.Sell)
	if err != nil {
		return out, errors.Wrap(err, "failure to check value of coin")
	}
	// get the collateral's price if it's different from the selling asset
	var collPrice float64
	if p.Collat != p.Sell {
		collPrice, err = arango.FetchLatestPrice(sesh, p.Collat)
		if err != nil {
			return out, errors.Wrap(err, "failure to check value of coin")
		}
	} else {
		collPrice = sellPrice
	}

	// find the percent change of the starting price
	currPrice := buyPrice / sellPrice
	percChange := (currPrice - p.Price) / p.Price
	dir := 1.0
	if !p.Long {
		dir = -1.0
	}
	delta := percChange * float64(p.Leverage) * dir
	out = PosVal{
		Time:     time.Now().Round(time.Second),
		Value:    (p.CollAmount * collPrice) + (delta * p.CollAmount * collPrice),
		Position: p.Key,
	}
	return out, nil
}

func (p *Position) LiquidationPrice() float64 {
	neededD := 1 / float64(p.Leverage)
	dir := float64(-1)
	if p.Long {
		dir = 1
	}
	return p.Price - (dir * neededD * p.Price)
}

func (p *Position) liquidationMessage() string {
	l := "long"
	if !p.Long {
		l = "short"
	}
	const message = `
	beloved meat bag, it is my burden to inform you that your favorite position, %s, %d x %s on %s relative to %s using %s as collateral, has reached the liquidation price and therefore met its fatefull end.
	`
	return fmt.Sprintf(message, p.Key, p.Leverage, l, p.Buy, p.Sell, p.Collat)
}

type CloseCondition struct {
	Upper float64 `json:"upper"`
	Lower float64 `json:"lower"`
}

func (p *Position) Check(sesh *arango.Sesh, val float64) (closed bool, upper string, err error) {
	if p.CloseCond == nil {
		return false, "", nil
	}
	// did the value cross the lower condition
	if val < p.CloseCond.Lower && p.CloseCond.Lower > 0 {
		err = p.Close(sesh, false)
		if err != nil {
			return false, "", errors.Wrap(err, "failure to update position")
		}
		return true, "lower", nil
	}

	// did the value cross the upper condition
	if val > p.CloseCond.Upper && p.CloseCond.Lower > 0 {
		err = p.Close(sesh, false)
		if err != nil {
			return false, "", errors.Wrap(err, "failure to update position")
		}
		return true, "upper", nil
	}
	return false, "", nil
}

// PosVal represents the value of a position at a point in time
type PosVal struct {
	Time     time.Time `json:"time"`
	Value    float64   `json:"value"` // value in USD
	Position string    `json:"position"`
}

type PosRender struct {
	*Position
	Dir string
}

func (p *Position) SetDir() {
	dir := "short"
	if p.Long {
		dir = "long"
	}
	p.Dir = dir
}
