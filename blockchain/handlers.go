package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/moonstream-to/seer/blockchain/arbitrum_one"
	"github.com/moonstream-to/seer/blockchain/arbitrum_sepolia"
	seer_common "github.com/moonstream-to/seer/blockchain/common"
	"github.com/moonstream-to/seer/blockchain/ethereum"
	"github.com/moonstream-to/seer/blockchain/game7_orbit_arbitrum_sepolia"
	"github.com/moonstream-to/seer/blockchain/mantle"
	"github.com/moonstream-to/seer/blockchain/mantle_sepolia"
	"github.com/moonstream-to/seer/blockchain/polygon"
	"github.com/moonstream-to/seer/blockchain/xai"
	"github.com/moonstream-to/seer/blockchain/xai_sepolia"
	"github.com/moonstream-to/seer/indexer"
	"google.golang.org/protobuf/proto"
)

func NewClient(chain, url string, timeout int) (BlockchainClient, error) {
	if chain == "ethereum" {
		client, err := ethereum.NewClient(url, timeout)
		return client, err
	} else if chain == "polygon" {
		client, err := polygon.NewClient(url, timeout)
		return client, err
	} else if chain == "arbitrum_one" {
		client, err := arbitrum_one.NewClient(url, timeout)
		return client, err
	} else if chain == "arbitrum_sepolia" {
		client, err := arbitrum_sepolia.NewClient(url, timeout)
		return client, err
	} else if chain == "game7_orbit_arbitrum_sepolia" {
		client, err := game7_orbit_arbitrum_sepolia.NewClient(url, timeout)
		return client, err
	} else if chain == "mantle" {
		client, err := mantle.NewClient(url, timeout)
		return client, err
	} else if chain == "mantle_sepolia" {
		client, err := mantle_sepolia.NewClient(url, timeout)
		return client, err
	} else if chain == "xai" {
		client, err := xai.NewClient(url, timeout)
		return client, err
	} else if chain == "xai_sepolia" {
		client, err := xai_sepolia.NewClient(url, timeout)
		return client, err
	} else {
		return nil, errors.New("unsupported chain type")
	}
}

type BlockData struct {
	BlockNumber    uint64
	BlockHash      string
	BlockTimestamp int64
	ChainID        string
	Data           map[string]interface{}
}

type BlockchainClient interface {
	GetLatestBlockNumber() (*big.Int, error)
	FetchAsProtoEvents(*big.Int, *big.Int, map[uint64]indexer.BlockCache, bool) ([]proto.Message, []indexer.LogIndex, error)
	FetchAsProtoBlocks(*big.Int, *big.Int, bool, int) ([]proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, map[uint64]indexer.BlockCache, error)
	FetchAsProtoEventsAndExtendBlock(*big.Int, *big.Int, []proto.Message, map[uint64]indexer.BlockCache, bool) ([]proto.Message, []indexer.LogIndex, error)
	DecodeProtoEntireBlock(*bytes.Buffer) ([]*seer_common.BlockJson, error)
	DecodeProtoEventsToLabels(*bytes.Buffer, map[uint64]uint64, map[string]map[string]map[string]string) ([]indexer.EventLabel, error)
	DecodeProtoTransactionsToLabels([]string, map[uint64]uint64, map[string]map[string]map[string]string) ([]indexer.TransactionLabel, error)
	ChainType() string
}

func CrawlEntireBlocks(client BlockchainClient, startBlock *big.Int, endBlock *big.Int, debug bool, maxRequests int) ([]proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, []indexer.LogIndex, error) {
	blocks, blocksIndex, txsIndex, blocksCache, pBlockErr := client.FetchAsProtoBlocks(startBlock, endBlock, debug, maxRequests)
	if pBlockErr != nil {
		return nil, nil, nil, nil, pBlockErr
	}

	blocks, eventsIndex, extErr := client.FetchAsProtoEventsAndExtendBlock(startBlock, endBlock, blocks, blocksCache, debug)
	if extErr != nil {
		return nil, nil, nil, nil, extErr
	}

	return blocks, blocksIndex, txsIndex, eventsIndex, nil
}

func CrawlBlocks(client BlockchainClient, startBlock *big.Int, endBlock *big.Int, debug bool, maxRequests int) ([]proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, map[uint64]indexer.BlockCache, error) {
	blocks, blocksIndex, txsIndex, blocksCache, err := client.FetchAsProtoBlocks(startBlock, endBlock, debug, maxRequests)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return blocks, blocksIndex, txsIndex, blocksCache, nil

}

func CrawlEvents(client BlockchainClient, startBlock *big.Int, endBlock *big.Int, BlockCahche map[uint64]indexer.BlockCache, debug bool) ([]proto.Message, []indexer.LogIndex, error) {
	events, eventsIndex, err := client.FetchAsProtoEvents(startBlock, endBlock, BlockCahche, debug)
	if err != nil {
		return nil, nil, err
	}

	return events, eventsIndex, nil
}

func DecodeTransactionInputData(contractABI *abi.ABI, data []byte) {
	methodSigData := data[:4]
	inputsSigData := data[4:]
	method, err := contractABI.MethodById(methodSigData)
	if err != nil {
		log.Fatal(err)
	}
	inputsMap := make(map[string]interface{})
	if err := method.Inputs.UnpackIntoMap(inputsMap, inputsSigData); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(inputsMap)
	}
	fmt.Printf("Method Name: %s\n", method.Name)
	fmt.Printf("Method inputs: %v\n", inputsMap)
}
