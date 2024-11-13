package subscription

import (
	"encoding/json"
	"log"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/gorilla/websocket"
)

type NewPendingTxResponse struct {
	JsonRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
		Subscription string `json:"subscription"`
		Result       string `json:"result"`
	} `json:"params"`
}

type SubscriptionResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  string `json:"result"`
}

func TestEthereumNodeWebSocket(t *testing.T) {
	eth := New()
	node := &Node{
		ethService: eth,
	}

	if err := node.StartRPC(); err != nil {
		t.Fatalf("Failed to start RPC server: %v", err)
	}

	time.Sleep(2 * time.Second)

	wsURL := "ws://localhost:8545"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_subscribe",
		"params":  []interface{}{"newPendingTransactions"},
	}

	reqBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal JSON-RPC request: %v", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, reqBytes); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}
	log.Println("Sent subscription request")

	subscriptionConfirmed := make(chan struct{})
	errCh := make(chan error, 1)

	var wgRead sync.WaitGroup
	wgRead.Add(9)

	go func() {
		defer close(errCh)
		for {
			defer wgRead.Done()
			time.Sleep(2 * time.Second)
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}

			var subResponse SubscriptionResponse
			if err := json.Unmarshal(msg, &subResponse); err == nil && subResponse.Result != "" {
				log.Println("Subscription confirmed: ", subResponse)
				close(subscriptionConfirmed)
			} else {
				var txResponse NewPendingTxResponse
				if err := json.Unmarshal(msg, &txResponse); err == nil && txResponse.Params.Result != "" {
					log.Println("New pending txs: ", txResponse)
				}
			}
		}
	}()

	var wgWrite sync.WaitGroup
	go func() {
	loop:
		for {
			select {
			case err := <-errCh:
				log.Fatalf("Error: %v", err)
			case <-subscriptionConfirmed:
				transactions := []*types.Transaction{
					types.NewTransaction(0, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
					types.NewTransaction(1, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
					types.NewTransaction(2, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
					types.NewTransaction(3, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
					types.NewTransaction(4, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
					types.NewTransaction(5, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
					types.NewTransaction(6, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
				}
				wgWrite.Add(len(transactions))

				for _, tx := range transactions {
					go func(tx *types.Transaction) {
						defer wgWrite.Done()
						ev := core.NewTxsEvent{Txs: []*types.Transaction{tx}}
						node.ethService.txPool.txFeed.Send(ev)
					}(tx)
				}
				break loop
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()

	wgRead.Wait()
	wgWrite.Wait()
}
