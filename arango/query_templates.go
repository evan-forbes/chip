package arango

import (
	"bytes"
	"text/template"
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

const LatestBalance = `
for b in balances
    sort b._key desc
    filter b.user == "%s"
    limit 1
    return b 
`
