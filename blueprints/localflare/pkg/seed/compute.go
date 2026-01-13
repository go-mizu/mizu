package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

func (s *Seeder) seedDurableObjects(ctx context.Context) error {
	slog.Info("seeding durable objects")

	nsCount := 0
	instanceCount := 0
	storageCount := 0

	// Durable Object Namespaces
	namespaces := []*store.DurableObjectNamespace{
		{
			ID:        generateID(),
			Name:      "COUNTERS",
			Script:    "counter-do",
			ClassName: "Counter",
			CreatedAt: s.timeAgo(30 * 24 * time.Hour),
		},
		{
			ID:        generateID(),
			Name:      "ROOMS",
			Script:    "chat-do",
			ClassName: "ChatRoom",
			CreatedAt: s.timeAgo(45 * 24 * time.Hour),
		},
		{
			ID:        generateID(),
			Name:      "SESSIONS",
			Script:    "session-do",
			ClassName: "UserSession",
			CreatedAt: s.timeAgo(60 * 24 * time.Hour),
		},
	}

	for _, ns := range namespaces {
		if err := s.store.DurableObjects().CreateNamespace(ctx, ns); err == nil {
			s.ids.DONamespaces[ns.Name] = ns.ID
			nsCount++
		}
	}

	// Durable Object Instances and Storage
	type instanceData struct {
		nsName     string
		objectID   string
		objectName string
		storage    map[string]interface{}
	}

	instances := []instanceData{
		{
			nsName:     "COUNTERS",
			objectID:   "global:requests",
			objectName: "global:requests",
			storage: map[string]interface{}{
				"count":      15234,
				"last_reset": s.timeAgo(24 * time.Hour).Unix(),
			},
		},
		{
			nsName:     "COUNTERS",
			objectID:   "ip:192.168.1.1",
			objectName: "ip:192.168.1.1",
			storage: map[string]interface{}{
				"count":        42,
				"window_start": s.timeAgo(30 * time.Minute).Unix(),
				"limit":        100,
			},
		},
		{
			nsName:     "COUNTERS",
			objectID:   "ip:192.168.1.50",
			objectName: "ip:192.168.1.50",
			storage: map[string]interface{}{
				"count":        8,
				"window_start": s.timeAgo(15 * time.Minute).Unix(),
				"limit":        100,
			},
		},
		{
			nsName:     "COUNTERS",
			objectID:   "user:123",
			objectName: "user:123",
			storage: map[string]interface{}{
				"count":        88,
				"window_start": s.timeAgo(1 * time.Hour).Unix(),
				"limit":        1000,
			},
		},
		{
			nsName:     "ROOMS",
			objectID:   "room:general",
			objectName: "room:general",
			storage: map[string]interface{}{
				"name":        "General Chat",
				"created_at":  s.timeAgo(7 * 24 * time.Hour).Unix(),
				"user_count":  12,
				"message_count": 1523,
				"users": []string{"user_001", "user_002", "user_003"},
			},
		},
		{
			nsName:     "ROOMS",
			objectID:   "room:support",
			objectName: "room:support",
			storage: map[string]interface{}{
				"name":        "Support Channel",
				"created_at":  s.timeAgo(14 * 24 * time.Hour).Unix(),
				"user_count":  5,
				"message_count": 342,
				"users": []string{"support_001", "user_005"},
			},
		},
		{
			nsName:     "ROOMS",
			objectID:   "room:dev",
			objectName: "room:dev",
			storage: map[string]interface{}{
				"name":        "Developer Lounge",
				"created_at":  s.timeAgo(3 * 24 * time.Hour).Unix(),
				"user_count":  8,
				"message_count": 256,
			},
		},
		{
			nsName:     "SESSIONS",
			objectID:   "user:abc123",
			objectName: "user:abc123",
			storage: map[string]interface{}{
				"cart": []map[string]interface{}{
					{"product_id": "ELEC-001", "quantity": 1, "price": 149.99},
					{"product_id": "BOOK-002", "quantity": 2, "price": 44.99},
				},
				"preferences": map[string]interface{}{
					"theme":        "dark",
					"language":     "en",
					"notifications": true,
				},
				"last_activity": s.timeAgo(5 * time.Minute).Unix(),
			},
		},
		{
			nsName:     "SESSIONS",
			objectID:   "user:def456",
			objectName: "user:def456",
			storage: map[string]interface{}{
				"cart": []map[string]interface{}{},
				"preferences": map[string]interface{}{
					"theme":        "light",
					"language":     "es",
					"notifications": false,
				},
				"last_activity": s.timeAgo(2 * time.Hour).Unix(),
			},
		},
		{
			nsName:     "SESSIONS",
			objectID:   "user:ghi789",
			objectName: "user:ghi789",
			storage: map[string]interface{}{
				"cart": []map[string]interface{}{
					{"product_id": "SPORT-001", "quantity": 1, "price": 29.99},
				},
				"preferences": map[string]interface{}{
					"theme":    "auto",
					"language": "en",
				},
				"last_activity": s.timeAgo(30 * time.Minute).Unix(),
			},
		},
	}

	for _, inst := range instances {
		nsID, ok := s.ids.DONamespaces[inst.nsName]
		if !ok {
			continue
		}

		// Create or get instance
		doInstance, err := s.store.DurableObjects().GetOrCreateInstance(ctx, nsID, inst.objectID, inst.objectName)
		if err != nil {
			continue
		}
		instanceCount++

		// Store data
		for key, value := range inst.storage {
			data, _ := json.Marshal(value)
			if err := s.store.DurableObjects().Put(ctx, doInstance.ID, key, data); err == nil {
				storageCount++
			}
		}
	}

	// Set some alarms
	if nsID, ok := s.ids.DONamespaces["COUNTERS"]; ok {
		if inst, err := s.store.DurableObjects().GetOrCreateInstance(ctx, nsID, "global:requests", "global:requests"); err == nil {
			// Reset counter at midnight
			s.store.DurableObjects().SetAlarm(ctx, inst.ID, s.timeFuture(24*time.Hour))
		}
	}

	slog.Info("durable objects seeded", "namespaces", nsCount, "instances", instanceCount, "storage_entries", storageCount)
	return nil
}

