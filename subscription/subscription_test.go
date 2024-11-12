package subscription

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
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
	// Step 1: Initialize Ethereum instance and Node
	eth := New()
	node := &Node{
		ethService: eth,
	}

	// Start RPC server
	if err := node.StartRPC(); err != nil {
		t.Fatalf("Failed to start RPC server: %v", err)
	}

	// Wait for the server to initialize (wait for RPC/WebSocket setup)
	time.Sleep(2 * time.Second)

	// Step 2: Connect to WebSocket
	wsURL := "ws://localhost:8545" // Ensure this matches your server's WebSocket URL
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Step 3: Send the subscription request
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

	// Step 4: Simulate event emission (mock txs for testing)
	transactions := []*types.Transaction{
		types.NewTransaction(0, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
		types.NewTransaction(1, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
		types.NewTransaction(2, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
		types.NewTransaction(3, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
		types.NewTransaction(4, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
		types.NewTransaction(5, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
		types.NewTransaction(6, common.HexToAddress("0xb794f5ea0ba39494ce83a213fffba74279579268"), new(big.Int), 0, new(big.Int), nil),
	}

	// Simulate sending transactions (mock event)
	go func() {
		ev := core.NewTxsEvent{Txs: transactions}
		api := NewPublicFilterAPI(eth.ApiBackend)
		api.events.txsCh <- ev
	}()
	// nsent := node.ethService.txPool.txFeed.Send(ev)
	// log.Println("Transactions have been simulated and sent ", nsent)

	// Step 5: Listen for events (on WebSocket connection)
	errCh := make(chan error, 1)
	messages := make(chan []byte)

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			messages <- msg
		}
	}()

	// Set a timeout for the test
	timeout := time.After(30 * time.Second)

	var receivedMessages [][]byte
loop:
	for {
		select {
		case err := <-errCh:
			t.Fatalf("Error in WebSocket reading: %v", err)
		case msg := <-messages:
			log.Println("Received message:", string(msg))
			receivedMessages = append(receivedMessages, msg)
			if len(receivedMessages) >= 2 { // Expecting at least two messages (one for subscription and one for result)
				break loop
			}
		case <-timeout:
			t.Fatalf("Test timed out waiting for WebSocket messages")
		}
	}

	// Step 6: Process the received messages
	for _, msg := range receivedMessages {
		var subResponse SubscriptionResponse
		var newPendingTx NewPendingTxResponse

		// Try unmarshaling as SubscriptionResponse
		if err := json.Unmarshal(msg, &subResponse); err == nil {
			// Successfully unmarshalled SubscriptionResponse
			fmt.Println("Received SubscriptionResponse:", subResponse)
		} else {
			// Try unmarshaling as NewPendingTxResponse
			if err := json.Unmarshal(msg, &newPendingTx); err == nil {
				// Successfully unmarshalled NewPendingTxResponse
				fmt.Println("Received NewPendingTxResponse:", newPendingTx)
			} else {
				// If both fail, print an error
				fmt.Println("Error unmarshalling message:", err)
			}
		}
	}
}
