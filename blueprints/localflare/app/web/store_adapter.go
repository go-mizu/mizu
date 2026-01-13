package web

import (
	"context"

	"github.com/go-mizu/blueprints/localflare/feature/ai"
	"github.com/go-mizu/blueprints/localflare/feature/ai_gateway"
	"github.com/go-mizu/blueprints/localflare/feature/analytics_engine"
	"github.com/go-mizu/blueprints/localflare/feature/auth"
	"github.com/go-mizu/blueprints/localflare/feature/cron"
	"github.com/go-mizu/blueprints/localflare/feature/d1"
	do "github.com/go-mizu/blueprints/localflare/feature/durable_objects"
	"github.com/go-mizu/blueprints/localflare/feature/hyperdrive"
	"github.com/go-mizu/blueprints/localflare/feature/kv"
	"github.com/go-mizu/blueprints/localflare/feature/queues"
	"github.com/go-mizu/blueprints/localflare/feature/r2"
	"github.com/go-mizu/blueprints/localflare/feature/vectorize"
	"github.com/go-mizu/blueprints/localflare/feature/workers"
	"github.com/go-mizu/blueprints/localflare/store"
)

// WorkersStoreAdapter adapts store.WorkerStore to workers.Store
type WorkersStoreAdapter struct {
	st store.WorkerStore
}

func (a *WorkersStoreAdapter) Create(ctx context.Context, w *workers.Worker) error {
	return a.st.Create(ctx, &store.Worker{
		ID: w.ID, Name: w.Name, Script: w.Script, Routes: w.Routes, Bindings: w.Bindings, Enabled: w.Enabled, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	})
}

func (a *WorkersStoreAdapter) GetByID(ctx context.Context, id string) (*workers.Worker, error) {
	w, err := a.st.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &workers.Worker{
		ID: w.ID, Name: w.Name, Script: w.Script, Routes: w.Routes, Bindings: w.Bindings, Enabled: w.Enabled, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}, nil
}

func (a *WorkersStoreAdapter) GetByName(ctx context.Context, name string) (*workers.Worker, error) {
	w, err := a.st.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return &workers.Worker{
		ID: w.ID, Name: w.Name, Script: w.Script, Routes: w.Routes, Bindings: w.Bindings, Enabled: w.Enabled, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	}, nil
}

func (a *WorkersStoreAdapter) List(ctx context.Context) ([]*workers.Worker, error) {
	list, err := a.st.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*workers.Worker, len(list))
	for i, w := range list {
		result[i] = &workers.Worker{
			ID: w.ID, Name: w.Name, Script: w.Script, Routes: w.Routes, Bindings: w.Bindings, Enabled: w.Enabled, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
		}
	}
	return result, nil
}

func (a *WorkersStoreAdapter) Update(ctx context.Context, w *workers.Worker) error {
	return a.st.Update(ctx, &store.Worker{
		ID: w.ID, Name: w.Name, Script: w.Script, Routes: w.Routes, Bindings: w.Bindings, Enabled: w.Enabled, CreatedAt: w.CreatedAt, UpdatedAt: w.UpdatedAt,
	})
}

func (a *WorkersStoreAdapter) Delete(ctx context.Context, id string) error {
	return a.st.Delete(ctx, id)
}

func (a *WorkersStoreAdapter) CreateRoute(ctx context.Context, r *workers.Route) error {
	return a.st.CreateRoute(ctx, &store.WorkerRoute{
		ID: r.ID, ZoneID: r.ZoneID, Pattern: r.Pattern, WorkerID: r.WorkerID, Enabled: r.Enabled,
	})
}

func (a *WorkersStoreAdapter) ListRoutes(ctx context.Context, zoneID string) ([]*workers.Route, error) {
	list, err := a.st.ListRoutes(ctx, zoneID)
	if err != nil {
		return nil, err
	}
	result := make([]*workers.Route, len(list))
	for i, r := range list {
		result[i] = &workers.Route{ID: r.ID, ZoneID: r.ZoneID, Pattern: r.Pattern, WorkerID: r.WorkerID, Enabled: r.Enabled}
	}
	return result, nil
}

func (a *WorkersStoreAdapter) DeleteRoute(ctx context.Context, id string) error {
	return a.st.DeleteRoute(ctx, id)
}

// KVStoreAdapter adapts store.KVStore to kv.Store
type KVStoreAdapter struct {
	st store.KVStore
}

func (a *KVStoreAdapter) CreateNamespace(ctx context.Context, ns *kv.Namespace) error {
	return a.st.CreateNamespace(ctx, &store.KVNamespace{ID: ns.ID, Title: ns.Title, CreatedAt: ns.CreatedAt})
}

func (a *KVStoreAdapter) GetNamespace(ctx context.Context, id string) (*kv.Namespace, error) {
	ns, err := a.st.GetNamespace(ctx, id)
	if err != nil {
		return nil, err
	}
	return &kv.Namespace{ID: ns.ID, Title: ns.Title, CreatedAt: ns.CreatedAt}, nil
}

