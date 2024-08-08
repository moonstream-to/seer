package synchronizer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	seer_blockchain "github.com/moonstream-to/seer/blockchain"
	"github.com/moonstream-to/seer/crawler"
	"github.com/moonstream-to/seer/indexer"
	"github.com/moonstream-to/seer/storage"
	"golang.org/x/exp/slices"
)

type Synchronizer struct {
	Client          seer_blockchain.BlockchainClient
	StorageInstance storage.Storer

	blockchain string
	startBlock uint64
	endBlock   uint64
	batchSize  uint64
	baseDir    string
	basePath   string
}

// NewSynchronizer creates a new synchronizer instance with the given blockchain handler.
func NewSynchronizer(blockchain, baseDir string, startBlock, endBlock, batchSize uint64, timeout int) (*Synchronizer, error) {
	var synchronizer Synchronizer

	basePath := filepath.Join(baseDir, crawler.SeerCrawlerStoragePrefix, "data", blockchain)
	storageInstance, err := storage.NewStorage(storage.SeerCrawlerStorageType, basePath)
	if err != nil {
		log.Fatalf("Failed to create storage instance: %v", err)
		panic(err)
	}

	client, err := seer_blockchain.NewClient(blockchain, crawler.BlockchainURLs[blockchain], timeout)
	if err != nil {
		log.Println("Error initializing blockchain client:", err)
		log.Fatal(err)
	}

	log.Printf("Initialized new synchronizer at blockchain: %s, startBlock: %d, endBlock: %d", blockchain, startBlock, endBlock)

	synchronizer = Synchronizer{
		Client:          client,
		StorageInstance: storageInstance,

		blockchain: blockchain,
		startBlock: startBlock,
		endBlock:   endBlock,
		batchSize:  batchSize,
		baseDir:    baseDir,
		basePath:   basePath,
	}

	return &synchronizer, nil
}

// Read index storage

// -------------------------------------------------------------------------------------------------------------------------------
//                 ________Moonstream blockstore________
// crawler 1 ---->|       Contain raw                   |
// 			      |        blocks                       |
// crawler 2 ---->|        transactions                 |
// 			      |        events                       |
// crawler 3 ---->|                                     |
// 			      |                                     |---->{synchonizer} --------> {user_RDS}
// crawler 4 ---->|                                     |     read from blockstore
// 			      |                                     |     via indexer
// crawler 5 ---->|                                     |     Decode transactions and events
// ....		      |									    |     using ABIs and addresses
// 			      |									    |     and write to user_RDS
// crawler n ---->|								     	|
// 			      |									    |
// 		    	  |_____________________________________|

// -------------------------------------------------------------------------------------------------------------------------------

func GetDBConnection(uuid string) (string, error) {

	// Create the request
	req, err := http.NewRequest("GET", MOONSTREAM_DB_V3_CONTROLLER_API+"/customers/"+uuid+"/instances/1/creds/seer/url", nil)
	if err != nil {
		return "", err
	}

	// Set the authorization header
	req.Header.Set("Authorization", "Bearer "+MOONSTREAM_DB_V3_CONTROLLER_SEER_ACCESS_TOKEN)

	// Perform the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Failed to get connection string for id", uuid, ":", MOONSTREAM_DB_V3_CONTROLLER_API+"/customers/"+uuid+"/instances/1/creds/seer/url")
		return "", fmt.Errorf("failed to get connection string for id %s: %s", uuid, resp.Status)
	}

	// Read string from body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	connectionString := strings.Trim(string(bodyBytes), "\"")

	// Validate and possibly correct the connection string
	connectionString, err = ensurePortInConnectionString(connectionString)
	if err != nil {
		return "", fmt.Errorf("invalid connection string for id %s: %v", uuid, err)
	}

	return connectionString, nil
}

func (d *Synchronizer) ReadAbiJobsFromDatabase(blockchain string) ([]indexer.AbiJob, error) {
	abiJobs, err := indexer.DBConnection.ReadABIJobs(blockchain)
	if err != nil {
		return nil, err
	}
	return abiJobs, nil
}

