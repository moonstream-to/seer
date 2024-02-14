package ethereum

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/moonstream-to/seer/indexer"
	"google.golang.org/protobuf/proto"
)

func NewClient(url string) (*Client, error) {
	rpcClient, err := rpc.DialContext(context.Background(), url)
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
	return "ethereum"
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
func (c *Client) GetBlockByNumber(ctx context.Context, number *big.Int) (*BlockJson, error) {

	var rawResponse json.RawMessage // Use RawMessage to capture the entire JSON response
	err := c.rpcClient.CallContext(ctx, &rawResponse, "eth_getBlockByNumber", "0x"+number.Text(16), true)
	if err != nil {
		fmt.Println("Error calling eth_getBlockByNumber: ", err)
		return nil, err
	}

	var response_json map[string]interface{}

	err = json.Unmarshal(rawResponse, &response_json)

	delete(response_json, "transactions")

	fmt.Println("Response: ", response_json)

	// // Optionally log the raw JSON response for debugging
	// fmt.Println("Raw JSON response: ", string(rawResponse))

	// chain_block, err := c.rpcClient.BlockByNumber(context.Background(), blockNumber)
	// if err != nil {
	// 	log.Printf("Failed to retrieve block %v: %v", blockNumber, err)
	// 	continue
	// }

	// blockData := map[string]interface{}{
	// 	"number":           header.Number.String(),
	// 	"hash":             header.Hash().Hex(),
	// 	"parentHash":       header.ParentHash.Hex(),
	// 	"nonce":            hexutil.EncodeUint64(header.Nonce.Uint64()),
	// 	"sha3Uncles":       header.UncleHash.Hex(),
	// 	"logsBloom":        header.Bloom.Bytes(),
	// 	"transactionsRoot": header.TxHash.Hex(),
	// 	"stateRoot":        header.Root.Hex(),
	// 	"miner":            header.Coinbase.Hex(),
	// 	"difficulty":       header.Difficulty.String(),
	// 	"extraData":        hexutil.Encode(header.Extra[:]),
	// 	"gasLimit":         header.GasLimit,
	// 	"gasUsed":          header.GasUsed,
	// 	"timestamp":        header.Time,
	// 	"mixHash":          header.MixDigest.Hex(),
	// 	"transactions":     body.Transactions,
	// }

	var block *BlockJson
	err = c.rpcClient.CallContext(ctx, &block, "eth_getBlockByNumber", "0x"+number.Text(16), true) // true to include transactions
	return block, err
}

// BlockByHash returns the block with the given hash.

func (c *Client) BlockByHash(ctx context.Context, hash common.Hash) (*BlockJson, error) {
	var block *BlockJson
	err := c.rpcClient.CallContext(ctx, &block, "eth_getBlockByHash", hash, true) // true to include transactions
	return block, err
}

// TransactionReceipt returns the receipt of a transaction by transaction hash.

func (c *Client) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	var receipt *types.Receipt
	err := c.rpcClient.CallContext(ctx, &receipt, "eth_getTransactionReceipt", hash)
	return receipt, err
}

// FilterLogs returns the logs that satisfy the specified filter query.

func (c *Client) FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	var logs []types.Log
	err := c.rpcClient.CallContext(ctx, &logs, "eth_getLogs", q)
	return logs, err
}

// fetchBlocks returns the blocks for a given range.

