package arango

import (
	"context"
	"encoding/json"
	"io/ioutil"

	driver "github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/pkg/errors"
)

// Sesh represents a connection to an arango instance
type Sesh struct {
	db          driver.Database
	client      driver.Client
	Collections map[string]driver.Collection
	Ctx         context.Context
}

// NewSesh establishes a connection to an arangodb instance
func NewSesh(ctx context.Context, db string) (*Sesh, error) {
	client, err := Connect("http://192.168.0.33:8529")
	if err != nil {
		return nil, err
	}
	database, err := client.Database(ctx, db)
	if err != nil {
		return nil, err
	}
	return &Sesh{
		db:     database,
		client: client,
		Ctx:    ctx,
	}, nil
}

// Execute completes a query and scans the result(s) into data
func (s *Sesh) Execute(query string, data interface{}) error {
	cursor, err := s.db.Query(s.Ctx, query, nil)
	if err != nil {
		return errors.Wrap(err, "Issue with query:")
	}
	defer cursor.Close()
	if data == nil {
		return nil
	}
	_, err = cursor.ReadDocument(s.Ctx, data)
	if err != nil {
		return errors.Wrap(err, "Could not read document")
	}
	return nil
}

// CreateDoc wraps the arango driver's collection CreateDoc method, supplementing
// with the Sesh's own context
func (s *Sesh) CreateDoc(col string, data interface{}) (err error) {
	collection, err := s.GetCol(col)
	if err != nil {
		return err
	}
	_, err = collection.CreateDocument(s.Ctx, data)
	return err
}

// GetCol retrieves a collection using established Sesh data
func (s *Sesh) GetCol(colName string) (driver.Collection, error) {
	col, err := s.db.Collection(s.Ctx, colName)
	if err != nil {
		return col, errors.Wrapf(err, "Could not access collection: %s", colName)
	}
	return col, nil
}

// Connect to an arangodb instance using Client
func Connect(host string) (driver.Client, error) {
	crds, err := creds()
	if err != nil {
		return nil, err
	}
	conn, err := http.NewConnection(
		http.ConnectionConfig{
			Endpoints: []string{host},
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "::Retry:: ::WriteLocal:: could not connect to arangodb")
	}
	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(crds.ArangoUser, crds.ArangoPass),
	})
	if err != nil {
		return nil, errors.Wrap(err, "::Retry:: ::WriteLocal:: could not connect to arangodb")
	}
	return c, nil
}

// GetCol will try and return a collection object from an arango isntance
func GetCol(ctx context.Context, client driver.Client, dbName, colName string) (driver.Database, driver.Collection, error) {
	db, err := client.Database(ctx, dbName)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Could not access databse: %s", dbName)
	}
	col, err := db.Collection(ctx, colName)
	if err != nil {
		return db, col, errors.Wrapf(err, "Could not access collection: %s", colName)
	}
	return db, col, nil
}

type cred struct {
	ArangoPass string `json:"ARANGO_PASS"`
	ArangoUser string `json:"ARANGO_USER"`
	CMCSecret  string `json:"CMC_API_KEY"`
}

// open creds file
func creds() (*cred, error) {
	var out cred
	jsonFile, err := ioutil.ReadFile("/home/evan/.creds/arango-cmc.json")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonFile, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// // Writer manages a connection to an arangodb instance, and writes
// // everything put into chan
// type Writer struct {
// 	client      driver.Client
// 	Sink        chan *parse.Parsed
// 	Errc        chan error
// 	ctx         context.Context
// 	Collections map[string]map[string]driver.Collection
// }

// func NewWriter(ctx context.Context, client driver.Client) (*Writer, <-chan error) {
// 	wr := Writer{
// 		Sink:   make(chan *parse.Parsed),
// 		Errc:   make(chan error, 1),
// 		ctx:    ctx,
// 		client: client,
// 	}
// 	return &wr, wr.Errc
// }

// func (wr *Writer) Conn(retries int, host string) error {
// 	var gerr error
// 	for i := 0; i < retries; i++ {
// 		c, err := Connect(host)
// 		if err != nil {
// 			time.Sleep(time.Second * time.Duration(i) * time.Duration(2))
// 			gerr = err
// 			continue
// 		}
// 		wr.client = c
// 		return nil
// 	}
// 	return errors.Wrapf(gerr, "could not connect to db after %d retries", retries)
// }

// func (wr *Writer) SetCol(dbname, colname string) error {
// 	db, err := wr.client.Database(wr.ctx, dbname)
// 	if err != nil {
// 		return err
// 	}
// 	col, err := db.Collection(wr.ctx, colname)
// 	if err != nil {
// 		return err
// 	}
// 	wr.Collections[dbname][colname] = col
// 	return nil
// }

// func (wr *Writer) StartWriting() {
// 	for {
// 		select {
// 		case data, ok := <-wr.Sink:
// 			if !ok {
// 				return
// 			}
// 			// find the desired collection
// 			// if not found notify via error channel

// 			meta, err := col.CreateDocument(wr.ctx, data.Data)
// 			if err != nil {
// 				fmt.Println(err)
// 				wr.Errc <- errors.Wrapf(
// 					err,
// 					"#:REBOOT:# could not write to db: %s col: %s",
// 					data.DB,
// 					data.Col,
// 				)
// 			}
// 			fmt.Println(meta.Key, "added")
// 		case <-wr.ctx.Done():
// 			return
// 		}
// 	}
// }