func (a *KVStoreAdapter) ListNamespaces(ctx context.Context) ([]*kv.Namespace, error) {
	list, err := a.st.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*kv.Namespace, len(list))
	for i, ns := range list {
		result[i] = &kv.Namespace{ID: ns.ID, Title: ns.Title, CreatedAt: ns.CreatedAt}
	}
	return result, nil
}

func (a *KVStoreAdapter) DeleteNamespace(ctx context.Context, id string) error {
	return a.st.DeleteNamespace(ctx, id)
}

func (a *KVStoreAdapter) Put(ctx context.Context, nsID string, pair *kv.Pair) error {
	return a.st.Put(ctx, nsID, &store.KVPair{
		Key: pair.Key, Value: pair.Value, Expiration: pair.Expiration, Metadata: pair.Metadata,
	})
}

func (a *KVStoreAdapter) Get(ctx context.Context, nsID, key string) (*kv.Pair, error) {
	p, err := a.st.Get(ctx, nsID, key)
	if err != nil {
		return nil, err
	}
	return &kv.Pair{Key: p.Key, Value: p.Value, Expiration: p.Expiration, Metadata: p.Metadata}, nil
}

func (a *KVStoreAdapter) Delete(ctx context.Context, nsID, key string) error {
	return a.st.Delete(ctx, nsID, key)
}

func (a *KVStoreAdapter) List(ctx context.Context, nsID, prefix string, limit int) ([]*kv.Pair, error) {
	list, err := a.st.List(ctx, nsID, prefix, limit)
	if err != nil {
		return nil, err
	}
	result := make([]*kv.Pair, len(list))
	for i, p := range list {
		result[i] = &kv.Pair{Key: p.Key, Value: p.Value, Expiration: p.Expiration, Metadata: p.Metadata}
	}
	return result, nil
}

// R2StoreAdapter adapts store.R2Store to r2.Store
type R2StoreAdapter struct {
	st store.R2Store
}

func (a *R2StoreAdapter) CreateBucket(ctx context.Context, b *r2.Bucket) error {
	return a.st.CreateBucket(ctx, &store.R2Bucket{ID: b.ID, Name: b.Name, Location: b.Location, CreatedAt: b.CreatedAt})
}

func (a *R2StoreAdapter) GetBucket(ctx context.Context, id string) (*r2.Bucket, error) {
	b, err := a.st.GetBucket(ctx, id)
	if err != nil {
		return nil, err
	}
	return &r2.Bucket{ID: b.ID, Name: b.Name, Location: b.Location, CreatedAt: b.CreatedAt}, nil
}

func (a *R2StoreAdapter) ListBuckets(ctx context.Context) ([]*r2.Bucket, error) {
	list, err := a.st.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*r2.Bucket, len(list))
	for i, b := range list {
		result[i] = &r2.Bucket{ID: b.ID, Name: b.Name, Location: b.Location, CreatedAt: b.CreatedAt}
	}
	return result, nil
}

func (a *R2StoreAdapter) DeleteBucket(ctx context.Context, id string) error {
	return a.st.DeleteBucket(ctx, id)
}

func (a *R2StoreAdapter) PutObject(ctx context.Context, bucketID, key string, data []byte, metadata map[string]string) error {
	// Convert legacy metadata map to R2PutOptions
	opts := &store.R2PutOptions{
		CustomMetadata: metadata,
	}
	// Extract content-type if present
	if ct, ok := metadata["content-type"]; ok {
		opts.HTTPMetadata = &store.R2HTTPMetadata{ContentType: ct}
	}
	_, err := a.st.PutObject(ctx, bucketID, key, data, opts)
	return err
}

func (a *R2StoreAdapter) GetObject(ctx context.Context, bucketID, key string) ([]byte, *r2.Object, error) {
	data, obj, err := a.st.GetObject(ctx, bucketID, key, nil)
	if err != nil {
		return nil, nil, err
	}
	contentType := "application/octet-stream"
	if obj.HTTPMetadata != nil && obj.HTTPMetadata.ContentType != "" {
		contentType = obj.HTTPMetadata.ContentType
	}
	return data, &r2.Object{
		Key: obj.Key, Size: obj.Size, ETag: obj.ETag, ContentType: contentType, LastModified: obj.Uploaded,
	}, nil
}

func (a *R2StoreAdapter) DeleteObject(ctx context.Context, bucketID, key string) error {
	return a.st.DeleteObject(ctx, bucketID, key)
}

