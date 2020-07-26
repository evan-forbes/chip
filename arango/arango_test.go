package arango

import (
	"context"
	"testing"
)

func TestConnect(t *testing.T) {
	_, err := Connect("192.168.0.33")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCol(t *testing.T) {
	client, err := Connect("http://192.168.0.33:8529")
	if err != nil {
		t.Error(err)
	}
	_, _, err = GetCol(context.Background(), client, "cookie", "stamps")
	if err != nil {
		t.Error(err)
	}
}

func TestFieldFilter(t *testing.T) {
	retVal := `{"symbol": stamp.symbol, "time": stamp.time, "cap": stamp.market_cap}`
	result, err := FieldFilter("symbol", "||", retVal, "ETH", "BTC", "LTC")
	if err != nil {
		t.Error(err)
	}
	t.Log(result)
}

const exptdFieldFilterQuery = `
for stamp in stamps
	filter stamp.symbol == "ETH" || stamp.symbol == "BTC" || stamp.symbol == "LTC"
`

// func TestDB(t *testing.T) {
// 	client, err := Connect("192.168.0.33:8529")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	// fmt.Println(client)
// 	_, err = client.Database(context.Background(), "eth_test")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	versionInfo, err := client.Version(nil)
// 	if err != nil {
// 		t.Errorf("Failed to get version info: %v", err)
// 	}
// 	fmt.Printf("Database has version '%s' and license '%s'\n", versionInfo.Version, versionInfo.License)
// }

// func handleTestErrc(t *testing.T, errc <-chan error) {
// 	for err := range errc {
// 		t.Error(err)
// 	}
// }

// type taddress struct {
// 	Address string `json:"_key"`
// 	Balance string `json:"balance"`
// }

// func TestWriter(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	client, err := Connect("192.168.0.33:8529")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	wr, errc := NewWriter(ctx, client)
// 	err = wr.SetCol("eth_test", "addresses")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	// handle the incoming errors.
// 	go handleTestErrc(t, errc)
// 	go wr.StartWriting()

// 	wr.Sink <- taddress{Address: "0x04555501", Balance: "1234123412341234"}
// 	time.Sleep(time.Second * 3)
// 	cancel()
// }
