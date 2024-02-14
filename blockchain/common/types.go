package common

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type EventLog struct {
	Address          common.Address `json:"address"`          // The address of the contract that generated the log.
	Topics           []common.Hash  `json:"topics"`           // Topics are indexed parameters during log generation.
	Data             []byte         `json:"data"`             // The data field from the log.
	BlockNumber      *big.Int       `json:"blockNumber"`      // The block number where this log was in.
	TransactionHash  common.Hash    `json:"transactionHash"`  // The hash of the transaction that generated this log.
	BlockHash        common.Hash    `json:"blockHash"`        // The hash of the block where this log was in.
	Removed          bool           `json:"removed"`          // True if the log was reverted due to a chain reorganization.
	LogIndex         uint64         `json:"logIndex"`         // The index of the log in the block.
	TransactionIndex uint64         `json:"transactionIndex"` // The index of the transaction in the block.

}

type Trace struct {
	Action              json.RawMessage `json:"action"`
	BlockHash           common.Hash     `json:"blockHash"`
	BlockNumber         *big.Int        `json:"blockNumber"`
	Result              *TraceResult    `json:"result,omitempty"`
	SubTraces           int             `json:"subtraces"`
	TraceAddress        []int           `json:"traceAddress"`
	TransactionHash     common.Hash     `json:"transactionHash"`
	TransactionPosition uint64          `json:"transactionPosition"`
	Type                string          `json:"type"`
}

type TraceActionCall struct {
	From     common.Address `json:"from"`
	CallType string         `json:"callType"`
	Gas      *big.Int       `json:"gas"`
	Input    string         `json:"input"`
	To       common.Address `json:"to"`
	Value    *big.Int       `json:"value"`
}

type TraceActionReward struct {
	Author     common.Address `json:"author"`
	RewardType string         `json:"rewardType"`
	Value      *big.Int       `json:"value"`
}

type TraceResult struct {
	GasUsed *big.Int `json:"gasUsed"`
	Output  string   `json:"output"`
}