func (a *R2StoreAdapter) ListObjects(ctx context.Context, bucketID, prefix, delimiter string, limit int) ([]*r2.Object, error) {
	opts := &store.R2ListOptions{
		Prefix:    prefix,
		Delimiter: delimiter,
		Limit:     limit,
	}
	listResult, err := a.st.ListObjects(ctx, bucketID, opts)
	if err != nil {
		return nil, err
	}
	result := make([]*r2.Object, len(listResult.Objects))
	for i, obj := range listResult.Objects {
		contentType := "application/octet-stream"
		if obj.HTTPMetadata != nil && obj.HTTPMetadata.ContentType != "" {
			contentType = obj.HTTPMetadata.ContentType
		}
		result[i] = &r2.Object{Key: obj.Key, Size: obj.Size, ETag: obj.ETag, ContentType: contentType, LastModified: obj.Uploaded}
	}
	return result, nil
}

// D1StoreAdapter adapts store.D1Store to d1.Store
type D1StoreAdapter struct {
	st store.D1Store
}

func (a *D1StoreAdapter) CreateDatabase(ctx context.Context, db *d1.Database) error {
	return a.st.CreateDatabase(ctx, &store.D1Database{
		ID: db.ID, Name: db.Name, Version: db.Version, NumTables: db.NumTables, FileSize: db.FileSize, CreatedAt: db.CreatedAt,
	})
}

func (a *D1StoreAdapter) GetDatabase(ctx context.Context, id string) (*d1.Database, error) {
	db, err := a.st.GetDatabase(ctx, id)
	if err != nil {
		return nil, err
	}
	return &d1.Database{ID: db.ID, Name: db.Name, Version: db.Version, NumTables: db.NumTables, FileSize: db.FileSize, CreatedAt: db.CreatedAt}, nil
}

func (a *D1StoreAdapter) ListDatabases(ctx context.Context) ([]*d1.Database, error) {
	list, err := a.st.ListDatabases(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*d1.Database, len(list))
	for i, db := range list {
		result[i] = &d1.Database{ID: db.ID, Name: db.Name, Version: db.Version, NumTables: db.NumTables, FileSize: db.FileSize, CreatedAt: db.CreatedAt}
	}
	return result, nil
}

func (a *D1StoreAdapter) DeleteDatabase(ctx context.Context, id string) error {
	return a.st.DeleteDatabase(ctx, id)
}

func (a *D1StoreAdapter) Query(ctx context.Context, dbID, sql string, params []interface{}) ([]map[string]interface{}, error) {
	return a.st.Query(ctx, dbID, sql, params)
}

func (a *D1StoreAdapter) Exec(ctx context.Context, dbID, sql string, params []interface{}) (int64, error) {
	return a.st.Exec(ctx, dbID, sql, params)
}

// DurableObjectsStoreAdapter adapts store.DurableObjectStore to do.Store
type DurableObjectsStoreAdapter struct {
	st store.DurableObjectStore
}

func (a *DurableObjectsStoreAdapter) CreateNamespace(ctx context.Context, ns *do.Namespace) error {
	return a.st.CreateNamespace(ctx, &store.DurableObjectNamespace{
		ID: ns.ID, Name: ns.Name, Script: ns.Script, ClassName: ns.ClassName, CreatedAt: ns.CreatedAt,
	})
}

func (a *DurableObjectsStoreAdapter) GetNamespace(ctx context.Context, id string) (*do.Namespace, error) {
	ns, err := a.st.GetNamespace(ctx, id)
	if err != nil {
		return nil, err
	}
	return &do.Namespace{ID: ns.ID, Name: ns.Name, Script: ns.Script, ClassName: ns.ClassName, CreatedAt: ns.CreatedAt}, nil
}

func (a *DurableObjectsStoreAdapter) ListNamespaces(ctx context.Context) ([]*do.Namespace, error) {
	list, err := a.st.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*do.Namespace, len(list))
	for i, ns := range list {
		result[i] = &do.Namespace{ID: ns.ID, Name: ns.Name, Script: ns.Script, ClassName: ns.ClassName, CreatedAt: ns.CreatedAt}
	}
	return result, nil
}

func (a *DurableObjectsStoreAdapter) DeleteNamespace(ctx context.Context, id string) error {
	return a.st.DeleteNamespace(ctx, id)
}

func (a *DurableObjectsStoreAdapter) ListInstances(ctx context.Context, nsID string) ([]*do.Instance, error) {
	list, err := a.st.ListInstances(ctx, nsID)
	if err != nil {
		return nil, err
	}
	result := make([]*do.Instance, len(list))
	for i, inst := range list {
		result[i] = &do.Instance{ID: inst.ID, NamespaceID: inst.NamespaceID, Name: inst.Name, HasStorage: inst.HasStorage, CreatedAt: inst.CreatedAt, LastAccess: inst.LastAccess}
	}
	return result, nil
}

// QueuesStoreAdapter adapts store.QueueStore to queues.Store
type QueuesStoreAdapter struct {
	st store.QueueStore
}

