// Package qdrant provides a Qdrant driver for the vectorize package.
// Import this package to register the "qdrant" driver.
package qdrant

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/vectorize"
	"github.com/go-mizu/blueprints/localflare/pkg/vectorize/driver"
	pb "github.com/qdrant/go-client/qdrant"
)

func init() {
	driver.Register("qdrant", &Driver{})
}

// Driver implements vectorize.Driver for Qdrant.
type Driver struct{}

// Open creates a new Qdrant connection.
// DSN format: host:port (e.g., "localhost:6334")
func (d *Driver) Open(dsn string) (vectorize.DB, error) {
	if dsn == "" {
		return nil, vectorize.ErrInvalidDSN
	}

	// Parse DSN
	parts := strings.Split(dsn, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: expected host:port format", vectorize.ErrInvalidDSN)
	}

	host := parts[0]
	port := parts[1]

	client, err := pb.NewClient(&pb.Config{
		Host: host,
		Port: mustParsePort(port),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", vectorize.ErrConnectionFailed, err)
	}

	return &DB{client: client}, nil
}

func mustParsePort(port string) int {
	var p int
	fmt.Sscanf(port, "%d", &p)
	if p == 0 {
		p = 6334
	}
	return p
}

// DB implements vectorize.DB for Qdrant.
type DB struct {
	client *pb.Client
}

// CreateIndex creates a new collection in Qdrant.
func (db *DB) CreateIndex(ctx context.Context, index *vectorize.Index) error {
	distance := pb.Distance_Cosine
	switch index.Metric {
	case vectorize.Cosine:
		distance = pb.Distance_Cosine
	case vectorize.Euclidean:
		distance = pb.Distance_Euclid
	case vectorize.DotProduct:
		distance = pb.Distance_Dot
	}

	err := db.client.CreateCollection(ctx, &pb.CreateCollection{
		CollectionName: index.Name,
		VectorsConfig: pb.NewVectorsConfig(&pb.VectorParams{
			Size:     uint64(index.Dimensions),
			Distance: distance,
		}),
	})
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return vectorize.ErrIndexExists
		}
		return err
	}

	return nil
}

// GetIndex retrieves collection information.
func (db *DB) GetIndex(ctx context.Context, name string) (*vectorize.Index, error) {
	info, err := db.client.GetCollectionInfo(ctx, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "doesn't exist") {
			return nil, vectorize.ErrIndexNotFound
		}
		return nil, err
	}

	// Get vector config
	var dimensions int
	var metric vectorize.DistanceMetric = vectorize.Cosine

	if info.Config != nil && info.Config.Params != nil {
		if vectorParams := info.Config.Params.GetVectorsConfig().GetParams(); vectorParams != nil {
			dimensions = int(vectorParams.Size)
			switch vectorParams.Distance {
			case pb.Distance_Cosine:
				metric = vectorize.Cosine
			case pb.Distance_Euclid:
				metric = vectorize.Euclidean
			case pb.Distance_Dot:
				metric = vectorize.DotProduct
			}
		}
	}

	var pointsCount int64
	if info.PointsCount != nil {
		pointsCount = int64(*info.PointsCount)
	}

	return &vectorize.Index{
		Name:        name,
		Dimensions:  dimensions,
		Metric:      metric,
		VectorCount: pointsCount,
	}, nil
}

// ListIndexes returns all collections.
func (db *DB) ListIndexes(ctx context.Context) ([]*vectorize.Index, error) {
	collections, err := db.client.ListCollections(ctx)
	if err != nil {
		return nil, err
	}

	indexes := make([]*vectorize.Index, 0, len(collections))
	for _, col := range collections {
		idx, err := db.GetIndex(ctx, col)
		if err != nil {
			continue
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// DeleteIndex removes a collection.
func (db *DB) DeleteIndex(ctx context.Context, name string) error {
	err := db.client.DeleteCollection(ctx, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "doesn't exist") {
			return vectorize.ErrIndexNotFound
		}
		return err
	}
	return nil
}

// Insert adds vectors to a collection.
func (db *DB) Insert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	if len(vectors) == 0 {
		return nil
	}

	points := make([]*pb.PointStruct, len(vectors))
	for i, v := range vectors {
		payload := make(map[string]*pb.Value)
		if v.Namespace != "" {
			payload["_namespace"] = pb.NewValueString(v.Namespace)
		}
		for k, val := range v.Metadata {
			payload[k] = toQdrantValue(val)
		}

		points[i] = &pb.PointStruct{
			Id:      pb.NewIDUUID(v.ID),
			Vectors: pb.NewVectors(v.Values...),
			Payload: payload,
		}
	}

	_, err := db.client.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: indexName,
		Points:         points,
	})
	return err
}

// Upsert adds or updates vectors.
func (db *DB) Upsert(ctx context.Context, indexName string, vectors []*vectorize.Vector) error {
	return db.Insert(ctx, indexName, vectors)
}

