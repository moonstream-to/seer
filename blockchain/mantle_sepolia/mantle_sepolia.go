package mantle_sepolia

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"

	seer_common "github.com/moonstream-to/seer/blockchain/common"
	"github.com/moonstream-to/seer/indexer"
)

func NewClient(url string, timeout int) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	rpcClient, err := rpc.DialContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return &Client{rpcClient: rpcClient}, nil
}

// Client is a wrapper around the Ethereum JSON-RPC client.

type Client struct {
	rpcClient *rpc.Client
}

// Client common

// ChainType returns the chain type.
func (c *Client) ChainType() string {
	return "mantle_sepolia"
}

// Close closes the underlying RPC client.
func (c *Client) Close() {
	c.rpcClient.Close()
}

// GetLatestBlockNumber returns the latest block number.
func (c *Client) GetLatestBlockNumber() (*big.Int, error) {
	var result string
	if err := c.rpcClient.CallContext(context.Background(), &result, "eth_blockNumber"); err != nil {
		return nil, err
	}

	// Convert the hex string to a big.Int
	blockNumber, ok := new(big.Int).SetString(result, 0) // The 0 base lets the function infer the base from the string prefix.
	if !ok {
		return nil, fmt.Errorf("invalid block number format: %s", result)
	}

	return blockNumber, nil
}

// BlockByNumber returns the block with the given number.
func (c *Client) GetBlockByNumber(ctx context.Context, number *big.Int) (*seer_common.BlockJson, error) {

	var rawResponse json.RawMessage // Use RawMessage to capture the entire JSON response
	err := c.rpcClient.CallContext(ctx, &rawResponse, "eth_getBlockByNumber", "0x"+number.Text(16), true)
	if err != nil {
		fmt.Println("Error calling eth_getBlockByNumber: ", err)
		return nil, err
	}

	var response_json map[string]interface{}

	err = json.Unmarshal(rawResponse, &response_json)

	delete(response_json, "transactions")

	var block *seer_common.BlockJson
	err = c.rpcClient.CallContext(ctx, &block, "eth_getBlockByNumber", "0x"+number.Text(16), true) // true to include transactions
	return block, err
}

// BlockByHash returns the block with the given hash.
func (c *Client) BlockByHash(ctx context.Context, hash common.Hash) (*seer_common.BlockJson, error) {
	var block *seer_common.BlockJson
	err := c.rpcClient.CallContext(ctx, &block, "eth_getBlockByHash", hash, true) // true to include transactions
	return block, err
}

// TransactionReceipt returns the receipt of a transaction by transaction hash.
func (c *Client) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	var receipt *types.Receipt
	err := c.rpcClient.CallContext(ctx, &receipt, "eth_getTransactionReceipt", hash)
	return receipt, err
}

func (c *Client) ClientFilterLogs(ctx context.Context, q ethereum.FilterQuery, debug bool) ([]*seer_common.EventJson, error) {
	var logs []*seer_common.EventJson
	fromBlock := q.FromBlock
	toBlock := q.ToBlock
	batchStep := new(big.Int).Sub(toBlock, fromBlock) // Calculate initial batch step

	for {
		// Calculate the next "lastBlock" within the batch step or adjust to "toBlock" if exceeding
		nextBlock := new(big.Int).Add(fromBlock, batchStep)
		if nextBlock.Cmp(toBlock) > 0 {
			nextBlock = new(big.Int).Set(toBlock)
		}

		var result []*seer_common.EventJson
		err := c.rpcClient.CallContext(ctx, &result, "eth_getLogs", struct {
			FromBlock string           `json:"fromBlock"`
			ToBlock   string           `json:"toBlock"`
			Addresses []common.Address `json:"addresses"`
			Topics    [][]common.Hash  `json:"topics"`
		}{
			FromBlock: toHex(fromBlock),
			ToBlock:   toHex(nextBlock),
			Addresses: q.Addresses,
			Topics:    q.Topics,
		})

		if err != nil {
			if strings.Contains(err.Error(), "query returned more than 10000 results") {
				// Halve the batch step if too many results and retry
				batchStep.Div(batchStep, big.NewInt(2))
				if batchStep.Cmp(big.NewInt(1)) < 0 {
					// If the batch step is too small we will skip that block
					fromBlock = new(big.Int).Add(nextBlock, big.NewInt(1))
					if fromBlock.Cmp(toBlock) > 0 {
						break
					}
					continue
				}
				continue
			} else {
				// For any other error, return immediately
				return nil, err
			}
		}

		// Append the results and adjust "fromBlock" for the next batch
		logs = append(logs, result...)
		fromBlock = new(big.Int).Add(nextBlock, big.NewInt(1))

		if debug {
			log.Printf("Fetched logs: %d", len(result))
		}

		// Break the loop if we've reached or exceeded "toBlock"
		if fromBlock.Cmp(toBlock) > 0 {
			break
		}
	}

	return logs, nil
}