func (a *QueuesStoreAdapter) CreateQueue(ctx context.Context, q *queues.Queue) error {
	return a.st.CreateQueue(ctx, &store.Queue{
		ID: q.ID, Name: q.Name, Settings: store.QueueSettings{
			DeliveryDelay: q.Settings.DeliveryDelay, MessageTTL: q.Settings.MessageTTL, MaxRetries: q.Settings.MaxRetries,
			MaxBatchSize: q.Settings.MaxBatchSize, MaxBatchTimeout: q.Settings.MaxBatchTimeout,
		}, CreatedAt: q.CreatedAt,
	})
}

func (a *QueuesStoreAdapter) GetQueue(ctx context.Context, id string) (*queues.Queue, error) {
	q, err := a.st.GetQueue(ctx, id)
	if err != nil {
		return nil, err
	}
	return &queues.Queue{
		ID: q.ID, Name: q.Name, Settings: queues.QueueSettings{
			DeliveryDelay: q.Settings.DeliveryDelay, MessageTTL: q.Settings.MessageTTL, MaxRetries: q.Settings.MaxRetries,
			MaxBatchSize: q.Settings.MaxBatchSize, MaxBatchTimeout: q.Settings.MaxBatchTimeout,
		}, CreatedAt: q.CreatedAt,
	}, nil
}

func (a *QueuesStoreAdapter) ListQueues(ctx context.Context) ([]*queues.Queue, error) {
	list, err := a.st.ListQueues(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*queues.Queue, len(list))
	for i, q := range list {
		result[i] = &queues.Queue{
			ID: q.ID, Name: q.Name, Settings: queues.QueueSettings{
				DeliveryDelay: q.Settings.DeliveryDelay, MessageTTL: q.Settings.MessageTTL, MaxRetries: q.Settings.MaxRetries,
				MaxBatchSize: q.Settings.MaxBatchSize, MaxBatchTimeout: q.Settings.MaxBatchTimeout,
			}, CreatedAt: q.CreatedAt,
		}
	}
	return result, nil
}

func (a *QueuesStoreAdapter) DeleteQueue(ctx context.Context, id string) error {
	return a.st.DeleteQueue(ctx, id)
}

func (a *QueuesStoreAdapter) SendMessage(ctx context.Context, queueID string, msg *queues.Message) error {
	return a.st.SendMessage(ctx, queueID, &store.QueueMessage{
		ID: msg.ID, QueueID: msg.QueueID, Body: msg.Body, ContentType: msg.ContentType, Attempts: msg.Attempts,
		CreatedAt: msg.CreatedAt, VisibleAt: msg.VisibleAt, ExpiresAt: msg.ExpiresAt,
	})
}

func (a *QueuesStoreAdapter) PullMessages(ctx context.Context, queueID string, batchSize, visibilityTimeout int) ([]*queues.Message, error) {
	list, err := a.st.PullMessages(ctx, queueID, batchSize, visibilityTimeout)
	if err != nil {
		return nil, err
	}
	result := make([]*queues.Message, len(list))
	for i, msg := range list {
		result[i] = &queues.Message{
			ID: msg.ID, QueueID: msg.QueueID, Body: msg.Body, ContentType: msg.ContentType, Attempts: msg.Attempts,
			CreatedAt: msg.CreatedAt, VisibleAt: msg.VisibleAt, ExpiresAt: msg.ExpiresAt,
		}
	}
	return result, nil
}

func (a *QueuesStoreAdapter) AckMessage(ctx context.Context, queueID, msgID string) error {
	return a.st.AckMessage(ctx, queueID, msgID)
}

func (a *QueuesStoreAdapter) AckBatch(ctx context.Context, queueID string, msgIDs []string) error {
	return a.st.AckBatch(ctx, queueID, msgIDs)
}

func (a *QueuesStoreAdapter) GetQueueStats(ctx context.Context, queueID string) (*queues.Stats, error) {
	s, err := a.st.GetQueueStats(ctx, queueID)
	if err != nil {
		return nil, err
	}
	return &queues.Stats{Messages: s.Messages, MessagesReady: s.MessagesReady, MessagesDelayed: s.MessagesDelayed}, nil
}

// VectorizeStoreAdapter adapts store.VectorizeStore to vectorize.Store
type VectorizeStoreAdapter struct {
	st store.VectorizeStore
}

func (a *VectorizeStoreAdapter) CreateIndex(ctx context.Context, idx *vectorize.Index) error {
	return a.st.CreateIndex(ctx, &store.VectorIndex{
		ID: idx.ID, Name: idx.Name, Description: idx.Description, Dimensions: idx.Dimensions, Metric: idx.Metric, CreatedAt: idx.CreatedAt,
	})
}

func (a *VectorizeStoreAdapter) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	idx, err := a.st.GetIndex(ctx, name)
	if err != nil {
		return nil, err
	}
	return &vectorize.Index{ID: idx.ID, Name: idx.Name, Description: idx.Description, Dimensions: idx.Dimensions, Metric: idx.Metric, CreatedAt: idx.CreatedAt}, nil
}