// Search finds similar vectors.
func (db *DB) Search(ctx context.Context, indexName string, vector []float32, opts *vectorize.SearchOptions) ([]*vectorize.Match, error) {
	if opts == nil {
		opts = &vectorize.SearchOptions{TopK: 10}
	}
	if opts.TopK <= 0 {
		opts.TopK = 10
	}

	req := &pb.QueryPoints{
		CollectionName: indexName,
		Query:          pb.NewQuery(vector...),
		Limit:          pb.PtrOf(uint64(opts.TopK)),
		WithPayload:    pb.NewWithPayload(opts.ReturnMetadata),
		WithVectors:    pb.NewWithVectors(opts.ReturnValues),
	}

	// Add namespace filter if specified
	if opts.Namespace != "" {
		req.Filter = &pb.Filter{
			Must: []*pb.Condition{
				{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: "_namespace",
							Match: &pb.Match{
								MatchValue: &pb.Match_Keyword{
									Keyword: opts.Namespace,
								},
							},
						},
					},
				},
			},
		}
	}

	// Add custom filters
	if len(opts.Filter) > 0 && req.Filter == nil {
		req.Filter = &pb.Filter{Must: []*pb.Condition{}}
	}
	for key, val := range opts.Filter {
		cond := &pb.Condition{
			ConditionOneOf: &pb.Condition_Field{
				Field: &pb.FieldCondition{
					Key:   key,
					Match: toQdrantMatchCondition(val),
				},
			},
		}
		req.Filter.Must = append(req.Filter.Must, cond)
	}

	results, err := db.client.Query(ctx, req)
	if err != nil {
		return nil, err
	}

	matches := make([]*vectorize.Match, len(results))
	for i, r := range results {
		match := &vectorize.Match{
			ID:    extractID(r.Id),
			Score: r.Score,
		}

		if opts.ReturnValues && r.Vectors != nil {
			match.Values = r.Vectors.GetVector().Data
		}

		if opts.ReturnMetadata && r.Payload != nil {
			match.Metadata = fromQdrantPayload(r.Payload)
		}

		// Apply score threshold
		if opts.ScoreThreshold > 0 && match.Score < opts.ScoreThreshold {
			matches = matches[:i]
			break
		}

		matches[i] = match
	}

	return matches, nil
}

// Get retrieves vectors by IDs.
func (db *DB) Get(ctx context.Context, indexName string, ids []string) ([]*vectorize.Vector, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	pointIDs := make([]*pb.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = pb.NewIDUUID(id)
	}

	points, err := db.client.Get(ctx, &pb.GetPoints{
		CollectionName: indexName,
		Ids:            pointIDs,
		WithPayload:    pb.NewWithPayload(true),
		WithVectors:    pb.NewWithVectors(true),
	})
	if err != nil {
		return nil, err
	}

	vectors := make([]*vectorize.Vector, len(points))
	for i, p := range points {
		vec := &vectorize.Vector{
			ID: extractID(p.Id),
		}

		if p.Vectors != nil {
			vec.Values = p.Vectors.GetVector().Data
		}

		if p.Payload != nil {
			vec.Metadata = fromQdrantPayload(p.Payload)
			if ns, ok := vec.Metadata["_namespace"].(string); ok {
				vec.Namespace = ns
				delete(vec.Metadata, "_namespace")
			}
		}

		vectors[i] = vec
	}

	return vectors, nil
}

// Delete removes vectors by IDs.
func (db *DB) Delete(ctx context.Context, indexName string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	pointIDs := make([]*pb.PointId, len(ids))
	for i, id := range ids {
		pointIDs[i] = pb.NewIDUUID(id)
	}

	_, err := db.client.Delete(ctx, &pb.DeletePoints{
		CollectionName: indexName,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{Ids: pointIDs},
			},
		},
	})
	return err
}

// Ping checks the connection.
func (db *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := db.client.ListCollections(ctx)
	return err
}

// Close releases resources.
func (db *DB) Close() error {
	return db.client.Close()
}

// Helper functions

func toQdrantValue(v any) *pb.Value {
	switch val := v.(type) {
	case string:
		return pb.NewValueString(val)
	case int:
		return pb.NewValueInt(int64(val))
	case int64:
		return pb.NewValueInt(val)
	case float64:
		return pb.NewValueDouble(val)
	case float32:
		return pb.NewValueDouble(float64(val))
	case bool:
		return pb.NewValueBool(val)
	default:
		return pb.NewValueString(fmt.Sprintf("%v", v))
	}
}

func toQdrantMatchCondition(v any) *pb.Match {
	switch val := v.(type) {
	case string:
		return &pb.Match{MatchValue: &pb.Match_Keyword{Keyword: val}}
	case int:
		return &pb.Match{MatchValue: &pb.Match_Integer{Integer: int64(val)}}
	case int64:
		return &pb.Match{MatchValue: &pb.Match_Integer{Integer: val}}
	case bool:
		return &pb.Match{MatchValue: &pb.Match_Boolean{Boolean: val}}
	default:
		return &pb.Match{MatchValue: &pb.Match_Keyword{Keyword: fmt.Sprintf("%v", v)}}
	}
}

func fromQdrantPayload(payload map[string]*pb.Value) map[string]any {
	result := make(map[string]any)
	for k, v := range payload {
		switch val := v.GetKind().(type) {
		case *pb.Value_StringValue:
			result[k] = val.StringValue
		case *pb.Value_IntegerValue:
			result[k] = val.IntegerValue
		case *pb.Value_DoubleValue:
			result[k] = val.DoubleValue
		case *pb.Value_BoolValue:
			result[k] = val.BoolValue
		}
	}
	return result
}

func extractID(id *pb.PointId) string {
	if id == nil {
		return ""
	}
	switch val := id.PointIdOptions.(type) {
	case *pb.PointId_Uuid:
		return val.Uuid
	case *pb.PointId_Num:
		return fmt.Sprintf("%d", val.Num)
	}
	return ""
}