// Utility function to convert big.Int to its hexadecimal representation.
func toHex(number *big.Int) string {
	return fmt.Sprintf("0x%x", number)
}

func fromHex(hex string) *big.Int {
	number := new(big.Int)
	number.SetString(hex, 0)
	return number
}

// FetchBlocksInRange fetches blocks within a specified range.
// This could be useful for batch processing or analysis.
func (c *Client) FetchBlocksInRange(from, to *big.Int, debug bool) ([]*seer_common.BlockJson, error) {
	var blocks []*seer_common.BlockJson
	ctx := context.Background() // For simplicity, using a background context; consider timeouts for production.

	for i := new(big.Int).Set(from); i.Cmp(to) <= 0; i.Add(i, big.NewInt(1)) {
		block, err := c.GetBlockByNumber(ctx, i)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)
		if debug {
			log.Printf("Fetched block number: %d", i)
		}
	}

	return blocks, nil
}

// FetchBlocksInRangeAsync fetches blocks within a specified range concurrently.
func (c *Client) FetchBlocksInRangeAsync(from, to *big.Int, debug bool, maxRequests int) ([]*seer_common.BlockJson, error) {
	var (
		blocks []*seer_common.BlockJson

		mu  sync.Mutex
		wg  sync.WaitGroup
		ctx = context.Background()
	)

	var blockNumbersRange []*big.Int
	for i := new(big.Int).Set(from); i.Cmp(to) <= 0; i.Add(i, big.NewInt(1)) {
		blockNumbersRange = append(blockNumbersRange, new(big.Int).Set(i))
	}

	sem := make(chan struct{}, maxRequests) // Semaphore to control concurrency
	errChan := make(chan error, 1)          // Handle errors to stop corrupted processing

	for _, b := range blockNumbersRange {
		wg.Add(1)
		go func(b *big.Int) {
			defer wg.Done()

			sem <- struct{}{} // Acquire semaphore

			block, getErr := c.GetBlockByNumber(ctx, b)
			if getErr != nil {
				log.Printf("Failed to fetch block number: %d, error: %v", b, getErr)
				select {
				case errChan <- getErr:
					// Record an error
				default:
					// Pass
				}
				return
			}

			mu.Lock()
			blocks = append(blocks, block)
			mu.Unlock()

			if debug {
				log.Printf("Fetched block number: %d", b)
			}

			<-sem
		}(b)
	}

	wg.Wait()
	close(sem)
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	return blocks, nil
}

// ParseBlocksWithTransactions parses blocks and their transactions into custom data structure.
// This method showcases how to handle and transform detailed block and transaction data.
func (c *Client) ParseBlocksWithTransactions(from, to *big.Int, debug bool, maxRequests int) ([]*MantleSepoliaBlock, error) {
	var blocksWithTxsJson []*seer_common.BlockJson
	var fetchErr error
	if maxRequests > 1 {
		blocksWithTxsJson, fetchErr = c.FetchBlocksInRangeAsync(from, to, debug, maxRequests)
	} else {
		blocksWithTxsJson, fetchErr = c.FetchBlocksInRange(from, to, debug)
	}
	if fetchErr != nil {
		return nil, fetchErr
	}

	var parsedBlocks []*MantleSepoliaBlock
	for _, blockAndTxsJson := range blocksWithTxsJson {
		// Convert BlockJson to Block and Transactions as required.
		parsedBlock := ToProtoSingleBlock(blockAndTxsJson)

		for _, txJson := range blockAndTxsJson.Transactions {
			txJson.BlockTimestamp = blockAndTxsJson.Timestamp

			parsedTransaction := ToProtoSingleTransaction(&txJson)
			parsedBlock.Transactions = append(parsedBlock.Transactions, parsedTransaction)
		}

		parsedBlocks = append(parsedBlocks, parsedBlock)
	}

	return parsedBlocks, nil
}

