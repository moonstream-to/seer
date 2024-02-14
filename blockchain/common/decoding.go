package common

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	// "time"
)

type BlockJson struct {
	Difficulty       int64  `json:"difficulty"`
	ExtraData        string `json:"extraData"`
	GasLimit         int64  `json:"gasLimit"`
	GasUsed          int64  `json:"gasUsed"`
	Hash             string `json:"hash"`
	LogsBloom        string `json:"logsBloom"`
	Miner            string `json:"miner"`
	MixHash          string `json:"mixHash"`
	Nonce            string `json:"nonce"`
	BlockNumber      int64  `json:"number"`
	ParentHash       string `json:"parentHash"`
	ReceiptRoot      string `json:"receiptRoot"`
	Sha3Uncles       string `json:"sha3Uncles"`
	StateRoot        string `json:"stateRoot"`
	Timestamp        int64  `json:"timestamp"`
	TotalDifficulty  string `json:"totalDifficulty"`
	TransactionsRoot string `json:"transactionsRoot"`
	Uncles           string `json:"uncles"`
	Size             int32  `json:"size"`
	BaseFeePerGas    string `json:"baseFeePerGas"`
	IndexedAt        int64  `json:"indexed_at"`
}

// SingleTransaction represents a single transaction within a block
/*

   {
       "accessList": [],
       "block_timestamp": 1705588511,
       "chainId": "0x1",
       "gas": "0x557bd",
       "gasPrice": null,
       "hash": "0x7e188362895c2180dc2e74f07888c1f1bbb4d74e9dd7d975b0f890cae99cfc9c",
       "input": "0x0162e2d000000000000000000000000000000000000000000000000000000000000000e000000000000000000000000000000000000000000000000000000000000001600000000000000000000000007a250d5630b4cf539739df2c5dacb4c659f2488d000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000000000000000000000000065a9371f00000000000000000000000000000000000000000000000000000000000000050000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000002386f26fc10000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc20000000000000000000000000ffeb22f687dbb4a3029de88cd19b13ebf3110ea",
       "maxFeePerGas": "0x13643c3268",
       "maxPriorityFeePerGas": "0xba43b7400",
       "nonce": "0x1aea",
       "r": "0x36923e7093bf9b72732868cf3d7f0584fae32c4b227ca0130be69f8b60eb0afa",
       "s": "0x7b979ab4d970d0babd9ba056259ffbf7b77523adeb52fd8372895896c1ee7a5b",
       "to": "0x3328f7f4a1d1c57c35df56bbf0c9dcafca309c49",
       "type": "0x2",
       "v": "0x1",
       "value": "0x3782dace9d90000",
       "yParity": "0x1"
   },
   {
       "accessList": [
           {
               "address": "0x9a6e85fbfd570ba0622caa7352060ab4fa47b42e",
               "storageKeys": [
                   "0x0000000000000000000000000000000000000000000000000000000000000008",
                   "0x000000000000000000000000000000000000000000000000000000000000000c",
                   "0x0000000000000000000000000000000000000000000000000000000000000006",
                   "0x0000000000000000000000000000000000000000000000000000000000000007",
                   "0x0000000000000000000000000000000000000000000000000000000000000009",
                   "0x000000000000000000000000000000000000000000000000000000000000000a"
               ]
           },
           {
               "address": "0x04c86baf148f9b1f3aee74fd8789151b4866dd83",
               "storageKeys": [
                   "0x505620c07e2942013614e9292259f65fa4de0fb7a09348ce8a8996c37a9b3a32",
                   "0x71e61966b67353d26a70364e1ae3a749613afeee98589296c12b88a20550a73d",
                   "0x000000000000000000000000000000000000000000000000000000000000000d",
                   "0x000000000000000000000000000000000000000000000000000000000000000f",
                   "0x000000000000000000000000000000000000000000000000000000000000000e",
                   "0x1b1f47e1b72202f6429d6ac72c5e9ecc6e7e541da2badeb18c23fda97faa5be9",
                   "0x0000000000000000000000000000000000000000000000000000000000000000",
                   "0x0000000000000000000000000000000000000000000000000000000000000008",
                   "0x1157eefad419eff4baef1b3125144d57090f58ee26070c5a61059de5a3ac5861",
                   "0x26d7cd113458189cbe21e389a1b88251b2af5598bfe7180f8952a88076604aba",
                   "0x110247826dcafe38ec5cdb9011db979a7d285b8f75dd7b3710b3da9994e12692",
                   "0x000000000000000000000000000000000000000000000000000000000000000a",
                   "0x0000000000000000000000000000000000000000000000000000000000000013",
                   "0x0000000000000000000000000000000000000000000000000000000000000012"
               ]
           },
           {
               "address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
               "storageKeys": [
                   "0x1157eefad419eff4baef1b3125144d57090f58ee26070c5a61059de5a3ac5861",
                   "0x80693802e5c6b000c35c44c3593461c122fa818665288e650ea38c2ba69c00f7"
               ]
           }
       ],
       "block_timestamp": 1705588511,
       "chainId": "0x1",
       "gas": "0x292d4",
       "gasPrice": null,
       "hash": "0x9704a51718b047d0507e66f3adb20f300fb4e3289f5a43b914e3746064859665",
       "input": "0x2f089a6e85fbfd570ba0622caa7352060ab4fa47b42e6008863499",
       "maxFeePerGas": "0x7c000be68",
       "maxPriorityFeePerGas": "0x0",
       "nonce": "0x1bc2",
       "r": "0x573db9bf0cc923a2986cb4961c3977201faf1a42230c85257c7ac6fa6c9aa3b0",
       "s": "0x276c17dedecdd2e7b0be61a5b6870331364e835366411b6eea89762cde1692f2",
       "to": "0x0000001d0000f38cc10d0028474a9c180058b091",
       "type": "0x2",
       "v": "0x1",
       "value": "0x17d7543f",
       "yParity": "0x1"
   },


*/
type SingleTransactionJson struct {
	AccessList           []AccessList `json:"accessList"`
	BlockHash            string       `json:"blockHash"`
	BlockNumber          int64        `json:"blockNumber"`
	ChainId              string       `json:"chainId"`
	FromAddress          string       `json:"from"`
	Gas                  string       `json:"gas"`
	GasPrice             string       `json:"gasPrice"`
	Hash                 string       `json:"hash"`
	Input                string       `json:"input"`
	MaxFeePerGas         string       `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string       `json:"maxPriorityFeePerGas"`
	Nonce                string       `json:"nonce"`
	R                    string       `json:"r"`
	S                    string       `json:"s"`
	ToAddress            string       `json:"to"`
	TransactionIndex     int64        `json:"transactionIndex"`
	TransactionType      int32        `json:"type"`
	Value                string       `json:"value"`
	YParity              string       `json:"yParity"`
	IndexedAt            int64        `json:"indexed_at"`
	BlockTimestamp       int64        `json:"block_timestamp"`
}

type AccessList struct {
	Address     string   `json:"address"`
	StorageKeys []string `json:"storageKeys"`
}

// SingleEvent represents a single event within a transaction
type SingleEventJson struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      int64    `json:"blockNumber"`
	TransactionHash  string   `json:"transactionHash"`
	BlockHash        string   `json:"blockHash"`
	Removed          bool     `json:"removed"`
	LogIndex         int64    `json:"logIndex"`
	TransactionIndex int64    `json:"transactionIndex"`
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
		fmt.Println("error:", err)
	}

	return blocks
}

// ReadJsonTransactions reads transactions from a JSON file
func ReadJsonTransactions() []*SingleTransactionJson {

	/*
			transform json to SingleTransactionJson
			json:
			[
		    {
		        "accessList": [],
		        "blockHash": "0xd6800c14106ee5ddd95597ae6b88da9fc0f5ef2c279f83858e9d3da44f10ba0b",
		        "blockNumber": "0x1227108",
		        "chainId": "0x1",
		        "from": "0x9696a5c3eb572de180aa7f76e39c0f4418a34af1",
		        "gas": "0x7a120",
		        "gasPrice": "0x7c000be68",
		        "hash": "0x7275d9513f6f4778d4cd32504c5e94baada9fd3898eb2a08433f7af514dc818e",
		        "input": "0x7ff36ab5000000000000000000000000000000000000000000007cccdd17771247d980000000000000000000000000000000000000000000000000000000000000000080000000000000000000000000479cd4c24c567cd18520f0d087acfa5f89fe6c890000000000000000000000000000000000000000000000000000000065a937900000000000000000000000000000000000000000000000000000000000000002000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2000000000000000000000000eb986da994e4a118d5956b02d8b7c3c7ce373674",
		        "maxFeePerGas": "0x9ef2b8ef2",
		        "maxPriorityFeePerGas": "0x0",
		        "nonce": "0x121a",
		        "r": "0x516ede02b7a1ae038e5e51e3beb37628ddbb5a713717e447e037086ee06cfca4",
		        "s": "0x4e15c0f8b9f52bc9c4de717b5ff9e1ca055ea4cfd1861ce90985ade145b030a8",
		        "to": "0x7a250d5630b4cf539739df2c5dacb4c659f2488d",
		        "transactionIndex": "0x0",
		        "type": "0x2",
		        "v": "0x1",
		        "value": "0x4f7df1a6d970000",
		        "yParity": "0x1",
		        "block_timestamp": "0x65a9371f"
		    },
		    {

	*/
	file, err := os.Open("data/transactions_go.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var transactions []*SingleTransactionJson
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&transactions)
	if err != nil {
		fmt.Println("error:", err)
	}

	return transactions
}

// ReadJsonEventLogs reads event logs from a JSON file
func ReadJsonEventLogs() []*SingleEventJson {

	/*
		{"address":"0x5132a183e9f3cb7c848b0aac5ae0c4f0491b7ab2","topics":["0x303446e6a8cb73c83dff421c0b1d5e5ce0719dab1bff13660fc254e58cc17fce","0x00000000000000000000000000000000000000000000000000000000001b353c"],"data":"0x","blockNumber":"0x122716b","transactionHash":"0xa127b9f16cc71ffbbafc77370b3ff5c99c78bb08d0a63c9cd4a817bdd17fe682","transactionIndex":"0xc2","blockHash":"0xdff413d7251ce015bb021fff544bc732c576decf6eef8c07739a05d5111492ee","logIndex":"0x155","removed":false}
	*/

	file, err := os.Open("data/event_logs_go.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var eventLogs []*SingleEventJson

	decoder := json.NewDecoder(file)

	fmt.Println("decoder:", decoder)

	err = decoder.Decode(&eventLogs)
	if err != nil {
		fmt.Println("error:", err)
	}

	fmt.Println("eventLogs:", eventLogs[0])

	return eventLogs
}