func (a *VectorizeStoreAdapter) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	list, err := a.st.ListIndexes(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*vectorize.Index, len(list))
	for i, idx := range list {
		result[i] = &vectorize.Index{ID: idx.ID, Name: idx.Name, Description: idx.Description, Dimensions: idx.Dimensions, Metric: idx.Metric, CreatedAt: idx.CreatedAt}
	}
	return result, nil
}

func (a *VectorizeStoreAdapter) DeleteIndex(ctx context.Context, name string) error {
	return a.st.DeleteIndex(ctx, name)
}

func (a *VectorizeStoreAdapter) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	stVectors := make([]*store.Vector, len(vectors))
	for i, v := range vectors {
		stVectors[i] = &store.Vector{ID: v.ID, Values: v.Values, Namespace: v.Namespace, Metadata: v.Metadata}
	}
	return a.st.Insert(ctx, indexName, stVectors)
}

func (a *VectorizeStoreAdapter) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	stVectors := make([]*store.Vector, len(vectors))
	for i, v := range vectors {
		stVectors[i] = &store.Vector{ID: v.ID, Values: v.Values, Namespace: v.Namespace, Metadata: v.Metadata}
	}
	return a.st.Upsert(ctx, indexName, stVectors)
}

func (a *VectorizeStoreAdapter) Query(ctx context.Context, indexName string, vector []float32, opts *vectorize.QueryOpts) ([]*vectorize.Match, error) {
	stOpts := &store.VectorQueryOptions{
		TopK: opts.TopK, Namespace: opts.Namespace, ReturnValues: opts.ReturnValues, ReturnMetadata: opts.ReturnMetadata, Filter: opts.Filter,
	}
	results, err := a.st.Query(ctx, indexName, vector, stOpts)
	if err != nil {
		return nil, err
	}
	matches := make([]*vectorize.Match, len(results))
	for i, r := range results {
		matches[i] = &vectorize.Match{ID: r.ID, Score: r.Score, Values: r.Values, Metadata: r.Metadata}
	}
	return matches, nil
}

func (a *VectorizeStoreAdapter) GetByIDs(ctx context.Context, indexName string, ids []string) ([]*vectorize.Vector, error) {
	list, err := a.st.GetByIDs(ctx, indexName, ids)
	if err != nil {
		return nil, err
	}
	result := make([]*vectorize.Vector, len(list))
	for i, v := range list {
		result[i] = &vectorize.Vector{ID: v.ID, Values: v.Values, Namespace: v.Namespace, Metadata: v.Metadata}
	}
	return result, nil
}

func (a *VectorizeStoreAdapter) DeleteByIDs(ctx context.Context, indexName string, ids []string) error {
	return a.st.DeleteByIDs(ctx, indexName, ids)
}

// AnalyticsEngineStoreAdapter adapts store.AnalyticsEngineStore to analytics_engine.Store
type AnalyticsEngineStoreAdapter struct {
	st store.AnalyticsEngineStore
}

func (a *AnalyticsEngineStoreAdapter) CreateDataset(ctx context.Context, ds *analytics_engine.Dataset) error {
	return a.st.CreateDataset(ctx, &store.AnalyticsEngineDataset{ID: ds.ID, Name: ds.Name, CreatedAt: ds.CreatedAt})
}

func (a *AnalyticsEngineStoreAdapter) GetDataset(ctx context.Context, name string) (*analytics_engine.Dataset, error) {
	ds, err := a.st.GetDataset(ctx, name)
	if err != nil {
		return nil, err
	}
	return &analytics_engine.Dataset{ID: ds.ID, Name: ds.Name, CreatedAt: ds.CreatedAt}, nil
}

func (a *AnalyticsEngineStoreAdapter) ListDatasets(ctx context.Context) ([]*analytics_engine.Dataset, error) {
	list, err := a.st.ListDatasets(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*analytics_engine.Dataset, len(list))
	for i, ds := range list {
		result[i] = &analytics_engine.Dataset{ID: ds.ID, Name: ds.Name, CreatedAt: ds.CreatedAt}
	}
	return result, nil
}

func (a *AnalyticsEngineStoreAdapter) DeleteDataset(ctx context.Context, name string) error {
	return a.st.DeleteDataset(ctx, name)
}

func (a *AnalyticsEngineStoreAdapter) WriteDataPoint(ctx context.Context, point *analytics_engine.DataPoint) error {
	return a.st.WriteDataPoint(ctx, &store.AnalyticsEngineDataPoint{
		Dataset: point.Dataset, Timestamp: point.Timestamp, Indexes: point.Indexes, Doubles: point.Doubles, Blobs: point.Blobs,
	})
}