func (c *Client) ParseEvents(from, to *big.Int, blocksCache map[uint64]indexer.BlockCache, debug bool) ([]*MantleSepoliaEventLog, error) {
	logs, err := c.ClientFilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: from,
		ToBlock:   to,
	}, debug)

	if err != nil {
		fmt.Println("Error fetching logs: ", err)
		return nil, err
	}

	var parsedEvents []*MantleSepoliaEventLog
	for _, log := range logs {
		parsedEvent := ToProtoSingleEventLog(log)
		parsedEvents = append(parsedEvents, parsedEvent)
	}

	return parsedEvents, nil
}

func (c *Client) FetchAsProtoBlocks(from, to *big.Int, debug bool, maxRequests int) ([]proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, map[uint64]indexer.BlockCache, error) {
	parsedBlocks, err := c.ParseBlocksWithTransactions(from, to, debug, maxRequests)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var blocksProto []proto.Message

	var blocksIndex []indexer.BlockIndex
	var transactionsIndex []indexer.TransactionIndex

	blocksCache := make(map[uint64]indexer.BlockCache)

	for bI, block := range parsedBlocks {
		blocksProto = append(blocksProto, block) // Assuming block is a proto.Message

		for txI, transaction := range block.Transactions {
			selector := "0x"

			if len(transaction.Input) > 10 {
				selector = transaction.Input[:10]
			}

			transactionsIndex = append(transactionsIndex, indexer.TransactionIndex{
				BlockNumber:      transaction.BlockNumber,
				BlockHash:        transaction.BlockHash,
				BlockTimestamp:   transaction.BlockTimestamp,
				FromAddress:      transaction.FromAddress,
				ToAddress:        transaction.ToAddress,
				RowID:            uint64(txI),
				Selector:         selector, // First 10 characters of the input data 0x + 4 bytes of the function signature
				TransactionHash:  transaction.Hash,
				TransactionIndex: transaction.TransactionIndex,
				Type:             transaction.TransactionType,
				Path:             "",
			})
		}

		blocksIndex = append(blocksIndex, indexer.NewBlockIndex("mantle_sepolia",
			block.BlockNumber,
			block.Hash,
			block.Timestamp,
			block.ParentHash,
			uint64(bI),
			"",
			0,
		))

		blocksCache[block.BlockNumber] = indexer.BlockCache{
			BlockNumber:    block.BlockNumber,
			BlockHash:      block.Hash,
			BlockTimestamp: block.Timestamp,
		} // Assuming block.BlockNumber is int64 and block.Hash is string
	}

	return blocksProto, blocksIndex, transactionsIndex, blocksCache, nil
}

