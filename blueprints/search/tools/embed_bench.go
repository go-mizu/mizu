package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type embReq struct {
	Input []string `json:"input"`
}

type embResp struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func main() {
	addrs := []string{
		"http://localhost:8087",
	}
	batchSize := 16
	workers := 8
	totalBatches := 200

	// Generate realistic texts (~500 chars each)
	text := "The quick brown fox jumps over the lazy dog. " +
		"This is a test of the embedding throughput for the search pipeline. " +
		"We want to measure the raw HTTP client performance without pipeline overhead. " +
		"Each chunk is approximately 500 characters which matches the production config. " +
		"The llamacpp server should be running with parallel=16 slots for maximum throughput. " +
		"End of test chunk."
	texts := make([]string, batchSize)
	for i := range texts {
		texts[i] = text
	}

	payload, _ := json.Marshal(embReq{Input: texts})

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        128,
			MaxIdleConnsPerHost: 64,
			MaxConnsPerHost:     64,
			IdleConnTimeout:     120 * time.Second,
		},
	}

	var total atomic.Int64
	var wg sync.WaitGroup
	ch := make(chan int, totalBatches)
	for i := 0; i < totalBatches; i++ {
		ch <- i
	}
	close(ch)

	start := time.Now()

	for w := 0; w < workers; w++ {
		addr := addrs[w%len(addrs)]
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			for range ch {
				req, _ := http.NewRequest("POST", addr+"/v1/embeddings", bytes.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					fmt.Printf("ERROR: %v\n", err)
					continue
				}
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				var r embResp
				json.Unmarshal(body, &r)
				total.Add(int64(len(r.Data)))
			}
		}(addr)
	}

	// Progress
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			t := total.Load()
			if t == 0 {
				continue
			}
			elapsed := time.Since(start).Seconds()
			fmt.Printf("\r  vecs=%d rate=%.0f vec/s", t, float64(t)/elapsed)
		}
	}()

	wg.Wait()
	elapsed := time.Since(start)
	t := total.Load()
	fmt.Printf("\n\nTotal: %d vectors in %s = %.0f vec/s (workers=%d batch=%d addrs=%d)\n",
		t, elapsed.Round(time.Millisecond), float64(t)/elapsed.Seconds(), workers, batchSize, len(addrs))
}
