package arango

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/pkg/errors"
)

// FilterMany feeds into the filterManyTemplate to generate a query to filter
// many
type FilterMany struct {
	Filters   []Filter
	ReturnVal string
}

// GenQuery generates an arangodb query from data stored in the FilterMany caller
func (fm *FilterMany) GenQuery() (string, error) {
	temp, err := template.New("filterMany").Parse(getManyTemp)
	if err != nil {
		return "", err
	}
	return strGen(temp, fm)
}

// Filter holds data for a filter operation in an arangodb query template
type Filter struct {
	// Field represents the name of the arangodb object's field
	Field    string
	Value    string
	Operator string
}

// this query template is not the most ideal, but sorting in arango is either slow or does
// not preserve order SO I'm sorting in golang instead...
const getManyTemp = `
let out = (
	for stamp in stamps
		filter {{range .Filters}} stamp.{{.Field}} == "{{.Value}}" {{.Operator}}{{end}}
		return {{.ReturnVal}}
)
return out
`

// FieldFilter generates an arangodb query with multiple filters on the same field.
// allows for easy obj.field == "value1" || obj.field == "value2"
func FieldFilter(field, op, returnVal string, values ...string) (string, error) {
	var out FilterMany
	out.ReturnVal = returnVal
	for i, val := range values {
		// set operator to nothing if this is the last filter
		if i+1 == len(values) {
			op = ""
		}
		out.Filters = append(out.Filters, Filter{Field: field, Value: val, Operator: op})
	}
	return out.GenQuery()
}

/////////////////////////////////
// Utility funcs
///////////////////////////////
func strGen(temp *template.Template, data interface{}) (string, error) {
	buf := new(bytes.Buffer)
	err := temp.Execute(buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

const LatestBalanceQ = `
for b in balances
    sort b._key desc
    filter b.user == "%s"
    limit 1
    return b 
`

func LatestBalance(sesh *Sesh, user string) (*Balance, error) {
	var bal Balance
	err := sesh.Execute(fmt.Sprintf(LatestBalanceQ, user), &bal)
	return &bal, err
}

const StampSeries = `
let out = (
	for s in stamps
		sort b._key asc
		filter b.time > "%s"
		filter b.time < "%s"
		return s
)
return out
`

const StampClean = `
for s in stamps
	filter s.market_cap == 0
	remove s._key in stamps
`

const LatestPrice = `
for s in stamps
    filter s.symbol == "BTC"
	sort s._key desc
	limit 1
	return s.price
`

func FetchLatestPrice(sesh *Sesh, symbol string) (float64, error) {
	var price float64
	err := sesh.Execute(fmt.Sprintf(LatestPrice, symbol), &price)
	return price, err
}

const UserChannelQ = `
for u in users
	filter u._key == "%s"
	return u.channel_id
`

func UserChanID(sesh *Sesh, user string) (string, error) {
	var id string
	err := sesh.Execute(fmt.Sprintf(LatestPrice, user), &id)
	return id, err
}

func RemoveLimit(sesh *Sesh, key string) error {
	col, err := sesh.GetCol("limits")
	if err != nil {
		return errors.Wrap(err, "failure to remove limit order:")
	}
	_, err = col.RemoveDocument(sesh.Ctx, key)
	if err != nil {
		return errors.Wrap(err, "failure to remove limit order:")
	}
	return nil
}

func ExportStamps(sesh *Sesh, incr int) ([]*Stamp, []string, error) {
	const query = `
	let out = (
		for s in stamps
			sort s._key asc
			limit %d
			return s
	)
	return out
	`
	var out []*Stamp
	err := sesh.Execute(fmt.Sprintf(query, incr), &out)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failure to export stamps:")
	}
	var keys []string
	for _, s := range out {
		keys = append(keys, s.Key)
	}
	return out, keys, nil
}

func RemoveStamps(sesh *Sesh, keys []string) error {
	col, err := sesh.GetCol("stamps")
	if err != nil {
		return err
	}
	_, _, err = col.RemoveDocuments(sesh.Ctx, keys)
	return err
}