func (c *Client) FetchAsProtoEventsAndExtendBlock(from, to *big.Int, blocksProto []proto.Message, blocksCache map[uint64]indexer.BlockCache, debug bool) ([]proto.Message, []indexer.LogIndex, error) {
	parsedEvents, err := c.ParseEvents(from, to, blocksCache, debug)
	if err != nil {
		return nil, nil, err
	}

	// Parse proto Message to block with transactions structure
	var blocks []*MantleSepoliaBlock
	for _, b := range blocksProto {
		block, ok := b.(*MantleSepoliaBlock)
		if !ok {
			log.Printf("Unexpected proto message type: %T", b)
			continue
		}
		blocks = append(blocks, block)
	}

	var eventsIndex []indexer.LogIndex
	for eI, event := range parsedEvents {
		for _, b := range blocks {
			if b.BlockNumber != event.BlockNumber {
				continue
			}
			for _, tx := range b.Transactions {
				if tx.Hash == event.TransactionHash {
					tx.Logs = append(tx.Logs, event)
				}
			}
		}

		var topic0, topic1, topic2 *string

		if len(event.Topics) == 0 {
			fmt.Println("No topics found for event: ", event)
		} else {
			topic0 = &event.Topics[0] // First topic
		}

		// Assign topics based on availability
		if len(event.Topics) > 1 {
			topic1 = &event.Topics[1] // Second topic, if present
		}
		if len(event.Topics) > 2 {
			topic2 = &event.Topics[2] // Third topic, if present
		}

		eventsIndex = append(eventsIndex, indexer.LogIndex{
			Address:         event.Address,
			BlockNumber:     event.BlockNumber,
			BlockHash:       event.BlockHash,
			BlockTimestamp:  blocksCache[event.BlockNumber].BlockTimestamp,
			TransactionHash: event.TransactionHash,
			Selector:        topic0, // First topic
			Topic1:          topic1,
			Topic2:          topic2,
			RowID:           uint64(eI),
			LogIndex:        event.LogIndex,
			Path:            "",
		})
	}

	var updatedBlocksProto []proto.Message
	for _, b := range blocks {
		updatedBlocksProto = append(updatedBlocksProto, b)
	}

	return updatedBlocksProto, eventsIndex, nil
}

func (c *Client) FetchAsProtoEvents(from, to *big.Int, blocksCahche map[uint64]indexer.BlockCache, debug bool) ([]proto.Message, []indexer.LogIndex, error) {
	parsedEvents, err := c.ParseEvents(from, to, blocksCahche, debug)

	if err != nil {
		return nil, nil, err
	}

	var eventsProto []proto.Message
	var eventsIndex []indexer.LogIndex
	for index, event := range parsedEvents {
		eventsProto = append(eventsProto, event) // Assuming event is already a proto.Message

		var topic0, topic1, topic2 *string

		if len(event.Topics) == 0 {
			fmt.Println("No topics found for event: ", event)
		} else {
			topic0 = &event.Topics[0] // First topic
		}

		// Assign topics based on availability
		if len(event.Topics) > 1 {
			topic1 = &event.Topics[1] // Second topic, if present
		}
		if len(event.Topics) > 2 {
			topic2 = &event.Topics[2] // Third topic, if present
		}

		eventsIndex = append(eventsIndex, indexer.LogIndex{
			Address:         event.Address,
			BlockNumber:     event.BlockNumber,
			BlockHash:       event.BlockHash,
			BlockTimestamp:  blocksCahche[event.BlockNumber].BlockTimestamp,
			TransactionHash: event.TransactionHash,
			Selector:        topic0, // First topic
			Topic1:          topic1,
			Topic2:          topic2,
			RowID:           uint64(index),
			LogIndex:        event.LogIndex,
			Path:            "",
		})
	}

	return eventsProto, eventsIndex, nil
}

