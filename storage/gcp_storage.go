package storage

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCS implements the Storer interface for Google Cloud Storage
type GCS struct {
	Client   *storage.Client
	BasePath string
}

// NewGCSStorage initializes a GCS storage with the provided client
func NewGCSStorage(client *storage.Client, basePath string) *GCS {
	return &GCS{
		Client:   client,
		BasePath: basePath,
	}
}

func (g *GCS) Save(batchDir, filename string, bf bytes.Buffer) error {
	key := filepath.Join(g.BasePath, batchDir, filename)

	ctx := context.Background()

	bucket := g.Client.Bucket(SeerCrawlerStorageBucket)

	obj := bucket.Object(key)

	wc := obj.NewWriter(ctx)
	wc.Metadata = map[string]string{
		"encoder": "varint-size-delimited",
	}

	if _, err := io.Copy(wc, &bf); err != nil {
		return fmt.Errorf("failed to write object to bucket: %v", err)
	}

	return nil
}

func (g *GCS) Read(key string) (bytes.Buffer, error) {

	ctx := context.Background()

	bucket := g.Client.Bucket(SeerCrawlerStorageBucket)

	obj := bucket.Object(key)

	r, err := obj.NewReader(ctx)
	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to create reader: %v", err)
	}
	defer r.Close()

	// Read the object data into a buffer
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return bytes.Buffer{}, fmt.Errorf("failed to read object data: %v", err)
	}

	return *buf, nil

}

var (
	GCSListReturnNameFunc = func(item any) string {
		if attr, ok := item.(*storage.ObjectAttrs); ok {
			return attr.Name
		}
		return ""
	}

	GCSListReturnPrefixFunc = func(item any) string {
		if attr, ok := item.(*storage.ObjectAttrs); ok {
			return attr.Prefix
		}
		return ""
	}
)

func (g *GCS) List(ctx context.Context, delim, blockBatch string, timeout int, returnFunc ListReturnFunc) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(timeout))
	defer cancel()

	// Prefixes and delimiters can be used to emulate directory listings.
	// Prefixes can be used to filter objects starting with prefix.
	// The delimiter argument can be used to restrict the results to only the
	// objects in the given "directory". Without the delimiter, the entire tree
	// under the prefix is returned.
	//
	// For example, given these blobs:
	//   /a/1.txt
	//   /a/b/2.txt
	//
	// If you just specify prefix="a/", you'll get back:
	//   /a/1.txt
	//   /a/b/2.txt
	//
	// However, if you specify prefix="a/" and delim="/", you'll get back:
	//   /a/1.txt
	prefix := fmt.Sprintf("%s/", g.BasePath)
	if blockBatch != "" {
		prefix = fmt.Sprintf("%s%s/", prefix, blockBatch)
	}
	log.Printf("Loading bucket items with prefix: %s and delim: %s", prefix, delim)

	it := g.Client.Bucket(SeerCrawlerStorageBucket).Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: delim,
	})

	var items []string
	itemsLen := 0

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []string{}, fmt.Errorf("Bucket(%q).Objects: %w", SeerCrawlerStorageBucket, err)
		}

		returnVal := returnFunc(attrs)
		if returnVal == "" {
			continue
		}

		itemsLen++
		items = append(items, returnVal)
	}

	log.Printf("Listed %d items", itemsLen)

	return items, nil
}

func (g *GCS) Delete(key string) error {

	ctx := context.Background()

	bucket := g.Client.Bucket(SeerCrawlerStorageBucket)

	obj := bucket.Object(key)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object from bucket: %v", err)
	}

	return nil

}

func (g *GCS) ReadBatch(readItems []ReadItem) (map[string][]string, error) {
	ctx := context.Background()

	bucket := g.Client.Bucket(SeerCrawlerStorageBucket)

	result := make(map[string][]string)

	for _, item := range readItems {
		obj := bucket.Object(item.Key)

		r, err := obj.NewReader(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create reader: %v", err)
		}
		defer r.Close()

		reader := bufio.NewReader(r)

		rowMap := make(map[uint64]bool)
		for _, id := range item.RowIds {
			rowMap[id] = true
		}

		var currentRow uint64 = 0

		for {
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read object from bucket: %v", err)
			}

			// Remove the newline character from the end of the line if it exists
			line = strings.TrimSuffix(line, "\n")

			if rowMap[currentRow] {
				if result[item.Key] == nil {
					result[item.Key] = make([]string, 0)
				}
				result[item.Key] = append(result[item.Key], line)
			}
			currentRow++
		}
	}

	return result, nil
}
