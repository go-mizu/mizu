package storebench

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/go-mizu/blueprints/table/feature/fields"
	"github.com/go-mizu/blueprints/table/feature/records"
	"github.com/go-mizu/blueprints/table/feature/tables"
)

// Record scenarios

func (r *Runner) runRecordScenarios(backend string, store Store, fixtures *TestFixtures) {
	scenarios := []struct {
		name string
		fn   func(Store, *TestFixtures, *Metrics)
	}{
		{"record_create", r.benchRecordCreate},
		{"record_get_by_id", r.benchRecordGetByID},
		{"record_update", r.benchRecordUpdate},
		{"record_update_cell", r.benchRecordUpdateCell},
		{"record_delete", r.benchRecordDelete},
	}

	for _, s := range scenarios {
		fmt.Printf("  Running %s...", s.name)
		metrics := NewMetrics()
		metrics.Start()
		s.fn(store, fixtures, metrics)
		metrics.End()
		stats := metrics.Stats()
		r.results.Add(Result{
			Scenario:  s.name,
			Backend:   backend,
			Stats:     stats,
			Timestamp: time.Now(),
		})
		fmt.Printf(" done (avg: %v, p99: %v, ops/s: %.2f)\n", stats.Avg, stats.P99, stats.OpsPerSec)
	}
}

func (r *Runner) benchRecordCreate(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()
	for i := 0; i < r.cfg.Iterations; i++ {
		rec := generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, i)
		start := time.Now()
		err := store.Records().Create(ctx, rec)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchRecordGetByID(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()

	// Pre-create records
	var recordIDs []string
	for i := 0; i < r.cfg.Iterations; i++ {
		rec := generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, i+10000)
		if err := store.Records().Create(ctx, rec); err == nil {
			recordIDs = append(recordIDs, rec.ID)
		}
	}

	// Benchmark get
	for _, id := range recordIDs {
		start := time.Now()
		_, err := store.Records().GetByID(ctx, id)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchRecordUpdate(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()

	// Pre-create records
	var recs []*records.Record
	for i := 0; i < r.cfg.Iterations; i++ {
		rec := generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, i+20000)
		if err := store.Records().Create(ctx, rec); err == nil {
			recs = append(recs, rec)
		}
	}

	// Benchmark update
	for i, rec := range recs {
		rec.Cells[fixtures.Fields[0].ID] = fmt.Sprintf("Updated Name %d", i)
		rec.UpdatedAt = time.Now()
		rec.UpdatedBy = fixtures.User.ID

		start := time.Now()
		err := store.Records().Update(ctx, rec)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchRecordUpdateCell(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()

	// Pre-create records
	var recordIDs []string
	for i := 0; i < r.cfg.Iterations; i++ {
		rec := generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, i+30000)
		if err := store.Records().Create(ctx, rec); err == nil {
			recordIDs = append(recordIDs, rec.ID)
		}
	}

	// Benchmark update cell
	fieldID := fixtures.Fields[3].ID // Priority field
	for i, id := range recordIDs {
		start := time.Now()
		err := store.Records().UpdateCell(ctx, id, fieldID, i*10)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchRecordDelete(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()

	// Pre-create records
	var recordIDs []string
	for i := 0; i < r.cfg.Iterations; i++ {
		rec := generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, i+40000)
		if err := store.Records().Create(ctx, rec); err == nil {
			recordIDs = append(recordIDs, rec.ID)
		}
	}

	// Benchmark delete
	for _, id := range recordIDs {
		start := time.Now()
		err := store.Records().Delete(ctx, id)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

// Batch scenarios

func (r *Runner) runBatchScenarios(backend string, store Store, fixtures *TestFixtures) {
	scenarios := []struct {
		name      string
		batchSize int
		iters     int
		fn        func(Store, *TestFixtures, *Metrics, int, int)
	}{
		{"batch_create_10", 10, r.cfg.Iterations, r.benchBatchCreate},
		{"batch_create_100", 100, r.cfg.Iterations / 2, r.benchBatchCreate},
		{"batch_create_500", 500, r.cfg.Iterations / 5, r.benchBatchCreate},
		{"batch_get_by_ids_10", 10, r.cfg.Iterations, r.benchBatchGetByIDs},
		{"batch_get_by_ids_100", 100, r.cfg.Iterations / 2, r.benchBatchGetByIDs},
		{"batch_delete_10", 10, r.cfg.Iterations, r.benchBatchDelete},
		{"batch_delete_100", 100, r.cfg.Iterations / 2, r.benchBatchDelete},
	}

	for _, s := range scenarios {
		fmt.Printf("  Running %s...", s.name)
		metrics := NewMetrics()
		metrics.Start()
		s.fn(store, fixtures, metrics, s.batchSize, s.iters)
		metrics.End()
		stats := metrics.Stats()
		r.results.Add(Result{
			Scenario:  s.name,
			Backend:   backend,
			Stats:     stats,
			Timestamp: time.Now(),
		})
		fmt.Printf(" done (avg: %v, p99: %v, records/s: %.2f)\n", stats.Avg, stats.P99, stats.RecordsPerSec)
	}
}

func (r *Runner) benchBatchCreate(store Store, fixtures *TestFixtures, metrics *Metrics, batchSize, iters int) {
	ctx := context.Background()
	baseOffset := 50000

	for i := 0; i < iters; i++ {
		batch := make([]*records.Record, batchSize)
		for j := 0; j < batchSize; j++ {
			batch[j] = generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, baseOffset+i*batchSize+j)
		}

		start := time.Now()
		err := store.Records().CreateBatch(ctx, batch)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.RecordWithCount(time.Since(start), int64(batchSize))
		}
	}
}

func (r *Runner) benchBatchGetByIDs(store Store, fixtures *TestFixtures, metrics *Metrics, batchSize, iters int) {
	ctx := context.Background()
	baseOffset := 100000

	// Pre-create all records needed
	allIDs := make([][]string, iters)
	for i := 0; i < iters; i++ {
		batch := make([]*records.Record, batchSize)
		ids := make([]string, batchSize)
		for j := 0; j < batchSize; j++ {
			batch[j] = generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, baseOffset+i*batchSize+j)
			ids[j] = batch[j].ID
		}
		store.Records().CreateBatch(ctx, batch)
		allIDs[i] = ids
	}

	// Benchmark batch get
	for _, ids := range allIDs {
		start := time.Now()
		_, err := store.Records().GetByIDs(ctx, ids)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.RecordWithCount(time.Since(start), int64(len(ids)))
		}
	}
}

func (r *Runner) benchBatchDelete(store Store, fixtures *TestFixtures, metrics *Metrics, batchSize, iters int) {
	ctx := context.Background()
	baseOffset := 150000

	// Pre-create all records needed
	allIDs := make([][]string, iters)
	for i := 0; i < iters; i++ {
		batch := make([]*records.Record, batchSize)
		ids := make([]string, batchSize)
		for j := 0; j < batchSize; j++ {
			batch[j] = generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, baseOffset+i*batchSize+j)
			ids[j] = batch[j].ID
		}
		store.Records().CreateBatch(ctx, batch)
		allIDs[i] = ids
	}

	// Benchmark batch delete
	for _, ids := range allIDs {
		start := time.Now()
		err := store.Records().DeleteBatch(ctx, ids)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.RecordWithCount(time.Since(start), int64(len(ids)))
		}
	}
}

