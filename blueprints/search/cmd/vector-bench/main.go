package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/scigolib/hdf5"

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

const phaseTS = "15:04:05"

type datasetSpec struct {
	Name      string
	URL       string
	Distance  string
	Dimension int
	TrainSize int
	TestSize  int
}

type driverSpec struct {
	Name    string
	Addr    string
	Probe   string
	IsTCP   bool
	Compose string
	Options map[string]string
}

type preparedMeta struct {
	Dataset        string `json:"dataset"`
	Distance       string `json:"distance"`
	Dim            int    `json:"dim"`
	TrainCount     int    `json:"train_count"`
	TestCount      int    `json:"test_count"`
	NeighborK      int    `json:"neighbor_k"`
	SourceTrain    int    `json:"source_train"`
	SourceTest     int    `json:"source_test"`
	SourceNeighbor int    `json:"source_neighbor"`
}

type preparedDataset struct {
	Meta     preparedMeta
	Train    []vector.Item
	Queries  []vector.Item
	TruthIDs [][]string
}

type benchResult struct {
	Dataset   string
	Distance  string
	Driver    string
	Status    string
	ComposeUp time.Duration
	Ready     time.Duration
	Index     time.Duration
	SearchP50 time.Duration
	SearchP95 time.Duration
	QPS       float64
	RecallAtK float64
	Error     string
}

func main() {
	var (
		datasetsCSV    string
		driversCSV     string
		reportPath     string
		cacheDir       string
		listDatasets   bool
		download       bool
		forceExtract   bool
		sampleTrain    int
		sampleTest     int
		neighborK      int
		k              int
		batchSize      int
		manageCompose  bool
		composeCmd     string
		composeTimeout time.Duration
		opTimeout      time.Duration
	)

	flag.StringVar(&datasetsCSV, "datasets", "all", "Comma-separated dataset names or 'all'")
	flag.StringVar(&driversCSV, "drivers", strings.Join(defaultDriverNames(), ","), "Comma-separated vector driver names")
	flag.StringVar(&reportPath, "report", "spec/0669_ann_benchmark.md", "Markdown report path")
	flag.StringVar(&cacheDir, "cache-dir", ".cache/vector-bench", "Cache directory for downloaded/extracted datasets")
	flag.BoolVar(&listDatasets, "list-datasets", false, "List all ANN-Benchmarks datasets and exit")
	flag.BoolVar(&download, "download", true, "Auto-download missing HDF5 datasets")
	flag.BoolVar(&forceExtract, "force-extract", false, "Force regenerate prepared subset files")
	flag.IntVar(&sampleTrain, "sample-train", 50000, "Max train vectors to extract per dataset (-1 for full)")
	flag.IntVar(&sampleTest, "sample-test", 500, "Max test vectors to extract per dataset (-1 for full)")
	flag.IntVar(&neighborK, "neighbor-k", 100, "Ground-truth neighbors to extract from HDF5")
	flag.IntVar(&k, "k", 10, "Search top-K used for benchmark metrics")
	flag.IntVar(&batchSize, "batch", 500, "Index batch size")
	flag.BoolVar(&manageCompose, "manage-compose", true, "Start/stop backend containers using compose")
	flag.StringVar(&composeCmd, "compose-cmd", "podman compose", "Compose command")
	flag.DurationVar(&composeTimeout, "compose-timeout", 10*time.Minute, "Compose command timeout")
	flag.DurationVar(&opTimeout, "op-timeout", 20*time.Minute, "Per benchmark operation timeout")
	flag.Parse()

	datasets := allDatasets()
	if listDatasets {
		printDatasetList(datasets)
		return
	}

	selectedDatasets := pickDatasets(datasets, datasetsCSV)
	selectedDrivers := parseCSV(driversCSV)
	dSpecs := defaultDriverSpecs()

	var results []benchResult
	for di, ds := range selectedDatasets {
		logPhase(ds.Name, "dataset-start", "dataset %d/%d (%s, dim=%d, distance=%s)", di+1, len(selectedDatasets), ds.Name, ds.Dimension, ds.Distance)
		prep, err := ensurePreparedDataset(ds, cacheDir, download, forceExtract, sampleTrain, sampleTest, neighborK)
		if err != nil {
			logPhase(ds.Name, "dataset-skip", "prepare failed: %v", err)
			results = append(results, benchResult{Dataset: ds.Name, Distance: ds.Distance, Driver: "ALL", Status: "SKIP", Error: err.Error()})
			continue
		}
		logPhase(ds.Name, "dataset-ready", "prepared train=%d test=%d dim=%d", len(prep.Train), len(prep.Queries), prep.Meta.Dim)

		for dj, drvName := range selectedDrivers {
			spec, ok := dSpecs[drvName]
			if !ok {
				logPhase(ds.Name, "driver-skip", "driver %d/%d unknown: %s", dj+1, len(selectedDrivers), drvName)
				results = append(results, benchResult{Dataset: ds.Name, Distance: ds.Distance, Driver: drvName, Status: "FAIL", Error: "unknown driver"})
				continue
			}
			logPhase(ds.Name+"/"+drvName, "driver-start", "driver %d/%d", dj+1, len(selectedDrivers))
			res := runBenchmark(prep, spec, k, batchSize, manageCompose, composeCmd, composeTimeout, opTimeout)
			results = append(results, res)
			logResult(res)
		}
	}

	logPhase("report", "write-start", "writing markdown report -> %s", reportPath)
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		fatalf("create report dir: %v", err)
	}
	if err := writeReport(reportPath, selectedDatasets, selectedDrivers, results, sampleTrain, sampleTest, neighborK, k, cacheDir); err != nil {
		fatalf("write report: %v", err)
	}
	logPhase("report", "write-done", "report written: %s", reportPath)
}