func (a *AnalyticsEngineStoreAdapter) WriteBatch(ctx context.Context, points []*analytics_engine.DataPoint) error {
	stPoints := make([]*store.AnalyticsEngineDataPoint, len(points))
	for i, p := range points {
		stPoints[i] = &store.AnalyticsEngineDataPoint{
			Dataset: p.Dataset, Timestamp: p.Timestamp, Indexes: p.Indexes, Doubles: p.Doubles, Blobs: p.Blobs,
		}
	}
	return a.st.WriteBatch(ctx, stPoints)
}

func (a *AnalyticsEngineStoreAdapter) Query(ctx context.Context, sql string) ([]map[string]any, error) {
	return a.st.Query(ctx, sql)
}

// AIStoreAdapter adapts store.AIStore to ai.Store
type AIStoreAdapter struct {
	st store.AIStore
}

func (a *AIStoreAdapter) ListModels(ctx context.Context, task string) ([]*ai.Model, error) {
	list, err := a.st.ListModels(ctx, task)
	if err != nil {
		return nil, err
	}
	result := make([]*ai.Model, len(list))
	for i, m := range list {
		result[i] = &ai.Model{ID: m.ID, Name: m.Name, Description: m.Description, Task: m.Task, Properties: m.Properties}
	}
	return result, nil
}

func (a *AIStoreAdapter) GetModel(ctx context.Context, name string) (*ai.Model, error) {
	m, err := a.st.GetModel(ctx, name)
	if err != nil {
		return nil, err
	}
	return &ai.Model{ID: m.ID, Name: m.Name, Description: m.Description, Task: m.Task, Properties: m.Properties}, nil
}

func (a *AIStoreAdapter) Run(ctx context.Context, model string, inputs map[string]any, options map[string]any) (*ai.InferenceResult, error) {
	result, err := a.st.Run(ctx, &store.AIInferenceRequest{Model: model, Inputs: inputs, Options: options})
	if err != nil {
		return nil, err
	}
	return &ai.InferenceResult{Result: result.Result}, nil
}

// AIGatewayStoreAdapter adapts store.AIGatewayStore to ai_gateway.Store
type AIGatewayStoreAdapter struct {
	st store.AIGatewayStore
}

func (a *AIGatewayStoreAdapter) CreateGateway(ctx context.Context, gw *ai_gateway.Gateway) error {
	return a.st.CreateGateway(ctx, &store.AIGateway{
		ID: gw.ID, Name: gw.Name, CollectLogs: gw.CollectLogs, CacheEnabled: gw.CacheEnabled, CacheTTL: gw.CacheTTL,
		RateLimitEnabled: gw.RateLimitEnabled, RateLimitCount: gw.RateLimitCount, RateLimitPeriod: gw.RateLimitPeriod, CreatedAt: gw.CreatedAt,
	})
}

func (a *AIGatewayStoreAdapter) GetGateway(ctx context.Context, id string) (*ai_gateway.Gateway, error) {
	gw, err := a.st.GetGateway(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ai_gateway.Gateway{
		ID: gw.ID, Name: gw.Name, CollectLogs: gw.CollectLogs, CacheEnabled: gw.CacheEnabled, CacheTTL: gw.CacheTTL,
		RateLimitEnabled: gw.RateLimitEnabled, RateLimitCount: gw.RateLimitCount, RateLimitPeriod: gw.RateLimitPeriod, CreatedAt: gw.CreatedAt,
	}, nil
}

func (a *AIGatewayStoreAdapter) ListGateways(ctx context.Context) ([]*ai_gateway.Gateway, error) {
	list, err := a.st.ListGateways(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*ai_gateway.Gateway, len(list))
	for i, gw := range list {
		result[i] = &ai_gateway.Gateway{
			ID: gw.ID, Name: gw.Name, CollectLogs: gw.CollectLogs, CacheEnabled: gw.CacheEnabled, CacheTTL: gw.CacheTTL,
			RateLimitEnabled: gw.RateLimitEnabled, RateLimitCount: gw.RateLimitCount, RateLimitPeriod: gw.RateLimitPeriod, CreatedAt: gw.CreatedAt,
		}
	}
	return result, nil
}

func (a *AIGatewayStoreAdapter) UpdateGateway(ctx context.Context, gw *ai_gateway.Gateway) error {
	return a.st.UpdateGateway(ctx, &store.AIGateway{
		ID: gw.ID, Name: gw.Name, CollectLogs: gw.CollectLogs, CacheEnabled: gw.CacheEnabled, CacheTTL: gw.CacheTTL,
		RateLimitEnabled: gw.RateLimitEnabled, RateLimitCount: gw.RateLimitCount, RateLimitPeriod: gw.RateLimitPeriod, CreatedAt: gw.CreatedAt,
	})
}

func (a *AIGatewayStoreAdapter) DeleteGateway(ctx context.Context, id string) error {
	return a.st.DeleteGateway(ctx, id)
}

func (a *AIGatewayStoreAdapter) GetLogs(ctx context.Context, gatewayID string, limit, offset int) ([]*ai_gateway.Log, error) {
	list, err := a.st.GetLogs(ctx, gatewayID, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]*ai_gateway.Log, len(list))
	for i, log := range list {
		result[i] = &ai_gateway.Log{
			ID: log.ID, GatewayID: log.GatewayID, Provider: log.Provider, Model: log.Model, Cached: log.Cached,
			Status: log.Status, Duration: log.Duration, Tokens: log.Tokens, Cost: log.Cost,
			Request: log.Request, Response: log.Response, Metadata: log.Metadata, CreatedAt: log.CreatedAt,
		}
	}
	return result, nil
}

