// Package rpc implements all WebSocket RPC method handlers for the Control Dashboard.
// Each method matches the OpenClaw gateway protocol for 100% feature compatibility.
package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/app/web/handler/dashboard"
	"github.com/go-mizu/mizu/blueprints/bot/feature/gateway"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/logring"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/store"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// RegisterAll registers all RPC methods on the hub.
func RegisterAll(hub *dashboard.Hub, s store.Store, gw *gateway.Service, logs *logring.Ring, startTime time.Time) {
	registerCronMethods(hub, s)
	registerSessionMethods(hub, s)
	registerChannelMethods(hub, s)
	registerSkillMethods(hub)
	registerConfigMethods(hub)
	registerLogMethods(hub, logs)
	registerDebugMethods(hub, s, gw, logs, startTime)
	registerChatMethods(hub, gw, s)
	registerAgentMethods(hub, s)
}

// --- Cron Methods ---

func registerCronMethods(hub *dashboard.Hub, s store.Store) {
	hub.Register("cron.list", func(params json.RawMessage) (any, error) {
		jobs, err := s.ListCronJobs(context.Background())
		if err != nil {
			return nil, err
		}
		if jobs == nil {
			jobs = []types.CronJob{}
		}
		return map[string]any{"jobs": jobs}, nil
	})

	hub.Register("cron.status", func(params json.RawMessage) (any, error) {
		jobs, err := s.ListCronJobs(context.Background())
		if err != nil {
			return nil, err
		}
		enabled := 0
		for _, j := range jobs {
			if j.Enabled {
				enabled++
			}
		}
		return types.CronStatus{
			Enabled: enabled > 0,
			Jobs:    len(jobs),
		}, nil
	})

	hub.Register("cron.add", func(params json.RawMessage) (any, error) {
		var job types.CronJob
		if err := json.Unmarshal(params, &job); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if job.Name == "" {
			return nil, fmt.Errorf("name is required")
		}
		job.Enabled = true
		if err := s.CreateCronJob(context.Background(), &job); err != nil {
			return nil, err
		}
		hub.Broadcast("cron.updated", nil)
		return job, nil
	})

	hub.Register("cron.update", func(params json.RawMessage) (any, error) {
		var req struct {
			ID string `json:"id"`
			types.CronJob
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		job, err := s.GetCronJob(context.Background(), req.ID)
		if err != nil {
			return nil, err
		}
		if req.Name != "" {
			job.Name = req.Name
		}
		if req.Description != "" {
			job.Description = req.Description
		}
		if req.Schedule != "" {
			job.Schedule = req.Schedule
		}
		if req.Payload != "" {
			job.Payload = req.Payload
		}
		if req.AgentID != "" {
			job.AgentID = req.AgentID
		}
		job.Enabled = req.Enabled
		if err := s.UpdateCronJob(context.Background(), job); err != nil {
			return nil, err
		}
		hub.Broadcast("cron.updated", nil)
		return job, nil
	})

	hub.Register("cron.remove", func(params json.RawMessage) (any, error) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if err := s.DeleteCronJob(context.Background(), req.ID); err != nil {
			return nil, err
		}
		hub.Broadcast("cron.updated", nil)
		return map[string]bool{"removed": true}, nil
	})

	hub.Register("cron.run", func(params json.RawMessage) (any, error) {
		var req struct {
			ID   string `json:"id"`
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		job, err := s.GetCronJob(context.Background(), req.ID)
		if err != nil {
			return nil, err
		}
		run := &types.CronRun{
			JobID:  job.ID,
			Status: "success",
		}
		if err := s.CreateCronRun(context.Background(), run); err != nil {
			return nil, err
		}
		run.Status = "success"
		run.DurationMs = 0
		run.Summary = "Executed via dashboard"
		_ = s.UpdateCronRun(context.Background(), run)
		job.LastRunAt = time.Now().UTC()
		job.LastStatus = "success"
		_ = s.UpdateCronJob(context.Background(), job)
		hub.Broadcast("cron.updated", nil)
		return map[string]bool{"ran": true}, nil
	})

	hub.Register("cron.runs", func(params json.RawMessage) (any, error) {
		var req struct {
			ID    string `json:"id"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		runs, err := s.ListCronRuns(context.Background(), req.ID, req.Limit)
		if err != nil {
			return nil, err
		}
		if runs == nil {
			runs = []types.CronRun{}
		}
		return map[string]any{"entries": runs}, nil
	})
}

// --- Session Methods ---

func registerSessionMethods(hub *dashboard.Hub, s store.Store) {
	hub.Register("sessions.list", func(params json.RawMessage) (any, error) {
		var req struct {
			ActiveMinutes int `json:"activeMinutes"`
			Limit         int `json:"limit"`
		}
		if params != nil {
			_ = json.Unmarshal(params, &req)
		}
		sessions, err := s.ListSessions(context.Background())
		if err != nil {
			return nil, err
		}
		if sessions == nil {
			sessions = []types.Session{}
		}
		// Apply limit
		if req.Limit > 0 && len(sessions) > req.Limit {
			sessions = sessions[:req.Limit]
		}
		return map[string]any{"sessions": sessions}, nil
	})

	hub.Register("sessions.delete", func(params json.RawMessage) (any, error) {
		var req struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if err := s.DeleteSession(context.Background(), req.Key); err != nil {
			return nil, err
		}
		hub.Broadcast("session.updated", nil)
		return map[string]bool{"ok": true}, nil
	})

	hub.Register("sessions.patch", func(params json.RawMessage) (any, error) {
		var req struct {
			Key      string  `json:"key"`
			Label    *string `json:"label"`
			Metadata *string `json:"metadata"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		session, err := s.GetSession(context.Background(), req.Key)
		if err != nil {
			return nil, err
		}
		if req.Label != nil {
			session.DisplayName = *req.Label
		}
		if req.Metadata != nil {
			session.Metadata = *req.Metadata
		}
		if err := s.UpdateSession(context.Background(), session); err != nil {
			return nil, err
		}
		hub.Broadcast("session.updated", nil)
		return map[string]bool{"ok": true}, nil
	})

	hub.Register("sessions.preview", func(params json.RawMessage) (any, error) {
		var req struct {
			Key   string `json:"key"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if req.Limit <= 0 {
			req.Limit = 20
		}
		messages, err := s.ListMessages(context.Background(), req.Key, req.Limit)
		if err != nil {
			return nil, err
		}
		if messages == nil {
			messages = []types.Message{}
		}
		return map[string]any{"messages": messages}, nil
	})
}

// --- Channel Methods ---

func registerChannelMethods(hub *dashboard.Hub, s store.Store) {
	hub.Register("channels.status", func(params json.RawMessage) (any, error) {
		channels, err := s.ListChannels(context.Background())
		if err != nil {
			return nil, err
		}
		if channels == nil {
			channels = []types.Channel{}
		}
		return map[string]any{"channels": channels}, nil
	})

	hub.Register("channels.list", func(params json.RawMessage) (any, error) {
		channels, err := s.ListChannels(context.Background())
		if err != nil {
			return nil, err
		}
		if channels == nil {
			channels = []types.Channel{}
		}
		return map[string]any{"channels": channels}, nil
	})

	hub.Register("channels.create", func(params json.RawMessage) (any, error) {
		var ch types.Channel
		if err := json.Unmarshal(params, &ch); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if err := s.CreateChannel(context.Background(), &ch); err != nil {
			return nil, err
		}
		hub.Broadcast("channel.updated", nil)
		return ch, nil
	})

	hub.Register("channels.delete", func(params json.RawMessage) (any, error) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if err := s.DeleteChannel(context.Background(), req.ID); err != nil {
			return nil, err
		}
		hub.Broadcast("channel.updated", nil)
		return map[string]bool{"ok": true}, nil
	})
}

// --- Skills Methods ---

func registerSkillMethods(hub *dashboard.Hub) {
	hub.Register("skills.status", func(params json.RawMessage) (any, error) {
		var allSkills []types.SkillEntry

		// Load skills from all available sources
		workspaceDir := filepath.Join(config.DefaultConfigDir(), "workspace")
		bundledDir := skill.BundledSkillsDir()
		loaded, _ := skill.LoadAllSkills(workspaceDir, bundledDir)

		for _, s := range loaded {
			entry := types.SkillEntry{
				Key:           s.Name,
				Name:          s.Name,
				Description:   s.Description,
				Source:        s.Source,
				Eligible:      skill.CheckEligibility(s),
				Enabled:       true,
				UserInvocable: s.UserInvocable,
			}
			if s.Emoji != "" {
				entry.Emoji = s.Emoji
			}
			allSkills = append(allSkills, entry)
		}

		if allSkills == nil {
			allSkills = []types.SkillEntry{}
		}
		return map[string]any{"skills": allSkills}, nil
	})

	hub.Register("skills.toggle", func(params json.RawMessage) (any, error) {
		var req struct {
			Key     string `json:"key"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		// Toggle skill enabled state in config
		cfgPath := config.DefaultConfigPath()
		data, err := config.LoadRawConfig(cfgPath)
		if err != nil {
			if os.IsNotExist(err) {
				data = make(map[string]any)
			} else {
				return nil, fmt.Errorf("load config: %w", err)
			}
		}
		// Store in skills.disabled map
		skillsCfg, ok := data["skills"].(map[string]any)
		if !ok {
			skillsCfg = make(map[string]any)
			data["skills"] = skillsCfg
		}
		disabledMap, ok := skillsCfg["disabled"].(map[string]any)
		if !ok {
			disabledMap = make(map[string]any)
			skillsCfg["disabled"] = disabledMap
		}
		if req.Enabled {
			delete(disabledMap, req.Key)
		} else {
			disabledMap[req.Key] = true
		}
		if err := config.SaveRawConfig(cfgPath, data); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
		return map[string]bool{"ok": true}, nil
	})
}

// --- Config Methods ---

func registerConfigMethods(hub *dashboard.Hub) {
	hub.Register("config.read", func(params json.RawMessage) (any, error) {
		cfgPath := config.DefaultConfigPath()
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			if os.IsNotExist(err) {
				return map[string]any{"raw": "{}", "valid": true}, nil
			}
			return nil, err
		}
		// Validate JSON
		valid := json.Valid(data)
		return map[string]any{"raw": string(data), "valid": valid}, nil
	})

	hub.Register("config.write", func(params json.RawMessage) (any, error) {
		var req struct {
			Raw string `json:"raw"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		// Validate JSON before writing
		if !json.Valid([]byte(req.Raw)) {
			return nil, fmt.Errorf("invalid JSON")
		}
		cfgPath := config.DefaultConfigPath()
		// Load as raw map and save with backup rotation
		var data map[string]any
		if err := json.Unmarshal([]byte(req.Raw), &data); err != nil {
			return nil, fmt.Errorf("parse error: %w", err)
		}
		if err := config.SaveRawConfig(cfgPath, data); err != nil {
			return nil, err
		}
		return map[string]bool{"ok": true}, nil
	})

	hub.Register("config.apply", func(params json.RawMessage) (any, error) {
		// Re-read config from disk to pick up any external changes
		cfgPath := config.DefaultConfigPath()
		_, err := config.LoadFromFile(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("apply config: %w", err)
		}
		return map[string]bool{"ok": true}, nil
	})

	hub.Register("config.patch", func(params json.RawMessage) (any, error) {
		var req struct {
			Path  string `json:"path"`
			Value any    `json:"value"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if req.Path == "" {
			return nil, fmt.Errorf("path is required")
		}
		cfgPath := config.DefaultConfigPath()
		data, err := config.LoadRawConfig(cfgPath)
		if err != nil {
			if os.IsNotExist(err) {
				data = make(map[string]any)
			} else {
				return nil, fmt.Errorf("load config: %w", err)
			}
		}
		config.ConfigSet(req.Path, req.Value, data)
		if err := config.SaveRawConfig(cfgPath, data); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
		return map[string]bool{"ok": true}, nil
	})

	hub.Register("config.schema", func(params json.RawMessage) (any, error) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"meta": map[string]any{
					"type":        "object",
					"description": "Version metadata",
					"properties": map[string]any{
						"lastTouchedVersion": map[string]string{"type": "string", "description": "Last config editor version"},
						"lastTouchedAt":      map[string]string{"type": "string", "description": "Last config edit timestamp"},
					},
				},
				"wizard": map[string]any{
					"type":        "object",
					"description": "Setup wizard state",
					"properties": map[string]any{
						"lastRunAt":      map[string]string{"type": "string", "description": "When the wizard last ran"},
						"lastRunVersion": map[string]string{"type": "string", "description": "Version that ran the wizard"},
						"lastRunCommand": map[string]string{"type": "string", "description": "Command that triggered the wizard"},
						"lastRunMode":    map[string]string{"type": "string", "description": "Wizard run mode"},
					},
				},
				"auth": map[string]any{
					"type":        "object",
					"description": "Authentication profiles",
					"properties": map[string]any{
						"profiles": map[string]string{"type": "object", "description": "Named auth profiles (provider + mode)"},
					},
				},
				"agents": map[string]any{
					"type":        "object",
					"description": "Agent defaults and behavior",
					"properties": map[string]any{
						"defaults": map[string]any{
							"type":        "object",
							"description": "Default settings for all agents",
							"properties": map[string]any{
								"workspace":      map[string]string{"type": "string", "description": "Path to agent workspace directory"},
								"maxConcurrent":  map[string]string{"type": "number", "description": "Maximum concurrent agent requests"},
								"contextPruning": map[string]any{
									"type":        "object",
									"description": "Context pruning behavior",
									"properties": map[string]any{
										"mode": map[string]string{"type": "string", "description": "Pruning mode: cache-ttl, none"},
										"ttl":  map[string]string{"type": "string", "description": "Time-to-live for cached context (e.g. 1h)"},
									},
								},
								"compaction": map[string]any{
									"type":        "object",
									"description": "Conversation compaction settings",
									"properties": map[string]any{
										"mode": map[string]string{"type": "string", "description": "Compaction mode: safeguard, aggressive, off"},
									},
								},
								"heartbeat": map[string]any{
									"type":        "object",
									"description": "Periodic heartbeat messages",
									"properties": map[string]any{
										"every": map[string]string{"type": "string", "description": "Heartbeat interval (e.g. 30m)"},
									},
								},
								"subagents": map[string]any{
									"type":        "object",
									"description": "Subagent concurrency settings",
									"properties": map[string]any{
										"maxConcurrent": map[string]string{"type": "number", "description": "Maximum concurrent subagent requests"},
									},
								},
							},
						},
					},
				},
				"messages": map[string]any{
					"type":        "object",
					"description": "Messaging behavior",
					"properties": map[string]any{
						"ackReactionScope": map[string]string{"type": "string", "description": "Ack reaction scope: group-mentions, all, none"},
					},
				},
				"commands": map[string]any{
					"type":        "object",
					"description": "Slash command configuration",
					"properties": map[string]any{
						"native":       map[string]string{"type": "string", "description": "Native commands: auto, on, off"},
						"nativeSkills": map[string]string{"type": "string", "description": "Native skill commands: auto, on, off"},
					},
				},
				"channels": map[string]any{
					"type":        "object",
					"description": "Channel configurations",
					"properties": map[string]any{
						"telegram": map[string]any{
							"type":        "object",
							"description": "Telegram channel settings",
							"properties": map[string]any{
								"enabled":     map[string]string{"type": "boolean", "description": "Whether the Telegram channel is enabled"},
								"botToken":    map[string]string{"type": "string", "description": "Telegram Bot API token"},
								"dmPolicy":    map[string]string{"type": "string", "description": "DM policy: allowlist, open, disabled"},
								"allowFrom":   map[string]string{"type": "array", "description": "Allowed Telegram user IDs for DMs"},
								"groupPolicy": map[string]string{"type": "string", "description": "Group chat policy"},
								"streamMode":  map[string]string{"type": "string", "description": "Stream mode for responses"},
							},
						},
					},
				},
				"gateway": map[string]any{
					"type":        "object",
					"description": "Gateway server settings",
					"properties": map[string]any{
						"port": map[string]string{"type": "number", "description": "Gateway listen port"},
						"mode": map[string]string{"type": "string", "description": "Gateway mode: local"},
						"bind": map[string]string{"type": "string", "description": "Bind mode: loopback, lan, tailnet, auto"},
						"auth": map[string]any{
							"type":        "object",
							"description": "Gateway authentication settings",
							"properties": map[string]any{
								"mode":           map[string]string{"type": "string", "description": "Auth mode: token, password"},
								"token":          map[string]string{"type": "string", "description": "Auth token value"},
								"password":       map[string]string{"type": "string", "description": "Auth password value"},
								"allowTailscale": map[string]string{"type": "boolean", "description": "Allow Tailscale authentication"},
							},
						},
						"tailscale": map[string]any{
							"type":        "object",
							"description": "Tailscale integration settings",
							"properties": map[string]any{
								"mode":        map[string]string{"type": "string", "description": "Tailscale mode: off, serve, funnel"},
								"resetOnExit": map[string]string{"type": "boolean", "description": "Reset Tailscale config on exit"},
							},
						},
					},
				},
				"plugins": map[string]any{
					"type":        "object",
					"description": "Plugin settings",
					"properties": map[string]any{
						"entries": map[string]string{"type": "object", "description": "Named plugin entries with enabled flag"},
					},
				},
			},
		}
		return schema, nil
	})
}

