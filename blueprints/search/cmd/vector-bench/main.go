package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/go-mizu/mizu/blueprints/search/pkg/vector"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/chroma"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/elasticsearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/meilisearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/milvus"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/opensearch"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/pgvector"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/qdrant"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/solr"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/typesense"
	_ "github.com/go-mizu/mizu/blueprints/search/pkg/vector/driver/weaviate"
)

type driverSpec struct {
	Name    string
	Addr    string
	Probe   string
	IsTCP   bool
	Compose string
	Options map[string]string
}

type benchResult struct {
	Name        string
	ComposeUp   time.Duration
	Ready       time.Duration
	Init        time.Duration
	Index       time.Duration
	SearchP50   time.Duration
	SearchP95   time.Duration
	SearchTotal time.Duration
	QPS         float64
	RecallAtK   float64
	Status      string
	Error       string
}

func main() {
	var (
		driversCSV string
		reportPath string
		k          int
		dim        int
		corpusN    int
		queriesN   int
		batchSize  int
		seed       int64

		manageCompose  bool
		composeCmd     string
		composeTimeout time.Duration
		opTimeout      time.Duration
	)

	flag.StringVar(&driversCSV, "drivers", strings.Join(defaultDriverNames(), ","), "Comma-separated driver names")
	flag.StringVar(&reportPath, "report", "spec/0668_benchmark_vector.md", "Markdown report output path")
	flag.IntVar(&k, "k", 10, "Top-K for search")
	flag.IntVar(&dim, "dim", 64, "Vector dimension")
	flag.IntVar(&corpusN, "corpus", 1000, "Number of indexed vectors")
	flag.IntVar(&queriesN, "queries", 50, "Number of query vectors")
	flag.IntVar(&batchSize, "batch", 200, "Index batch size")
	flag.Int64Var(&seed, "seed", 42, "Random seed")
	flag.BoolVar(&manageCompose, "manage-compose", true, "Start/stop backend containers with compose")
	flag.StringVar(&composeCmd, "compose-cmd", "podman compose", "Compose command")
	flag.DurationVar(&composeTimeout, "compose-timeout", 5*time.Minute, "Compose command timeout")
	flag.DurationVar(&opTimeout, "op-timeout", 180*time.Second, "Per-driver benchmark timeout")
	flag.Parse()

	if k <= 0 || dim <= 0 || corpusN <= 0 || queriesN <= 0 || batchSize <= 0 {
		fatalf("invalid numeric flags: all must be > 0")
	}

	specs := defaultSpecs()
	drivers := parseDrivers(driversCSV)
	corpus, queries := makeDataset(seed, corpusN, queriesN, dim)
	truth := groundTruthTopK(corpus, queries, k)

	results := make([]benchResult, 0, len(drivers))
	for _, name := range drivers {
		spec, ok := specs[name]
		if !ok {
			results = append(results, benchResult{Name: name, Status: "FAIL", Error: "unknown driver"})
			continue
		}
		res := benchDriver(spec, corpus, queries, truth, k, batchSize, manageCompose, composeCmd, composeTimeout, opTimeout)
		results = append(results, res)
		if res.Status == "PASS" {
			fmt.Printf("[%s] PASS recall@%d=%.3f p50=%s p95=%s qps=%.1f\n", res.Name, k, res.RecallAtK, res.SearchP50.Round(time.Millisecond), res.SearchP95.Round(time.Millisecond), res.QPS)
		} else {
			fmt.Printf("[%s] FAIL %s\n", res.Name, res.Error)
		}
	}

	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		fatalf("create report dir: %v", err)
	}
	if err := writeReport(reportPath, results, dim, corpusN, queriesN, k, seed); err != nil {
		fatalf("write report: %v", err)
	}
	fmt.Printf("report written: %s\n", reportPath)
}