func ensurePortInConnectionString(connStr string) (string, error) {
	parsedURL, err := url.Parse(connStr)
	if err != nil {
		return "", err
	}

	host := parsedURL.Host
	if !strings.Contains(host, ":") {
		host = host + ":5432" // Add the default PostgreSQL port
		parsedURL.Host = host
	}

	return parsedURL.String(), nil
}

type CustomerDBConnection struct {
	Uri string

	Pgx *indexer.PostgreSQLpgx
}

// getCustomers fetch ABI jobs, customer IDs and database URLs
func (d *Synchronizer) getCustomers(customerDbUriFlag string) (map[string]CustomerDBConnection, []string, error) {
	customerDBConnections := make(map[string]CustomerDBConnection)
	var customerIds []string

	// Read ABI jobs from database
	abiJobs, err := d.ReadAbiJobsFromDatabase(d.blockchain)
	if err != nil {
		return customerDBConnections, customerIds, err
	}

	// Create a set of customer IDs from ABI jobs to remove duplicates
	customerIdsSet := make(map[string]struct{})
	for _, job := range abiJobs {
		customerIdsSet[job.CustomerID] = struct{}{}
	}

	for id := range customerIdsSet {
		var connectionString string
		var dbConnErr error
		if customerDbUriFlag == "" {
			connectionString, dbConnErr = GetDBConnection(id)
			if dbConnErr != nil {
				log.Printf("Unable to get connection database URI for %s customer, err: %v", id, dbConnErr)
				continue
			}
		} else {
			connectionString = customerDbUriFlag
		}

		pgx, pgxErr := indexer.NewPostgreSQLpgxWithCustomURI(connectionString)
		if pgxErr != nil {
			log.Printf("Error creating RDS connection for %s customer, err: %v", id, pgxErr)
			continue
		}

		customerDBConnections[id] = CustomerDBConnection{
			Uri: connectionString,
			Pgx: pgx,
		}
		customerIds = append(customerIds, id)

	}
	log.Println("Customer IDs to sync:", customerIds)

	return customerDBConnections, customerIds, nil
}

func (d *Synchronizer) Start(customerDbUriFlag string) {
	var isEnd bool

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	isEnd, err := d.SyncCycle(customerDbUriFlag)
	if err != nil {
		fmt.Println("Error during first synchronization cycle:", err)
	}
	if isEnd {
		return
	}

	for {
		select {
		case <-ticker.C:
			isEnd, err := d.SyncCycle(customerDbUriFlag)
			if err != nil {
				fmt.Println("Error during synchronization cycle:", err)
			}
			if isEnd {
				return
			}
		}
	}
}