func runBenchmark(prep preparedDataset, spec driverSpec, k, batchSize int, manageCompose bool, composeCmd string, composeTimeout, opTimeout time.Duration) benchResult {
	scope := prep.Meta.Dataset + "/" + spec.Name
	res := benchResult{Dataset: prep.Meta.Dataset, Distance: prep.Meta.Distance, Driver: spec.Name, Status: "PASS"}
	if manageCompose {
		logPhase(scope, "compose-down", "stopping stack")
		_ = runCompose(composeCmd, composeTimeout, spec.Compose, "down", "-v")
		logPhase(scope, "compose-up", "starting stack")
		t := time.Now()
		if err := runCompose(composeCmd, composeTimeout, spec.Compose, "up", "-d"); err != nil {
			res.Status = "FAIL"
			res.Error = "compose up: " + err.Error()
			return res
		}
		res.ComposeUp = time.Since(t)
		logPhase(scope, "compose-up-done", "done in %s", res.ComposeUp.Round(time.Millisecond))

		logPhase(scope, "ready-wait", "waiting for readiness probe %s", spec.Probe)
		t = time.Now()
		if err := waitReady(spec, 180*time.Second); err != nil {
			res.Status = "FAIL"
			res.Error = "ready: " + err.Error()
			_ = runCompose(composeCmd, composeTimeout, spec.Compose, "down", "-v")
			return res
		}
		res.Ready = time.Since(t)
		logPhase(scope, "ready-done", "ready in %s", res.Ready.Round(time.Millisecond))
	}

	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	logPhase(scope, "store-open", "opening driver connection")
	st, err := openWithRetry(spec, 60, 1*time.Second)
	if err != nil {
		res.Status = "FAIL"
		res.Error = "open: " + err.Error()
		cleanup(spec, manageCompose, composeCmd, composeTimeout)
		return res
	}
	logPhase(scope, "store-open-done", "store connection established")
	if c, ok := st.(vector.Closer); ok {
		defer c.Close()
	}

	coll := st.Collection(safeCollectionName(fmt.Sprintf("vb_%s_%s_%d", prep.Meta.Dataset, spec.Name, time.Now().UnixNano())))

	totalBatches := (len(prep.Train) + batchSize - 1) / batchSize
	logPhase(scope, "index-start", "indexing %d vectors in %d batches", len(prep.Train), totalBatches)
	t := time.Now()
	lastPct := -1
	for i := 0; i < len(prep.Train); i += batchSize {
		j := i + batchSize
		if j > len(prep.Train) {
			j = len(prep.Train)
		}
		batchNo := i/batchSize + 1
		err := retry(20, 500*time.Millisecond, func() error { return coll.Index(ctx, prep.Train[i:j]) })
		if err != nil {
			res.Status = "FAIL"
			res.Error = fmt.Sprintf("index %d-%d: %v", i, j, err)
			cleanup(spec, manageCompose, composeCmd, composeTimeout)
			return res
		}
		if shouldLogProgress(batchNo, totalBatches, &lastPct) {
			logPhase(scope, "index-progress", "batch %d/%d (%d%%)", batchNo, totalBatches, lastPct)
		}
	}
	res.Index = time.Since(t)
	logPhase(scope, "index-done", "done in %s", res.Index.Round(time.Millisecond))

	latencies := make([]time.Duration, 0, len(prep.Queries))
	recallSum := 0.0
	logPhase(scope, "search-start", "searching %d queries (k=%d)", len(prep.Queries), k)
	t = time.Now()
	lastPct = -1
	for qi, q := range prep.Queries {
		start := time.Now()
		var out vector.Results
		err := retry(10, 300*time.Millisecond, func() error {
			var e error
			out, e = coll.Search(ctx, vector.Query{Vector: q.Vector, K: k})
			return e
		})
		latencies = append(latencies, time.Since(start))
		if err != nil {
			res.Status = "FAIL"
			res.Error = fmt.Sprintf("search q%d: %v", qi, err)
			cleanup(spec, manageCompose, composeCmd, composeTimeout)
			return res
		}
		recallSum += recallAtK(spec.Name, prep.TruthIDs[qi], out.Hits, k)
		if shouldLogProgress(qi+1, len(prep.Queries), &lastPct) {
			logPhase(scope, "search-progress", "query %d/%d (%d%%)", qi+1, len(prep.Queries), lastPct)
		}
	}
	searchTotal := time.Since(t)
	if searchTotal > 0 {
		res.QPS = float64(len(prep.Queries)) / searchTotal.Seconds()
	}
	res.SearchP50 = percentile(latencies, 50)
	res.SearchP95 = percentile(latencies, 95)
	res.RecallAtK = recallSum / float64(len(prep.Queries))
	logPhase(scope, "search-done", "done in %s (qps=%.1f recall@%d=%.3f)", searchTotal.Round(time.Millisecond), res.QPS, k, res.RecallAtK)

	cleanup(spec, manageCompose, composeCmd, composeTimeout)
	logPhase(scope, "cleanup", "stack cleaned up")
	return res
}

