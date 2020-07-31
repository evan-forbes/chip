package arango

import (
	"bytes"
	"fmt"
	"log"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/pkg/errors"
)

/*
data schema

/////////// Collections ////////////
balances #the balances of the users # updated everytime a trade occurs
	key = simple
	data: {"user": "Boo", "timestamp": "time here", "balances": {"ABC": 10000, "XYZ": 500}}

trades # all of the successfull trades # updated everytime a trade occurs
	key = default
	data: {"user": "Boo", "timestamp": "time here", executed: "time here", "buying": "ABC", "selling": "XYZ", "price": 1234.56, "buy_amount": 222, "sell_amount": 444.44}

pending # all pending trades, ported to to trades everytime the price is updated.

*/

// Balance represents the state of a user portfolio at a give time
type Balance struct {
	User      string             `json:"user"`
	Balances  map[string]float64 `json:"balances"`
	Timestamp time.Time          `json:"timestamp"`
	Prices    map[string]float64
	Total     float64
}

// Total calculates the total prices given that the prices
func (b *Balance) CalcTotal() (float64, error) {
	var total float64
	for coin, amount := range b.Balances {
		price, has := b.Prices[coin]
		if !has {
			return 0, errors.Errorf("no price value found for coin %s", coin)
		}
		total = total + (amount * price)
	}
	b.Total = total
	return total, nil
}

// Clean deletes coins that are too small to matter
func (b *Balance) Clean(sesh *Sesh) {
	for coin, bal := range b.Balances {
		if bal < 0.0000009 {
			delete(b.Balances, coin)
		}
	}
	if sesh != nil {
		err := sesh.CreateDoc("balances", b)
		if err != nil {
			log.Println("failure to insert clean balance", err)
		}
	}
}

// LookupPrices searches for the most recent prices for each asset in b.Balances
func (b *Balance) LookupPrices(sesh *Sesh) error {
	const query = `
	for s in fulltext(stamps, "symbol", "%s")
		sort s._key desc
		limit 1
		return s.price
	`
	b.Prices = make(map[string]float64)
	for coin := range b.Balances {
		// fetch the latest price for the coin
		var price float64
		err := sesh.Execute(fmt.Sprintf(query, coin), &price)
		if err != nil {
			log.Println("failure to get price for", coin, b.User, err)
			return errors.Wrap(err, "failure to lookup prices")
		}
		b.Prices[coin] = price
	}
	return nil
}

// Render returns a formatted string that descibes the state of the balance
func (b *Balance) Render() string {
	var buf bytes.Buffer
	twr := tabwriter.NewWriter(&buf, 1, 1, 2, ' ', 0)
	// make and execute the template
	t := template.Must(template.New("portfolio").Parse(balanceTempl))
	err := t.Execute(twr, b)
	if err != nil {
		fmt.Println("error in template exec:", err)
	}

	err = twr.Flush()
	if err != nil {
		fmt.Println("failure to render balance", err)
	}
	out := buf.String()
	fmt.Println(out)
	return out
}

const balanceTempl = `
@{{.User}}{{ range $asset, $bal := .Balances}}
{{$bal}}	{{$asset}}	${{ index $.Prices $asset}}{{end}}
TOTAL	${{.Total}}
`

// Trade represents a pending or successful trade. Trades become successful after
// execution.
type Trade struct {
	Key        string    `json:"_key,omitempty"`
	Sell       string    `json:"sell"`
	Buy        string    `json:"buy"`
	User       string    `json:"user"`
	BuyAmount  float64   `json:"buy_amount,omitempty"`
	SellAmount float64   `json:"sell_amount,omitempty"`
	Price      float64   `json:"price,omitempty"`
	ChanID     string    `json:"channel_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// Stamp holds data for a coin's score at a given time
type Stamp struct {
	Key               string    `json:"_key,omitempty"`
	Name              string    `json:"name"`
	Symbol            string    `json:"symbol"`
	Cap               float64   `json:"market_cap"`
	CirculatingSupply float64   `json:"circulating_supply"`
	TotalSupply       float64   `json:"total_supply"`
	MaxSupply         float64   `json:"max_supply"`
	Price             float64   `json:"price"`
	Volume            float64   `json:"volume24"`
	Time              time.Time `json:"time"`
}