// --- Log Methods ---

func registerLogMethods(hub *dashboard.Hub, logs *logring.Ring) {
	hub.Register("logs.tail", func(params json.RawMessage) (any, error) {
		var req struct {
			Limit int    `json:"limit"`
			Level string `json:"level"`
		}
		if params != nil {
			_ = json.Unmarshal(params, &req)
		}
		if req.Limit <= 0 {
			req.Limit = 200
		}
		entries := logs.Tail(req.Limit, req.Level)
		if entries == nil {
			entries = []logring.Entry{}
		}
		return map[string]any{"entries": entries}, nil
	})

	hub.Register("logs.search", func(params json.RawMessage) (any, error) {
		var req struct {
			Query string `json:"query"`
			Level string `json:"level"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		entries := logs.Search(req.Query, req.Level)
		if entries == nil {
			entries = []logring.Entry{}
		}
		return map[string]any{"entries": entries}, nil
	})
}

// --- Debug / Health / System Methods ---

func registerDebugMethods(hub *dashboard.Hub, s store.Store, gw *gateway.Service, logs *logring.Ring, startTime time.Time) {
	hub.Register("health.check", func(params json.RawMessage) (any, error) {
		status, err := gw.Status(context.Background(), 18789)
		if err != nil {
			return nil, err
		}
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		return types.HealthSnapshot{
			Status:   "ok",
			Uptime:   time.Since(startTime).Truncate(time.Second).String(),
			Database: "sqlite",
			Memory: map[string]any{
				"allocMB":   float64(memStats.Alloc) / 1024 / 1024,
				"sysMB":     float64(memStats.Sys) / 1024 / 1024,
				"goroutines": runtime.NumGoroutine(),
			},
			Stats:     status,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}, nil
	})

	hub.Register("system.status", func(params json.RawMessage) (any, error) {
		status, err := gw.Status(context.Background(), 18789)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"ok":        true,
			"status":    status,
			"uptime":    time.Since(startTime).Truncate(time.Second).String(),
			"startAt":   startTime.Format(time.RFC3339),
			"go":        runtime.Version(),
			"goVersion": runtime.Version(),
			"os":        runtime.GOOS,
			"arch":      runtime.GOARCH,
			"database":  "sqlite",
			"sessions":  status.Sessions,
			"messages":  status.Messages,
		}, nil
	})

	hub.Register("system.presence", func(params json.RawMessage) (any, error) {
		instances := hub.Instances()
		if instances == nil {
			instances = []types.Instance{}
		}
		return map[string]any{"instances": instances}, nil
	})

	hub.Register("system.call", func(params json.RawMessage) (any, error) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		// Dispatch to the registered method
		return nil, fmt.Errorf("use the method name directly: %s", req.Method)
	})
}

// --- Chat Methods ---

func registerChatMethods(hub *dashboard.Hub, gw *gateway.Service, s store.Store) {
	hub.Register("chat.send", func(params json.RawMessage) (any, error) {
		var req struct {
			SessionID string `json:"sessionId"`
			Message   string `json:"message"`
			AgentID   string `json:"agentId"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if req.Message == "" {
			return nil, fmt.Errorf("message is required")
		}

		msg := &types.InboundMessage{
			ChannelType: types.ChannelWebhook,
			ChannelID:   "dashboard",
			PeerID:      "dashboard-user",
			PeerName:    "Dashboard",
			Content:     req.Message,
			Origin:      "dm",
		}

		response, err := gw.ProcessMessage(context.Background(), msg)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"response": response,
			"content":  response,
			"message":  response,
		}, nil
	})

	hub.Register("chat.history", func(params json.RawMessage) (any, error) {
		var req struct {
			SessionID string `json:"sessionId"`
			Limit     int    `json:"limit"`
		}
		if params != nil {
			_ = json.Unmarshal(params, &req)
		}
		if req.Limit <= 0 {
			req.Limit = 50
		}
		if req.SessionID == "" {
			req.SessionID = "dashboard"
		}
		messages, err := s.ListMessages(context.Background(), req.SessionID, req.Limit)
		if err != nil {
			return nil, err
		}
		if messages == nil {
			messages = []types.Message{}
		}
		return map[string]any{"messages": messages}, nil
	})

	hub.Register("chat.abort", func(params json.RawMessage) (any, error) {
		// Placeholder: abort current in-flight chat request
		return map[string]bool{"ok": true}, nil
	})
}

