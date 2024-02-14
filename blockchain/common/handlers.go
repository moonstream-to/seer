package common

import (
	"errors"
	"math/big"

	"github.com/moonstream-to/seer/blockchain/ethereum"
	// "github.com/moonstream-to/seer/blockchain/polygon"
	// "github.com/moonstream-to/seer/blockchain/starknet"
	"github.com/moonstream-to/seer/indexer"
	"google.golang.org/protobuf/proto"
)

func wrapEthereumClient(url string) (BlockchainClient, error) {
	client, err := ethereum.NewClient(url)
	return client, err // Ensure ethereum.Client implements BlockchainClient.
}

var (
	ErrUnsupportedChainType = errors.New("unsupported chain type")
	blockchainClients       = map[string]func(string) (BlockchainClient, error){
		//"ethereum": ethereum.NewClient,
		"ethereum": wrapEthereumClient,
		// "polygon":  polygon.NewClient,
		// "starknet": starknet.NewClient,
	}
	Blocks = make(map[string][]BlockData)
)

type BlockData struct {
	BlockNumber    uint64
	BlockHash      string
	BlockTimestamp int64
	ChainID        string
	Data           map[string]interface{}
}

type BlockchainClient interface {
	// GetBlockByNumber(ctx context.Context, number *big.Int) (*proto.Message, error)
	GetLatestBlockNumber() (*big.Int, error)
	FetchAsProtoBlocks(*big.Int, *big.Int) ([]proto.Message, []proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, error)
	ChainType() string
}

// return client depends on the blockchain type
func NewClient(chainType string, url string) (BlockchainClient, error) {
	clientFunc, ok := blockchainClients[chainType]
	if !ok {
		return nil, ErrUnsupportedChainType
	}

	return clientFunc(url)
}

// crawl blocks
func CrawlBlocks(client BlockchainClient, startBlock *big.Int, endBlock *big.Int) ([]proto.Message, []proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, error) {
	blocks, transactions, blocksIndex, transactionsIndex, err := client.FetchAsProtoBlocks(startBlock, endBlock)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// index blocks and transactions

	return blocks, transactions, blocksIndex, transactionsIndex, nil

}