func (s *Seeder) seedQueues(ctx context.Context) error {
	slog.Info("seeding queues")

	queueCount := 0
	messageCount := 0
	consumerCount := 0

	// Queues
	queues := []*store.Queue{
		{
			ID:        generateID(),
			Name:      "events",
			CreatedAt: s.timeAgo(30 * 24 * time.Hour),
			Settings: store.QueueSettings{
				DeliveryDelay:   0,
				MessageTTL:      604800, // 7 days
				MaxRetries:      3,
				MaxBatchSize:    100,
				MaxBatchTimeout: 5,
			},
		},
		{
			ID:        generateID(),
			Name:      "emails",
			CreatedAt: s.timeAgo(45 * 24 * time.Hour),
			Settings: store.QueueSettings{
				DeliveryDelay:   0,
				MessageTTL:      86400, // 1 day
				MaxRetries:      5,
				MaxBatchSize:    50,
				MaxBatchTimeout: 10,
			},
		},
		{
			ID:        generateID(),
			Name:      "webhooks",
			CreatedAt: s.timeAgo(20 * 24 * time.Hour),
			Settings: store.QueueSettings{
				DeliveryDelay:   0,
				MessageTTL:      259200, // 3 days
				MaxRetries:      10,
				MaxBatchSize:    25,
				MaxBatchTimeout: 30,
			},
		},
		{
			ID:        generateID(),
			Name:      "dead-letter",
			CreatedAt: s.timeAgo(60 * 24 * time.Hour),
			Settings: store.QueueSettings{
				DeliveryDelay:   0,
				MessageTTL:      2592000, // 30 days
				MaxRetries:      0,
				MaxBatchSize:    10,
				MaxBatchTimeout: 60,
			},
		},
	}

	for _, q := range queues {
		if err := s.store.Queues().CreateQueue(ctx, q); err == nil {
			s.ids.Queues[q.Name] = q.ID
			queueCount++
		}
	}

	// Sample messages for events queue
	if queueID, ok := s.ids.Queues["events"]; ok {
		eventMessages := []map[string]interface{}{
			{"type": "page_view", "url": "/products/123", "user_id": "abc", "timestamp": s.timeAgo(5 * time.Minute).Unix()},
			{"type": "page_view", "url": "/cart", "user_id": "abc", "timestamp": s.timeAgo(4 * time.Minute).Unix()},
			{"type": "purchase", "order_id": "ord_123", "amount": 99.99, "timestamp": s.timeAgo(3 * time.Minute).Unix()},
			{"type": "signup", "user_id": "def", "source": "google", "timestamp": s.timeAgo(10 * time.Minute).Unix()},
			{"type": "page_view", "url": "/", "user_id": "ghi", "timestamp": s.timeAgo(2 * time.Minute).Unix()},
			{"type": "click", "element": "add-to-cart", "product_id": "ELEC-002", "timestamp": s.timeAgo(1 * time.Minute).Unix()},
		}
		for _, event := range eventMessages {
			body, _ := json.Marshal(event)
			msg := &store.QueueMessage{
				ID:          generateID(),
				QueueID:     queueID,
				Body:        body,
				ContentType: "json",
				Attempts:    0,
				CreatedAt:   s.now,
				VisibleAt:   s.now,
				ExpiresAt:   s.timeFuture(7 * 24 * time.Hour),
			}
			if err := s.store.Queues().SendMessage(ctx, queueID, msg); err == nil {
				messageCount++
			}
		}
	}

	// Sample messages for emails queue
	if queueID, ok := s.ids.Queues["emails"]; ok {
		emailMessages := []map[string]interface{}{
			{"template": "welcome", "to": "newuser@example.com", "data": map[string]string{"name": "John"}},
			{"template": "order_confirmation", "to": "buyer@example.com", "data": map[string]string{"order_id": "ord_123", "total": "$99.99"}},
			{"template": "password_reset", "to": "forgot@example.com", "data": map[string]string{"reset_link": "https://example.com/reset/abc"}},
		}
		for _, email := range emailMessages {
			body, _ := json.Marshal(email)
			msg := &store.QueueMessage{
				ID:          generateID(),
				QueueID:     queueID,
				Body:        body,
				ContentType: "json",
				Attempts:    0,
				CreatedAt:   s.now,
				VisibleAt:   s.now,
				ExpiresAt:   s.timeFuture(24 * time.Hour),
			}
			if err := s.store.Queues().SendMessage(ctx, queueID, msg); err == nil {
				messageCount++
			}
		}
	}

	// Sample messages for webhooks queue
	if queueID, ok := s.ids.Queues["webhooks"]; ok {
		webhookMessages := []map[string]interface{}{
			{"url": "https://partner.example.com/webhook", "event": "order.created", "payload": map[string]interface{}{"order_id": "ord_123"}},
			{"url": "https://analytics.example.com/track", "event": "user.signup", "payload": map[string]interface{}{"user_id": "def"}},
		}
		for _, webhook := range webhookMessages {
			body, _ := json.Marshal(webhook)
			msg := &store.QueueMessage{
				ID:          generateID(),
				QueueID:     queueID,
				Body:        body,
				ContentType: "json",
				Attempts:    0,
				CreatedAt:   s.now,
				VisibleAt:   s.now,
				ExpiresAt:   s.timeFuture(3 * 24 * time.Hour),
			}
			if err := s.store.Queues().SendMessage(ctx, queueID, msg); err == nil {
				messageCount++
			}
		}
	}

	// Consumers
	consumers := []struct {
		queueName string
		consumer  *store.QueueConsumer
	}{
		{
			queueName: "events",
			consumer: &store.QueueConsumer{
				ID:              generateID(),
				ScriptName:      "analytics-worker",
				Type:            "worker",
				MaxBatchSize:    100,
				MaxBatchTimeout: 5,
				MaxRetries:      3,
				DeadLetterQueue: s.ids.Queues["dead-letter"],
				CreatedAt:       s.now,
			},
		},
		{
			queueName: "emails",
			consumer: &store.QueueConsumer{
				ID:              generateID(),
				ScriptName:      "email-worker",
				Type:            "worker",
				MaxBatchSize:    50,
				MaxBatchTimeout: 10,
				MaxRetries:      5,
				DeadLetterQueue: s.ids.Queues["dead-letter"],
				CreatedAt:       s.now,
			},
		},
		{
			queueName: "webhooks",
			consumer: &store.QueueConsumer{
				ID:              generateID(),
				ScriptName:      "webhook-worker",
				Type:            "worker",
				MaxBatchSize:    25,
				MaxBatchTimeout: 30,
				MaxRetries:      10,
				DeadLetterQueue: s.ids.Queues["dead-letter"],
				CreatedAt:       s.now,
			},
		},
	}

	for _, c := range consumers {
		queueID, ok := s.ids.Queues[c.queueName]
		if !ok {
			continue
		}
		c.consumer.QueueID = queueID
		if err := s.store.Queues().CreateConsumer(ctx, c.consumer); err == nil {
			consumerCount++
		}
	}

	slog.Info("queues seeded", "queues", queueCount, "messages", messageCount, "consumers", consumerCount)
	return nil
}

