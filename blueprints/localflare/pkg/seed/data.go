package seed

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedAnalytics(ctx context.Context) error {
	slog.Info("seeding analytics data")

	dataCount := 0

	// Base traffic patterns for each zone
	zoneTraffic := map[string]struct {
		baseRequests  int64
		baseBandwidth int64
		baseVisitors  int64
		cacheHitRate  float64
		threatRate    float64
	}{
		"example.com": {
			baseRequests:  5000,
			baseBandwidth: 100 * 1024 * 1024, // 100 MB
			baseVisitors:  500,
			cacheHitRate:  0.85,
			threatRate:    0.001,
		},
		"api.myapp.io": {
			baseRequests:  20000,
			baseBandwidth: 50 * 1024 * 1024, // 50 MB
			baseVisitors:  2000,
			cacheHitRate:  0.6,
			threatRate:    0.005,
		},
		"store.acme.co": {
			baseRequests:  15000,
			baseBandwidth: 200 * 1024 * 1024, // 200 MB
			baseVisitors:  3000,
			cacheHitRate:  0.75,
			threatRate:    0.002,
		},
		"internal.corp": {
			baseRequests:  1000,
			baseBandwidth: 20 * 1024 * 1024, // 20 MB
			baseVisitors:  50,
			cacheHitRate:  0.3,
			threatRate:    0.0,
		},
	}

	// Generate 7 days of hourly data for each zone
	for zoneName, traffic := range zoneTraffic {
		zoneID, ok := s.ids.Zones[zoneName]
		if !ok {
			continue
		}

		// 7 days * 24 hours = 168 data points
		for h := 0; h < 168; h++ {
			ts := s.timeAgo(time.Duration(h) * time.Hour)

			// Apply time-of-day multiplier (peak hours: 9-17, low: 0-6)
			hour := ts.Hour()
			var multiplier float64
			switch {
			case hour >= 9 && hour <= 17:
				multiplier = 1.5
			case hour >= 18 && hour <= 22:
				multiplier = 1.2
			case hour >= 0 && hour <= 6:
				multiplier = 0.3
			default:
				multiplier = 0.8
			}

			// Apply weekend reduction
			if ts.Weekday() == time.Saturday || ts.Weekday() == time.Sunday {
				multiplier *= 0.6
			}

			// Add some randomness
			randomFactor := 0.8 + rand.Float64()*0.4 // 0.8 to 1.2

			requests := int64(float64(traffic.baseRequests) * multiplier * randomFactor)
			bandwidth := int64(float64(traffic.baseBandwidth) * multiplier * randomFactor)
			visitors := int64(float64(traffic.baseVisitors) * multiplier * randomFactor)

			cacheHits := int64(float64(requests) * traffic.cacheHitRate)
			cacheMisses := requests - cacheHits
			threats := int64(float64(requests) * traffic.threatRate)
			pageViews := int64(float64(visitors) * (2.0 + rand.Float64()*2.0)) // 2-4 pages per visitor

			// Status code distribution
			status2xx := int64(float64(requests) * (0.90 + rand.Float64()*0.05))
			status3xx := int64(float64(requests) * (0.03 + rand.Float64()*0.02))
			status4xx := int64(float64(requests) * (0.02 + rand.Float64()*0.02))
			status5xx := requests - status2xx - status3xx - status4xx

			data := &store.AnalyticsData{
				Timestamp:    ts,
				Requests:     requests,
				Bandwidth:    bandwidth,
				Threats:      threats,
				PageViews:    pageViews,
				UniqueVisits: visitors,
				CacheHits:    cacheHits,
				CacheMisses:  cacheMisses,
				StatusCodes: map[int]int64{
					200: int64(float64(status2xx) * 0.95),
					201: int64(float64(status2xx) * 0.03),
					204: int64(float64(status2xx) * 0.02),
					301: int64(float64(status3xx) * 0.3),
					304: int64(float64(status3xx) * 0.7),
					400: int64(float64(status4xx) * 0.2),
					401: int64(float64(status4xx) * 0.15),
					403: int64(float64(status4xx) * 0.1),
					404: int64(float64(status4xx) * 0.5),
					429: int64(float64(status4xx) * 0.05),
					500: int64(float64(status5xx) * 0.6),
					502: int64(float64(status5xx) * 0.2),
					503: int64(float64(status5xx) * 0.15),
					504: int64(float64(status5xx) * 0.05),
				},
			}

			if err := s.store.Analytics().Record(ctx, zoneID, data); err == nil {
				dataCount++
			}
		}
	}

	slog.Info("analytics seeded", "data_points", dataCount)
	return nil
}

