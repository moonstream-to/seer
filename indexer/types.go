package indexer

// Define the interface for handling general index types such as blocks, transactions, and logs for fast access to blockchain data.

type BlockIndex struct {
	BlockNumber    int64
	BlockHash      string
	BlockTimestamp int64
	ParentHash     string
	Filepath       string
}

type TransactionIndex struct {
	BlockNumber          int64
	BlockHash            string
	BlockTimestamp       int64
	TransactionHash      string
	TransactionIndex     int64
	TransactionTimestamp int64
	Filepath             string
}

type LogIndex struct {
	BlockNumber     int64
	BlockHash       string
	BlockTimestamp  int64
	TransactionHash string
	LogIndex        int64
	Filepath        string
}
