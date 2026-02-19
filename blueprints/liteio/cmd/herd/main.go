// Command herd starts a single-binary S3-compatible object storage cluster
// with master (routing), volume (storage), and filter (bloom) components.
//
// Usage:
//
//	herd [flags]
//	  -listen :9230       S3 API listen address
//	  -data-dir /tmp/herd Data directory
//	  -stripes 16         Number of storage stripes
//	  -sync none          Sync mode: none|batch|full
//	  -inline-kb 8        Inline threshold (KB)
//	  -prealloc 1024      Preallocate per stripe (MB)
//	  -access-key herd    S3 access key
//	  -secret-key herd123 S3 secret key
//	  -no-auth            Disable authentication
//	  -no-log             Disable request logging
//	  -pprof              Enable pprof endpoints
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/liteio-dev/liteio/pkg/storage/server"

	// Register herd driver.
	_ "github.com/liteio-dev/liteio/pkg/storage/driver/zoo/herd"
)

func main() {
	var (
		listen    = flag.String("listen", ":9230", "S3 API listen address (host:port)")
		dataDir   = flag.String("data-dir", "/tmp/herd-data", "Data directory")
		stripes   = flag.Int("stripes", 16, "Number of storage stripes")
		syncMode  = flag.String("sync", "none", "Sync mode: none|batch|full")
		inlineKB  = flag.Int("inline-kb", 8, "Inline threshold (KB)")
		preallocMB = flag.Int("prealloc", 1024, "Preallocate per stripe (MB)")
		bufSize   = flag.Int("bufsize", 8*1024*1024, "Write buffer size per stripe (bytes)")
		accessKey = flag.String("access-key", "herd", "S3 access key ID")
		secretKey = flag.String("secret-key", "herd123", "S3 secret access key")
		noAuth    = flag.Bool("no-auth", false, "Disable S3 authentication")
		noLog     = flag.Bool("no-log", false, "Disable request logging")
		pprof     = flag.Bool("pprof", true, "Enable pprof endpoints")
	)
	flag.Parse()

	// Build DSN from flags.
	dsn := fmt.Sprintf("herd://%s?stripes=%d&sync=%s&inline_kb=%d&prealloc=%d&bufsize=%d",
		*dataDir, *stripes, *syncMode, *inlineKB, *preallocMB, *bufSize)

	// Parse host/port from listen address.
	host := "0.0.0.0"
	port := 9230
	if _, err := fmt.Sscanf(*listen, ":%d", &port); err != nil {
		// Try host:port format.
		fmt.Sscanf(*listen, "%s:%d", &host, &port)
	}

	cfg := &server.Config{
		Host:           host,
		Port:           port,
		DSN:            dsn,
		AccessKeyID:    *accessKey,
		SecretAccessKey: *secretKey,
		EnablePprof:    *pprof,
	}

	if *noAuth {
		cfg.SkipAuth = true
	}
	if *noLog {
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

	fmt.Printf("Herd S3 server listening on %s (data-dir=%s, stripes=%d, sync=%s, inline-kb=%d)\n",
		*listen, *dataDir, *stripes, *syncMode, *inlineKB)
	fmt.Printf("  DSN: %s\n", dsn)
	fmt.Printf("  Auth: access-key=%s\n", *accessKey)

	if err := srv.Start(); err != nil {
		log.Fatalf("herd: server error: %v", err)
	}
}
