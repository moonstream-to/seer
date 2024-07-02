package storage

import (
	"bytes"
	"context"
)

type ListReturnFunc func(any) string

type Storer interface {
	Save(batchDir, filename string, bf bytes.Buffer) error
	Read(key string) (bytes.Buffer, error)
	ReadBatch(readItems []ReadItem) (map[string][]string, error)
	Delete(key string) error
	List(ctx context.Context, delim, blockBatch string, timeout int, returnFunc ListReturnFunc) ([]string, error)
}

type ReadItem struct {
	Key    string
	RowIds []uint64
}