func ToEntireBlockFromLogProto(obj *MantleSepoliaBlock) *seer_common.BlockJson {
	var txs []seer_common.TransactionJson
	for _, tx := range obj.Transactions {
		var accessList []seer_common.AccessList
		for _, al := range tx.AccessList {
			accessList = append(accessList, seer_common.AccessList{
				Address:     al.Address,
				StorageKeys: al.StorageKeys,
			})
		}
		var events []seer_common.EventJson
		for _, e := range tx.Logs {
			events = append(events, seer_common.EventJson{
				Address:          e.Address,
				Topics:           e.Topics,
				Data:             e.Data,
				BlockNumber:      fmt.Sprintf("%d", e.BlockNumber),
				TransactionHash:  e.TransactionHash,
				BlockHash:        e.BlockHash,
				Removed:          e.Removed,
				LogIndex:         fmt.Sprintf("%d", e.LogIndex),
				TransactionIndex: fmt.Sprintf("%d", e.TransactionIndex),
			})
		}
		txs = append(txs, seer_common.TransactionJson{
			BlockHash:            tx.BlockHash,
			BlockNumber:          fmt.Sprintf("%d", tx.BlockNumber),
			ChainId:              tx.ChainId,
			FromAddress:          tx.FromAddress,
			Gas:                  tx.Gas,
			GasPrice:             tx.GasPrice,
			Hash:                 tx.Hash,
			Input:                tx.Input,
			MaxFeePerGas:         tx.MaxFeePerGas,
			MaxPriorityFeePerGas: tx.MaxPriorityFeePerGas,
			Nonce:                tx.Nonce,
			V:                    tx.V,
			R:                    tx.R,
			S:                    tx.S,
			ToAddress:            tx.ToAddress,
			TransactionIndex:     fmt.Sprintf("%d", tx.TransactionIndex),
			TransactionType:      fmt.Sprintf("%d", tx.TransactionType),
			Value:                tx.Value,
			IndexedAt:            fmt.Sprintf("%d", tx.IndexedAt),
			BlockTimestamp:       fmt.Sprintf("%d", tx.BlockTimestamp),
			AccessList:           accessList,
			YParity:              tx.YParity,

			Events: events,
		})
	}

	return &seer_common.BlockJson{
		Difficulty:       fmt.Sprintf("%d", obj.Difficulty),
		ExtraData:        obj.ExtraData,
		GasLimit:         fmt.Sprintf("%d", obj.GasLimit),
		GasUsed:          fmt.Sprintf("%d", obj.GasUsed),
		Hash:             obj.Hash,
		LogsBloom:        obj.LogsBloom,
		Miner:            obj.Miner,
		Nonce:            obj.Nonce,
		BlockNumber:      fmt.Sprintf("%d", obj.BlockNumber),
		ParentHash:       obj.ParentHash,
		ReceiptsRoot:     obj.ReceiptsRoot,
		Sha3Uncles:       obj.Sha3Uncles,
		StateRoot:        obj.StateRoot,
		Timestamp:        fmt.Sprintf("%d", obj.Timestamp),
		TotalDifficulty:  obj.TotalDifficulty,
		TransactionsRoot: obj.TransactionsRoot,
		Size:             fmt.Sprintf("%d", obj.Size),
		BaseFeePerGas:    obj.BaseFeePerGas,
		IndexedAt:        fmt.Sprintf("%d", obj.IndexedAt),

		Transactions: txs,
	}
}

func ToProtoSingleBlock(obj *seer_common.BlockJson) *MantleSepoliaBlock {
	return &MantleSepoliaBlock{
		BlockNumber:      fromHex(obj.BlockNumber).Uint64(),
		Difficulty:       fromHex(obj.Difficulty).Uint64(),
		ExtraData:        obj.ExtraData,
		GasLimit:         fromHex(obj.GasLimit).Uint64(),
		GasUsed:          fromHex(obj.GasUsed).Uint64(),
		BaseFeePerGas:    obj.BaseFeePerGas,
		Hash:             obj.Hash,
		LogsBloom:        obj.LogsBloom,
		Miner:            obj.Miner,
		Nonce:            obj.Nonce,
		ParentHash:       obj.ParentHash,
		ReceiptsRoot:     obj.ReceiptsRoot,
		Sha3Uncles:       obj.Sha3Uncles,
		Size:             fromHex(obj.Size).Uint64(),
		StateRoot:        obj.StateRoot,
		Timestamp:        fromHex(obj.Timestamp).Uint64(),
		TotalDifficulty:  obj.TotalDifficulty,
		TransactionsRoot: obj.TransactionsRoot,
		IndexedAt:        fromHex(obj.IndexedAt).Uint64(),
	}
}

