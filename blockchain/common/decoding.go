package common

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type BlockJson struct {
	Difficulty       string `json:"difficulty"`
	ExtraData        string `json:"extraData"`
	GasLimit         string `json:"gasLimit"`
	GasUsed          string `json:"gasUsed"`
	Hash             string `json:"hash"`
	LogsBloom        string `json:"logsBloom"`
	Miner            string `json:"miner"`
	Nonce            string `json:"nonce"`
	BlockNumber      string `json:"number"`
	ParentHash       string `json:"parentHash"`
	ReceiptsRoot     string `json:"receiptsRoot"`
	Sha3Uncles       string `json:"sha3Uncles"`
	StateRoot        string `json:"stateRoot"`
	Timestamp        string `json:"timestamp"`
	TotalDifficulty  string `json:"totalDifficulty"`
	TransactionsRoot string `json:"transactionsRoot"`
	Size             string `json:"size"`
	BaseFeePerGas    string `json:"baseFeePerGas"`
	IndexedAt        string `json:"indexed_at"`

	MixHash       string `json:"mixHash,omitempty"`
	SendCount     string `json:"sendCount,omitempty"`
	SendRoot      string `json:"sendRoot,omitempty"`
	L1BlockNumber string `json:"l1BlockNumber,omitempty"`

	Transactions []TransactionJson `json:"transactions,omitempty"`
}

type TransactionJson struct {
	BlockHash            string `json:"blockHash"`
	BlockNumber          string `json:"blockNumber"`
	ChainId              string `json:"chainId"`
	FromAddress          string `json:"from"`
	Gas                  string `json:"gas"`
	GasPrice             string `json:"gasPrice"`
	Hash                 string `json:"hash"`
	Input                string `json:"input"`
	MaxFeePerGas         string `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
	Nonce                string `json:"nonce"`
	V                    string `json:"v"`
	R                    string `json:"r"`
	S                    string `json:"s"`
	ToAddress            string `json:"to"`
	TransactionIndex     string `json:"transactionIndex"`
	TransactionType      string `json:"type"`
	Value                string `json:"value"`
	IndexedAt            string `json:"indexed_at"`
	BlockTimestamp       string `json:"block_timestamp"`

	AccessList []AccessList `json:"accessList,omitempty"`
	YParity    string       `json:"yParity,omitempty"`

	Events []EventJson `json:"events,omitempty"`
}

type AccessList struct {
	Address     string   `json:"address"`
	StorageKeys []string `json:"storageKeys"`
}

// SingleEvent represents a single event within a transaction
type EventJson struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      string   `json:"blockNumber"`
	TransactionHash  string   `json:"transactionHash"`
	BlockHash        string   `json:"blockHash"`
	Removed          bool     `json:"removed"`
	LogIndex         string   `json:"logIndex"`
	TransactionIndex string   `json:"transactionIndex"`
}

type QueryFilter struct {
	BlockHash string     `json:"blockHash"`
	FromBlock string     `json:"fromBlock"`
	ToBlock   string     `json:"toBlock"`
	Address   []string   `json:"address"`
	Topics    [][]string `json:"topics"`
}

// ReadJsonBlocks reads blocks from a JSON file
func ReadJsonBlocks() []*BlockJson {
	file, err := os.Open("data/blocks_go.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var blocks []*BlockJson
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&blocks)
	if err != nil {
		log.Printf("error: %v", err)
	}

	return blocks
}

// ReadJsonTransactions reads transactions from a JSON file
func ReadJsonTransactions() []*TransactionJson {

	file, err := os.Open("data/transactions_go.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var transactions []*TransactionJson
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&transactions)
	if err != nil {
		log.Printf("error: %v", err)
	}

	return transactions
}

// ReadJsonEventLogs reads event logs from a JSON file
func ReadJsonEventLogs() []*EventJson {

	file, err := os.Open("data/event_logs_go.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var eventLogs []*EventJson

	decoder := json.NewDecoder(file)

	log.Printf("decoder: %v", decoder)

	err = decoder.Decode(&eventLogs)
	if err != nil {
		log.Printf("error: %v", err)
	}

	log.Printf("eventLogs: %v", eventLogs[0])

	return eventLogs
}

func DecodeTransactionInputDataToInterface(contractABI *abi.ABI, data []byte) (map[string]interface{}, error) {
	methodSigData := data[:4]
	inputsSigData := data[4:]
	method, err := contractABI.MethodById(methodSigData)
	if err != nil {
		log.Fatal(err)
	}
	inputsMap := make(map[string]interface{})
	if err := method.Inputs.UnpackIntoMap(inputsMap, inputsSigData); err != nil {
		return nil, err
	}

	// Prepare the extended map
	labelData := make(map[string]interface{})
	labelData["type"] = "tx_call"
	labelData["gas_used"] = 0
	labelData["args"] = inputsMap

	// check if labeData is valid json
	_, err = json.Marshal(labelData)
	if err != nil {
		fmt.Println("Error marshalling labelData: ", labelData)
		return nil, err
	}

	return labelData, nil
}

func DecodeLogArgsToLabelData(contractABI *abi.ABI, topics []string, data string) (map[string]interface{}, error) {

	topic0 := topics[0]

	// Convert the topic0 string to common.Hash
	topic0Hash := common.HexToHash(topic0)

	event, err := contractABI.EventByID(
		topic0Hash,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Decode the data string from hex to bytes
	dataBytes, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		log.Fatalf("Failed to decode data string: %v", err)
	}

	// Prepare the map to hold the input data
	labelData := make(map[string]interface{})
	labelData["type"] = "event"
	labelData["name"] = event.Name
	labelData["args"] = make(map[string]interface{})

	i := 1
	// Extract indexed parameters from topics
	for _, input := range event.Inputs {
		var arg interface{}
		if input.Indexed {
			// Note: topic[0] is the event signature, so indexed params start from topic[1]
			switch input.Type.T {
			case abi.AddressTy:
				arg = common.HexToAddress(topics[i]).Hex()
			case abi.BytesTy:
				arg = common.HexToHash(topics[i]).Hex()
			case abi.FixedBytesTy:
				if input.Type.Size == 32 {
					arg = common.HexToHash(topics[i]).Hex()
				} else {
					arg = common.BytesToHash(common.Hex2Bytes(topics[i][2:])).Hex() // for other fixed sizes
				}
			case abi.UintTy:
				arg = new(big.Int).SetBytes(common.Hex2Bytes(topics[i][2:]))
			case abi.BoolTy:
				arg = new(big.Int).SetBytes(common.Hex2Bytes(topics[i][2:])).Cmp(big.NewInt(0)) != 0
			case abi.StringTy:
				argBytes, err := hex.DecodeString(strings.TrimPrefix(topics[i], "0x"))
				if err != nil {
					return nil, fmt.Errorf("failed to decode hex string to normal string: %v", err)
				}
				arg = string(argBytes)
			default:
				log.Fatalf("Unsupported indexed type: %s", input.Type.String())
			}
			i++
		} else {
			arg = "NON-INDEXED" // Placeholder for non-indexed arguments, which will be unpacked later
		}
		labelData["args"].(map[string]interface{})[input.Name] = arg
	}

	// Unpack the data bytes into the args map
	if err := event.Inputs.UnpackIntoMap(labelData["args"].(map[string]interface{}), dataBytes); err != nil {
		return nil, err
	}

	return labelData, nil
}
