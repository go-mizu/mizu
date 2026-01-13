package seed

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedVectorize(ctx context.Context) error {
	slog.Info("seeding vectorize indexes")

	indexCount := 0
	vectorCount := 0

	// Vector Indexes
	indexes := []*store.VectorIndex{
		{
			ID:          generateID(),
			Name:        "products",
			Description: "Product search embeddings",
			Dimensions:  384,
			Metric:      "cosine",
			CreatedAt:   s.timeAgo(30 * 24 * time.Hour),
			VectorCount: 0,
		},
		{
			ID:          generateID(),
			Name:        "documents",
			Description: "Documentation search",
			Dimensions:  1536,
			Metric:      "cosine",
			CreatedAt:   s.timeAgo(45 * 24 * time.Hour),
			VectorCount: 0,
		},
		{
			ID:          generateID(),
			Name:        "images",
			Description: "Image similarity search",
			Dimensions:  512,
			Metric:      "euclidean",
			CreatedAt:   s.timeAgo(20 * 24 * time.Hour),
			VectorCount: 0,
		},
	}

	for _, idx := range indexes {
		if err := s.store.Vectorize().CreateIndex(ctx, idx); err == nil {
			s.ids.VectorIndexes[idx.Name] = idx.ID
			indexCount++
		}
	}

	// Sample vectors for products index
	if _, ok := s.ids.VectorIndexes["products"]; ok {
		products := []struct {
			id       string
			name     string
			category string
		}{
			{"prod_001", "Wireless Bluetooth Headphones", "Electronics"},
			{"prod_002", "USB-C Charging Cable", "Electronics"},
			{"prod_003", "Portable Power Bank", "Electronics"},
			{"prod_004", "Smart Watch Pro", "Electronics"},
			{"prod_005", "Classic Cotton T-Shirt", "Clothing"},
			{"prod_006", "Denim Jeans", "Clothing"},
			{"prod_007", "Running Shoes", "Clothing"},
			{"prod_008", "Clean Code Book", "Books"},
			{"prod_009", "Coffee Maker", "Home"},
			{"prod_010", "Yoga Mat", "Sports"},
		}

		var vectors []*store.Vector
		for _, p := range products {
			// Generate random but reproducible embedding
			values := make([]float32, 384)
			src := rand.NewSource(int64(hashString(p.id)))
			r := rand.New(src)
			for i := range values {
				values[i] = r.Float32()*2 - 1 // -1 to 1
			}
			// Normalize
			var norm float32
			for _, v := range values {
				norm += v * v
			}
			norm = float32(1.0 / float64(norm))
			for i := range values {
				values[i] *= norm
			}

			vectors = append(vectors, &store.Vector{
				ID:        p.id,
				Values:    values,
				Namespace: p.category,
				Metadata: map[string]interface{}{
					"name":     p.name,
					"category": p.category,
				},
			})
		}

		if err := s.store.Vectorize().Insert(ctx, "products", vectors); err == nil {
			vectorCount += len(vectors)
		}
	}

	// Sample vectors for documents index
	if _, ok := s.ids.VectorIndexes["documents"]; ok {
		docs := []struct {
			id    string
			title string
			topic string
		}{
			{"doc_001", "Getting Started Guide", "tutorial"},
			{"doc_002", "API Reference", "reference"},
			{"doc_003", "Authentication Guide", "security"},
			{"doc_004", "Workers Documentation", "compute"},
			{"doc_005", "R2 Storage Guide", "storage"},
		}

		var vectors []*store.Vector
		for _, d := range docs {
			values := make([]float32, 1536)
			src := rand.NewSource(int64(hashString(d.id)))
			r := rand.New(src)
			for i := range values {
				values[i] = r.Float32()*2 - 1
			}

			vectors = append(vectors, &store.Vector{
				ID:        d.id,
				Values:    values,
				Namespace: d.topic,
				Metadata: map[string]interface{}{
					"title": d.title,
					"topic": d.topic,
				},
			})
		}

		if err := s.store.Vectorize().Insert(ctx, "documents", vectors); err == nil {
			vectorCount += len(vectors)
		}
	}

	// Sample vectors for images index
	if _, ok := s.ids.VectorIndexes["images"]; ok {
		images := []struct {
			id   string
			name string
			tags []string
		}{
			{"img_001", "product_hero.jpg", []string{"hero", "product"}},
			{"img_002", "team_photo.jpg", []string{"team", "people"}},
			{"img_003", "office.jpg", []string{"office", "interior"}},
			{"img_004", "logo.png", []string{"logo", "branding"}},
			{"img_005", "banner.jpg", []string{"banner", "marketing"}},
		}

		var vectors []*store.Vector
		for _, img := range images {
			values := make([]float32, 512)
			src := rand.NewSource(int64(hashString(img.id)))
			r := rand.New(src)
			for i := range values {
				values[i] = r.Float32()*2 - 1
			}

			vectors = append(vectors, &store.Vector{
				ID:        img.id,
				Values:    values,
				Namespace: "images",
				Metadata: map[string]interface{}{
					"filename": img.name,
					"tags":     img.tags,
				},
			})
		}

		if err := s.store.Vectorize().Insert(ctx, "images", vectors); err == nil {
			vectorCount += len(vectors)
		}
	}

	slog.Info("vectorize seeded", "indexes", indexCount, "vectors", vectorCount)
	return nil
}

