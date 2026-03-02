package crawl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

)

// SwarmEngine spawns DroneCount drone sub-processes, distributes seeds by domain hash
// for connection-reuse locality, and aggregates final stats from each drone stdout.
//
// Requires cfg.SwarmResultDir and cfg.SearchBinary to be set; otherwise falls back to KeepAliveEngine.
type SwarmEngine struct{}

// droneStats is the JSON a drone writes to stdout every 500ms.
type droneStats struct {
	OK      int64   `json:"ok"`
	Failed  int64   `json:"failed"`
	Timeout int64   `json:"timeout"`
	Total   int64   `json:"total"`
	RPS     float64 `json:"rps"`
}

// dnsFrame is the DNS snapshot sent from queen to each drone via stdin (gob-encoded).
type dnsFrame struct {
	Resolved map[string]string // host → first IP
	Dead     map[string]bool   // host → NXDOMAIN
}

// buildDNSFrame iterates over seeds and snapshots Lookup/IsDead for every unique host.
func buildDNSFrame(seeds []SeedURL, dns DNSCache) dnsFrame {
	resolved := make(map[string]string, 1024)
	dead := make(map[string]bool, 256)
	seen := make(map[string]struct{}, 1024)
	for _, s := range seeds {
		h := s.Host
		if h == "" {
			h = s.Domain
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		if dns.IsDead(h) {
			dead[h] = true
		} else if ip, ok := dns.Lookup(h); ok {
			resolved[h] = ip
		}
	}
	return dnsFrame{Resolved: resolved, Dead: dead}
}

// writeDNSFrame encodes frame as gob prefixed by a 4-byte big-endian length.
func writeDNSFrame(w io.Writer, frame dnsFrame) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(frame); err != nil {
		return err
	}
	b := buf.Bytes()
	if err := binary.Write(w, binary.BigEndian, uint32(len(b))); err != nil {
		return err
	}
	_, err := w.Write(b)
	return err
}

func (e *SwarmEngine) Run(ctx context.Context, seeds []SeedURL,
	dns DNSCache, cfg Config, results ResultWriter, failures FailureWriter) (*Stats, error) {

	if cfg.SearchBinary == "" || cfg.DroneCount <= 1 || cfg.SwarmResultDir == "" {
		return (&KeepAliveEngine{}).Run(ctx, seeds, dns, cfg, results, failures)
	}

	n := cfg.DroneCount
	buckets := make([][]SeedURL, n)
	for _, s := range seeds {
		idx := int(fnvHash(s.Domain) % uint32(n))
		buckets[idx] = append(buckets[idx], s)
	}

	frame := buildDNSFrame(seeds, dns)

	var (
		totalOK      atomic.Int64
		totalFailed  atomic.Int64
		totalTimeout atomic.Int64
		totalReqs    atomic.Int64
	)

	start := time.Now()
	peak := &peakTracker{}
	var wg sync.WaitGroup

	// ProgressFunc relay: report cumulative totals every 500ms while drones run.
	if cfg.ProgressFunc != nil {
		progressCtx, cancelProgress := context.WithCancel(ctx)
		progressDone := make(chan struct{})
		go func() {
			defer close(progressDone)
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-progressCtx.Done():
					return
				case <-ticker.C:
					cfg.ProgressFunc(totalOK.Load(), totalFailed.Load(), totalTimeout.Load())
				}
			}
		}()
		defer func() {
			cancelProgress()
			<-progressDone
		}()
	}

	for i := range n {
		wg.Add(1)
		go func(droneIdx int, droneSeeds []SeedURL) {
			defer wg.Done()
			if len(droneSeeds) == 0 {
				return
			}
			droneResultDir := filepath.Join(cfg.SwarmResultDir, fmt.Sprintf("d%d", droneIdx))
			var droneFailedDB string
			if cfg.SwarmFailedDir != "" {
				droneFailedDB = filepath.Join(cfg.SwarmFailedDir, fmt.Sprintf("failed_%d.duckdb", droneIdx))
			}
			if err := runDroneProcess(ctx, cfg, droneIdx, droneSeeds, frame,
				droneResultDir, droneFailedDB,
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

func runDroneProcess(ctx context.Context, cfg Config, idx int, seeds []SeedURL,
	frame dnsFrame, resultDir, failedDB string,
	ok, failed, timeout, total *atomic.Int64, peak *peakTracker) error {

	args := []string{
		"cc", "recrawl-drone",
		fmt.Sprintf("--drone-id=%d", idx),
		fmt.Sprintf("--workers=%d", cfg.Workers),
		fmt.Sprintf("--timeout=%d", cfg.Timeout.Milliseconds()),
		fmt.Sprintf("--max-conns-per-domain=%d", cfg.MaxConnsPerDomain),
		fmt.Sprintf("--status-only=%v", cfg.StatusOnly),
		fmt.Sprintf("--batch-size=%d", max(cfg.BatchSize, 1000)),
		fmt.Sprintf("--result-dir=%s", resultDir),
		fmt.Sprintf("--failed-db=%s", failedDB),
		fmt.Sprintf("--domain-fail-threshold=%d", cfg.DomainFailThreshold),
		fmt.Sprintf("--domain-timeout=%d", cfg.DomainTimeout.Milliseconds()),
	}

	cmd := exec.CommandContext(ctx, cfg.SearchBinary, args...)
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

	// Write DNS frame then seed JSON lines to drone stdin, then close.
	go func() {
		defer stdin.Close()
		if err := writeDNSFrame(stdin, frame); err != nil {
			fmt.Fprintf(os.Stderr, "[swarm] drone %d frame write error: %v\n", idx, err)
			return
		}
		enc := json.NewEncoder(stdin)
		for _, s := range seeds {
			if err := enc.Encode(s); err != nil {
				break
			}
		}
	}()

	// Drain stdout; accumulate stats as deltas so totals stay live during the run.
	var prevOK, prevFailed, prevTimeout int64
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var ds droneStats
		if json.Unmarshal(scanner.Bytes(), &ds) == nil {
			ok.Add(ds.OK - prevOK)
			failed.Add(ds.Failed - prevFailed)
			timeout.Add(ds.Timeout - prevTimeout)
			total.Add((ds.OK + ds.Failed + ds.Timeout) - (prevOK + prevFailed + prevTimeout))
			prevOK = ds.OK
			prevFailed = ds.Failed
			prevTimeout = ds.Timeout
		}
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "[swarm] drone %d exited with error: %v\n", idx, err)
	}
	return nil
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