// HyperdriveStoreAdapter adapts store.HyperdriveStore to hyperdrive.Store
type HyperdriveStoreAdapter struct {
	st store.HyperdriveStore
}

func (a *HyperdriveStoreAdapter) CreateConfig(ctx context.Context, cfg *hyperdrive.Config) error {
	return a.st.CreateConfig(ctx, &store.HyperdriveConfig{
		ID: cfg.ID, Name: cfg.Name, Origin: store.HyperdriveOrigin{
			Database: cfg.Origin.Database, Host: cfg.Origin.Host, Port: cfg.Origin.Port, Scheme: cfg.Origin.Scheme, User: cfg.Origin.User, Password: cfg.Origin.Password,
		}, Caching: store.HyperdriveCaching{
			Disabled: cfg.Caching.Disabled, MaxAge: cfg.Caching.MaxAge, StaleWhileRevalidate: cfg.Caching.StaleWhileRevalidate,
		}, CreatedAt: cfg.CreatedAt,
	})
}

func (a *HyperdriveStoreAdapter) GetConfig(ctx context.Context, id string) (*hyperdrive.Config, error) {
	cfg, err := a.st.GetConfig(ctx, id)
	if err != nil {
		return nil, err
	}
	return &hyperdrive.Config{
		ID: cfg.ID, Name: cfg.Name, Origin: hyperdrive.Origin{
			Database: cfg.Origin.Database, Host: cfg.Origin.Host, Port: cfg.Origin.Port, Scheme: cfg.Origin.Scheme, User: cfg.Origin.User, Password: cfg.Origin.Password,
		}, Caching: hyperdrive.Caching{
			Disabled: cfg.Caching.Disabled, MaxAge: cfg.Caching.MaxAge, StaleWhileRevalidate: cfg.Caching.StaleWhileRevalidate,
		}, CreatedAt: cfg.CreatedAt,
	}, nil
}

func (a *HyperdriveStoreAdapter) ListConfigs(ctx context.Context) ([]*hyperdrive.Config, error) {
	list, err := a.st.ListConfigs(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*hyperdrive.Config, len(list))
	for i, cfg := range list {
		result[i] = &hyperdrive.Config{
			ID: cfg.ID, Name: cfg.Name, Origin: hyperdrive.Origin{
				Database: cfg.Origin.Database, Host: cfg.Origin.Host, Port: cfg.Origin.Port, Scheme: cfg.Origin.Scheme, User: cfg.Origin.User, Password: cfg.Origin.Password,
			}, Caching: hyperdrive.Caching{
				Disabled: cfg.Caching.Disabled, MaxAge: cfg.Caching.MaxAge, StaleWhileRevalidate: cfg.Caching.StaleWhileRevalidate,
			}, CreatedAt: cfg.CreatedAt,
		}
	}
	return result, nil
}

func (a *HyperdriveStoreAdapter) UpdateConfig(ctx context.Context, cfg *hyperdrive.Config) error {
	return a.st.UpdateConfig(ctx, &store.HyperdriveConfig{
		ID: cfg.ID, Name: cfg.Name, Origin: store.HyperdriveOrigin{
			Database: cfg.Origin.Database, Host: cfg.Origin.Host, Port: cfg.Origin.Port, Scheme: cfg.Origin.Scheme, User: cfg.Origin.User, Password: cfg.Origin.Password,
		}, Caching: store.HyperdriveCaching{
			Disabled: cfg.Caching.Disabled, MaxAge: cfg.Caching.MaxAge, StaleWhileRevalidate: cfg.Caching.StaleWhileRevalidate,
		}, CreatedAt: cfg.CreatedAt,
	})
}

func (a *HyperdriveStoreAdapter) DeleteConfig(ctx context.Context, id string) error {
	return a.st.DeleteConfig(ctx, id)
}

func (a *HyperdriveStoreAdapter) GetStats(ctx context.Context, configID string) (*hyperdrive.Stats, error) {
	s, err := a.st.GetStats(ctx, configID)
	if err != nil {
		return nil, err
	}
	return &hyperdrive.Stats{
		ActiveConnections: s.ActiveConnections, IdleConnections: s.IdleConnections, TotalConnections: s.TotalConnections,
		QueriesPerSecond: s.QueriesPerSecond, CacheHitRate: s.CacheHitRate,
	}, nil
}

