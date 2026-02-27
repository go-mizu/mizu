package crawl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/pkg/archived/recrawler"
)

// SwarmEngine spawns N drone sub-processes (one per CPU), distributes seeds by
// domain hash for locality (all URLs of a domain go to the same drone), and
// aggregates stats from each drone's stdout.
// If SearchBinary is empty or DroneCount <= 1, falls back to KeepAliveEngine.
type SwarmEngine struct{}

// droneStats is the JSON a drone writes to stdout every 500ms.
type droneStats struct {
	OK      int64   `json:"ok"`
	Failed  int64   `json:"failed"`
	Timeout int64   `json:"timeout"`
	Total   int64   `json:"total"`
	RPS     float64 `json:"rps"`
}

func (e *SwarmEngine) Run(ctx context.Context, seeds []recrawler.SeedURL,
	dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

	if cfg.SearchBinary == "" || cfg.DroneCount <= 1 {
		return (&KeepAliveEngine{}).Run(ctx, seeds, dns, cfg, results, failures)
	}

	n := cfg.DroneCount
	buckets := make([][]recrawler.SeedURL, n)
	for _, s := range seeds {
		h := fnvHash(s.Domain)
		idx := int(h % uint32(n))
		buckets[idx] = append(buckets[idx], s)
	}

	var (
		totalOK      atomic.Int64
		totalFailed  atomic.Int64
		totalTimeout atomic.Int64
		totalReqs    atomic.Int64
	)

	start := time.Now()
	peak := &peakTracker{}
	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)
		go func(droneIdx int, droneSeeds []recrawler.SeedURL) {
			defer wg.Done()
			if err := runDroneProcess(ctx, cfg.SearchBinary, droneIdx, droneSeeds,
				&totalOK, &totalFailed, &totalTimeout, &totalReqs, peak); err != nil {
				fmt.Fprintf(os.Stderr, "[swarm] drone %d error: %v\n", droneIdx, err)
			}
		}(i, buckets[i])
	}
	wg.Wait()

	dur := time.Since(start)
	tot := totalReqs.Load()
	avgRPS := 0.0
	if dur.Seconds() > 0 {
		avgRPS = float64(tot) / dur.Seconds()
	}
	return &Stats{
		Total:    tot,
		OK:       totalOK.Load(),
		Failed:   totalFailed.Load(),
		Timeout:  totalTimeout.Load(),
		PeakRPS:  peak.Peak(),
		AvgRPS:   avgRPS,
		Duration: dur,
		MemRSS:   rssNow(),
	}, nil
}

func runDroneProcess(ctx context.Context, binary string, idx int, seeds []recrawler.SeedURL,
	ok, failed, timeout, total *atomic.Int64, peak *peakTracker) error {

	if len(seeds) == 0 {
		return nil
	}
	cmd := exec.CommandContext(ctx, binary, "cc", "recrawl-drone",
		fmt.Sprintf("--drone-id=%d", idx))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	// Write seeds as JSON lines to drone stdin, then close
	enc := json.NewEncoder(stdin)
	for _, s := range seeds {
		if err := enc.Encode(s); err != nil {
			break
		}
	}
	stdin.Close()

	// Read droneStats JSON lines from drone stdout
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var ds droneStats
		if err := json.Unmarshal(scanner.Bytes(), &ds); err != nil {
			continue
		}
		ok.Add(ds.OK)
		failed.Add(ds.Failed)
		timeout.Add(ds.Timeout)
		total.Add(ds.Total)
		peak.Record()
	}

	return cmd.Wait()
}

// fnvHash computes FNV-1a hash of s, used for domain→drone assignment.
func fnvHash(s string) uint32 {
	h := uint32(2166136261)
	for i := range len(s) {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