func (s *Seeder) seedAIGateway(ctx context.Context) error {
	slog.Info("seeding AI gateways")

	gwCount := 0
	logCount := 0

	// AI Gateways
	gateways := []*store.AIGateway{
		{
			ID:               generateID(),
			Name:             "openai-prod",
			CollectLogs:      true,
			CacheEnabled:     true,
			CacheTTL:         3600, // 1 hour
			RateLimitEnabled: true,
			RateLimitCount:   100,
			RateLimitPeriod:  60, // per minute
			CreatedAt:        s.timeAgo(30 * 24 * time.Hour),
		},
		{
			ID:               generateID(),
			Name:             "anthropic-dev",
			CollectLogs:      true,
			CacheEnabled:     false,
			CacheTTL:         0,
			RateLimitEnabled: true,
			RateLimitCount:   50,
			RateLimitPeriod:  60,
			CreatedAt:        s.timeAgo(20 * 24 * time.Hour),
		},
		{
			ID:               generateID(),
			Name:             "huggingface",
			CollectLogs:      false,
			CacheEnabled:     true,
			CacheTTL:         86400, // 24 hours
			RateLimitEnabled: true,
			RateLimitCount:   1000,
			RateLimitPeriod:  60,
			CreatedAt:        s.timeAgo(15 * 24 * time.Hour),
		},
	}

	for _, gw := range gateways {
		if err := s.store.AIGateway().CreateGateway(ctx, gw); err == nil {
			s.ids.AIGateways[gw.Name] = gw.ID
			gwCount++
		}
	}

	// Sample logs for gateways
	type logTemplate struct {
		provider string
		model    string
		cached   bool
		status   int
		tokens   int
		cost     float64
		duration int
	}

	// Generate logs for openai-prod gateway
	if gwID, ok := s.ids.AIGateways["openai-prod"]; ok {
		logTemplates := []logTemplate{
			{"openai", "gpt-4-turbo", false, 200, 1523, 0.0456, 2340},
			{"openai", "gpt-4-turbo", true, 200, 1523, 0.0, 45},
			{"openai", "gpt-3.5-turbo", false, 200, 856, 0.0012, 890},
			{"openai", "gpt-3.5-turbo", false, 200, 423, 0.0006, 567},
			{"openai", "text-embedding-3-small", false, 200, 256, 0.0001, 123},
			{"openai", "gpt-4-turbo", false, 429, 0, 0.0, 50},
			{"openai", "gpt-4-turbo", true, 200, 2048, 0.0, 38},
		}

		for i := 0; i < 20; i++ {
			tmpl := logTemplates[i%len(logTemplates)]
			req, _ := json.Marshal(map[string]interface{}{
				"model":    tmpl.model,
				"messages": []map[string]string{{"role": "user", "content": "Sample prompt"}},
			})
			resp, _ := json.Marshal(map[string]interface{}{
				"id":      "chatcmpl-xxx",
				"choices": []map[string]interface{}{{"message": map[string]string{"content": "Sample response"}}},
			})

			log := &store.AIGatewayLog{
				ID:        generateID(),
				GatewayID: gwID,
				Provider:  tmpl.provider,
				Model:     tmpl.model,
				Cached:    tmpl.cached,
				Status:    tmpl.status,
				Duration:  tmpl.duration + (i * 10),
				Tokens:    tmpl.tokens,
				Cost:      tmpl.cost,
				Request:   req,
				Response:  resp,
				Metadata:  map[string]string{"user": "user_001"},
				CreatedAt: s.timeAgo(time.Duration(i*5) * time.Minute),
			}
			if err := s.store.AIGateway().LogRequest(ctx, log); err == nil {
				logCount++
			}
		}
	}

	// Generate logs for anthropic-dev gateway
	if gwID, ok := s.ids.AIGateways["anthropic-dev"]; ok {
		logTemplates := []logTemplate{
			{"anthropic", "claude-3-opus", false, 200, 2048, 0.075, 3200},
			{"anthropic", "claude-3-sonnet", false, 200, 1024, 0.018, 1800},
			{"anthropic", "claude-3-haiku", false, 200, 512, 0.0012, 450},
			{"anthropic", "claude-3-sonnet", false, 500, 0, 0.0, 100},
		}

		for i := 0; i < 15; i++ {
			tmpl := logTemplates[i%len(logTemplates)]
			req, _ := json.Marshal(map[string]interface{}{
				"model":      tmpl.model,
				"max_tokens": 1024,
				"messages":   []map[string]string{{"role": "user", "content": "Test prompt"}},
			})
			resp, _ := json.Marshal(map[string]interface{}{
				"id":      "msg_xxx",
				"content": []map[string]string{{"text": "Test response"}},
			})

			log := &store.AIGatewayLog{
				ID:        generateID(),
				GatewayID: gwID,
				Provider:  tmpl.provider,
				Model:     tmpl.model,
				Cached:    false,
				Status:    tmpl.status,
				Duration:  tmpl.duration + (i * 15),
				Tokens:    tmpl.tokens,
				Cost:      tmpl.cost,
				Request:   req,
				Response:  resp,
				Metadata:  map[string]string{"env": "development"},
				CreatedAt: s.timeAgo(time.Duration(i*10) * time.Minute),
			}
			if err := s.store.AIGateway().LogRequest(ctx, log); err == nil {
				logCount++
			}
		}
	}

	slog.Info("AI gateway seeded", "gateways", gwCount, "logs", logCount)
	return nil
}

// hashString creates a simple hash for reproducible random generation
func hashString(s string) uint32 {
	var h uint32
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}
