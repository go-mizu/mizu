package crawl

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// HWMonitor continuously samples disk and network I/O throughput.
//
// On Linux, reads /proc/diskstats (disk sectors read/written) and /proc/net/dev
// (interface bytes received/transmitted).  On other platforms, always returns zero.
//
// Stats are stored as float64 bits in atomic uint64 fields, allowing lock-free
// reads from the status display goroutine while the sampler goroutine writes.
type HWMonitor struct {
	diskRdBits uint64 // float64 bits, atomic
	diskWrBits uint64
	netRxBits  uint64
	netTxBits  uint64

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// HWStats is a point-in-time hardware throughput sample.
type HWStats struct {
	DiskReadMBps  float64 // aggregate disk read across all whole-disk devices
	DiskWriteMBps float64 // aggregate disk write
	NetRxMBps     float64 // aggregate RX across all non-loopback interfaces
	NetTxMBps     float64 // aggregate TX
}

// NewHWMonitor creates a monitor that samples hardware I/O every interval.
// The sampling goroutine starts immediately.
func NewHWMonitor(interval time.Duration) *HWMonitor {
	m := &HWMonitor{stopCh: make(chan struct{})}
	m.wg.Add(1)
	go m.run(interval)
	return m
}

// Stats returns the most recent hardware throughput sample. Safe to call
// concurrently from any goroutine.
func (m *HWMonitor) Stats() HWStats {
	return HWStats{
		DiskReadMBps:  math.Float64frombits(atomic.LoadUint64(&m.diskRdBits)),
		DiskWriteMBps: math.Float64frombits(atomic.LoadUint64(&m.diskWrBits)),
		NetRxMBps:     math.Float64frombits(atomic.LoadUint64(&m.netRxBits)),
		NetTxMBps:     math.Float64frombits(atomic.LoadUint64(&m.netTxBits)),
	}
}

// Stop stops the sampling goroutine and waits for it to exit.
func (m *HWMonitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

func (m *HWMonitor) run(interval time.Duration) {
	defer m.wg.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	prev := hwRawRead()
	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			cur := hwRawRead()
			dt := cur.ts.Sub(prev.ts).Seconds()
			if dt > 0 {
				const sectorBytes = 512.0
				rd := float64(cur.diskRead-prev.diskRead) * sectorBytes / dt / (1 << 20)
				wr := float64(cur.diskWrite-prev.diskWrite) * sectorBytes / dt / (1 << 20)
				rx := float64(cur.netRx-prev.netRx) / dt / (1 << 20)
				tx := float64(cur.netTx-prev.netTx) / dt / (1 << 20)
				hwStoreF64(&m.diskRdBits, hwMaxF64(0, rd))
				hwStoreF64(&m.diskWrBits, hwMaxF64(0, wr))
				hwStoreF64(&m.netRxBits, hwMaxF64(0, rx))
				hwStoreF64(&m.netTxBits, hwMaxF64(0, tx))
			}
			prev = cur
		}
	}
}

func hwStoreF64(addr *uint64, v float64) {
	atomic.StoreUint64(addr, math.Float64bits(v))
}

func hwMaxF64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