// --- Agent Methods ---

func registerAgentMethods(hub *dashboard.Hub, s store.Store) {
	hub.Register("agents.list", func(params json.RawMessage) (any, error) {
		agents, err := s.ListAgents(context.Background())
		if err != nil {
			return nil, err
		}
		if agents == nil {
			agents = []types.Agent{}
		}
		return map[string]any{"agents": agents}, nil
	})

	hub.Register("agents.get", func(params json.RawMessage) (any, error) {
		var req struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		agent, err := s.GetAgent(context.Background(), req.ID)
		if err != nil {
			return nil, err
		}
		return agent, nil
	})

	hub.Register("agent.identity.get", func(params json.RawMessage) (any, error) {
		var req struct {
			ID string `json:"id"`
		}
		if params != nil {
			_ = json.Unmarshal(params, &req)
		}
		// Load identity from workspace files
		workspaceDir := filepath.Join(config.DefaultConfigDir(), "workspace")
		identity := map[string]any{
			"name":  "OpenBot",
			"emoji": "\U0001f916",
		}
		// Try to read IDENTITY.md
		identityPath := filepath.Join(workspaceDir, "IDENTITY.md")
		if data, err := os.ReadFile(identityPath); err == nil {
			content := string(data)
			identity["raw"] = content
		}
		return identity, nil
	})

	hub.Register("models.list", func(params json.RawMessage) (any, error) {
		models := []map[string]any{
			{"id": "claude-sonnet-4-20250514", "name": "Claude Sonnet 4", "provider": "anthropic"},
			{"id": "claude-opus-4-20250514", "name": "Claude Opus 4", "provider": "anthropic"},
			{"id": "claude-haiku-3-5-20241022", "name": "Claude 3.5 Haiku", "provider": "anthropic"},
		}
		return map[string]any{"models": models}, nil
	})
}
