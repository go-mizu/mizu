package crawl

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

)

// PipelineConfig holds configuration for the Mode C pipeline crawler.
type PipelineConfig struct {
	Cfg       Config
	DNS       DNSCache
	Results   ResultWriter
	Failures  FailureWriter
	RDB       ShardReopener
	SeedPath  string
	BatchSize int // domains per batch (0=auto from AvailMB)
	PageSize  int // seeds per SeedCursor page (0=10000)
	AvailMB   int
}

// RunPipeline executes the Mode C staged goroutine pipeline crawl.
// Stage 1 (DomainBatcher): reads SeedCursor pages, groups by domain, emits batches.
// Stage 2 (CrawlStage): runs keepalive engine per batch, reopens shards between batches.
func RunPipeline(ctx context.Context, pcfg PipelineConfig) (*Stats, error) {
	batchSize := pcfg.BatchSize
	if batchSize <= 0 {
		batchSize = AutoBatchDomains(pcfg.AvailMB, 3, 256)
	}
	pageSize := pcfg.PageSize
	if pageSize <= 0 {
		pageSize = 10_000
	}

	dns := pcfg.DNS
	if dns == nil {
		dns = &NoopDNS{}
	}

	cursor, err := NewSeedCursor(pcfg.SeedPath, pageSize)
	if err != nil {
		return nil, fmt.Errorf("pipeline: seed cursor: %w", err)
	}
	defer cursor.Close()

	// batchCh carries domain-grouped seed slices; cap=1 means CrawlStage
	// applies back-pressure to DomainBatcher when it falls behind.
	batchCh := make(chan []SeedURL, 1)

	var wg sync.WaitGroup
	var batcherErr error

	var statsMu sync.Mutex
	var combined *Stats

	// Stage 1: DomainBatcher
	wg.Add(1)
	go func() {
		defer close(batchCh)
		defer wg.Done()

		domainMap := make(map[string][]SeedURL)
		var domainOrder []string

		emit := func() {
			if len(domainOrder) == 0 {
				return
			}
			batch := make([]SeedURL, 0)
			for _, d := range domainOrder {
				batch = append(batch, domainMap[d]...)
			}
			select {
			case batchCh <- batch:
			case <-ctx.Done():
				return
			}
			domainMap = make(map[string][]SeedURL)
			domainOrder = domainOrder[:0]
		}

		for {
			page, err := cursor.Next(ctx)
			if err != nil {
				batcherErr = fmt.Errorf("pipeline batcher: %w", err)
				return
			}
			if len(page) == 0 {
				break // EOF
			}
			for _, s := range page {
				if _, ok := domainMap[s.Domain]; !ok {
					domainOrder = append(domainOrder, s.Domain)
				}
				domainMap[s.Domain] = append(domainMap[s.Domain], s)
			}
			for len(domainOrder) >= batchSize {
				emit()
				if ctx.Err() != nil {
					return
				}
			}
		}
		emit() // flush remainder
	}()

	// Stage 2: CrawlStage
	var crawlErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		eng := &KeepAliveEngine{}
		for batch := range batchCh {
			if ctx.Err() != nil {
				return
			}
			batchStats, runErr := eng.Run(ctx, batch, dns, pcfg.Cfg, pcfg.Results, pcfg.Failures)
			if runErr != nil && ctx.Err() == nil {
				crawlErr = runErr
				return
			}
			if batchStats != nil {
				statsMu.Lock()
				if combined == nil {
					combined = batchStats
				} else {
					combined.OK += batchStats.OK
					combined.Total += batchStats.Total
					combined.Failed += batchStats.Failed
					combined.Bytes += batchStats.Bytes
					if batchStats.PeakRPS > combined.PeakRPS {
						combined.PeakRPS = batchStats.PeakRPS
					}
				}
				statsMu.Unlock()
			}
			if pcfg.RDB != nil {
				if reopenErr := pcfg.RDB.ReopenShards(); reopenErr != nil {
					fmt.Printf("  [warn] ReopenShards: %v\n", reopenErr)
				}
			}
			debug.FreeOSMemory()
		}
	}()

	wg.Wait()
	if batcherErr != nil {
		return combined, batcherErr
	}
	if crawlErr != nil {
		return combined, crawlErr
	}
	if ctx.Err() != nil {
		return combined, ctx.Err()
	}
	return combined, nil
}
