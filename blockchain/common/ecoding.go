package common

import "math/big"

// BlockchainHandler defines the interface for handling blockchain data fetching.
type BlockchainHandler interface {
	FetchData(blockNumber *big.Int) ([]EventData, error)
}

// EventData outlines the structure for blockchain event data.
type EventData struct {
	// Common fields...
	BlockNumber uint64
	Data        []byte
	ChainID     string
	Extra       interface{} // Use for blockchain-specific additional data
}