// CronStoreAdapter adapts store.CronStore to cron.Store
type CronStoreAdapter struct {
	st store.CronStore
}

func (a *CronStoreAdapter) CreateTrigger(ctx context.Context, t *cron.Trigger) error {
	return a.st.CreateTrigger(ctx, &store.CronTrigger{
		ID: t.ID, ScriptName: t.ScriptName, Cron: t.Cron, Enabled: t.Enabled, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	})
}

func (a *CronStoreAdapter) GetTrigger(ctx context.Context, id string) (*cron.Trigger, error) {
	t, err := a.st.GetTrigger(ctx, id)
	if err != nil {
		return nil, err
	}
	return &cron.Trigger{ID: t.ID, ScriptName: t.ScriptName, Cron: t.Cron, Enabled: t.Enabled, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}, nil
}

func (a *CronStoreAdapter) ListTriggers(ctx context.Context) ([]*cron.Trigger, error) {
	list, err := a.st.ListTriggers(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*cron.Trigger, len(list))
	for i, t := range list {
		result[i] = &cron.Trigger{ID: t.ID, ScriptName: t.ScriptName, Cron: t.Cron, Enabled: t.Enabled, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
	}
	return result, nil
}

func (a *CronStoreAdapter) ListTriggersByScript(ctx context.Context, scriptName string) ([]*cron.Trigger, error) {
	list, err := a.st.ListTriggersByScript(ctx, scriptName)
	if err != nil {
		return nil, err
	}
	result := make([]*cron.Trigger, len(list))
	for i, t := range list {
		result[i] = &cron.Trigger{ID: t.ID, ScriptName: t.ScriptName, Cron: t.Cron, Enabled: t.Enabled, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
	}
	return result, nil
}

func (a *CronStoreAdapter) UpdateTrigger(ctx context.Context, t *cron.Trigger) error {
	return a.st.UpdateTrigger(ctx, &store.CronTrigger{
		ID: t.ID, ScriptName: t.ScriptName, Cron: t.Cron, Enabled: t.Enabled, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	})
}

func (a *CronStoreAdapter) DeleteTrigger(ctx context.Context, id string) error {
	return a.st.DeleteTrigger(ctx, id)
}

func (a *CronStoreAdapter) GetRecentExecutions(ctx context.Context, triggerID string, limit int) ([]*cron.Execution, error) {
	list, err := a.st.GetRecentExecutions(ctx, triggerID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]*cron.Execution, len(list))
	for i, e := range list {
		result[i] = &cron.Execution{
			ID: e.ID, TriggerID: e.TriggerID, ScheduledAt: e.ScheduledAt, StartedAt: e.StartedAt,
			FinishedAt: e.FinishedAt, Status: e.Status, Error: e.Error,
		}
	}
	return result, nil
}

// AuthStoreAdapter adapts store.UserStore to auth.Store
type AuthStoreAdapter struct {
	st store.UserStore
}

func (a *AuthStoreAdapter) Create(ctx context.Context, u *auth.User) error {
	return a.st.Create(ctx, &store.User{
		ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: u.PasswordHash, Role: u.Role, CreatedAt: u.CreatedAt,
	})
}

func (a *AuthStoreAdapter) GetByID(ctx context.Context, id string) (*auth.User, error) {
	u, err := a.st.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &auth.User{ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: u.PasswordHash, Role: u.Role, CreatedAt: u.CreatedAt}, nil
}

func (a *AuthStoreAdapter) GetByEmail(ctx context.Context, email string) (*auth.User, error) {
	u, err := a.st.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return &auth.User{ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: u.PasswordHash, Role: u.Role, CreatedAt: u.CreatedAt}, nil
}

func (a *AuthStoreAdapter) Update(ctx context.Context, u *auth.User) error {
	return a.st.Update(ctx, &store.User{
		ID: u.ID, Email: u.Email, Name: u.Name, PasswordHash: u.PasswordHash, Role: u.Role, CreatedAt: u.CreatedAt,
	})
}

func (a *AuthStoreAdapter) CreateSession(ctx context.Context, s *auth.Session) error {
	return a.st.CreateSession(ctx, &store.Session{
		ID: s.ID, UserID: s.UserID, Token: s.Token, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt,
	})
}

func (a *AuthStoreAdapter) GetSession(ctx context.Context, token string) (*auth.Session, error) {
	s, err := a.st.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}
	return &auth.Session{ID: s.ID, UserID: s.UserID, Token: s.Token, ExpiresAt: s.ExpiresAt, CreatedAt: s.CreatedAt}, nil
}

func (a *AuthStoreAdapter) DeleteSession(ctx context.Context, token string) error {
	return a.st.DeleteSession(ctx, token)
}
