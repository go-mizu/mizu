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
	"syscall"

	"github.com/liteio-dev/liteio/pkg/storage/driver/zoo/zebra"
)

func main() {
	listen := flag.String("listen", ":9601", "bind address")
	dataDir := flag.String("data-dir", "/tmp/zebra-node", "data directory")
	stripes := flag.Int("stripes", 8, "number of stripes")
	syncMode := flag.String("sync", "none", "sync mode: none|batch|full")
	inlineKB := flag.Int("inline-kb", 4, "inline threshold in KB")
	preallocMB := flag.Int("prealloc", 1024, "preallocate per stripe in MB")
	flag.Parse()

	q := url.Values{}
	q.Set("stripes", strconv.Itoa(*stripes))
	q.Set("sync", *syncMode)
	q.Set("inline_kb", strconv.Itoa(*inlineKB))
	q.Set("prealloc", strconv.Itoa(*preallocMB))

	dsn := fmt.Sprintf("zebra:///%s?%s", *dataDir, q.Encode())

	d := &zebra.Driver{}
	st, err := d.Open(context.Background(), dsn)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	// Type assert to get the underlying store for the node server.
	engine, ok := st.(zebra.StoreEngine)
	if !ok {
		log.Fatal("store does not implement StoreEngine")
	}

	srv := zebra.NewNodeServerFromEngine(engine)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("shutting down...")
		srv.Close()
		st.Close()
		os.Exit(0)
	}()

	log.Printf("zebra node listening on %s (data=%s, stripes=%d, sync=%s, inline=%dKB)",
		*listen, *dataDir, *stripes, *syncMode, *inlineKB)

	if err := srv.ListenAndServe(*listen); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
