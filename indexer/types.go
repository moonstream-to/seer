package indexer

import "time"

// gorm is a Go ORM library for working with databases

// Define the interface for handling general index types such as blocks, transactions, and logs for fast access to blockchain data.

type BlockIndex struct {
	chain          string
	BlockNumber    uint64
	BlockHash      string
	BlockTimestamp uint64
	ParentHash     string
	RowID          uint64
	Path           string
	L1BlockNumber  uint64
}

func (b *BlockIndex) SetChain(chain string) {
	b.chain = chain
}

// NewBlockIndex creates a new instance of BlockIndex with the chain set.
func NewBlockIndex(chain string, blockNumber uint64, blockHash string, blockTimestamp uint64, parentHash string, row_id uint64, path string, l1BlockNumber uint64) BlockIndex {
	return BlockIndex{
		chain:          chain,
		BlockNumber:    blockNumber,
		BlockHash:      blockHash,
		BlockTimestamp: blockTimestamp,
		ParentHash:     parentHash,
		RowID:          row_id,
		Path:           path,
		L1BlockNumber:  l1BlockNumber,
	}
}

type TransactionIndex struct {
	chain            string
	BlockNumber      uint64
	BlockHash        string
	BlockTimestamp   uint64
	FromAddress      string
	ToAddress        string
	RowID            uint64
	Selector         string
	TransactionHash  string
	TransactionIndex uint64
	Type             uint64
	Path             string
}

func (t TransactionIndex) TableName() string {
	// This is the table name in the database for particular chain
	return t.chain + "_transaction_index"
}

func NewTransactionIndex(chain string, blockNumber uint64, blockHash string, blockTimestamp uint64, fromAddress string, toAddress string, selector string, row_id uint64, transactionHash string, transactionIndex uint64, tx_type uint64, path string) TransactionIndex {
	return TransactionIndex{
		chain:            chain,
		BlockNumber:      blockNumber,
		BlockHash:        blockHash,
		BlockTimestamp:   blockTimestamp,
		FromAddress:      fromAddress,
		ToAddress:        toAddress,
		RowID:            row_id,
		Selector:         selector,
		TransactionHash:  transactionHash,
		TransactionIndex: transactionIndex,
		Type:             tx_type,
		Path:             path,
	}
}

type LogIndex struct {
	chain           string
	BlockNumber     uint64
	BlockHash       string
	BlockTimestamp  uint64
	Address         string
	TransactionHash string
	Selector        *string // TODO: 1) Add Topic1, Topic2. 2) Rename Topic0 to selector
	Topic1          *string
	Topic2          *string
	RowID           uint64
	LogIndex        uint64
	Path            string
}

func NewLogIndex(chain string, address string, blockNumber uint64, blockHash string, transactionHash string, BlockTimestamp uint64, topic0 *string, topic1 *string, topic2 *string, row_id uint64, logIndex uint64, path string) LogIndex {
	return LogIndex{
		chain:           chain,
		Address:         address,
		BlockNumber:     blockNumber,
		BlockHash:       blockHash,
		BlockTimestamp:  BlockTimestamp,
		TransactionHash: transactionHash,
		Selector:        topic0,
		Topic1:          topic1,
		Topic2:          topic2,
		RowID:           row_id,
		LogIndex:        logIndex,
		Path:            path,
	}
}

type IndexType string

const (
	BlockIndexType       IndexType = "blocks"
	TransactionIndexType IndexType = "transactions"
	LogIndexType         IndexType = "logs"
)

type BlockCache struct {
	BlockNumber    uint64
	BlockTimestamp uint64
	BlockHash      string
}

type AbiJob struct {
	ID                    string
	Address               []byte
	UserID                string
	CustomerID            string
	AbiSelector           string
	Chain                 string
	AbiName               string
	Status                string
	HistoricalCrawlStatus string
	Progress              int
	MoonwormTaskPickedup  bool
	Abi                   string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type CustomerUpdates struct {
	CustomerID  string                                  `json:"customer_id"`
	Abis        map[string]map[string]map[string]string `json:"abis"`
	BlocksCache map[uint64]uint64                       `json:"blocks_cache"`
	Data        RawChainData                            `json:"data"`
}

type TaskForTransaction struct {
	Hash     string `json:"hash"`
	Address  string `json:"address"`
	Selector string `json:"selector"`
	ABI      string `json:"abi"`
	RowID    uint64 `json:"row_id"`
	Path     string `json:"path"`
}

type TaskForLog struct { // Assuming structure similar to TransactionIndex
	Hash     string `json:"hash"`
	Address  string `json:"address"`
	Selector string `json:"selector"`
	ABI      string `json:"abi"`
	RowID    uint64 `json:"row_id"`
	Path     string `json:"path"`
}

type RawChainData struct {
	Transactions []TaskForTransaction `json:"transactions"`
	Events       []TaskForLog         `json:"events"`
}

type EventLabel struct {
	Address         string
	BlockNumber     uint64
	BlockHash       string
	CallerAddress   string
	Label           string
	LabelName       string
	LabelType       string
	OriginAddress   string
	TransactionHash string
	LabelData       string
	BlockTimestamp  uint64
	LogIndex        uint64
}

type TransactionLabel struct {
	Address         string
	BlockNumber     uint64
	BlockHash       string
	CallerAddress   string
	Label           string
	LabelName       string
	LabelType       string
	OriginAddress   string
	TransactionHash string
	LabelData       string
	BlockTimestamp  uint64
}

type protoEventsWithAbi struct {
	Events [][]byte
	Abi    string
}

type protoTransactionsWithAbi struct {
	Transactions [][]byte
	Abi          string
}