func (s *Seeder) seedAnalyticsEngine(ctx context.Context) error {
	slog.Info("seeding analytics engine datasets")

	datasetCount := 0
	pointCount := 0

	// Analytics Engine Datasets
	datasets := []*store.AnalyticsEngineDataset{
		{ID: generateID(), Name: "web_analytics", CreatedAt: s.timeAgo(60 * 24 * time.Hour)},
		{ID: generateID(), Name: "api_metrics", CreatedAt: s.timeAgo(45 * 24 * time.Hour)},
		{ID: generateID(), Name: "business_events", CreatedAt: s.timeAgo(30 * 24 * time.Hour)},
	}

	for _, ds := range datasets {
		if err := s.store.AnalyticsEngine().CreateDataset(ctx, ds); err == nil {
			s.ids.Datasets[ds.Name] = ds.ID
			datasetCount++
		}
	}

	// Sample data points for web_analytics
	pages := []string{"/", "/products", "/about", "/contact", "/blog", "/pricing", "/cart", "/checkout"}
	countries := []string{"US", "GB", "DE", "FR", "JP", "AU", "CA", "BR"}
	devices := []string{"desktop", "mobile", "tablet"}
	browsers := []string{"Chrome", "Firefox", "Safari", "Edge"}

	// Generate 100 data points per hour for last 24 hours = 2400 points
	for h := 0; h < 24; h++ {
		baseTime := s.timeAgo(time.Duration(h) * time.Hour)

		for i := 0; i < 100; i++ {
			ts := baseTime.Add(time.Duration(rand.Intn(3600)) * time.Second)
			loadTime := 100 + rand.Float64()*2000  // 100ms to 2100ms
			ttfb := 20 + rand.Float64()*200        // 20ms to 220ms

			point := &store.AnalyticsEngineDataPoint{
				Dataset:   "web_analytics",
				Timestamp: ts,
				Indexes: []string{
					pages[rand.Intn(len(pages))],
					countries[rand.Intn(len(countries))],
					devices[rand.Intn(len(devices))],
				},
				Doubles: []float64{
					loadTime,
					ttfb,
				},
				Blobs: [][]byte{
					[]byte(browsers[rand.Intn(len(browsers))]),
				},
			}

			if err := s.store.AnalyticsEngine().WriteDataPoint(ctx, point); err == nil {
				pointCount++
			}
		}
	}

	// Sample data points for api_metrics
	endpoints := []string{"/api/users", "/api/products", "/api/orders", "/api/auth", "/api/search"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	statusCodes := []string{"200", "201", "400", "401", "404", "500"}

	for h := 0; h < 24; h++ {
		baseTime := s.timeAgo(time.Duration(h) * time.Hour)

		for i := 0; i < 50; i++ {
			ts := baseTime.Add(time.Duration(rand.Intn(3600)) * time.Second)
			latency := 5 + rand.Float64()*500    // 5ms to 505ms
			bodySize := float64(rand.Intn(50000)) // 0 to 50KB

			point := &store.AnalyticsEngineDataPoint{
				Dataset:   "api_metrics",
				Timestamp: ts,
				Indexes: []string{
					endpoints[rand.Intn(len(endpoints))],
					methods[rand.Intn(len(methods))],
					statusCodes[rand.Intn(len(statusCodes))],
				},
				Doubles: []float64{
					latency,
					bodySize,
				},
				Blobs: nil,
			}

			if err := s.store.AnalyticsEngine().WriteDataPoint(ctx, point); err == nil {
				pointCount++
			}
		}
	}

	// Sample data points for business_events
	eventTypes := []string{"page_view", "signup", "login", "purchase", "add_to_cart", "checkout_started"}
	sources := []string{"organic", "paid", "referral", "direct", "social", "email"}

	for h := 0; h < 24; h++ {
		baseTime := s.timeAgo(time.Duration(h) * time.Hour)

		for i := 0; i < 30; i++ {
			ts := baseTime.Add(time.Duration(rand.Intn(3600)) * time.Second)

			var value float64
			eventType := eventTypes[rand.Intn(len(eventTypes))]
			if eventType == "purchase" {
				value = 10 + rand.Float64()*500 // $10 to $510
			}

			point := &store.AnalyticsEngineDataPoint{
				Dataset:   "business_events",
				Timestamp: ts,
				Indexes: []string{
					eventType,
					sources[rand.Intn(len(sources))],
				},
				Doubles: []float64{
					value,
					float64(rand.Intn(10) + 1), // count
				},
				Blobs: [][]byte{
					[]byte(fmt.Sprintf("user_%03d", rand.Intn(100)+1)),
				},
			}

			if err := s.store.AnalyticsEngine().WriteDataPoint(ctx, point); err == nil {
				pointCount++
			}
		}
	}

	slog.Info("analytics engine seeded", "datasets", datasetCount, "points", pointCount)
	return nil
}