func (c *Client) fetchBlocks(ctx context.Context, from, to *big.Int) ([]*BlockJson, error) {
	var blocks []*BlockJson

	for i := from; i.Cmp(to) <= 0; i.Add(i, big.NewInt(1)) {
		block, err := c.GetBlockByNumber(ctx, i)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
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
func (c *Client) FetchBlocksInRange(from, to *big.Int) ([]*BlockJson, error) {
	var blocks []*BlockJson
	ctx := context.Background() // For simplicity, using a background context; consider timeouts for production.

	for i := new(big.Int).Set(from); i.Cmp(to) <= 0; i.Add(i, big.NewInt(1)) {
		block, err := c.GetBlockByNumber(ctx, i)
		fmt.Println("Block number: ", i)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// ParseBlocksAndTransactions parses blocks and their transactions into custom data structures.
// This method showcases how to handle and transform detailed block and transaction data.
func (c *Client) ParseBlocksAndTransactions(from, to *big.Int) ([]*Block, []*SingleTransaction, error) {
	blocksJson, err := c.FetchBlocksInRange(from, to)
	if err != nil {
		return nil, nil, err
	}

	var parsedBlocks []*Block
	var parsedTransactions []*SingleTransaction
	for _, blockJson := range blocksJson {
		// Convert BlockJson to Block and Transactions as required.
		parsedBlock := ToProtoSingleBlock(blockJson)

		// Example: Parsing transactions within the block
		for _, txJson := range blockJson.Transactions {

			parsedTransaction := ToProtoSingleTransaction(txJson)
			parsedTransactions = append(parsedTransactions, parsedTransaction)
		}

		parsedBlocks = append(parsedBlocks, parsedBlock)
	}

	return parsedBlocks, parsedTransactions, nil
}

func (c *Client) FetchAsProtoBlocks(from, to *big.Int) ([]proto.Message, []proto.Message, []indexer.BlockIndex, []indexer.TransactionIndex, error) {
	parsedBlocks, parsedTransactions, err := c.ParseBlocksAndTransactions(from, to)

	if err != nil {
		return nil, nil, nil, nil, err
	}

	blockHashes := make(map[int64]string)

	var blocksProto []proto.Message
	var blockIndex []indexer.BlockIndex
	for _, block := range parsedBlocks {
		blocksProto = append(blocksProto, block)    // Assuming block is already a proto.Message
		blockHashes[block.BlockNumber] = block.Hash // Assuming block.BlockNumber is int64 and block.Hash is string
		blockIndex = append(blockIndex, indexer.BlockIndex{
			BlockNumber:    block.BlockNumber,
			BlockHash:      block.Hash,
			BlockTimestamp: block.Timestamp,
			ParentHash:     block.ParentHash,
			Filepath:       "",
		})
	}

	var transactionsProto []proto.Message
	var transactionIndex []indexer.TransactionIndex
	for _, transaction := range parsedTransactions {
		transactionsProto = append(transactionsProto, transaction) // Assuming transaction is also a proto.Message
		transactionIndex = append(transactionIndex, indexer.TransactionIndex{
			BlockNumber:          transaction.BlockNumber,
			BlockHash:            blockHashes[transaction.BlockNumber],
			BlockTimestamp:       transaction.BlockTimestamp,
			TransactionHash:      transaction.Hash,
			TransactionIndex:     transaction.TransactionIndex,
			TransactionTimestamp: transaction.BlockTimestamp,
			Filepath:             "",
		})
	}

	return blocksProto, transactionsProto, blockIndex, transactionIndex, nil
}

func ToProtoSingleBlock(obj *BlockJson) *Block {
	return &Block{
		BlockNumber:   fromHex(obj.BlockNumber).Int64(),
		Difficulty:    fromHex(obj.Difficulty).Int64(),
		ExtraData:     obj.ExtraData,
		GasLimit:      fromHex(obj.GasLimit).Int64(),
		GasUsed:       fromHex(obj.GasUsed).Int64(),
		BaseFeePerGas: obj.BaseFeePerGas,
		Hash:          obj.Hash,
		LogsBloom:     obj.LogsBloom,
		Miner:         obj.Miner,
		Nonce:         obj.Nonce,
		ParentHash:    obj.ParentHash,
		ReceiptRoot:   obj.ReceiptRoot,
		Uncles:        strings.Join(obj.Uncles, ","),
		// convert hex to int32
		Size:             int32(fromHex(obj.Size).Int64()),
		StateRoot:        obj.StateRoot,
		Timestamp:        fromHex(obj.Timestamp).Int64(),
		TotalDifficulty:  obj.TotalDifficulty,
		TransactionsRoot: obj.TransactionsRoot,
		IndexedAt:        obj.IndexedAt,
	}
}

func ToProtoSingleTransaction(obj *SingleTransactionJson) *SingleTransaction {
	return &SingleTransaction{
		Hash:                 obj.Hash,
		BlockNumber:          fromHex(obj.BlockNumber).Int64(),
		FromAddress:          obj.FromAddress,
		ToAddress:            obj.ToAddress,
		Gas:                  obj.Gas,
		GasPrice:             obj.GasPrice,
		MaxFeePerGas:         obj.MaxFeePerGas,
		MaxPriorityFeePerGas: obj.MaxPriorityFeePerGas,
		Input:                obj.Input,
		Nonce:                obj.Nonce,
		TransactionIndex:     fromHex(obj.TransactionIndex).Int64(),
		TransactionType:      int32(fromHex(obj.TransactionType).Int64()),
		Value:                obj.Value,
		IndexedAt:            fromHex(obj.IndexedAt).Int64(),
		BlockTimestamp:       fromHex(obj.BlockTimestamp).Int64(),
	}
}

func ToProtoSingleEventLog(obj *SingleEventJson) *EventLog {
	return &EventLog{
		Address:         obj.Address,
		Topics:          obj.Topics,
		Data:            obj.Data,
		BlockNumber:     obj.BlockNumber,
		TransactionHash: obj.TransactionHash,
		BlockHash:       obj.BlockHash,
		Removed:         obj.Removed,
	}
}