// Query scenarios

func (r *Runner) runQueryScenarios(backend string, store Store, fixtures *TestFixtures) {
	ctx := context.Background()

	// Create a dedicated table with lots of records for query tests
	queryTable := &tables.Table{
		ID:        newID(),
		BaseID:    fixtures.Base.ID,
		Name:      "Query Benchmark Table",
		Position:  2,
		CreatedBy: fixtures.User.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Tables().Create(ctx, queryTable)

	// Create fields for query table
	queryFields := make([]*fields.Field, len(fixtures.Fields))
	for i, f := range fixtures.Fields {
		newField := &fields.Field{
			ID:        newID(),
			TableID:   queryTable.ID,
			Name:      f.Name,
			Type:      f.Type,
			Options:   f.Options,
			Position:  f.Position,
			IsPrimary: f.IsPrimary,
			Width:     f.Width,
			CreatedBy: fixtures.User.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		store.Fields().Create(ctx, newField)
		queryFields[i] = newField
	}
	store.Tables().SetPrimaryField(ctx, queryTable.ID, queryFields[0].ID)

	// Seed with records
	recordCount := 1000
	fmt.Printf("  Seeding %d records for query tests...\n", recordCount)
	batchSize := 100
	for i := 0; i < recordCount/batchSize; i++ {
		batch := make([]*records.Record, batchSize)
		for j := 0; j < batchSize; j++ {
			batch[j] = generateRecord(queryTable.ID, queryFields, fixtures.User.ID, 200000+i*batchSize+j)
		}
		store.Records().CreateBatch(ctx, batch)
	}

	scenarios := []struct {
		name string
		fn   func(Store, string, []*fields.Field, *Metrics)
	}{
		{"list_100_records", r.benchList100},
		{"list_500_records", r.benchList500},
		{"list_with_sort", r.benchListWithSort},
		{"list_with_filter", r.benchListWithFilter},
	}

	for _, s := range scenarios {
		fmt.Printf("  Running %s...", s.name)
		metrics := NewMetrics()
		metrics.Start()
		s.fn(store, queryTable.ID, queryFields, metrics)
		metrics.End()
		stats := metrics.Stats()
		r.results.Add(Result{
			Scenario:  s.name,
			Backend:   backend,
			Stats:     stats,
			Timestamp: time.Now(),
		})
		fmt.Printf(" done (avg: %v, p99: %v, ops/s: %.2f)\n", stats.Avg, stats.P99, stats.OpsPerSec)
	}
}

func (r *Runner) benchList100(store Store, tableID string, _ []*fields.Field, metrics *Metrics) {
	ctx := context.Background()
	opts := records.ListOpts{Limit: 100}

	for i := 0; i < r.cfg.Iterations; i++ {
		opts.Offset = (i % 10) * 100 // Vary offset
		start := time.Now()
		_, err := store.Records().List(ctx, tableID, opts)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchList500(store Store, tableID string, _ []*fields.Field, metrics *Metrics) {
	ctx := context.Background()
	opts := records.ListOpts{Limit: 500}

	for i := 0; i < r.cfg.Iterations/2; i++ {
		opts.Offset = (i % 2) * 500
		start := time.Now()
		_, err := store.Records().List(ctx, tableID, opts)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchListWithSort(store Store, tableID string, queryFields []*fields.Field, metrics *Metrics) {
	ctx := context.Background()
	opts := records.ListOpts{
		Limit: 100,
		SortBy: []records.SortSpec{
			{FieldID: queryFields[3].ID, Direction: "desc"}, // Sort by Priority
		},
	}

	for i := 0; i < r.cfg.Iterations; i++ {
		opts.Offset = (i % 10) * 100
		start := time.Now()
		_, err := store.Records().List(ctx, tableID, opts)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchListWithFilter(store Store, tableID string, queryFields []*fields.Field, metrics *Metrics) {
	ctx := context.Background()
	opts := records.ListOpts{
		Limit: 100,
		Filters: []records.Filter{
			{FieldID: queryFields[3].ID, Operator: records.OpGreaterThan, Value: 50},
		},
	}

	for i := 0; i < r.cfg.Iterations; i++ {
		start := time.Now()
		_, err := store.Records().List(ctx, tableID, opts)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

// Field scenarios

func (r *Runner) runFieldScenarios(backend string, store Store, fixtures *TestFixtures) {
	scenarios := []struct {
		name string
		fn   func(Store, *TestFixtures, *Metrics)
	}{
		{"field_create", r.benchFieldCreate},
		{"field_list_by_table", r.benchFieldListByTable},
		{"field_update", r.benchFieldUpdate},
		{"select_choice_add", r.benchSelectChoiceAdd},
		{"select_choice_list", r.benchSelectChoiceList},
	}

	for _, s := range scenarios {
		fmt.Printf("  Running %s...", s.name)
		metrics := NewMetrics()
		metrics.Start()
		s.fn(store, fixtures, metrics)
		metrics.End()
		stats := metrics.Stats()
		r.results.Add(Result{
			Scenario:  s.name,
			Backend:   backend,
			Stats:     stats,
			Timestamp: time.Now(),
		})
		fmt.Printf(" done (avg: %v, p99: %v, ops/s: %.2f)\n", stats.Avg, stats.P99, stats.OpsPerSec)
	}
}

func (r *Runner) benchFieldCreate(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()

	// Create a new table for each batch of fields
	tbl := &tables.Table{
		ID:        newID(),
		BaseID:    fixtures.Base.ID,
		Name:      "Field Benchmark Table",
		Position:  3,
		CreatedBy: fixtures.User.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Tables().Create(ctx, tbl)

	for i := 0; i < r.cfg.Iterations; i++ {
		f := &fields.Field{
			ID:        newID(),
			TableID:   tbl.ID,
			Name:      fmt.Sprintf("Field %d", i),
			Type:      fields.TypeSingleLineText,
			Options:   []byte("{}"),
			Position:  i + 1,
			Width:     200,
			CreatedBy: fixtures.User.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		start := time.Now()
		err := store.Fields().Create(ctx, f)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchFieldListByTable(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()

	for i := 0; i < r.cfg.Iterations; i++ {
		start := time.Now()
		_, err := store.Fields().ListByTable(ctx, fixtures.Table.ID)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchFieldUpdate(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()

	// Create fields to update
	tbl := &tables.Table{
		ID:        newID(),
		BaseID:    fixtures.Base.ID,
		Name:      "Field Update Table",
		Position:  4,
		CreatedBy: fixtures.User.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Tables().Create(ctx, tbl)

	var createdFields []*fields.Field
	for i := 0; i < r.cfg.Iterations; i++ {
		f := &fields.Field{
			ID:        newID(),
			TableID:   tbl.ID,
			Name:      fmt.Sprintf("Update Field %d", i),
			Type:      fields.TypeSingleLineText,
			Options:   []byte("{}"),
			Position:  i + 1,
			Width:     200,
			CreatedBy: fixtures.User.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		store.Fields().Create(ctx, f)
		createdFields = append(createdFields, f)
	}

	for i, f := range createdFields {
		f.Name = fmt.Sprintf("Updated Field %d", i)
		f.UpdatedAt = time.Now()

		start := time.Now()
		err := store.Fields().Update(ctx, f)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchSelectChoiceAdd(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()

	// Create a select field
	tbl := &tables.Table{
		ID:        newID(),
		BaseID:    fixtures.Base.ID,
		Name:      "Select Choice Table",
		Position:  5,
		CreatedBy: fixtures.User.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Tables().Create(ctx, tbl)

	selectField := &fields.Field{
		ID:        newID(),
		TableID:   tbl.ID,
		Name:      "Status",
		Type:      fields.TypeSingleSelect,
		Options:   []byte("{}"),
		Position:  1,
		Width:     200,
		CreatedBy: fixtures.User.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Fields().Create(ctx, selectField)

	colors := []string{"#e53e3e", "#38a169", "#3182ce", "#d69e2e", "#805ad5"}

	for i := 0; i < r.cfg.Iterations; i++ {
		choice := &fields.SelectChoice{
			ID:       newID(),
			FieldID:  selectField.ID,
			Name:     fmt.Sprintf("Choice %d", i),
			Color:    colors[i%len(colors)],
			Position: i + 1,
		}

		start := time.Now()
		err := store.Fields().AddSelectChoice(ctx, choice)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

func (r *Runner) benchSelectChoiceList(store Store, fixtures *TestFixtures, metrics *Metrics) {
	ctx := context.Background()

	// Create a select field with choices
	tbl := &tables.Table{
		ID:        newID(),
		BaseID:    fixtures.Base.ID,
		Name:      "Select List Table",
		Position:  6,
		CreatedBy: fixtures.User.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Tables().Create(ctx, tbl)

	selectField := &fields.Field{
		ID:        newID(),
		TableID:   tbl.ID,
		Name:      "Category",
		Type:      fields.TypeSingleSelect,
		Options:   []byte("{}"),
		Position:  1,
		Width:     200,
		CreatedBy: fixtures.User.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.Fields().Create(ctx, selectField)

	// Add some choices
	for i := 0; i < 20; i++ {
		choice := &fields.SelectChoice{
			ID:       newID(),
			FieldID:  selectField.ID,
			Name:     fmt.Sprintf("Category %d", i),
			Color:    "#3182ce",
			Position: i + 1,
		}
		store.Fields().AddSelectChoice(ctx, choice)
	}

	for i := 0; i < r.cfg.Iterations; i++ {
		start := time.Now()
		_, err := store.Fields().ListSelectChoices(ctx, selectField.ID)
		if err != nil {
			metrics.RecordError()
		} else {
			metrics.Record(time.Since(start))
		}
	}
}

// Concurrent scenarios

func (r *Runner) runConcurrentScenarios(backend string, store Store, fixtures *TestFixtures) {
	concurrencyLevels := []int{10, 25, 50}
	if r.cfg.Concurrency > 50 {
		concurrencyLevels = append(concurrencyLevels, r.cfg.Concurrency)
	}

	for _, conc := range concurrencyLevels {
		scenarios := []struct {
			name string
			fn   func(Store, *TestFixtures, *Metrics, int)
		}{
			{fmt.Sprintf("concurrent_reads_%d", conc), r.benchConcurrentReads},
			{fmt.Sprintf("concurrent_writes_%d", conc), r.benchConcurrentWrites},
			{fmt.Sprintf("concurrent_mixed_%d", conc), r.benchConcurrentMixed},
		}

		for _, s := range scenarios {
			fmt.Printf("  Running %s...", s.name)
			metrics := NewMetrics()
			metrics.Start()
			s.fn(store, fixtures, metrics, conc)
			metrics.End()
			stats := metrics.Stats()
			r.results.Add(Result{
				Scenario:  s.name,
				Backend:   backend,
				Stats:     stats,
				Timestamp: time.Now(),
			})
			fmt.Printf(" done (avg: %v, p99: %v, ops/s: %.2f)\n", stats.Avg, stats.P99, stats.OpsPerSec)
		}
	}
}

func (r *Runner) benchConcurrentReads(store Store, fixtures *TestFixtures, metrics *Metrics, concurrency int) {
	ctx := context.Background()

	// Pre-create records
	var recordIDs []string
	batch := make([]*records.Record, 100)
	for i := 0; i < 100; i++ {
		batch[i] = generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, 300000+i)
		recordIDs = append(recordIDs, batch[i].ID)
	}
	store.Records().CreateBatch(ctx, batch)

	opsPerWorker := r.cfg.Iterations / concurrency
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for w := 0; w < concurrency; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				id := recordIDs[rand.Intn(len(recordIDs))]
				start := time.Now()
				_, err := store.Records().GetByID(ctx, id)
				if err != nil {
					metrics.RecordError()
				} else {
					metrics.Record(time.Since(start))
				}
			}
		}()
	}

	wg.Wait()
}

func (r *Runner) benchConcurrentWrites(store Store, fixtures *TestFixtures, metrics *Metrics, concurrency int) {
	ctx := context.Background()

	opsPerWorker := r.cfg.Iterations / concurrency
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for w := 0; w < concurrency; w++ {
		workerID := w
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				rec := generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, 400000+workerID*10000+i)
				start := time.Now()
				err := store.Records().Create(ctx, rec)
				if err != nil {
					metrics.RecordError()
				} else {
					metrics.Record(time.Since(start))
				}
			}
		}()
	}

	wg.Wait()
}

func (r *Runner) benchConcurrentMixed(store Store, fixtures *TestFixtures, metrics *Metrics, concurrency int) {
	ctx := context.Background()

	// Pre-create records for reads
	var recordIDs []string
	batch := make([]*records.Record, 100)
	for i := 0; i < 100; i++ {
		batch[i] = generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, 500000+i)
		recordIDs = append(recordIDs, batch[i].ID)
	}
	store.Records().CreateBatch(ctx, batch)

	opsPerWorker := r.cfg.Iterations / concurrency
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for w := 0; w < concurrency; w++ {
		workerID := w
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				start := time.Now()
				var err error

				if rand.Float32() < 0.8 { // 80% reads
					id := recordIDs[rand.Intn(len(recordIDs))]
					_, err = store.Records().GetByID(ctx, id)
				} else { // 20% writes
					rec := generateRecord(fixtures.Table.ID, fixtures.Fields, fixtures.User.ID, 600000+workerID*10000+i)
					err = store.Records().Create(ctx, rec)
				}

				if err != nil {
					metrics.RecordError()
				} else {
					metrics.Record(time.Since(start))
				}
			}
		}()
	}

	wg.Wait()
}

// Helper functions

func generateRecord(tableID string, fieldDefs []*fields.Field, userID string, seed int) *records.Record {
	cells := make(map[string]interface{})

	for _, f := range fieldDefs {
		switch f.Type {
		case fields.TypeSingleLineText:
			cells[f.ID] = fmt.Sprintf("Text value %d", seed)
		case fields.TypeLongText:
			cells[f.ID] = fmt.Sprintf("This is a longer text value for record %d with more content to simulate real data.", seed)
		case fields.TypeNumber:
			cells[f.ID] = seed % 100
		case fields.TypeCurrency:
			cells[f.ID] = float64(seed%10000) / 100.0
		case fields.TypeDate:
			cells[f.ID] = time.Now().AddDate(0, 0, seed%365).Format("2006-01-02")
		case fields.TypeEmail:
			cells[f.ID] = fmt.Sprintf("user%d@example.com", seed)
		case fields.TypeURL:
			cells[f.ID] = fmt.Sprintf("https://example.com/page/%d", seed)
		case fields.TypeRating:
			cells[f.ID] = (seed % 5) + 1
		case fields.TypeCheckbox:
			cells[f.ID] = seed%2 == 0
		case fields.TypeSingleSelect, fields.TypeMultiSelect:
			// Skip select fields as they need valid choice IDs
		}
	}

	return &records.Record{
		ID:        newID(),
		TableID:   tableID,
		Cells:     cells,
		Position:  seed,
		CreatedBy: userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