func (s *Seeder) seedCron(ctx context.Context) error {
	slog.Info("seeding cron triggers")

	triggerCount := 0
	execCount := 0

	// Cron Triggers
	triggers := []*store.CronTrigger{
		{
			ID:         generateID(),
			ScriptName: "daily-cleanup",
			Cron:       "0 0 * * *",
			Enabled:    true,
			CreatedAt:  s.timeAgo(30 * 24 * time.Hour),
			UpdatedAt:  s.now,
		},
		{
			ID:         generateID(),
			ScriptName: "hourly-sync",
			Cron:       "0 * * * *",
			Enabled:    true,
			CreatedAt:  s.timeAgo(45 * 24 * time.Hour),
			UpdatedAt:  s.now,
		},
		{
			ID:         generateID(),
			ScriptName: "weekly-report",
			Cron:       "0 9 * * 1",
			Enabled:    true,
			CreatedAt:  s.timeAgo(60 * 24 * time.Hour),
			UpdatedAt:  s.now,
		},
		{
			ID:         generateID(),
			ScriptName: "health-check",
			Cron:       "*/5 * * * *",
			Enabled:    true,
			CreatedAt:  s.timeAgo(90 * 24 * time.Hour),
			UpdatedAt:  s.now,
		},
		{
			ID:         generateID(),
			ScriptName: "maintenance",
			Cron:       "0 3 * * 0",
			Enabled:    false,
			CreatedAt:  s.timeAgo(120 * 24 * time.Hour),
			UpdatedAt:  s.now,
		},
	}

	for _, trigger := range triggers {
		if err := s.store.Cron().CreateTrigger(ctx, trigger); err == nil {
			s.ids.CronTriggers[fmt.Sprintf("%s:%s", trigger.ScriptName, trigger.Cron)] = trigger.ID
			triggerCount++
		}
	}

	// Generate execution history
	type execConfig struct {
		triggerKey  string
		count       int
		interval    time.Duration
		failureRate float64
	}

	execConfigs := []execConfig{
		{"daily-cleanup:0 0 * * *", 1, 24 * time.Hour, 0.0},           // 1 execution, no failures
		{"hourly-sync:0 * * * *", 24, 1 * time.Hour, 0.08},            // 24 executions, ~8% failure
		{"health-check:*/5 * * * *", 288, 5 * time.Minute, 0.01},      // 288 executions, ~1% failure
	}

	for _, ec := range execConfigs {
		triggerID, ok := s.ids.CronTriggers[ec.triggerKey]
		if !ok {
			continue
		}

		for i := 0; i < ec.count; i++ {
			scheduledAt := s.timeAgo(time.Duration(i) * ec.interval)
			startedAt := scheduledAt.Add(100 * time.Millisecond)
			finishedAt := startedAt.Add(time.Duration(50+i%200) * time.Millisecond)

			// Determine if this execution failed
			status := "success"
			errMsg := ""
			if float64(i%100)/100.0 < ec.failureRate {
				status = "failed"
				errMsg = "Connection timeout"
			}

			exec := &store.CronExecution{
				ID:          generateID(),
				TriggerID:   triggerID,
				ScheduledAt: scheduledAt,
				StartedAt:   startedAt,
				FinishedAt:  &finishedAt,
				Status:      status,
				Error:       errMsg,
			}
			if err := s.store.Cron().RecordExecution(ctx, exec); err == nil {
				execCount++
			}
		}
	}

	slog.Info("cron seeded", "triggers", triggerCount, "executions", execCount)
	return nil
}
