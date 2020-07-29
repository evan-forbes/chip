package trade

import (
	"time"

	"github.com/evan-forbes/chip/arango"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2/disc"
)

// UpdatePositions checks for liquidations and updates the historic value of each position
func UpdatePositions(srv *disc.Server, sesh *arango.Sesh) error {

}

// Position describes all pertinant data for a position
type Position struct {
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	Alive      bool      `json:"alive"`
	LiqPrice   float64   `json:"liquidation"`
	Liquidated bool      `json:"liquidated"`
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
	return nil
}

// Value calculates the current worth of the position and indicates if it subject to liquidation
func (p *Position) Value(sesh *arango.Sesh) (float64, bool, error) {
	// get fresh price data
	buyPrice, err := arango.FetchLatestPrice(sesh, p.Buy)
	if err != nil {
		return 0, false, errors.Wrap(err, "failure to check value of coin")
	}
	sellPrice, err := arango.FetchLatestPrice(sesh, p.Sell)
	if err != nil {
		return 0, false, errors.Wrap(err, "failure to check value of coin")
	}
	// find the percent change of the starting price
	currPrice := buyPrice / sellPrice
	percChange := (currPrice - p.Price) / p.Price
	dir := 1.0
	if !p.Long {
		dir = -1.0
	}
	delta := percChange * float64(p.Leverage) * dir
	var liq bool
	if delta < -1.0 {
		liq = true
	}
	return delta * p.CollAmount, liq, nil
}

// PosVal represents the value of a position at a point in time
type PosVal struct {
	Time     time.Time `json:"time"`
	Value    float64   `json:"value"`
	Position string    `json:"position"`
}
