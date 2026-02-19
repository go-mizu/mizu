// Command herd starts a single-binary S3-compatible object storage server
// or a TCP node server for cluster mode.
//
// Usage:
//
//	# S3 server (default, single embedded store):
//	herd [flags]
//	  -listen :9230       S3 API listen address
//	  -data-dir /tmp/herd Data directory
//	  -stripes 16         Number of storage stripes
//	  -sync none          Sync mode: none|batch|full
//	  -inline-kb 8        Inline threshold (KB)
//	  -prealloc 1024      Preallocate per stripe (MB)
//	  -access-key herd    S3 access key
//	  -secret-key herd123 S3 secret key
//
//	# Embedded multi-node S3 server:
//	herd -nodes 3 [flags]
//
//	# TCP node server (cluster mode):
//	herd -node [flags]
//	  -listen :9241       TCP listen address
//
//	# TCP node server with gossip membership:
//	herd -node -seeds 127.0.0.1:7241 -gossip-port 7241 [flags]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd"
	"github.com/liteio-dev/liteio/pkg/storage/server"
)

func main() {
	var (
		listen     = flag.String("listen", "", "Listen address (default: :9230 for S3, :9241 for node)")
		dataDir    = flag.String("data-dir", "/tmp/herd-data", "Data directory")
		stripes    = flag.Int("stripes", 16, "Number of storage stripes")
		syncMode   = flag.String("sync", "none", "Sync mode: none|batch|full")
		inlineKB   = flag.Int("inline-kb", 8, "Inline threshold (KB)")
		preallocMB = flag.Int("prealloc", 1024, "Preallocate per stripe (MB)")
		bufSize    = flag.Int("bufsize", 8*1024*1024, "Write buffer size per stripe (bytes)")
		accessKey  = flag.String("access-key", "herd", "S3 access key ID")
		secretKey  = flag.String("secret-key", "herd123", "S3 secret access key")
		noAuth     = flag.Bool("no-auth", false, "Disable S3 authentication")
		noLog      = flag.Bool("no-log", false, "Disable request logging")
		pprof      = flag.Bool("pprof", true, "Enable pprof endpoints")
		nodeMode   = flag.Bool("node", false, "Run as TCP node server (binary protocol)")
		nodes      = flag.Int("nodes", 0, "Embedded multi-node count (0 = single store)")
		seeds      = flag.String("seeds", "", "Gossip seed addresses (comma-separated)")
		gossipPort = flag.Int("gossip-port", 7241, "Gossip bind port")
	)
	flag.Parse()

	if *nodeMode {
		runNodeServer(*listen, *dataDir, *stripes, *syncMode, *inlineKB, *preallocMB, *bufSize,
			*seeds, *gossipPort)
		return
	}

	runS3Server(*listen, *dataDir, *stripes, *syncMode, *inlineKB, *preallocMB, *bufSize,
		*accessKey, *secretKey, *noAuth, *noLog, *pprof, *nodes)
}

func runNodeServer(listen, dataDir string, numStripes int, syncMode string, inlineKB, preallocMB, bufSize int,
	seedsStr string, gossipPort int) {
	if listen == "" {
		listen = ":9241"
	}

	q := url.Values{}
	q.Set("stripes", strconv.Itoa(numStripes))
	q.Set("sync", syncMode)
	q.Set("inline_kb", strconv.Itoa(inlineKB))
	q.Set("prealloc", strconv.Itoa(preallocMB))
	q.Set("bufsize", strconv.Itoa(bufSize))

	dsn := fmt.Sprintf("herd:///%s?%s", dataDir, q.Encode())

	d := &herd.Driver{}
	st, err := d.Open(context.Background(), dsn)
	if err != nil {
		log.Fatalf("herd: open store: %v", err)
	}

	engine, ok := st.(herd.StoreEngine)
	if !ok {
		log.Fatal("herd: store does not implement StoreEngine")
	}

	srv := herd.NewNodeServerFromEngine(engine)

	// Start gossip membership if seeds provided.
	var membership *herd.Membership
	if seedsStr != "" {
		seeds := strings.Split(seedsStr, ",")
		membership, err = herd.NewMembership(herd.GossipConfig{
			BindPort: gossipPort,
			DataAddr: listen,
			Seeds:    seeds,
		})
		if err != nil {
			log.Fatalf("herd: gossip: %v", err)
		}
		log.Printf("herd gossip on port %d, seeds=%v", gossipPort, seeds)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("shutting down...")
		if membership != nil {
			membership.Leave(5 * 1e9) // 5 seconds
			membership.Shutdown()
		}
		srv.Close()
		st.Close()
		os.Exit(0)
	}()

	log.Printf("herd node listening on %s (data=%s, stripes=%d, sync=%s, inline=%dKB)",
		listen, dataDir, numStripes, syncMode, inlineKB)

	if err := srv.ListenAndServe(listen); err != nil {
		log.Fatalf("herd: serve: %v", err)
	}
}

func runS3Server(listen, dataDir string, numStripes int, syncMode string, inlineKB, preallocMB, bufSize int,
	accessKey, secretKey string, noAuth, noLog, enablePprof bool, numNodes int) {
	if listen == "" {
		listen = ":9230"
	}

	var dsn string
	if numNodes > 0 {
		// Embedded multi-node mode.
		dsn = fmt.Sprintf("herd://%s?nodes=%d&stripes=%d&sync=%s&inline_kb=%d&prealloc=%d&bufsize=%d",
			dataDir, numNodes, numStripes, syncMode, inlineKB, preallocMB, bufSize)
	} else {
		dsn = fmt.Sprintf("herd://%s?stripes=%d&sync=%s&inline_kb=%d&prealloc=%d&bufsize=%d",
			dataDir, numStripes, syncMode, inlineKB, preallocMB, bufSize)
	}

	// Parse host/port from listen address.
	host := "0.0.0.0"
	port := 9230
	if _, err := fmt.Sscanf(listen, ":%d", &port); err != nil {
		// Try host:port format.
		fmt.Sscanf(listen, "%s:%d", &host, &port)
	}

	cfg := &server.Config{
		Host:            host,
		Port:            port,
		DSN:             dsn,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		EnablePprof:     enablePprof,
	}

	if noAuth {
		cfg.SkipAuth = true
	}
	if noLog {
		cfg.NoLog = true
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("herd: create server: %v", err)
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		fmt.Printf("\nReceived %v, shutting down...\n", sig)
		srv.Stop()
	}()

	if numNodes > 0 {
		fmt.Printf("Herd S3 server listening on %s (nodes=%d, data-dir=%s, stripes=%d, sync=%s, inline-kb=%d)\n",
			listen, numNodes, dataDir, numStripes, syncMode, inlineKB)
	} else {
		fmt.Printf("Herd S3 server listening on %s (data-dir=%s, stripes=%d, sync=%s, inline-kb=%d)\n",
			listen, dataDir, numStripes, syncMode, inlineKB)
	}
	fmt.Printf("  DSN: %s\n", dsn)
	fmt.Printf("  Auth: access-key=%s\n", accessKey)

	if err := srv.Start(); err != nil {
		log.Fatalf("herd: server error: %v", err)
	}
}
