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
	"github.com/moonstream-to/seer/blockchain/game7_testnet"
	"github.com/moonstream-to/seer/blockchain/imx_zkevm"
	"github.com/moonstream-to/seer/blockchain/imx_zkevm_sepolia"
	"github.com/moonstream-to/seer/blockchain/mantle"
	"github.com/moonstream-to/seer/blockchain/mantle_sepolia"
	"github.com/moonstream-to/seer/blockchain/polygon"
	"github.com/moonstream-to/seer/blockchain/sepolia"
	"github.com/moonstream-to/seer/blockchain/xai"
	"github.com/moonstream-to/seer/blockchain/xai_sepolia"
	"github.com/moonstream-to/seer/indexer"
	"google.golang.org/protobuf/proto"
)

func NewClient(chain, url string, timeout int) (BlockchainClient, error) {
	if chain == "ethereum" {
		client, err := ethereum.NewClient(url, timeout)
		return client, err
	} else if chain == "sepolia" {
		client, err := sepolia.NewClient(url, timeout)
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
	} else if chain == "game7_testnet" {
		client, err := game7_testnet.NewClient(url, timeout)
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
	} else if chain == "imx_zkevm" {
		client, err := imx_zkevm.NewClient(url, timeout)
		return client, err
	} else if chain == "imx_zkevm_sepolia" {
		client, err := imx_zkevm_sepolia.NewClient(url, timeout)
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
	FetchAsProtoBlocksWithEvents(*big.Int, *big.Int, bool, int) ([]proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, []indexer.LogIndex, uint64, error)
	ProcessBlocksToBatch([]proto.Message) (proto.Message, error)
	DecodeProtoEntireBlockToJson(*bytes.Buffer) (*seer_common.BlocksBatchJson, error)
	DecodeProtoEntireBlockToLabels(*bytes.Buffer, map[string]map[string]map[string]string) ([]indexer.EventLabel, []indexer.TransactionLabel, error)
	DecodeProtoTransactionsToLabels([]string, map[uint64]uint64, map[string]map[string]map[string]string) ([]indexer.TransactionLabel, error)
	ChainType() string
}

func CrawlEntireBlocks(client BlockchainClient, startBlock *big.Int, endBlock *big.Int, debug bool, maxRequests int) ([]proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, []indexer.LogIndex, uint64, error) {
	blocks, blocksIndex, txsIndex, eventsIndex, blocksSize, pBlockErr := client.FetchAsProtoBlocksWithEvents(startBlock, endBlock, debug, maxRequests)
	if pBlockErr != nil {
		return nil, nil, nil, nil, 0, pBlockErr
	}

	return blocks, blocksIndex, txsIndex, eventsIndex, blocksSize, nil
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