func benchDriver(spec driverSpec, corpus, queries []vector.Item, truth [][]string, k, batchSize int, manageCompose bool, composeCmd string, composeTimeout, opTimeout time.Duration) benchResult {
	res := benchResult{Name: spec.Name, Status: "PASS"}
	if manageCompose {
		_ = runCompose(composeCmd, composeTimeout, spec.Compose, "down", "-v")
		t := time.Now()
		if err := runCompose(composeCmd, composeTimeout, spec.Compose, "up", "-d"); err != nil {
			res.Status = "FAIL"
			res.Error = "compose up: " + err.Error()
			return res
		}
		res.ComposeUp = time.Since(t)

		t = time.Now()
		if err := waitReady(spec, 120*time.Second); err != nil {
			res.Status = "FAIL"
			res.Error = "ready: " + err.Error()
			_ = runCompose(composeCmd, composeTimeout, spec.Compose, "down", "-v")
			return res
		}
		res.Ready = time.Since(t)
	}

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	st, err := vector.Open(spec.Name, vector.Config{Addr: spec.Addr, Options: spec.Options})
	if err != nil {
		lastErr := err
		for i := 0; i < 30; i++ {
			if !isTransient(lastErr) {
				break
			}
			time.Sleep(1 * time.Second)
			st, err = vector.Open(spec.Name, vector.Config{Addr: spec.Addr, Options: spec.Options})
			if err == nil {
				lastErr = nil
				break
			}
			lastErr = err
		}
		if lastErr != nil {
			res.Status = "FAIL"
			res.Error = "open: " + lastErr.Error()
			cleanup(spec, manageCompose, composeCmd, composeTimeout)
			return res
		}
	}
	if st == nil {
		res.Status = "FAIL"
		res.Error = "open: nil store"
		cleanup(spec, manageCompose, composeCmd, composeTimeout)
		return res
	}
	if c, ok := st.(vector.Closer); ok {
		defer c.Close()
	}

	coll := st.Collection(fmt.Sprintf("vb_%s_%d", spec.Name, time.Now().UnixNano()))

	t := time.Now()
	for i := 0; i < len(corpus); i += batchSize {
		j := i + batchSize
		if j > len(corpus) {
			j = len(corpus)
		}
		err := retry(15, 1*time.Second, func() error { return coll.Index(ctx, corpus[i:j]) })
		if err != nil {
			res.Status = "FAIL"
			res.Error = fmt.Sprintf("index %d-%d: %v", i, j, err)
			cleanup(spec, manageCompose, composeCmd, composeTimeout)
			return res
		}
	}
	res.Index = time.Since(t)

	lat := make([]time.Duration, 0, len(queries))
	recallSum := 0.0
	t = time.Now()
	for qi, q := range queries {
		start := time.Now()
		var out vector.Results
		err := retry(8, 500*time.Millisecond, func() error {
			var e error
			out, e = coll.Search(ctx, vector.Query{Vector: q.Vector, K: k})
			return e
		})
		lat = append(lat, time.Since(start))
		if err != nil {
			res.Status = "FAIL"
			res.Error = fmt.Sprintf("search q%d: %v", qi, err)
			cleanup(spec, manageCompose, composeCmd, composeTimeout)
			return res
		}
		recallSum += recallAtK(spec.Name, truth[qi], out.Hits, k)
	}
	res.SearchTotal = time.Since(t)
	res.SearchP50 = percentileDuration(lat, 50)
	res.SearchP95 = percentileDuration(lat, 95)
	res.RecallAtK = recallSum / float64(len(queries))
	if res.SearchTotal > 0 {
		res.QPS = float64(len(queries)) / res.SearchTotal.Seconds()
	}

	cleanup(spec, manageCompose, composeCmd, composeTimeout)
	return res
}

func retry(attempts int, wait time.Duration, fn func() error) error {
	var last error
	for i := 0; i < attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		last = err
		if !isTransient(err) {
			return err
		}
		time.Sleep(wait)
	}
	return last
}

func isTransient(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "starting up") ||
		strings.Contains(s, "solrcore is loading") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "unexpected eof") ||
		strings.Contains(s, "failed to receive message")
}

func cleanup(spec driverSpec, manageCompose bool, composeCmd string, composeTimeout time.Duration) {
	if manageCompose {
		_ = runCompose(composeCmd, composeTimeout, spec.Compose, "down", "-v")
	}
}

func runCompose(composeCmd string, timeout time.Duration, composeFile string, args ...string) error {
	parts := strings.Fields(composeCmd)
	if len(parts) == 0 {
		return errors.New("empty compose command")
	}
	cmdArgs := append(parts[1:], "-f", composeFile)
	cmdArgs = append(cmdArgs, args...)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, parts[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "DOCKER_CONFIG=/tmp/podman-docker-config")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w\n%s", strings.Join(append([]string{parts[0]}, cmdArgs...), " "), err, string(out))
	}
	return nil
}

