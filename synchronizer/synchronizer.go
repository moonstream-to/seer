package synchronizer

import (
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
			log.Println("Error creating RDS connection for %s customer, err: %v", id, pgxErr)
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
			d.startBlock = uint64(crawler.SetDefaultStartBlock(0, latestBlockNumber))
		}
	}

	// Get the latest block from indexes database
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
	tempEndBlock := d.startBlock + d.batchSize
	var isCycleFinished bool
	for {
		tempEndBlock = d.startBlock + d.batchSize
		if d.endBlock != 0 {
			if tempEndBlock >= d.endBlock {
				tempEndBlock = d.endBlock
				isEnd = true
				isCycleFinished = true
				log.Printf("End block %d almost reached", tempEndBlock)
			}
		}
		if tempEndBlock >= indexedLatestBlock {
			tempEndBlock = indexedLatestBlock
			isCycleFinished = true
		}

		if crawler.SEER_CRAWLER_DEBUG {
			log.Printf("Syncing %d blocks from %d to %d\n", tempEndBlock-d.startBlock, d.startBlock, tempEndBlock)
		}

		// Read updates from the indexer db
		// This function will return a list of customer updates 1 update is 1 customer
		updates, err := indexer.DBConnection.ReadUpdates(d.blockchain, d.startBlock, tempEndBlock, customerIds)
		if err != nil {
			return isEnd, fmt.Errorf("error reading updates: %w", err)
		}

		log.Printf("Read %d users updates from the indexer db\n", len(updates))

		var wg sync.WaitGroup

		sem := make(chan struct{}, 5)  // Semaphore to control concurrency
		errChan := make(chan error, 1) // Buffered channel for error handling

		for _, update := range updates {
			wg.Add(1)
			go func(update indexer.CustomerUpdates) {
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

				// Events group by path for batch reading
				groupByPathEvents := make(map[string][]uint64)

				for _, event := range update.Data.Events {

					if _, ok := groupByPathEvents[event.Path]; !ok {
						groupByPathEvents[event.Path] = []uint64{}
					}

					groupByPathEvents[event.Path] = append(groupByPathEvents[event.Path], event.RowID)
				}

				eventsReadMap := []storage.ReadItem{}

				for path, rowIds := range groupByPathEvents {
					eventsReadMap = append(eventsReadMap, storage.ReadItem{
						Key:    path,
						RowIds: rowIds,
					})
				}

				var decodedEvents []indexer.EventLabel

				for _, item := range eventsReadMap {

					if crawler.SEER_CRAWLER_DEBUG {
						log.Printf("Key: %s", item.Key)
					}

					// Read events from storage
					rawEvents, readErr := d.StorageInstance.Read(item.Key)
					if readErr != nil {
						errChan <- fmt.Errorf("error reading events for customer %s: %w", update.CustomerID, readErr)
						return
					}

					// Decode the events using ABIs
					decodedEventsPack, decErr := d.Client.DecodeProtoEventsToLabels(&rawEvents, update.BlocksCache, update.Abis)
					if decErr != nil {
						fmt.Println("Error decoding events: ", decErr)
						errChan <- fmt.Errorf("error decoding events for customer %s: %w", update.CustomerID, decErr)
						return
					}

					decodedEvents = append(decodedEvents, decodedEventsPack...)
				}

				// Write events to user RDS
				customer.Pgx.WriteEvents(
					d.blockchain,
					decodedEvents,
				)

				// Transactions
				groupByPathTransactions := make(map[string][]uint64)

				for _, transaction := range update.Data.Transactions {

					if _, ok := groupByPathTransactions[transaction.Path]; !ok {
						groupByPathTransactions[transaction.Path] = []uint64{}
					}

					groupByPathTransactions[transaction.Path] = append(groupByPathTransactions[transaction.Path], transaction.RowID)
				}

				transactionsReadMap := []storage.ReadItem{}

				for path, rowIds := range groupByPathTransactions {
					transactionsReadMap = append(transactionsReadMap, storage.ReadItem{
						Key:    path,
						RowIds: rowIds,
					})
				}
				encodedTransactions, err := d.StorageInstance.ReadBatch(transactionsReadMap)

				if err != nil {
					errChan <- fmt.Errorf("error reading transactions for customer %s: %w", update.CustomerID, err)
					return
				}

				// remap the transactions to a single slice

				var all_transactions []string

				for _, data := range encodedTransactions {
					all_transactions = append(all_transactions, data...)
				}

				decodedTransactions, err := d.Client.DecodeProtoTransactionsToLabels(all_transactions, update.BlocksCache, update.Abis)

				if err != nil {
					errChan <- fmt.Errorf("error decoding transactions for customer %s: %w", update.CustomerID, err)
					return
				}

				// Write transactions to user RDS
				customer.Pgx.WriteTransactions(
					d.blockchain,
					decodedTransactions,
				)

				<-sem
			}(update)
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

		d.startBlock = tempEndBlock + 1

		if isCycleFinished {
			break
		}
	}

	return isEnd, nil
}