func ToProtoSingleTransaction(obj *seer_common.TransactionJson) *MantleSepoliaTransaction {
	var accessList []*MantleSepoliaTransactionAccessList
	for _, al := range obj.AccessList {
		accessList = append(accessList, &MantleSepoliaTransactionAccessList{
			Address:     al.Address,
			StorageKeys: al.StorageKeys,
		})
	}

	return &MantleSepoliaTransaction{
		Hash:                 obj.Hash,
		BlockNumber:          fromHex(obj.BlockNumber).Uint64(),
		BlockHash:            obj.BlockHash,
		FromAddress:          obj.FromAddress,
		ToAddress:            obj.ToAddress,
		Gas:                  obj.Gas,
		GasPrice:             obj.GasPrice,
		MaxFeePerGas:         obj.MaxFeePerGas,
		MaxPriorityFeePerGas: obj.MaxPriorityFeePerGas,
		Input:                obj.Input,
		Nonce:                obj.Nonce,
		TransactionIndex:     fromHex(obj.TransactionIndex).Uint64(),
		TransactionType:      fromHex(obj.TransactionType).Uint64(),
		Value:                obj.Value,
		IndexedAt:            fromHex(obj.IndexedAt).Uint64(),
		BlockTimestamp:       fromHex(obj.BlockTimestamp).Uint64(),

		ChainId: obj.ChainId,
		V:       obj.V,
		R:       obj.R,
		S:       obj.S,

		AccessList: accessList,
		YParity:    obj.YParity,
	}
}

func ToEvenFromLogProto(obj *MantleSepoliaEventLog) *seer_common.EventJson {
	return &seer_common.EventJson{
		Address:         obj.Address,
		Topics:          obj.Topics,
		Data:            obj.Data,
		BlockNumber:     fmt.Sprintf("%d", obj.BlockNumber),
		TransactionHash: obj.TransactionHash,
		LogIndex:        fmt.Sprintf("%d", obj.LogIndex),
		BlockHash:       obj.BlockHash,
		Removed:         obj.Removed,
	}
}

func ToProtoSingleEventLog(obj *seer_common.EventJson) *MantleSepoliaEventLog {
	return &MantleSepoliaEventLog{
		Address:         obj.Address,
		Topics:          obj.Topics,
		Data:            obj.Data,
		BlockNumber:     fromHex(obj.BlockNumber).Uint64(),
		TransactionHash: obj.TransactionHash,
		LogIndex:        fromHex(obj.LogIndex).Uint64(),
		BlockHash:       obj.BlockHash,
		Removed:         obj.Removed,
	}
}

func (c *Client) DecodeProtoEventLogs(data []string) ([]*MantleSepoliaEventLog, error) {
	var events []*MantleSepoliaEventLog
	for _, d := range data {
		var event MantleSepoliaEventLog
		base64Decoded, err := base64.StdEncoding.DecodeString(d)
		if err != nil {
			return nil, err
		}
		if err := proto.Unmarshal(base64Decoded, &event); err != nil {
			return nil, err
		}
		events = append(events, &event)
	}
	return events, nil
}

func (c *Client) DecodeProtoTransactions(data []string) ([]*MantleSepoliaTransaction, error) {
	var transactions []*MantleSepoliaTransaction
	for _, d := range data {
		var transaction MantleSepoliaTransaction
		base64Decoded, err := base64.StdEncoding.DecodeString(d)
		if err != nil {
			return nil, err
		}
		if err := proto.Unmarshal(base64Decoded, &transaction); err != nil {
			return nil, err
		}
		transactions = append(transactions, &transaction)
	}
	return transactions, nil
}

func (c *Client) DecodeProtoBlocks(data []string) ([]*MantleSepoliaBlock, error) {
	var blocks []*MantleSepoliaBlock
	for _, d := range data {
		var block MantleSepoliaBlock
		base64Decoded, err := base64.StdEncoding.DecodeString(d)
		if err != nil {
			return nil, err
		}
		if err := proto.Unmarshal(base64Decoded, &block); err != nil {
			return nil, err
		}
		blocks = append(blocks, &block)
	}
	return blocks, nil
}

func (c *Client) DecodeProtoEntireBlock(rawData *bytes.Buffer) ([]*seer_common.BlockJson, error) {
	var blocks []*seer_common.BlockJson

	for {
		protoBlock := &MantleSepoliaBlock{}
		if unmErr := protodelim.UnmarshalFrom(rawData, protoBlock); unmErr != nil {
			if unmErr == io.EOF {
				break // End of the buffer
			}
			return nil, unmErr
		}

		block := ToEntireBlockFromLogProto(protoBlock)
		blocks = append(blocks, block)
	}

	return blocks, nil
}