func waitReady(spec driverSpec, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isReady(spec) {
			return nil
		}
		time.Sleep(400 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", spec.Name)
}

func isReady(spec driverSpec) bool {
	if spec.IsTCP {
		conn, err := net.DialTimeout("tcp", spec.Probe, 900*time.Millisecond)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, spec.Probe, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}

func recallAtK(driver string, truth []string, hits []vector.Hit, k int) float64 {
	if k <= 0 {
		return 0
	}
	truthSet := make(map[string]struct{}, len(truth))
	for i := 0; i < len(truth) && i < k; i++ {
		truthSet[normalizeIDForDriver(driver, truth[i])] = struct{}{}
	}
	if len(truthSet) == 0 {
		return 0
	}
	found := 0
	for i := 0; i < len(hits) && i < k; i++ {
		if _, ok := truthSet[hits[i].ID]; ok {
			found++
		}
	}
	return float64(found) / float64(len(truthSet))
}

func normalizeIDForDriver(driver, id string) string {
	switch driver {
	case "qdrant", "weaviate":
		return uuid.NewSHA1(uuid.Nil, []byte(id)).String()
	case "milvus":
		if n, err := strconv.ParseInt(id, 10, 64); err == nil {
			return strconv.FormatInt(n, 10)
		}
		h := fnv.New64a()
		_, _ = h.Write([]byte(id))
		return strconv.FormatInt(int64(h.Sum64()&0x7fffffffffffffff), 10)
	default:
		return id
	}
}

func percentileDuration(vals []time.Duration, p float64) time.Duration {
	if len(vals) == 0 {
		return 0
	}
	cp := append([]time.Duration(nil), vals...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(math.Ceil((p/100.0)*float64(len(cp)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func makeDataset(seed int64, corpusN, queriesN, dim int) ([]vector.Item, []vector.Item) {
	rng := rand.New(rand.NewSource(seed))
	corpus := make([]vector.Item, corpusN)
	for i := 0; i < corpusN; i++ {
		v := make([]float32, dim)
		for d := range v {
			v[d] = float32(rng.NormFloat64())
		}
		normalize(v)
		corpus[i] = vector.Item{ID: fmt.Sprintf("id-%06d", i), Vector: v, Metadata: map[string]string{"bucket": fmt.Sprintf("b%d", i%8)}}
	}
	queries := make([]vector.Item, queriesN)
	for i := 0; i < queriesN; i++ {
		base := corpus[rng.Intn(corpusN)].Vector
		q := make([]float32, dim)
		for d := range q {
			q[d] = base[d] + 0.03*float32(rng.NormFloat64())
		}
		normalize(q)
		queries[i] = vector.Item{ID: fmt.Sprintf("q-%06d", i), Vector: q}
	}
	return corpus, queries
}

func groundTruthTopK(corpus, queries []vector.Item, k int) [][]string {
	type pair struct {
		id    string
		score float64
	}
	out := make([][]string, len(queries))
	for qi, q := range queries {
		arr := make([]pair, len(corpus))
		for i, c := range corpus {
			arr[i] = pair{id: c.ID, score: cosine(q.Vector, c.Vector)}
		}
		sort.Slice(arr, func(i, j int) bool { return arr[i].score > arr[j].score })
		n := k
		if n > len(arr) {
			n = len(arr)
		}
		ids := make([]string, n)
		for i := 0; i < n; i++ {
			ids[i] = arr[i].id
		}
		out[qi] = ids
	}
	return out
}

func cosine(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	dot := 0.0
	na := 0.0
	nb := 0.0
	for i := range a {
		aa := float64(a[i])
		bb := float64(b[i])
		dot += aa * bb
		na += aa * aa
		nb += bb * bb
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func normalize(v []float32) {
	norm := 0.0
	for _, x := range v {
		norm += float64(x * x)
	}
	if norm == 0 {
		return
	}
	inv := float32(1.0 / math.Sqrt(norm))
	for i := range v {
		v[i] *= inv
	}
}

func parseDrivers(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(strings.ToLower(p))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func defaultDriverNames() []string {
	return []string{"qdrant", "weaviate", "milvus", "chroma", "elasticsearch", "opensearch", "meilisearch", "typesense", "pgvector", "solr"}
}

func defaultSpecs() map[string]driverSpec {
	return map[string]driverSpec{
		"qdrant":        {Name: "qdrant", Addr: "http://localhost:6333", Probe: "http://localhost:6333/healthz", Compose: "docker/qdrant/docker-compose.yaml"},
		"weaviate":      {Name: "weaviate", Addr: "http://localhost:8080", Probe: "http://localhost:8080/v1/.well-known/ready", Compose: "docker/weaviate/docker-compose.yaml"},
		"milvus":        {Name: "milvus", Addr: "http://localhost:19530", Probe: "http://localhost:9091/healthz", Compose: "docker/milvus/docker-compose.yaml", Options: map[string]string{"token": "root:Milvus"}},
		"chroma":        {Name: "chroma", Addr: "http://localhost:8000", Probe: "http://localhost:8000/api/v2/heartbeat", Compose: "docker/chroma/docker-compose.yaml"},
		"elasticsearch": {Name: "elasticsearch", Addr: "http://localhost:9201", Probe: "http://localhost:9201", Compose: "docker/elasticsearch/docker-compose.yaml"},
		"opensearch":    {Name: "opensearch", Addr: "http://localhost:9200", Probe: "http://localhost:9200", Compose: "docker/opensearch/docker-compose.yaml"},
		"meilisearch":   {Name: "meilisearch", Addr: "http://localhost:7700", Probe: "http://localhost:7700/health", Compose: "docker/meilisearch/docker-compose.yaml"},
		"typesense":     {Name: "typesense", Addr: "http://localhost:8108", Probe: "http://localhost:8108/health", Compose: "docker/typesense/docker-compose.yaml", Options: map[string]string{"api_key": "mizu-typesense-key"}},
		"pgvector":      {Name: "pgvector", Addr: "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable", Probe: "localhost:5432", IsTCP: true, Compose: "docker/pgvector/docker-compose.yaml"},
		"solr":          {Name: "solr", Addr: "http://localhost:8983", Probe: "http://localhost:8983/solr/admin/info/system?wt=json", Compose: "docker/solr/docker-compose.yaml"},
	}
}

func writeReport(path string, results []benchResult, dim, corpusN, queriesN, k int, seed int64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	now := time.Now().Format("2006-01-02 15:04:05 MST")
	fmt.Fprintf(f, "# 0668 Benchmark Vector\n\n")
	fmt.Fprintf(f, "Date: %s\n\n", now)

	fmt.Fprintf(f, "## Best Open Source Benchmark\n\n")
	fmt.Fprintf(f, "Best baseline: **ANN-Benchmarks** ([GitHub](https://github.com/erikbern/ann-benchmarks), [Website](https://ann-benchmarks.com/)).\n\n")
	fmt.Fprintf(f, "Reason:\n")
	fmt.Fprintf(f, "- Widely accepted ANN benchmark baseline in the community.\n")
	fmt.Fprintf(f, "- Emphasizes recall/latency/QPS trade-offs.\n")
	fmt.Fprintf(f, "- Vendor-neutral and reproducible.\n\n")
	fmt.Fprintf(f, "Alternatives considered:\n")
	fmt.Fprintf(f, "- VectorDBBench ([GitHub](https://github.com/zilliztech/VectorDBBench)) for DB operational scenarios.\n")
	fmt.Fprintf(f, "- Big ANN Benchmarks ([GitHub](https://github.com/harsha-simhadri/big-ann-benchmarks)) for billion-scale tracks.\n\n")

	fmt.Fprintf(f, "## Harness\n\n")
	fmt.Fprintf(f, "Implemented command: `cmd/vector-bench`.\n\n")
	fmt.Fprintf(f, "Workload parameters:\n")
	fmt.Fprintf(f, "- corpus=%d\n", corpusN)
	fmt.Fprintf(f, "- queries=%d\n", queriesN)
	fmt.Fprintf(f, "- dim=%d\n", dim)
	fmt.Fprintf(f, "- k=%d\n", k)
	fmt.Fprintf(f, "- seed=%d\n\n", seed)

	fmt.Fprintf(f, "## Results\n\n")
	fmt.Fprintf(f, "| Driver | Status | compose_up_s | ready_s | init_ms | index_ms | search_p50_ms | search_p95_ms | qps | recall@%d |\n", k)
	fmt.Fprintf(f, "|---|---|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range results {
		fmt.Fprintf(f, "| %s | %s | %.3f | %.3f | %.1f | %.1f | %.1f | %.1f | %.1f | %.3f |\n",
			r.Name, r.Status,
			r.ComposeUp.Seconds(), r.Ready.Seconds(),
			float64(r.Init.Microseconds())/1000.0,
			float64(r.Index.Microseconds())/1000.0,
			float64(r.SearchP50.Microseconds())/1000.0,
			float64(r.SearchP95.Microseconds())/1000.0,
			r.QPS, r.RecallAtK,
		)
		if r.Error != "" {
			fmt.Fprintf(f, "\nError (%s): `%s`\n\n", r.Name, strings.ReplaceAll(r.Error, "|", "/"))
		}
	}

	fmt.Fprintf(f, "## Reproduce\n\n")
	fmt.Fprintf(f, "```bash\ngo run ./cmd/vector-bench \\\n  -manage-compose=true \\\n  -drivers %s \\\n  -corpus %d -queries %d -dim %d -k %d -seed %d \\\n  -report %s\n```\n", strings.Join(defaultDriverNames(), ","), corpusN, queriesN, dim, k, seed, path)
	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
