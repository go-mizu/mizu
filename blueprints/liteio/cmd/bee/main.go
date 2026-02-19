package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/liteio-dev/liteio/pkg/storage/driver/zoo/bee"
)

func main() {
	var (
		listen   = flag.String("listen", ":9401", "HTTP listen address")
		dataDir  = flag.String("data-dir", "/tmp/bee-node", "Node data directory")
		syncMode = flag.String("sync", "none", "Sync mode: none|batch|full")
		inlineKB = flag.Int("inline-kb", 64, "Inline cache threshold (KB)")
	)
	flag.Parse()

	node, err := bee.NewHTTPNodeServer(*dataDir, *syncMode, *inlineKB)
	if err != nil {
		log.Fatalf("create bee node server: %v", err)
	}
	defer node.Close()

	srv := &http.Server{
		Addr:              *listen,
		Handler:           node.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       120 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	fmt.Printf("Bee node listening on %s (data-dir=%s, sync=%s, inline-kb=%d)\n", *listen, *dataDir, *syncMode, *inlineKB)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("bee server error: %v", err)
	}
}
