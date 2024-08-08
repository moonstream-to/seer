package game7_testnet

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	"google.golang.org/protobuf/proto"

	seer_common "github.com/moonstream-to/seer/blockchain/common"
	"github.com/moonstream-to/seer/indexer"
	"github.com/moonstream-to/seer/version"
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
	return "game7_testnet"
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

	sem := make(chan struct{}, maxRequests)             // Semaphore to control concurrency
	errChan := make(chan error, len(blockNumbersRange)) // Handle errors to stop corrupted processing

	for _, b := range blockNumbersRange {
		wg.Add(1)
		go func(b *big.Int) {
			defer wg.Done()

			sem <- struct{}{} // Acquire semaphore

			block, getErr := c.GetBlockByNumber(ctx, b)
			if getErr != nil {
				log.Printf("Failed to fetch block number: %d, error: %v", b, getErr)
				errChan <- getErr
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

	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return blocks, nil
}

// ParseBlocksWithTransactions parses blocks and their transactions into custom data structure.
// This method showcases how to handle and transform detailed block and transaction data.
func (c *Client) ParseBlocksWithTransactions(from, to *big.Int, debug bool, maxRequests int) ([]*Game7TestnetBlock, error) {
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

	var parsedBlocks []*Game7TestnetBlock
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

func (c *Client) ParseEvents(from, to *big.Int, blocksCache map[uint64]indexer.BlockCache, debug bool) ([]*Game7TestnetEventLog, []indexer.LogIndex, error) {
	logs, err := c.ClientFilterLogs(context.Background(), ethereum.FilterQuery{
		FromBlock: from,
		ToBlock:   to,
	}, debug)

	if err != nil {
		fmt.Println("Error fetching logs: ", err)
		return nil, nil, err
	}

	var parsedEvents []*Game7TestnetEventLog
	var eventsIndex []indexer.LogIndex

	for i, log := range logs {
		parsedEvent := ToProtoSingleEventLog(log)
		parsedEvents = append(parsedEvents, parsedEvent)

		// Prepare events to index
		var topic0, topic1, topic2, topic3 *string

		if len(parsedEvent.Topics) == 0 {
			// Anonymous events
			fmt.Printf("No topics found for event with tx hash: %s and log index: %d\n", parsedEvent.TransactionHash, parsedEvent.LogIndex)
		} else {
			topic0 = &parsedEvent.Topics[0] // First topic
		}

		// Assign topics based on availability
		if len(parsedEvent.Topics) > 1 {
			topic1 = &parsedEvent.Topics[1] // Second topic, if present
		}
		if len(parsedEvent.Topics) > 2 {
			topic2 = &parsedEvent.Topics[2] // Third topic, if present
		}

		if len(parsedEvent.Topics) > 3 {
			topic3 = &parsedEvent.Topics[3] // Fourth topic, if present
		}

		eventsIndex = append(eventsIndex, indexer.LogIndex{
			Address:         parsedEvent.Address,
			BlockNumber:     parsedEvent.BlockNumber,
			BlockHash:       parsedEvent.BlockHash,
			BlockTimestamp:  blocksCache[parsedEvent.BlockNumber].BlockTimestamp,
			TransactionHash: parsedEvent.TransactionHash,
			Selector:        topic0, // First topic
			Topic1:          topic1,
			Topic2:          topic2,
			Topic3:          topic3,
			RowID:           uint64(i), // TODO: Remove
			LogIndex:        parsedEvent.LogIndex,
			Path:            "",
		})
	}

	return parsedEvents, eventsIndex, nil
}

func (c *Client) FetchAsProtoBlocksWithEvents(from, to *big.Int, debug bool, maxRequests int) ([]proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, []indexer.LogIndex, uint64, error) {
	blocks, err := c.ParseBlocksWithTransactions(from, to, debug, maxRequests)
	if err != nil {
		return nil, nil, nil, nil, 0, err
	}

	var blocksSize uint64

	blocksCache := make(map[uint64]indexer.BlockCache)

	for _, block := range blocks {
		blocksCache[block.BlockNumber] = indexer.BlockCache{
			BlockNumber:    block.BlockNumber,
			BlockHash:      block.Hash,
			BlockTimestamp: block.Timestamp,
		} // Assuming block.BlockNumber is int64 and block.Hash is string
	}

	events, eventsIndex, err := c.ParseEvents(from, to, blocksCache, debug)
	if err != nil {
		return nil, nil, nil, nil, 0, err
	}

	var blocksProto []proto.Message
	var blocksIndex []indexer.BlockIndex
	var txsIndex []indexer.TransactionIndex

	for bI, block := range blocks {
		for txI, tx := range block.Transactions {
			for _, event := range events {
				if tx.Hash == event.TransactionHash {
					tx.Logs = append(tx.Logs, event)
				}
			}

			// Prepare transactions to index
			txSelector := "0x"

			if len(tx.Input) > 10 {
				txSelector = tx.Input[:10]
			}

			txsIndex = append(txsIndex, indexer.TransactionIndex{
				BlockNumber:      tx.BlockNumber,
				BlockHash:        tx.BlockHash,
				BlockTimestamp:   tx.BlockTimestamp,
				FromAddress:      tx.FromAddress,
				ToAddress:        tx.ToAddress,
				RowID:            uint64(txI),
				Selector:         txSelector, // First 10 characters of the input data 0x + 4 bytes of the function signature
				TransactionHash:  tx.Hash,
				TransactionIndex: tx.TransactionIndex,
				Type:             tx.TransactionType,
				Path:             "",
			})
		}

		// Prepare blocks to index
		blocksIndex = append(blocksIndex, indexer.NewBlockIndex("game7_testnet",
			block.BlockNumber,
			block.Hash,
			block.Timestamp,
			block.ParentHash,
			uint64(bI),
			"",
			block.L1BlockNumber,
		))

		blocksSize += uint64(proto.Size(block))
		blocksProto = append(blocksProto, block) // Assuming block is already a proto.Message
	}

	return blocksProto, blocksIndex, txsIndex, eventsIndex, blocksSize, nil
}

func (c *Client) ProcessBlocksToBatch(msgs []proto.Message) (proto.Message, error) {
	var blocks []*Game7TestnetBlock
	for _, msg := range msgs {
		block, ok := msg.(*Game7TestnetBlock)
		if !ok {
			return nil, fmt.Errorf("failed to type assert proto.Message to *Game7TestnetBlock")
		}
		blocks = append(blocks, block)
	}

	return &Game7TestnetBlocksBatch{
		Blocks:      blocks,
		SeerVersion: version.SeerVersion,
	}, nil
}

func ToEntireBlocksBatchFromLogProto(obj *Game7TestnetBlocksBatch) *seer_common.BlocksBatchJson {
	blocksBatchJson := seer_common.BlocksBatchJson{
		Blocks:      []seer_common.BlockJson{},
		SeerVersion: obj.SeerVersion,
	}

	for _, b := range obj.Blocks {
		var txs []seer_common.TransactionJson
		for _, tx := range b.Transactions {
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

		blocksBatchJson.Blocks = append(blocksBatchJson.Blocks, seer_common.BlockJson{
			Difficulty:       fmt.Sprintf("%d", b.Difficulty),
			ExtraData:        b.ExtraData,
			GasLimit:         fmt.Sprintf("%d", b.GasLimit),
			GasUsed:          fmt.Sprintf("%d", b.GasUsed),
			Hash:             b.Hash,
			LogsBloom:        b.LogsBloom,
			Miner:            b.Miner,
			Nonce:            b.Nonce,
			BlockNumber:      fmt.Sprintf("%d", b.BlockNumber),
			ParentHash:       b.ParentHash,
			ReceiptsRoot:     b.ReceiptsRoot,
			Sha3Uncles:       b.Sha3Uncles,
			StateRoot:        b.StateRoot,
			Timestamp:        fmt.Sprintf("%d", b.Timestamp),
			TotalDifficulty:  b.TotalDifficulty,
			TransactionsRoot: b.TransactionsRoot,
			Size:             fmt.Sprintf("%d", b.Size),
			BaseFeePerGas:    b.BaseFeePerGas,
			IndexedAt:        fmt.Sprintf("%d", b.IndexedAt),

			MixHash:       b.MixHash,
			SendCount:     b.SendCount,
			SendRoot:      b.SendRoot,
			L1BlockNumber: fmt.Sprintf("%d", b.L1BlockNumber),

			Transactions: txs,
		})
	}

	return &blocksBatchJson
}

func ToProtoSingleBlock(obj *seer_common.BlockJson) *Game7TestnetBlock {
	return &Game7TestnetBlock{
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

		MixHash:       obj.MixHash,
		SendCount:     obj.SendCount,
		SendRoot:      obj.SendRoot,
		L1BlockNumber: fromHex(obj.L1BlockNumber).Uint64(),
	}
}

func ToProtoSingleTransaction(obj *seer_common.TransactionJson) *Game7TestnetTransaction {
	var accessList []*Game7TestnetTransactionAccessList
	for _, al := range obj.AccessList {
		accessList = append(accessList, &Game7TestnetTransactionAccessList{
			Address:     al.Address,
			StorageKeys: al.StorageKeys,
		})
	}

	return &Game7TestnetTransaction{
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

func ToEvenFromLogProto(obj *Game7TestnetEventLog) *seer_common.EventJson {
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

func ToProtoSingleEventLog(obj *seer_common.EventJson) *Game7TestnetEventLog {
	return &Game7TestnetEventLog{
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

func (c *Client) DecodeProtoEventLogs(data []string) ([]*Game7TestnetEventLog, error) {
	var events []*Game7TestnetEventLog
	for _, d := range data {
		var event Game7TestnetEventLog
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

func (c *Client) DecodeProtoTransactions(data []string) ([]*Game7TestnetTransaction, error) {
	var transactions []*Game7TestnetTransaction
	for _, d := range data {
		var transaction Game7TestnetTransaction
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

func (c *Client) DecodeProtoBlocks(data []string) ([]*Game7TestnetBlock, error) {
	var blocks []*Game7TestnetBlock
	for _, d := range data {
		var block Game7TestnetBlock
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

func (c *Client) DecodeProtoEntireBlockToJson(rawData *bytes.Buffer) (*seer_common.BlocksBatchJson, error) {
	var protoBlocksBatch Game7TestnetBlocksBatch

	dataBytes := rawData.Bytes()

	err := proto.Unmarshal(dataBytes, &protoBlocksBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %v", err)
	}

	blocksBatchJson := ToEntireBlocksBatchFromLogProto(&protoBlocksBatch)

	return blocksBatchJson, nil
}

func (c *Client) DecodeProtoEntireBlockToLabels(rawData *bytes.Buffer, abiMap map[string]map[string]map[string]string) ([]indexer.EventLabel, []indexer.TransactionLabel, error) {
	var protoBlocksBatch Game7TestnetBlocksBatch

	dataBytes := rawData.Bytes()

	err := proto.Unmarshal(dataBytes, &protoBlocksBatch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal data: %v", err)
	}

	var labels []indexer.EventLabel
	var txLabels []indexer.TransactionLabel
	var decodeErr error

	for _, b := range protoBlocksBatch.Blocks {
		for _, tx := range b.Transactions {
			var decodedArgsTx map[string]interface{}

			label := indexer.SeerCrawlerLabel

			if len(tx.Input) < 10 { // If input is less than 3 characters then it direct transfer
				continue
			}

			// Process transaction labels
			selector := tx.Input[:10]

			if abiMap[tx.ToAddress] != nil && abiMap[tx.ToAddress][selector] != nil {
				txContractAbi, err := abi.JSON(strings.NewReader(abiMap[tx.ToAddress][selector]["abi"]))
				if err != nil {
					fmt.Println("Error initializing contract ABI transactions: ", err)
					return nil, nil, err
				}

				inputData, err := hex.DecodeString(tx.Input[2:])
				if err != nil {
					fmt.Println("Error decoding input data: ", err)
					return nil, nil, err
				}

				decodedArgsTx, decodeErr = seer_common.DecodeTransactionInputDataToInterface(&txContractAbi, inputData)
				if decodeErr != nil {
					fmt.Println("Error decoding transaction not decoded data: ", tx.Hash, decodeErr)
					decodedArgsTx = map[string]interface{}{
						"input_raw": tx,
						"abi":       abiMap[tx.ToAddress][selector]["abi"],
						"selector":  selector,
						"error":     decodeErr,
					}
					label = indexer.SeerCrawlerRawLabel
				}

				txLabelDataBytes, err := json.Marshal(decodedArgsTx)
				if err != nil {
					fmt.Println("Error converting decodedArgsTx to JSON: ", err)
					return nil, nil, err
				}

				// Convert transaction to label
				transactionLabel := indexer.TransactionLabel{
					Address:         tx.ToAddress,
					BlockNumber:     tx.BlockNumber,
					BlockHash:       tx.BlockHash,
					CallerAddress:   tx.FromAddress,
					LabelName:       abiMap[tx.ToAddress][selector]["abi_name"],
					LabelType:       "tx_call",
					OriginAddress:   tx.FromAddress,
					Label:           label,
					TransactionHash: tx.Hash,
					LabelData:       string(txLabelDataBytes), // Convert JSON byte slice to string
					BlockTimestamp:  b.Timestamp,
				}

				txLabels = append(txLabels, transactionLabel)
			}

			// Process events
			for _, e := range tx.Logs {
				var decodedArgsLogs map[string]interface{}
				label = indexer.SeerCrawlerLabel

				var topicSelector string

				if len(e.Topics) > 0 {
					topicSelector = e.Topics[0]
				} else {
					// 0x0 is the default topic selector
					topicSelector = "0x0"
				}

				if abiMap[e.Address] == nil || abiMap[e.Address][topicSelector] == nil {
					continue
				}

				// Get the ABI string
				contractAbi, err := abi.JSON(strings.NewReader(abiMap[e.Address][topicSelector]["abi"]))
				if err != nil {
					fmt.Println("Error initializing contract ABI: ", err)
					return nil, nil, err
				}

				// Decode the event data
				decodedArgsLogs, decodeErr = seer_common.DecodeLogArgsToLabelData(&contractAbi, e.Topics, e.Data)
				if decodeErr != nil {
					fmt.Println("Error decoding event not decoded data: ", e.TransactionHash, decodeErr)
					decodedArgsLogs = map[string]interface{}{
						"input_raw": e,
						"abi":       abiMap[e.Address][topicSelector]["abi"],
						"selector":  topicSelector,
						"error":     decodeErr,
					}
					label = indexer.SeerCrawlerRawLabel
				}

				// Convert decodedArgsLogs map to JSON
				labelDataBytes, err := json.Marshal(decodedArgsLogs)
				if err != nil {
					fmt.Println("Error converting decodedArgsLogs to JSON: ", err)
					return nil, nil, err
				}

				// Convert event to label
				eventLabel := indexer.EventLabel{
					Label:           label,
					LabelName:       abiMap[e.Address][topicSelector]["abi_name"],
					LabelType:       "event",
					BlockNumber:     e.BlockNumber,
					BlockHash:       e.BlockHash,
					Address:         e.Address,
					OriginAddress:   tx.FromAddress,
					TransactionHash: e.TransactionHash,
					LabelData:       string(labelDataBytes), // Convert JSON byte slice to string
					BlockTimestamp:  b.Timestamp,
					LogIndex:        e.LogIndex,
				}

				labels = append(labels, eventLabel)
			}
		}
	}

	return labels, txLabels, nil
}

func (c *Client) DecodeProtoTransactionsToLabels(transactions []string, blocksCache map[uint64]uint64, abiMap map[string]map[string]map[string]string) ([]indexer.TransactionLabel, error) {

	decodedTransactions, err := c.DecodeProtoTransactions(transactions)

	if err != nil {
		return nil, err
	}

	var labels []indexer.TransactionLabel
	var decodedArgs map[string]interface{}
	var decodeErr error

	for _, transaction := range decodedTransactions {

		label := indexer.SeerCrawlerLabel

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

		decodedArgs, decodeErr = seer_common.DecodeTransactionInputDataToInterface(&contractAbi, inputData)

		if decodeErr != nil {
			fmt.Println("Error decoding transaction not decoded data: ", transaction.Hash, decodeErr)
			decodedArgs = map[string]interface{}{
				"input_raw": transaction,
				"abi":       abiMap[transaction.ToAddress][selector]["abi"],
				"selector":  selector,
				"error":     decodeErr,
			}
			label = indexer.SeerCrawlerRawLabel
		}

		labelDataBytes, err := json.Marshal(decodedArgs)
		if err != nil {
			fmt.Println("Error converting decodedArgs to JSON: ", err)
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
			Label:           label,
			TransactionHash: transaction.Hash,
			LabelData:       labelDataString,
			BlockTimestamp:  blocksCache[transaction.BlockNumber],
		}

		labels = append(labels, transactionLabel)

	}

	return labels, nil
}
