// Tool to extract text from parquet files for Zig benchmarking
// Run: go run extract_texts.go -parquet ~/data/fineweb-2/vie_Latn/train -output texts.bin -limit 100000
package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"
)

type PureTextParquet struct {
	Text string `parquet:"text"`
}

func main() {
	parquetDir := flag.String("parquet", "", "Parquet directory")
	output := flag.String("output", "texts.bin", "Output binary file")
	limit := flag.Int("limit", 0, "Max documents (0=all)")
	flag.Parse()

	if *parquetDir == "" {
		home, _ := os.UserHomeDir()
		*parquetDir = filepath.Join(home, "data", "fineweb-2", "vie_Latn", "train")
	}

	log.Printf("Reading parquet from: %s", *parquetDir)
	log.Printf("Output: %s", *output)
	if *limit > 0 {
		log.Printf("Limit: %d docs", *limit)
	}

	// Find parquet files
	entries, err := os.ReadDir(*parquetDir)
	if err != nil {
		log.Fatalf("Failed to read directory: %v", err)
	}

	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".parquet") {
			files = append(files, filepath.Join(*parquetDir, e.Name()))
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		log.Fatalf("No parquet files found in %s", *parquetDir)
	}
	log.Printf("Found %d parquet files", len(files))

	// Create output file
	out, err := os.Create(*output)
	if err != nil {
		log.Fatalf("Failed to create output: %v", err)
	}
	defer out.Close()

	// Write header placeholder (will update later)
	// Format: [4 bytes: num_docs] [8 bytes: total_bytes]
	header := make([]byte, 12)
	out.Write(header)

	ctx := context.Background()
	startTime := time.Now()
	var totalDocs int64
	var totalBytes int64

	for _, file := range files {
		if *limit > 0 && totalDocs >= int64(*limit) {
			break
		}

		f, err := os.Open(file)
		if err != nil {
			log.Printf("Failed to open %s: %v", file, err)
			continue
		}

		stat, _ := f.Stat()
		pf, err := parquet.OpenFile(f, stat.Size())
		if err != nil {
			f.Close()
			log.Printf("Failed to parse parquet %s: %v", file, err)
			continue
		}

		reader := parquet.NewGenericReader[PureTextParquet](pf)
		batch := make([]PureTextParquet, 10000)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, err := reader.Read(batch)
			if err != nil && err != io.EOF {
				log.Printf("Read error: %v", err)
				break
			}

			for i := 0; i < n; i++ {
				if *limit > 0 && totalDocs >= int64(*limit) {
					break
				}

				text := batch[i].Text
				textBytes := []byte(text)
				textLen := len(textBytes)

				// Write: [4 bytes: length] [text bytes]
				lenBuf := make([]byte, 4)
				binary.LittleEndian.PutUint32(lenBuf, uint32(textLen))
				out.Write(lenBuf)
				out.Write(textBytes)

				totalDocs++
				totalBytes += int64(textLen)

				if totalDocs%100000 == 0 {
					elapsed := time.Since(startTime)
					rate := float64(totalDocs) / elapsed.Seconds()
					log.Printf("Progress: %d docs (%.0f docs/sec)", totalDocs, rate)
				}
			}

			if err == io.EOF || n == 0 {
				break
			}
		}

		reader.Close()
		f.Close()
	}

	// Update header
	out.Seek(0, 0)
	binary.Write(out, binary.LittleEndian, uint32(totalDocs))
	binary.Write(out, binary.LittleEndian, totalBytes)

	elapsed := time.Since(startTime)
	log.Printf("Done! %d documents, %.2f MB in %.2fs (%.0f docs/sec)",
		totalDocs, float64(totalBytes)/(1024*1024), elapsed.Seconds(),
		float64(totalDocs)/elapsed.Seconds())
}