func openWithRetry(spec driverSpec, tries int, wait time.Duration) (vector.Store, error) {
	var last error
	for i := 0; i < tries; i++ {
		st, err := vector.Open(spec.Name, vector.Config{Addr: spec.Addr, Options: spec.Options})
		if err == nil {
			return st, nil
		}
		last = err
		if !isTransient(err) {
			break
		}
		time.Sleep(wait)
	}
	if last == nil {
		last = errors.New("open failed")
	}
	return nil, last
}

func ensurePreparedDataset(ds datasetSpec, cacheDir string, download, forceExtract bool, sampleTrain, sampleTest, neighborK int) (preparedDataset, error) {
	hdf5Path := filepath.Join(cacheDir, "hdf5", ds.Name+".hdf5")
	prepDir := filepath.Join(cacheDir, "prepared", ds.Name)
	logPhase(ds.Name, "prepare-start", "cache=%s", cacheDir)
	if err := os.MkdirAll(filepath.Dir(hdf5Path), 0o755); err != nil {
		return preparedDataset{}, err
	}
	if _, err := os.Stat(hdf5Path); err != nil {
		if !download {
			return preparedDataset{}, fmt.Errorf("missing dataset file %s (use -download=true)", hdf5Path)
		}
		logPhase(ds.Name, "download-start", "%s -> %s", ds.URL, hdf5Path)
		t := time.Now()
		if err := downloadFile(ds.URL, hdf5Path); err != nil {
			return preparedDataset{}, fmt.Errorf("download %s: %w", ds.Name, err)
		}
		logPhase(ds.Name, "download-done", "completed in %s", time.Since(t).Round(time.Millisecond))
	} else {
		logPhase(ds.Name, "download-skip", "using cached file %s", hdf5Path)
	}
	t := time.Now()
	if err := ensureExtracted(ds, hdf5Path, prepDir, forceExtract, sampleTrain, sampleTest, neighborK); err != nil {
		return preparedDataset{}, err
	}
	logPhase(ds.Name, "extract-done", "prepared data at %s in %s", prepDir, time.Since(t).Round(time.Millisecond))
	logPhase(ds.Name, "prepared-load", "loading prepared binaries")
	p, err := loadPrepared(prepDir)
	if err != nil {
		return preparedDataset{}, err
	}
	logPhase(ds.Name, "prepared-load-done", "train=%d test=%d dim=%d", len(p.Train), len(p.Queries), p.Meta.Dim)
	return p, nil
}