func (c *Client) DecodeProtoEventsToLabels(rawData *bytes.Buffer, blocksCache map[uint64]uint64, abiMap map[string]map[string]map[string]string) ([]indexer.EventLabel, error) {
	var protoEvents []*MantleSepoliaEventLog
	for {
		protoEvent := &MantleSepoliaEventLog{}
		if unmErr := protodelim.UnmarshalFrom(rawData, protoEvent); unmErr != nil {
			if unmErr == io.EOF {
				break // End of the buffer
			}
			return nil, unmErr
		}

		protoEvents = append(protoEvents, protoEvent)
	}

	var labels []indexer.EventLabel

	for _, event := range protoEvents {

		var topicSelector string

		if len(event.Topics) > 0 {
			topicSelector = event.Topics[0]
		} else {
			// 0x0 is the default topic selector
			topicSelector = "0x0"
		}

		// Get the ABI string
		contractAbi, err := abi.JSON(strings.NewReader(abiMap[event.Address][topicSelector]["abi"]))
		if err != nil {
			fmt.Println("Error initializing contract ABI: ", err)
			return nil, err
		}

		// Decode the event data
		decodedArgs, err := seer_common.DecodeLogArgsToLabelData(&contractAbi, event.Topics, event.Data)

		if err != nil {
			fmt.Println("Error decoding event data: ", err)
			return nil, err
		}

		// Convert decodedArgs map to JSON
		labelDataBytes, err := json.Marshal(decodedArgs)
		if err != nil {
			return nil, err
		}

		// Convert JSON byte slice to string
		labelDataString := string(labelDataBytes)

		// Convert event to label
		eventLabel := indexer.EventLabel{
			Label:           indexer.SeerCrawlerLabel,
			LabelName:       abiMap[event.Address][topicSelector]["abi_name"],
			LabelType:       "event",
			BlockNumber:     event.BlockNumber,
			BlockHash:       event.BlockHash,
			Address:         event.Address,
			TransactionHash: event.TransactionHash,
			LabelData:       labelDataString,
			BlockTimestamp:  blocksCache[event.BlockNumber],
			LogIndex:        event.LogIndex,
		}

		labels = append(labels, eventLabel)

	}

	return labels, nil
}

func (c *Client) DecodeProtoTransactionsToLabels(transactions []string, blocksCache map[uint64]uint64, abiMap map[string]map[string]map[string]string) ([]indexer.TransactionLabel, error) {

	decodedTransactions, err := c.DecodeProtoTransactions(transactions)

	if err != nil {
		return nil, err
	}

	var labels []indexer.TransactionLabel

	for _, transaction := range decodedTransactions {

		selector := transaction.Input[:10]

		contractAbi, err := abi.JSON(strings.NewReader(abiMap[transaction.ToAddress][selector]["abi"]))

		if err != nil {
			return nil, err
		}

		inputData, err := hex.DecodeString(transaction.Input[2:])
		if err != nil {
			fmt.Println("Error decoding input data: ", err)
			return nil, err
		}

		decodedArgs, err := seer_common.DecodeTransactionInputDataToInterface(&contractAbi, inputData)

		if err != nil {
			fmt.Println("Error decoding transaction input data: ", err)
			return nil, err
		}

		labelDataBytes, err := json.Marshal(decodedArgs)
		if err != nil {
			return nil, err
		}

		// Convert JSON byte slice to string
		labelDataString := string(labelDataBytes)

		// Convert transaction to label
		transactionLabel := indexer.TransactionLabel{
			Address:         transaction.ToAddress,
			BlockNumber:     transaction.BlockNumber,
			BlockHash:       transaction.BlockHash,
			CallerAddress:   transaction.FromAddress,
			LabelName:       abiMap[transaction.ToAddress][selector]["abi_name"],
			LabelType:       "tx_call",
			OriginAddress:   transaction.FromAddress,
			Label:           indexer.SeerCrawlerLabel,
			TransactionHash: transaction.Hash,
			LabelData:       labelDataString,
			BlockTimestamp:  blocksCache[transaction.BlockNumber],
		}

		labels = append(labels, transactionLabel)

	}

	return labels, nil
}