func (d *Synchronizer) SyncCycle(customerDbUriFlag string) (bool, error) {
	var isEnd bool

	customerDBConnections, customerIds, customersErr := d.getCustomers(customerDbUriFlag)
	if customersErr != nil {
		return isEnd, customersErr
	}

	// Set startBlocks as latest labeled block from customers or from the blockchain if customers are not indexed yet
	if d.startBlock == 0 {
		var latestCustomerBlocks []uint64
		for id, customer := range customerDBConnections {

			pool := customer.Pgx.GetPool()
			conn, err := pool.Acquire(context.Background())
			if err != nil {
				log.Println("Error acquiring pool connection: ", err)
				return isEnd, err
			}
			defer conn.Release()

			latestLabelBlock, err := customer.Pgx.ReadLastLabel(d.blockchain)
			if err != nil {
				log.Println("Error reading latest block: ", err)
				return isEnd, err
			}
			latestCustomerBlocks = append(latestCustomerBlocks, latestLabelBlock)
			log.Printf("Latest block for customer %s is: %d\n", id, latestLabelBlock)
		}

		// Determine the start block as the maximum of the latest blocks of all customers
		var maxCustomerLatestBlock uint64
		if len(latestCustomerBlocks) != 0 {
			maxCustomerLatestBlock = slices.Max(latestCustomerBlocks)
		}
		if maxCustomerLatestBlock != 0 {
			d.startBlock = maxCustomerLatestBlock - 100
		} else {
			// In case start block is still 0, get the latest block from the blockchain minus shift
			latestBlockNumber, latestErr := d.Client.GetLatestBlockNumber()
			if latestErr != nil {
				return isEnd, fmt.Errorf("failed to get latest block number: %v", latestErr)
			}
			d.startBlock = uint64(crawler.SetDefaultStartBlock(0, latestBlockNumber)) - 100
		}
	}

	// Get the latest block from the indexer db
	indexedLatestBlock, idxLatestErr := indexer.DBConnection.GetLatestDBBlockNumber(d.blockchain)
	if idxLatestErr != nil {
		return isEnd, idxLatestErr
	}

	if d.endBlock != 0 && indexedLatestBlock > d.endBlock {
		indexedLatestBlock = d.endBlock
	}

	if d.startBlock >= indexedLatestBlock {
		log.Printf("Value in startBlock %d greater or equal indexedLatestBlock %d, waiting next iteration..", d.startBlock, indexedLatestBlock)
		return isEnd, nil
	}

	// Main loop Steps:
	// 1. Read updates from the indexer db
	// 2. For each update, read the original event data from storage
	// 3. Decode input data using ABIs
	// 4. Write updates to the user RDS
	var isCycleFinished bool
	for {
		if d.endBlock != 0 {
			if d.startBlock >= d.endBlock {
				isEnd = true
				isCycleFinished = true
				log.Printf("End block %d almost reached", d.endBlock)
			}
		}
		if d.endBlock >= indexedLatestBlock {
			isCycleFinished = true
		}

		// Read updates from the indexer db
		// This function will return a list of customer updates 1 update is 1 customer
		lastBlockOfChank, path, updates, err := indexer.DBConnection.ReadUpdates(d.blockchain, d.startBlock, customerIds)

		if err != nil {
			return isEnd, fmt.Errorf("error reading updates: %w", err)
		}

		fmt.Println("Last block of chank: ", lastBlockOfChank)

		// Read the raw data from the storage for current path
		rawData, readErr := d.StorageInstance.Read(path)
		if readErr != nil {
			return isEnd, fmt.Errorf("error reading events for customers %s: %w", readErr)
		}

		if err != nil {
			return isEnd, fmt.Errorf("error reading updates: %w", err)
		}

		log.Printf("Read %d users updates from the indexer db in range of blocks %d-%d\n", len(updates), d.startBlock, lastBlockOfChank)
		var wg sync.WaitGroup

		sem := make(chan struct{}, 5)  // Semaphore to control concurrency
		errChan := make(chan error, 1) // Buffered channel for error handling

		for _, update := range updates {
			wg.Add(1)
			go func(update indexer.CustomerUpdates, rawData bytes.Buffer) {
				defer wg.Done()

				sem <- struct{}{} // Acquire semaphore

				// Get the RDS connection for the customer
				customer := customerDBConnections[update.CustomerID]

				// Create a connection to the user RDS
				pool := customer.Pgx.GetPool()
				conn, err := pool.Acquire(context.Background())
				if err != nil {
					errChan <- fmt.Errorf("error acquiring connection for customer %s: %w", update.CustomerID, err)
					return
				}
				defer conn.Release()

				var decodedEventsPack []indexer.EventLabel
				var decodedTransactionsPack []indexer.TransactionLabel

				if crawler.SEER_CRAWLER_DEBUG {
					log.Printf("Key: %s", update.Path)
				}

				// decodedEvents, decodedTransactions, decErr
				decodedEvents, decodedTransactions, decErr := d.Client.DecodeProtoEntireBlockToLabels(&rawData, update.Abis)
				if decErr != nil {
					fmt.Println("Error decoding events: ", decErr)
					errChan <- fmt.Errorf("error decoding events for customer %s: %w", update.CustomerID, decErr)
					return
				}

				decodedEventsPack = append(decodedEventsPack, decodedEvents...)
				decodedTransactionsPack = append(decodedTransactionsPack, decodedTransactions...)

				customer.Pgx.WriteLabes(d.blockchain, decodedTransactionsPack, decodedEventsPack)

				<-sem
			}(update, rawData)
		}

		wg.Wait()

		close(sem)
		close(errChan) // Close the channel to signal that all goroutines have finished

		// Check for errors from goroutines
		for err := range errChan {
			fmt.Println("Error during synchronization cycle:", err)
			if err != nil {
				return isEnd, err
			}
		}

		d.startBlock = lastBlockOfChank + 1

		if isCycleFinished {
			break
		}
	}

	return isEnd, nil
}
