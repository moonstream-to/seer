package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/moonstream-to/seer/blockchain/common"
	"github.com/moonstream-to/seer/indexer"
	"google.golang.org/protobuf/proto"
)

// Crawler defines the crawler structure.
type Crawler struct {
	blockchain     string
	startBlock     uint64
	maxBlocksBatch int
	minBlocksBatch int
	confirmations  int
	endBlock       uint64
	force          bool
	baseDir        string
}

// NewCrawler creates a new crawler instance with the given blockchain handler.
func NewCrawler(chain string, startBlock uint64, endBlock uint64, timeout int, batchSize int, confirmations int, baseDir string, force bool) *Crawler {
	fmt.Printf("Start block: %d\n", startBlock)
	return &Crawler{
		blockchain:     chain,
		startBlock:     startBlock,
		endBlock:       endBlock,
		maxBlocksBatch: batchSize,
		minBlocksBatch: 1,
		confirmations:  confirmations,
		baseDir:        baseDir,
		force:          force,
	}

}

func writeProtoMessagesToFile(messages []proto.Message, filePath string) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	for _, msg := range messages {
		data, err := proto.Marshal(msg)
		if err != nil {
			log.Fatalf("Failed to marshal proto message: %v", err)
		}
		if _, err := file.Write(data); err != nil {
			log.Fatalf("Failed to write data to file: %v", err)
		}
	}
}

func updateBlockIndexFilepaths(indices []indexer.BlockIndex, baseDir string) {
	for i, _ := range indices {
		indices[i].Filepath = filepath.Join(baseDir, "blocks.proto")
	}
}

func updateTransactionIndexFilepaths(indices []indexer.TransactionIndex, baseDir string) {
	for i, _ := range indices {
		indices[i].Filepath = filepath.Join(baseDir, "transactions.proto")
	}
}

// writeIndicesToFile serializes and writes a batch of indices to a specified JSON file.
func writeIndicesToFile(indices []interface{}, filePath string) { // small cheating
	// Open or create the file for appending
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644) // that cool
	if err != nil {
		log.Fatalf("Failed to open file %s for writing: %v", filePath, err)
	}
	defer file.Close()

	// Serialize and write each index in the batch
	encoder := json.NewEncoder(file)
	for _, index := range indices {
		if err := encoder.Encode(index); err != nil {
			log.Fatalf("Failed to encode index to %s: %v", filePath, err)
		}
	}
}

// read latest indexed block
// readLatestIndexedBlock reads the latest indexed block from a JSON file
func readLatestIndexedBlock() (uint64, error) {
	// Open the file for reading
	file, err := os.Open("data/block_index.json")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Initialize a decoder that reads from the file
	decoder := json.NewDecoder(file)
	var blockIndex indexer.BlockIndex
	var lastBlockNumber uint64 // Variable to store the last read block number

	// Read the file line by line
	for {
		// Decode the next JSON object into the BlockIndex struct
		if err := decoder.Decode(&blockIndex); err != nil {
			if err == io.EOF {
				// If EOF is reached, break out of the loop
				break
			} else {
				// For any other error, return it
				return 0, err
			}
		}
		// Successfully decoded, update lastBlockNumber
		lastBlockNumber = uint64(blockIndex.BlockNumber)
	}

	// Return the last block number read from the file
	return lastBlockNumber, nil
}

// Start initiates the crawling process for the configured blockchain.
func (c *Crawler) Start() {

	chainType := "ethereum"
	url := os.Getenv("INFURA_URL")

	log.Printf("Starting crawler using blockchain URL: %s", url)
	client, err := common.NewClient(chainType, url)
	if err != nil {
		log.Fatal(err)
	}

	latestBlockNumber, err := client.GetLatestBlockNumber()
	// Initial setup
	baseDir := "data"
	crawlBatchSize := uint64(10) // Crawl in batches of 10
	confirmations := uint64(10)  // Consider 10 confirmations to ensure block finality

	fmt.Printf("Latest block number: %d\n", latestBlockNumber)
	fmt.Printf("Start block: %d\n", c.startBlock)

	fmt.Printf("force: %v\n", c.force)

	if c.force {
		// try extract from index
		if c.startBlock == uint64(0) {
			startBlock := latestBlockNumber.Uint64() - uint64(confirmations) - 100
			c.startBlock = startBlock

		}

	} else {
		latestIndexedBlock, err := readLatestIndexedBlock()

		if err != nil {
			log.Fatalf("Failed to read latest indexed block: %v", err)
		}

		if latestIndexedBlock != uint64(0) {
			c.startBlock = latestIndexedBlock
		}
	}

	endBlock := uint64(0)
	safeBlock := uint64(0)

	for {
		// Fetch the latest block number at the start of each iteration
		latestBlockNumber, err = client.GetLatestBlockNumber()

		if err != nil {
			log.Fatalf("Failed to get latest block number: %v", err)
		}

		safeBlock = latestBlockNumber.Uint64() - confirmations

		endBlock = c.startBlock + crawlBatchSize

		if endBlock > safeBlock {
			//wait for new blocks
			time.Sleep(1 * time.Minute) // Adjust sleep time as needed
			continue
		}

		// Fetch the blocks and transactions for the range
		blocks, transactions, blockIndex, transactionIndex, err := common.CrawlBlocks(client, big.NewInt(int64(c.startBlock)), big.NewInt(int64(endBlock)))
		if err != nil {
			log.Fatalf("Failed to get blocks: %v", err)
		}

		fmt.Printf("Crawled transactions: %d\n", len(transactions))

		// Compute batch directory for this iteration
		batchDir := filepath.Join(c.baseDir, fmt.Sprintf("%d-%d", c.startBlock, endBlock))
		if err := os.MkdirAll(batchDir, 0755); err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		}

		// Write the blocks, transactions, and their indices to disk
		writeProtoMessagesToFile(blocks, filepath.Join(batchDir, "blocks.proto"))
		writeProtoMessagesToFile(transactions, filepath.Join(batchDir, "transactions.proto"))
		updateBlockIndexFilepaths(blockIndex, batchDir)
		updateTransactionIndexFilepaths(transactionIndex, batchDir)

		// Write updated indices to their respective files
		// Convert blockIndex to []interface{}
		interfaceBlockIndex := make([]interface{}, len(blockIndex))
		for i, v := range blockIndex {
			interfaceBlockIndex[i] = v
		}
		writeIndicesToFile(interfaceBlockIndex, filepath.Join(baseDir, "block_index.json"))

		interfaceTransactionIndex := make([]interface{}, len(transactionIndex))
		for i, v := range transactionIndex {
			interfaceTransactionIndex[i] = v
		}
		writeIndicesToFile(interfaceTransactionIndex, filepath.Join(baseDir, "transaction_index.json"))

		// Convert transactionIndex to []interface{}

		nextStartBlock := endBlock + 1
		c.startBlock = nextStartBlock
	}
}

// You can add more methods here for additional functionalities
// such as handling reconnection logic, managing crawler state, etc.
