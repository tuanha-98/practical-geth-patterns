package subscription

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

// core/tx_pool.go
type TxPool struct {
	txFeed event.Feed
}

// api backend methods
// eth/api_backend.go
type EthApiBackend struct {
	eth *Ethereum
}

func (b *EthApiBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.eth.txPool.txFeed.Subscribe(ch)
}

// ethereum methods
// eth/backend.go
type Ethereum struct {
	txPool     *TxPool
	ApiBackend *EthApiBackend
}

func New() *Ethereum {
	eth := &Ethereum{
		txPool: &TxPool{},
	}
	eth.ApiBackend = &EthApiBackend{eth}
	return eth
}

func (e *Ethereum) TxPool() *TxPool { return e.txPool }

func (e *Ethereum) APIs() []rpc.API {
	apis := []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicFilterAPI(e.ApiBackend),
		},
	}

	return apis
}

// eth/filters/filter_system.go
type Type byte

const (
	UnknownSubscription Type = iota
	LogsSubscription
	PendingLogsSubscription
	MinedAndPendingLogsSubscription
	PendingTransactionsSubscription
	BlocksSubscription
	LastIndexSubscription
)

type subscription struct {
	id        rpc.ID
	typ       Type
	created   time.Time
	logsCrit  ethereum.FilterQuery
	logs      chan []*types.Log
	txs       chan []*types.Transaction
	headers   chan *types.Header
	installed chan struct{}
	err       chan error
}

type Backend interface {
	SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription
}

type EventSystem struct {
	backend   Backend
	txsSub    event.Subscription
	txsCh     chan core.NewTxsEvent
	install   chan *subscription
	uninstall chan *subscription
}

func NewEventSystem(backend Backend) *EventSystem {
	m := &EventSystem{
		backend:   backend,
		txsCh:     make(chan core.NewTxsEvent, 4096),
		install:   make(chan *subscription),
		uninstall: make(chan *subscription),
	}

	m.txsSub = m.backend.SubscribeNewTxsEvent(m.txsCh)

	go m.eventLoop()

	return m
}

type PublicFilterAPI struct {
	backend Backend
	events  *EventSystem
}

func NewPublicFilterAPI(backend Backend) *PublicFilterAPI {
	api := &PublicFilterAPI{
		backend: backend,
		events:  NewEventSystem(backend),
	}
	return api
}

func (api *PublicFilterAPI) NewPendingTransactions(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	fmt.Println("NewPendingTransactions subscription created ", rpcSub)

	go func() {
		fmt.Println("Inside goroutine for pending transactions.")
		txs := make(chan []*types.Transaction, 128)
		fmt.Println(txs)

		pendingTxSub := api.events.SubscribePendingTxEvents(txs)
		fmt.Println("Pending transaction subscription started ", pendingTxSub)
		for {
			select {
			case txs := <-txs:
				fmt.Println("Received transactions:", len(txs))
				if len(txs) == 0 {
					fmt.Println("Received empty transaction list.")
				}
				for _, tx := range txs {
					fmt.Println("Notifying transaction:", tx.Hash().Hex())
					notifier.Notify(rpcSub.ID, tx.Hash())
				}
			case <-rpcSub.Err():
				pendingTxSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

type FilterCriteria struct {
	FromBlock *big.Int
	ToBlock   *big.Int
	Addresses []common.Address
	Topics    [][]common.Hash
}

type Subscription struct {
	ID        rpc.ID
	f         *subscription
	es        *EventSystem
	unsubOnce sync.Once
}

func (sub *Subscription) Err() <-chan error {
	return sub.f.err
}

func (sub *Subscription) Unsubscribe() {
	sub.unsubOnce.Do(func() {
	uninstallLoop:
		for {
			select {
			case sub.es.uninstall <- sub.f:
				break uninstallLoop
			case <-sub.f.logs:
			}
		}
		<-sub.Err()
	})
}

func (es *EventSystem) subscribe(sub *subscription) *Subscription {
	es.install <- sub
	<-sub.installed
	return &Subscription{ID: sub.id, f: sub, es: es}
}

func (es *EventSystem) SubscribePendingTxEvents(txs chan []*types.Transaction) *Subscription {
	sub := &subscription{
		id:        rpc.NewID(),
		typ:       PendingTransactionsSubscription,
		created:   time.Now(),
		logs:      make(chan []*types.Log),
		txs:       txs,
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

type filterIndex map[Type]map[rpc.ID]*subscription

func (es *EventSystem) handleTxsEvent(filters filterIndex, ev core.NewTxsEvent) {
	for _, f := range filters[PendingTransactionsSubscription] {
		f.txs <- ev.Txs
	}
}

func (es *EventSystem) SubscribeLogs(crit ethereum.FilterQuery, logs chan []*types.Log) (*Subscription, error) {
	var from, to rpc.BlockNumber
	if crit.FromBlock == nil {
		from = rpc.LatestBlockNumber
	} else {
		from = rpc.BlockNumber(crit.FromBlock.Int64())
	}
	if crit.ToBlock == nil {
		to = rpc.LatestBlockNumber
	} else {
		to = rpc.BlockNumber(crit.ToBlock.Int64())
	}

	// only interested in pending logs
	if from == rpc.PendingBlockNumber && to == rpc.PendingBlockNumber {
		return es.subscribePendingLogs(crit, logs), nil
	}

	return nil, fmt.Errorf("invalid from and to block combination: from > to")
}

func (es *EventSystem) subscribePendingLogs(crit ethereum.FilterQuery, logs chan []*types.Log) *Subscription {
	sub := &subscription{
		id:        rpc.NewID(),
		typ:       PendingLogsSubscription,
		logsCrit:  crit,
		created:   time.Now(),
		logs:      logs,
		txs:       make(chan []*types.Transaction),
		headers:   make(chan *types.Header),
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

func (es *EventSystem) eventLoop() {
	defer es.txsSub.Unsubscribe()
	index := make(filterIndex)

	for i := UnknownSubscription; i < LastIndexSubscription; i++ {
		index[i] = make(map[rpc.ID]*subscription)
	}

	for {
		select {
		case ev := <-es.txsCh:
			fmt.Println("Received txs event:", len(ev.Txs), "transactions")
			es.handleTxsEvent(index, ev)
		case <-es.txsSub.Err():
			return
		}
	}
}

// node/node.go
type Node struct {
	ethService *Ethereum
	wsHandler  *rpc.Server
}

func (n *Node) Start() error {
	if err := n.StartRPC(); err != nil {
		return err
	}
	return nil
}

func (n *Node) StartRPC() error {
	log.Println("Starting WebSocket server on ws://localhost:8545")
	handler := rpc.NewServer()

	for _, api := range n.ethService.APIs() {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			log.Printf("Failed to register API %s: %v", api.Namespace, err)
		}
	}
	log.Println("WebSocket handler registered successfully")

	go func() {
		if err := http.ListenAndServe(":8545", handler.WebsocketHandler([]string{"localhost:8545"})); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	n.wsHandler = handler

	return nil
}