func ensureExtracted(ds datasetSpec, hdf5Path, prepDir string, force bool, sampleTrain, sampleTest, neighborK int) error {
	metaPath := filepath.Join(prepDir, "meta.json")
	if !force {
		if _, err := os.Stat(metaPath); err == nil {
			logPhase(ds.Name, "extract-skip", "meta already exists (%s)", metaPath)
			return nil
		}
	}
	logPhase(ds.Name, "extract-start", "hdf5 -> prepared (force=%v)", force)
	if err := os.MkdirAll(prepDir, 0o755); err != nil {
		return err
	}

	f, err := hdf5.Open(hdf5Path)
	if err != nil {
		return fmt.Errorf("open hdf5 %s: %w", hdf5Path, err)
	}
	defer f.Close()

	var (
		trainDS, testDS, neighborsDS *hdf5.Dataset
	)
	f.Walk(func(path string, obj hdf5.Object) {
		d, ok := obj.(*hdf5.Dataset)
		if !ok {
			return
		}
		switch strings.TrimPrefix(path, "/") {
		case "train":
			trainDS = d
		case "test":
			testDS = d
		case "neighbors":
			neighborsDS = d
		}
	})
	if trainDS == nil || testDS == nil || neighborsDS == nil {
		return fmt.Errorf("extract %s: missing required datasets train/test/neighbors", ds.Name)
	}
	logPhase(ds.Name, "extract-discover", "found datasets train/test/neighbors")

	trainRows, trainDim, err := datasetShape(trainDS)
	if err != nil {
		return fmt.Errorf("extract %s: train shape: %w", ds.Name, err)
	}
	testRows, testDim, err := datasetShape(testDS)
	if err != nil {
		return fmt.Errorf("extract %s: test shape: %w", ds.Name, err)
	}
	neighborRows, sourceNeighbor, err := datasetShape(neighborsDS)
	if err != nil {
		return fmt.Errorf("extract %s: neighbors shape: %w", ds.Name, err)
	}
	if testDim != trainDim {
		return fmt.Errorf("extract %s: train/test dim mismatch (%d vs %d)", ds.Name, trainDim, testDim)
	}
	if neighborRows != testRows {
		return fmt.Errorf("extract %s: neighbors rows %d != test rows %d", ds.Name, neighborRows, testRows)
	}
	if trainDim <= 0 {
		return fmt.Errorf("extract %s: invalid dimension %d", ds.Name, trainDim)
	}
	logPhase(ds.Name, "extract-shape", "source train=%d test=%d dim=%d neighbors=%d", trainRows, testRows, trainDim, sourceNeighbor)

	trainCount := clampSample(trainRows, sampleTrain)
	testLimit := clampSample(testRows, sampleTest)
	nk := sourceNeighbor
	if neighborK >= 0 && neighborK < nk {
		nk = neighborK
	}
	if nk <= 0 {
		return fmt.Errorf("extract %s: invalid neighbor-k %d", ds.Name, nk)
	}
	logPhase(ds.Name, "extract-sample", "sample train=%d/%d test_limit=%d/%d neighbor_k=%d", trainCount, trainRows, testLimit, testRows, nk)

	validRows := make([]int, 0, testLimit)
	lastPct := -1
	for i := 0; i < testRows; i++ {
		row, err := readI32Row(neighborsDS, i, nk)
		if err != nil {
			return fmt.Errorf("extract %s: read neighbors row %d: %w", ds.Name, i, err)
		}
		ok := true
		for _, idx := range row {
			if int(idx) >= trainCount {
				ok = false
				break
			}
		}
		if ok {
			validRows = append(validRows, i)
			if sampleTest >= 0 && len(validRows) >= testLimit {
				break
			}
		}
		if shouldLogProgress(i+1, testRows, &lastPct) {
			logPhase(ds.Name, "extract-filter-progress", "neighbors filtered %d/%d (%d%%), accepted=%d", i+1, testRows, lastPct, len(validRows))
		}
	}
	if len(validRows) == 0 {
		for i := 0; i < testLimit; i++ {
			validRows = append(validRows, i)
		}
	}

	trainPath := filepath.Join(prepDir, "train.f32")
	testPath := filepath.Join(prepDir, "test.f32")
	neighborsPath := filepath.Join(prepDir, "neighbors.i32")

	trainFile, err := os.Create(trainPath)
	if err != nil {
		return err
	}
	defer trainFile.Close()
	testFile, err := os.Create(testPath)
	if err != nil {
		return err
	}
	defer testFile.Close()
	neighborsFile, err := os.Create(neighborsPath)
	if err != nil {
		return err
	}
	defer neighborsFile.Close()

	trainW := bufio.NewWriterSize(trainFile, 4*1024*1024)
	defer trainW.Flush()
	testW := bufio.NewWriterSize(testFile, 4*1024*1024)
	defer testW.Flush()
	neighborsW := bufio.NewWriterSize(neighborsFile, 4*1024*1024)
	defer neighborsW.Flush()

	lastPct = -1
	for i := 0; i < trainCount; i++ {
		row, err := readF32Row(trainDS, i, trainDim)
		if err != nil {
			return fmt.Errorf("extract %s: read train row %d: %w", ds.Name, i, err)
		}
		if err := writeF32Row(trainW, row); err != nil {
			return err
		}
		if shouldLogProgress(i+1, trainCount, &lastPct) {
			logPhase(ds.Name, "extract-train-progress", "wrote train rows %d/%d (%d%%)", i+1, trainCount, lastPct)
		}
	}

	lastPct = -1
	for outPos, rowIdx := range validRows {
		row, err := readF32Row(testDS, rowIdx, testDim)
		if err != nil {
			return fmt.Errorf("extract %s: read test row %d: %w", ds.Name, rowIdx, err)
		}
		if err := writeF32Row(testW, row); err != nil {
			return err
		}
		neighborsRow, err := readI32Row(neighborsDS, rowIdx, nk)
		if err != nil {
			return fmt.Errorf("extract %s: read neighbors row %d: %w", ds.Name, rowIdx, err)
		}
		for i := range neighborsRow {
			if neighborsRow[i] < 0 {
				neighborsRow[i] = 0
			}
			if int(neighborsRow[i]) >= trainCount {
				neighborsRow[i] = int32(trainCount - 1)
			}
		}
		if err := writeI32Row(neighborsW, neighborsRow); err != nil {
			return err
		}
		if shouldLogProgress(outPos+1, len(validRows), &lastPct) {
			logPhase(ds.Name, "extract-test-progress", "wrote test rows %d/%d (%d%%)", outPos+1, len(validRows), lastPct)
		}
	}

	meta := preparedMeta{
		Dataset:        ds.Name,
		Distance:       ds.Distance,
		Dim:            trainDim,
		TrainCount:     trainCount,
		TestCount:      len(validRows),
		NeighborK:      nk,
		SourceTrain:    trainRows,
		SourceTest:     testRows,
		SourceNeighbor: sourceNeighbor,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	if err := os.WriteFile(metaPath, metaJSON, 0o644); err != nil {
		return err
	}
	logPhase(ds.Name, "extract-meta", "meta saved: %s", metaPath)
	return nil
}

func clampSample(total, sample int) int {
	if sample < 0 || sample > total {
		return total
	}
	return sample
}

var shapeRE = regexp.MustCompile(`\[(\d+)\s*x\s*(\d+)\]`)

func datasetShape(ds *hdf5.Dataset) (rows, cols int, err error) {
	info, err := ds.Info()
	if err != nil {
		return 0, 0, err
	}
	m := shapeRE.FindStringSubmatch(info)
	if len(m) != 3 {
		return 0, 0, fmt.Errorf("unable to parse dataset shape from: %s", info)
	}
	rows, err = strconv.Atoi(m[1])
	if err != nil {
		return 0, 0, err
	}
	cols, err = strconv.Atoi(m[2])
	if err != nil {
		return 0, 0, err
	}
	return rows, cols, nil
}

func readF32Row(ds *hdf5.Dataset, row, cols int) ([]float32, error) {
	raw, err := ds.ReadSlice([]uint64{uint64(row), 0}, []uint64{1, uint64(cols)})
	if err != nil {
		return nil, err
	}
	return toFloat32Slice(raw)
}

func readI32Row(ds *hdf5.Dataset, row, cols int) ([]int32, error) {
	raw, err := ds.ReadSlice([]uint64{uint64(row), 0}, []uint64{1, uint64(cols)})
	if err != nil {
		return nil, err
	}
	return toInt32Slice(raw)
}

func toFloat32Slice(v interface{}) ([]float32, error) {
	switch x := v.(type) {
	case []float64:
		out := make([]float32, len(x))
		for i := range x {
			out[i] = float32(x[i])
		}
		return out, nil
	case []float32:
		return x, nil
	case []int:
		out := make([]float32, len(x))
		for i := range x {
			out[i] = float32(x[i])
		}
		return out, nil
	case []int64:
		out := make([]float32, len(x))
		for i := range x {
			out[i] = float32(x[i])
		}
		return out, nil
	case []int32:
		out := make([]float32, len(x))
		for i := range x {
			out[i] = float32(x[i])
		}
		return out, nil
	case []uint64:
		out := make([]float32, len(x))
		for i := range x {
			out[i] = float32(x[i])
		}
		return out, nil
	case []uint32:
		out := make([]float32, len(x))
		for i := range x {
			out[i] = float32(x[i])
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported row type %T", v)
	}
}

func toInt32Slice(v interface{}) ([]int32, error) {
	switch x := v.(type) {
	case []int32:
		return x, nil
	case []int64:
		out := make([]int32, len(x))
		for i := range x {
			out[i] = int32(x[i])
		}
		return out, nil
	case []int:
		out := make([]int32, len(x))
		for i := range x {
			out[i] = int32(x[i])
		}
		return out, nil
	case []float64:
		out := make([]int32, len(x))
		for i := range x {
			out[i] = int32(x[i])
		}
		return out, nil
	case []float32:
		out := make([]int32, len(x))
		for i := range x {
			out[i] = int32(x[i])
		}
		return out, nil
	case []uint64:
		out := make([]int32, len(x))
		for i := range x {
			out[i] = int32(x[i])
		}
		return out, nil
	case []uint32:
		out := make([]int32, len(x))
		for i := range x {
			out[i] = int32(x[i])
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported row type %T", v)
	}
}

func writeF32Row(w io.Writer, row []float32) error {
	buf := make([]byte, len(row)*4)
	for i, v := range row {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	_, err := w.Write(buf)
	return err
}

func writeI32Row(w io.Writer, row []int32) error {
	buf := make([]byte, len(row)*4)
	for i, v := range row {
		binary.LittleEndian.PutUint32(buf[i*4:], uint32(v))
	}
	_, err := w.Write(buf)
	return err
}

func loadPrepared(dir string) (preparedDataset, error) {
	metaPath := filepath.Join(dir, "meta.json")
	trainPath := filepath.Join(dir, "train.f32")
	testPath := filepath.Join(dir, "test.f32")
	neighborsPath := filepath.Join(dir, "neighbors.i32")

	var meta preparedMeta
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return preparedDataset{}, err
	}
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return preparedDataset{}, err
	}

	train, err := readF32Vectors(trainPath, meta.TrainCount, meta.Dim)
	if err != nil {
		return preparedDataset{}, err
	}
	test, err := readF32Vectors(testPath, meta.TestCount, meta.Dim)
	if err != nil {
		return preparedDataset{}, err
	}
	neighbors, err := readI32Matrix(neighborsPath, meta.TestCount, meta.NeighborK)
	if err != nil {
		return preparedDataset{}, err
	}

	trainItems := make([]vector.Item, meta.TrainCount)
	for i := 0; i < meta.TrainCount; i++ {
		trainItems[i] = vector.Item{ID: fmt.Sprintf("id-%09d", i), Vector: train[i]}
	}
	queryItems := make([]vector.Item, meta.TestCount)
	truth := make([][]string, meta.TestCount)
	for i := 0; i < meta.TestCount; i++ {
		queryItems[i] = vector.Item{ID: fmt.Sprintf("q-%09d", i), Vector: test[i]}
		truth[i] = make([]string, meta.NeighborK)
		for j := 0; j < meta.NeighborK; j++ {
			truth[i][j] = fmt.Sprintf("id-%09d", neighbors[i][j])
		}
	}

	return preparedDataset{Meta: meta, Train: trainItems, Queries: queryItems, TruthIDs: truth}, nil
}

func readF32Vectors(path string, rows, dim int) ([][]float32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := make([][]float32, rows)
	buf := make([]byte, dim*4)
	for i := 0; i < rows; i++ {
		if _, err := io.ReadFull(f, buf); err != nil {
			return nil, err
		}
		v := make([]float32, dim)
		for d := 0; d < dim; d++ {
			v[d] = math.Float32frombits(binary.LittleEndian.Uint32(buf[d*4:]))
		}
		out[i] = v
	}
	return out, nil
}

func readI32Matrix(path string, rows, cols int) ([][]int32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := make([][]int32, rows)
	buf := make([]byte, cols*4)
	for i := 0; i < rows; i++ {
		if _, err := io.ReadFull(f, buf); err != nil {
			return nil, err
		}
		row := make([]int32, cols)
		for j := 0; j < cols; j++ {
			row[j] = int32(binary.LittleEndian.Uint32(buf[j*4:]))
		}
		out[i] = row
	}
	return out, nil
}

func downloadFile(url, path string) error {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriterSize(f, 4*1024*1024)
	defer w.Flush()
	buf := make([]byte, 1024*1024)
	var downloaded int64
	lastLog := time.Now()
	total := resp.ContentLength
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := w.Write(buf[:n]); err != nil {
				return err
			}
			downloaded += int64(n)
		}
		if time.Since(lastLog) >= 3*time.Second {
			logDownloadProgress(path, downloaded, total)
			lastLog = time.Now()
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			return rerr
		}
	}
	logDownloadProgress(path, downloaded, total)
	return nil
}

func runCompose(composeCmd string, timeout time.Duration, composeFile string, args ...string) error {
	parts := strings.Fields(composeCmd)
	if len(parts) == 0 {
		return errors.New("empty compose command")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	argv := append(parts[1:], "-f", composeFile)
	argv = append(argv, args...)
	cmd := exec.CommandContext(ctx, parts[0], argv...)
	cmd.Env = append(os.Environ(), "DOCKER_CONFIG=/tmp/podman-docker-config")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", parts[0], strings.Join(argv, " "), err, string(out))
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
		conn, err := net.DialTimeout("tcp", spec.Probe, 1*time.Second)
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

func retry(attempts int, wait time.Duration, fn func() error) error {
	var last error
	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			last = err
			if !isTransient(err) {
				return err
			}
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
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "unexpected eof") ||
		strings.Contains(s, "timeout")
}

func cleanup(spec driverSpec, manageCompose bool, composeCmd string, composeTimeout time.Duration) {
	if manageCompose {
		_ = runCompose(composeCmd, composeTimeout, spec.Compose, "down", "-v")
	}
}

func recallAtK(driver string, truth []string, hits []vector.Hit, k int) float64 {
	truthSet := make(map[string]struct{}, k)
	for i := 0; i < len(truth) && i < k; i++ {
		truthSet[normalizeID(driver, truth[i])] = struct{}{}
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

func normalizeID(driver, id string) string {
	switch driver {
	case "qdrant", "weaviate":
		return uuidSHA1(id)
	case "milvus":
		return milvusIntID(id)
	default:
		return id
	}
}

func milvusIntID(id string) string {
	if n, err := strconv.ParseInt(strings.TrimPrefix(id, "id-"), 10, 64); err == nil {
		return strconv.FormatInt(n, 10)
	}
	if n, err := strconv.ParseInt(id, 10, 64); err == nil {
		return strconv.FormatInt(n, 10)
	}
	h := fnv64a(id)
	return strconv.FormatInt(int64(h&0x7fffffffffffffff), 10)
}

func uuidSHA1(s string) string {
	return uuid.NewSHA1(uuid.Nil, []byte(s)).String()
}

func fnv64a(s string) uint64 {
	const offset64 = 14695981039346656037
	const prime64 = 1099511628211
	h := uint64(offset64)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime64
	}
	return h
}

func percentile(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	cp := append([]time.Duration(nil), values...)
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

func logResult(r benchResult) {
	if r.Status == "PASS" {
		fmt.Printf("[%s/%s] PASS recall=%.3f p50=%s p95=%s qps=%.1f\n", r.Dataset, r.Driver, r.RecallAtK, r.SearchP50.Round(time.Millisecond), r.SearchP95.Round(time.Millisecond), r.QPS)
	} else {
		fmt.Printf("[%s/%s] %s %s\n", r.Dataset, r.Driver, r.Status, r.Error)
	}
}

func logPhase(scope, phase, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("[%s] [%s] [%s] %s\n", time.Now().Format(phaseTS), scope, phase, msg)
}

func shouldLogProgress(done, total int, lastPct *int) bool {
	if total <= 0 {
		return false
	}
	pct := int(float64(done) * 100.0 / float64(total))
	if pct >= *lastPct+10 || done == total {
		*lastPct = pct
		return true
	}
	return false
}

func logDownloadProgress(path string, downloaded, total int64) {
	if total > 0 {
		pct := float64(downloaded) * 100.0 / float64(total)
		logPhase(filepath.Base(path), "download-progress", "%.1f%% (%s/%s)", pct, humanBytes(downloaded), humanBytes(total))
		return
	}
	logPhase(filepath.Base(path), "download-progress", "%s downloaded", humanBytes(downloaded))
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func writeReport(path string, datasets []datasetSpec, drivers []string, results []benchResult, sampleTrain, sampleTest, neighborK, k int, cacheDir string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	now := time.Now().Format(time.RFC3339)
	fmt.Fprintf(f, "# 0669 ANN Benchmark Pipeline and Results\n\n")
	fmt.Fprintf(f, "Date: %s\n\n", now)
	fmt.Fprintf(f, "Reference learned from: [erikbern/ann-benchmarks](https://github.com/erikbern/ann-benchmarks) and dataset table in README.\n\n")
	fmt.Fprintf(f, "## Dataset Coverage\n\n")
	fmt.Fprintf(f, "All ANN-Benchmarks datasets are wired into the pipeline:\n\n")
	fmt.Fprintf(f, "| Dataset | Distance | Dim | Train | Test | URL |\n")
	fmt.Fprintf(f, "|---|---|---:|---:|---:|---|\n")
	for _, ds := range datasets {
		fmt.Fprintf(f, "| %s | %s | %d | %d | %d | %s |\n", ds.Name, ds.Distance, ds.Dimension, ds.TrainSize, ds.TestSize, ds.URL)
	}

	fmt.Fprintf(f, "\n## Pipeline Implemented\n\n")
	fmt.Fprintf(f, "1. Download HDF5 dataset to `%s/hdf5/<dataset>.hdf5`.\n", cacheDir)
	fmt.Fprintf(f, "2. Extract prepared benchmark subset via pure-Go HDF5 reader to `%s/prepared/<dataset>/`:\n", cacheDir)
	fmt.Fprintf(f, "- `train.f32`, `test.f32`, `neighbors.i32`, `meta.json`\n")
	fmt.Fprintf(f, "- query rows are filtered so top-k neighbors are valid in sampled train subset\n")
	fmt.Fprintf(f, "3. For each dataset x driver:\n")
	fmt.Fprintf(f, "- optional compose `down -v`, `up -d`, readiness check\n")
	fmt.Fprintf(f, "- index sampled train vectors\n")
	fmt.Fprintf(f, "- search sampled test vectors\n")
	fmt.Fprintf(f, "- compute recall@%d against provided ANN-Benchmarks ground truth\n", k)
	fmt.Fprintf(f, "- collect p50/p95 latency and QPS\n")

	fmt.Fprintf(f, "\n## Run Settings\n\n")
	fmt.Fprintf(f, "- drivers: `%s`\n", strings.Join(drivers, ","))
	fmt.Fprintf(f, "- sample-train: %d\n", sampleTrain)
	fmt.Fprintf(f, "- sample-test: %d\n", sampleTest)
	fmt.Fprintf(f, "- neighbor-k extracted: %d\n", neighborK)
	fmt.Fprintf(f, "- evaluated k: %d\n", k)

	fmt.Fprintf(f, "\n## Results\n\n")
	fmt.Fprintf(f, "| Dataset | Driver | Status | compose_up_s | ready_s | index_s | p50_ms | p95_ms | qps | recall@%d |\n", k)
	fmt.Fprintf(f, "|---|---|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, r := range results {
		fmt.Fprintf(f, "| %s | %s | %s | %.3f | %.3f | %.3f | %.2f | %.2f | %.2f | %.3f |\n",
			r.Dataset, r.Driver, r.Status,
			r.ComposeUp.Seconds(), r.Ready.Seconds(), r.Index.Seconds(),
			float64(r.SearchP50.Microseconds())/1000.0, float64(r.SearchP95.Microseconds())/1000.0,
			r.QPS, r.RecallAtK,
		)
		if r.Error != "" {
			fmt.Fprintf(f, "\nError (%s/%s): `%s`\n\n", r.Dataset, r.Driver, strings.ReplaceAll(r.Error, "|", "/"))
		}
	}

	fmt.Fprintf(f, "\n## Reproduce\n\n")
	fmt.Fprintf(f, "```bash\ngo run ./cmd/vector-bench \\\n  -datasets all \\\n  -drivers %s \\\n  -download true \\\n  -sample-train %d -sample-test %d -neighbor-k %d -k %d \\\n  -report %s\n```\n", strings.Join(defaultDriverNames(), ","), sampleTrain, sampleTest, neighborK, k, path)
	return nil
}

func printDatasetList(datasets []datasetSpec) {
	for _, d := range datasets {
		fmt.Printf("%s\t%s\t%d\ttrain=%d\ttest=%d\t%s\n", d.Name, d.Distance, d.Dimension, d.TrainSize, d.TestSize, d.URL)
	}
}

func parseCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(strings.ToLower(p))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func pickDatasets(all []datasetSpec, csv string) []datasetSpec {
	if strings.EqualFold(strings.TrimSpace(csv), "all") {
		return all
	}
	want := parseCSV(csv)
	by := make(map[string]datasetSpec, len(all))
	for _, d := range all {
		by[d.Name] = d
	}
	out := make([]datasetSpec, 0, len(want))
	for _, name := range want {
		if d, ok := by[name]; ok {
			out = append(out, d)
		}
	}
	return out
}

func defaultDriverNames() []string {
	return []string{"qdrant", "weaviate", "milvus", "chroma", "elasticsearch", "opensearch", "meilisearch", "typesense", "pgvector", "solr"}
}

func defaultDriverSpecs() map[string]driverSpec {
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

func allDatasets() []datasetSpec {
	return []datasetSpec{
		{Name: "deep-image-96-angular", Dimension: 96, TrainSize: 9990000, TestSize: 10000, Distance: "angular", URL: "https://ann-benchmarks.com/deep-image-96-angular.hdf5"},
		{Name: "fashion-mnist-784-euclidean", Dimension: 784, TrainSize: 60000, TestSize: 10000, Distance: "euclidean", URL: "https://ann-benchmarks.com/fashion-mnist-784-euclidean.hdf5"},
		{Name: "gist-960-euclidean", Dimension: 960, TrainSize: 1000000, TestSize: 1000, Distance: "euclidean", URL: "https://ann-benchmarks.com/gist-960-euclidean.hdf5"},
		{Name: "glove-25-angular", Dimension: 25, TrainSize: 1183514, TestSize: 10000, Distance: "angular", URL: "https://ann-benchmarks.com/glove-25-angular.hdf5"},
		{Name: "glove-50-angular", Dimension: 50, TrainSize: 1183514, TestSize: 10000, Distance: "angular", URL: "https://ann-benchmarks.com/glove-50-angular.hdf5"},
		{Name: "glove-100-angular", Dimension: 100, TrainSize: 1183514, TestSize: 10000, Distance: "angular", URL: "https://ann-benchmarks.com/glove-100-angular.hdf5"},
		{Name: "glove-200-angular", Dimension: 200, TrainSize: 1183514, TestSize: 10000, Distance: "angular", URL: "https://ann-benchmarks.com/glove-200-angular.hdf5"},
		{Name: "kosarak-jaccard", Dimension: 27983, TrainSize: 74962, TestSize: 500, Distance: "jaccard", URL: "https://ann-benchmarks.com/kosarak-jaccard.hdf5"},
		{Name: "mnist-784-euclidean", Dimension: 784, TrainSize: 60000, TestSize: 10000, Distance: "euclidean", URL: "https://ann-benchmarks.com/mnist-784-euclidean.hdf5"},
		{Name: "movielens10m-jaccard", Dimension: 65134, TrainSize: 69363, TestSize: 500, Distance: "jaccard", URL: "https://ann-benchmarks.com/movielens10m-jaccard.hdf5"},
		{Name: "nytimes-256-angular", Dimension: 256, TrainSize: 290000, TestSize: 10000, Distance: "angular", URL: "https://ann-benchmarks.com/nytimes-256-angular.hdf5"},
		{Name: "sift-128-euclidean", Dimension: 128, TrainSize: 1000000, TestSize: 10000, Distance: "euclidean", URL: "https://ann-benchmarks.com/sift-128-euclidean.hdf5"},
		{Name: "lastfm-64-dot", Dimension: 65, TrainSize: 292385, TestSize: 50000, Distance: "angular", URL: "https://ann-benchmarks.com/lastfm-64-dot.hdf5"},
		{Name: "coco-i2i-512-angular", Dimension: 512, TrainSize: 113287, TestSize: 10000, Distance: "angular", URL: "https://github.com/fabiocarrara/str-encoders/releases/download/v0.1.3/coco-i2i-512-angular.hdf5"},
		{Name: "coco-t2i-512-angular", Dimension: 512, TrainSize: 113287, TestSize: 10000, Distance: "angular", URL: "https://github.com/fabiocarrara/str-encoders/releases/download/v0.1.3/coco-t2i-512-angular.hdf5"},
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func safeCollectionName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			b.WriteByte(c)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "vb_collection"
	}
	return out
}
