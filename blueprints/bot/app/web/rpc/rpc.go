// Package rpc implements all WebSocket RPC method handlers for the Control Dashboard.
// Each method matches the OpenBot gateway protocol for full feature compatibility.
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
	registerSessionMethods(hub, s, gw)
	registerChannelMethods(hub, s)
	registerSkillMethods(hub)
	registerConfigMethods(hub)
	registerLogMethods(hub, logs)
	registerDebugMethods(hub, s, gw, logs, startTime)
	registerChatMethods(hub, gw, s)
	registerMemoryMethods(hub, gw)
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

func registerSessionMethods(hub *dashboard.Hub, s store.Store, gw *gateway.Service) {
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
			Key            string  `json:"key"`
			SessionID      string  `json:"sessionId"`
			Label          *string `json:"label"`
			ThinkingLevel  *string `json:"thinkingLevel"`
			VerboseLevel   *string `json:"verboseLevel"`
			ReasoningLevel *string `json:"reasoningLevel"`
			Model          *string `json:"model"`
			ResponseUsage  *string `json:"responseUsage"`
			SendPolicy     *string `json:"sendPolicy"`
			Metadata       *string `json:"metadata"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}

		// Resolve session ID.
		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = req.Key
		}
		if sessionID == "" {
			return nil, fmt.Errorf("sessionId or key required")
		}

		// Build updates map.
		updates := make(map[string]any)
		if req.Label != nil {
			updates["label"] = *req.Label
		}
		if req.ThinkingLevel != nil {
			updates["thinking_level"] = *req.ThinkingLevel
		}
		if req.VerboseLevel != nil {
			updates["verbose_level"] = *req.VerboseLevel
		}
		if req.ReasoningLevel != nil {
			updates["reasoning_level"] = *req.ReasoningLevel
		}
		if req.Model != nil {
			updates["model"] = *req.Model
		}
		if req.ResponseUsage != nil {
			updates["response_usage"] = *req.ResponseUsage
		}
		if req.SendPolicy != nil {
			updates["send_policy"] = *req.SendPolicy
		}
		if req.Metadata != nil {
			updates["metadata"] = *req.Metadata
		}

		if len(updates) == 0 {
			return map[string]bool{"ok": true}, nil
		}

		if err := s.PatchSession(context.Background(), sessionID, updates); err != nil {
			return nil, err
		}
		hub.Broadcast("session.updated", nil)
		return map[string]bool{"ok": true}, nil
	})

	hub.Register("sessions.reset", func(params json.RawMessage) (any, error) {
		var req struct {
			Key       string `json:"key"`
			SessionID string `json:"sessionId"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}

		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = req.Key
		}
		if sessionID == "" {
			return nil, fmt.Errorf("sessionId or key required")
		}

		// Get existing session to preserve metadata.
		session, err := s.GetSession(context.Background(), sessionID)
		if err != nil {
			return nil, fmt.Errorf("session not found: %w", err)
		}

		// Generate new session ID.
		newID := fmt.Sprintf("%s-reset-%d", sessionID[:8], time.Now().UnixMilli())

		// Create new session with same metadata.
		newSession := &types.Session{
			ID:          newID,
			AgentID:     session.AgentID,
			ChannelID:   session.ChannelID,
			ChannelType: session.ChannelType,
			PeerID:      session.PeerID,
			DisplayName: session.DisplayName,
			Origin:      session.Origin,
			Status:      "active",
			Model:       session.Model,
		}
		// Mark old session as expired.
		session.Status = "expired"
		_ = s.UpdateSession(context.Background(), session)

		if err := s.CreateSession(context.Background(), newSession); err != nil {
			return nil, err
		}

		hub.Broadcast("session.updated", nil)
		return map[string]any{
			"ok":        true,
			"sessionId": newID,
		}, nil
	})

	hub.Register("sessions.compact", func(params json.RawMessage) (any, error) {
		var req struct {
			Key       string `json:"key"`
			SessionID string `json:"sessionId"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}

		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = req.Key
		}
		if sessionID == "" {
			return nil, fmt.Errorf("sessionId or key required")
		}

		// Get session and increment compaction count.
		session, err := s.GetSession(context.Background(), sessionID)
		if err != nil {
			return nil, fmt.Errorf("session not found: %w", err)
		}

		session.CompactionCount++
		if err := s.UpdateSession(context.Background(), session); err != nil {
			return nil, err
		}

		// Set memory flush timestamp.
		updates := map[string]any{
			"memory_flush_at":               time.Now().UnixMilli(),
			"memory_flush_compaction_count": session.CompactionCount,
		}
		_ = s.PatchSession(context.Background(), sessionID, updates)

		// Trigger memory re-index after compaction to keep search current.
		if agent, err := s.GetAgent(context.Background(), session.AgentID); err == nil && agent.Workspace != "" {
			go func() {
				if err := gw.ReIndex(agent.Workspace); err != nil {
					fmt.Fprintf(os.Stderr, "memory re-index after compact: %v\n", err)
				}
			}()
		}

		hub.Broadcast("session.updated", nil)
		return map[string]any{
			"ok":              true,
			"compactionCount": session.CompactionCount,
		}, nil
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

// loadSkillsWithConfig loads all skills and the raw config, returning both.
// It respects skills.load.extraDirs from the config.
func loadSkillsWithConfig() ([]*skill.Skill, map[string]any, error) {
	workspaceDir := filepath.Join(config.DefaultConfigDir(), "workspace")
	bundledDir := skill.BundledSkillsDir()

	// Load config first so we can extract extra dirs.
	cfgPath := config.DefaultConfigPath()
	data, err := config.LoadRawConfig(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			data = make(map[string]any)
		} else {
			return nil, nil, err
		}
	}

	extraDirs := skill.ParseExtraDirs(data)
	loaded, _ := skill.LoadAllSkillsWithExtras(workspaceDir, extraDirs, bundledDir)

	return loaded, data, nil
}

func registerSkillMethods(hub *dashboard.Hub) {
	// skills.status - Full skill status report matching OpenClaw schema.
	hub.Register("skills.status", func(params json.RawMessage) (any, error) {
		loaded, data, err := loadSkillsWithConfig()
		if err != nil {
			return nil, fmt.Errorf("load config: %w", err)
		}

		var allSkills []types.SkillEntry
		for _, s := range loaded {
			skillKey := s.SkillKey
			if skillKey == "" {
				skillKey = s.Name
			}
			skillCfg := config.ResolveSkillConfig(data, skillKey)
			if skillCfg == nil {
				skillCfg = map[string]any{}
			}
			entry := skill.BuildSkillStatus(s, data, skillCfg)

			// Attach install options.
			specs := skill.ParseInstallSpecs(s)
			if opts := skill.ToInstallOpts(specs); len(opts) > 0 {
				entry.Install = opts
			}

			allSkills = append(allSkills, entry)
		}

		if allSkills == nil {
			allSkills = []types.SkillEntry{}
		}

		workspaceDir := filepath.Join(config.DefaultConfigDir(), "workspace")
		managedSkillsDir := filepath.Join(config.DefaultConfigDir(), "skills")
		return map[string]any{
			"workspaceDir":     workspaceDir,
			"managedSkillsDir": managedSkillsDir,
			"skills":           allSkills,
		}, nil
	})

	// skills.update - Update per-skill config (enabled, apiKey, env).
	// Replaces the old skills.toggle with full OpenClaw-compatible update.
	hub.Register("skills.update", func(params json.RawMessage) (any, error) {
		var req struct {
			SkillKey string            `json:"skillKey"`
			Enabled  *bool             `json:"enabled"`
			ApiKey   *string           `json:"apiKey"`
			Env      map[string]string `json:"env"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if req.SkillKey == "" {
			return nil, fmt.Errorf("skillKey is required")
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

		config.UpdateSkillConfig(data, req.SkillKey, req.Enabled, req.ApiKey, req.Env)

		if err := config.SaveRawConfig(cfgPath, data); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}

		hub.Broadcast("skills.updated", nil)
		return map[string]any{
			"ok":       true,
			"skillKey": req.SkillKey,
			"config":   config.ResolveSkillConfig(data, req.SkillKey),
		}, nil
	})

	// skills.toggle - Legacy toggle, delegates to skills.update.
	hub.Register("skills.toggle", func(params json.RawMessage) (any, error) {
		var req struct {
			Key     string `json:"key"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
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

		enabled := req.Enabled
		config.UpdateSkillConfig(data, req.Key, &enabled, nil, nil)

		if err := config.SaveRawConfig(cfgPath, data); err != nil {
			return nil, fmt.Errorf("save config: %w", err)
		}
		hub.Broadcast("skills.updated", nil)
		return map[string]bool{"ok": true}, nil
	})

	// skills.bins - Returns all required binaries across all loaded skills.
	hub.Register("skills.bins", func(params json.RawMessage) (any, error) {
		loaded, _, err := loadSkillsWithConfig()
		if err != nil {
			return nil, fmt.Errorf("load skills: %w", err)
		}

		seen := map[string]bool{}
		var bins []string
		for _, s := range loaded {
			for _, bin := range s.Requires.Binaries {
				if !seen[bin] {
					seen[bin] = true
					bins = append(bins, bin)
				}
			}
			for _, bin := range s.Requires.AnyBins {
				if !seen[bin] {
					seen[bin] = true
					bins = append(bins, bin)
				}
			}
		}
		if bins == nil {
			bins = []string{}
		}
		return map[string]any{"bins": bins}, nil
	})

	// skills.install - Install a skill's dependency.
	hub.Register("skills.install", func(params json.RawMessage) (any, error) {
		var req struct {
			Name      string `json:"name"`
			InstallID string `json:"installId"`
			TimeoutMs int    `json:"timeoutMs"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if req.Name == "" || req.InstallID == "" {
			return nil, fmt.Errorf("name and installId are required")
		}

		loaded, _, err := loadSkillsWithConfig()
		if err != nil {
			return nil, fmt.Errorf("load skills: %w", err)
		}

		// Load install preferences from config.
		prefs := skill.DefaultInstallPrefs()
		rawCfg, cfgErr := config.LoadRawConfig(config.DefaultConfigPath())
		if cfgErr == nil {
			prefs = skill.ParseInstallPrefs(rawCfg)
		}

		result, err := skill.InstallSkillDep(loaded, req.Name, req.InstallID, req.TimeoutMs, prefs)
		if err != nil {
			return nil, err
		}

		hub.Broadcast("skills.updated", nil)
		return result, nil
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

// generateID produces a short unique ID for runs and messages.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func registerChatMethods(hub *dashboard.Hub, gw *gateway.Service, s store.Store) {
	// chat.send — OpenClaw-compatible: accepts sessionKey, idempotencyKey, thinking.
	// Returns {runId, status} immediately. Processing happens synchronously for Phase 1
	// but the response format matches OpenClaw's async ACK protocol.
	hub.Register("chat.send", func(params json.RawMessage) (any, error) {
		var req struct {
			// OpenClaw params
			SessionKey     string `json:"sessionKey"`
			Message        string `json:"message"`
			IdempotencyKey string `json:"idempotencyKey"`
			Thinking       string `json:"thinking"`
			TimeoutMs      int    `json:"timeoutMs"`
			// Legacy params
			SessionID string `json:"sessionId"`
			AgentID   string `json:"agentId"`
			PeerName  string `json:"peerName"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if req.Message == "" {
			return nil, fmt.Errorf("message is required")
		}

		// Resolve sessionKey: prefer OpenClaw sessionKey, fall back to legacy sessionId
		sessionKey := req.SessionKey
		if sessionKey == "" && req.SessionID != "" {
			sessionKey = "agent:main:" + req.SessionID
		}
		if sessionKey == "" {
			agentID := req.AgentID
			if agentID == "" {
				agentID = "main"
			}
			sessionKey = "agent:" + agentID + ":main"
		}

		// Resolve runId from idempotencyKey
		runID := req.IdempotencyKey
		if runID == "" {
			runID = generateID()
		}

		// Check idempotency cache
		if cached, ok := gw.CheckDedupe("chat:" + runID); ok {
			return cached, nil
		}

		peerName := req.PeerName
		if peerName == "" {
			peerName = "Dashboard"
		}

		msg := &types.InboundMessage{
			ChannelType: types.ChannelWebhook,
			ChannelID:   "dashboard",
			PeerID:      "dashboard-user",
			PeerName:    peerName,
			Content:     req.Message,
			Origin:      "dm",
			RunID:       runID,
			SessionKey:  sessionKey,
		}

		result, err := gw.ProcessMessage(context.Background(), msg)
		if err != nil {
			// Broadcast error in OpenClaw format
			hub.Broadcast("chat", map[string]any{
				"runId": runID, "sessionKey": sessionKey,
				"seq": 1, "state": "error",
				"errorMessage": err.Error(),
			})
			return nil, err
		}

		// Build OpenClaw-compatible response
		resp := map[string]any{
			"runId":     runID,
			"status":    "ok",
			"sessionId": result.SessionID,
			"messageId": result.MessageID,
			"content":   result.Content,
			"agentId":   result.AgentID,
			"model":     result.Model,
		}

		// Cache for idempotency
		gw.SetDedupe("chat:"+runID, resp)

		return resp, nil
	})

	// chat.history — OpenClaw-compatible: accepts sessionKey or sessionId.
	// Returns messages in OpenClaw format with thinkingLevel and timestamps.
	hub.Register("chat.history", func(params json.RawMessage) (any, error) {
		var req struct {
			SessionKey string `json:"sessionKey"`
			SessionID  string `json:"sessionId"`
			Limit      int    `json:"limit"`
		}
		if params != nil {
			_ = json.Unmarshal(params, &req)
		}
		if req.Limit <= 0 {
			req.Limit = 200
		}
		if req.Limit > 1000 {
			req.Limit = 1000
		}

		// Resolve session: prefer sessionKey, fall back to sessionId
		sessionID := req.SessionID
		sessionKey := req.SessionKey
		if sessionKey != "" {
			// Resolve sessionKey to internal session
			agentID, channelType, peerID := gateway.SessionKeyToQuery(sessionKey)
			session, err := s.GetOrCreateSession(context.Background(), agentID, "dashboard", channelType, peerID, "", "dm")
			if err == nil && session != nil {
				sessionID = session.ID
			}
		}
		if sessionID == "" && sessionKey == "" {
			return nil, fmt.Errorf("sessionKey or sessionId is required")
		}

		messages, err := s.ListMessages(context.Background(), sessionID, req.Limit)
		if err != nil {
			return nil, err
		}
		if messages == nil {
			messages = []types.Message{}
		}

		// Convert to OpenClaw message format
		ocMessages := make([]map[string]any, len(messages))
		for i, m := range messages {
			msg := map[string]any{
				"role":      m.Role,
				"timestamp": m.CreatedAt.UnixMilli(),
			}
			if m.Role == types.RoleAssistant {
				msg["content"] = []any{map[string]string{"type": "text", "text": m.Content}}
				msg["stopReason"] = "end_turn"
			} else {
				msg["content"] = m.Content
			}
			ocMessages[i] = msg
		}

		resp := map[string]any{
			"messages":      ocMessages,
			"sessionId":     sessionID,
			"sessionKey":    sessionKey,
			"thinkingLevel": "off",
		}
		return resp, nil
	})

	// chat.abort — OpenClaw-compatible: accepts runId and/or sessionKey.
	// Returns {ok, aborted, runIds}.
	hub.Register("chat.abort", func(params json.RawMessage) (any, error) {
		var req struct {
			SessionKey string `json:"sessionKey"`
			SessionID  string `json:"sessionId"`
			RunID      string `json:"runId"`
		}
		if params != nil {
			_ = json.Unmarshal(params, &req)
		}

		// If runId specified, abort that specific run
		if req.RunID != "" {
			aborted, err := gw.AbortByRunID(req.RunID, req.SessionKey)
			if err != nil {
				return nil, err
			}
			runIDs := []string{}
			if aborted {
				runIDs = []string{req.RunID}
			}
			return map[string]any{"ok": true, "aborted": aborted, "runIds": runIDs}, nil
		}

		// If sessionKey specified, abort all runs for that session
		if req.SessionKey != "" {
			runIDs, aborted := gw.AbortBySessionKey(req.SessionKey)
			if runIDs == nil {
				runIDs = []string{}
			}
			return map[string]any{"ok": true, "aborted": aborted, "runIds": runIDs}, nil
		}

		// Legacy: abort by sessionId
		if req.SessionID != "" {
			aborted := gw.Abort(req.SessionID)
			return map[string]any{"ok": true, "aborted": aborted, "runIds": []string{}}, nil
		}

		return map[string]any{"ok": true, "aborted": false, "runIds": []string{}}, nil
	})

	// chat.inject — NEW: OpenClaw-compatible. Injects an assistant message
	// into a session without invoking the LLM.
	hub.Register("chat.inject", func(params json.RawMessage) (any, error) {
		var req struct {
			SessionKey string `json:"sessionKey"`
			SessionID  string `json:"sessionId"`
			Message    string `json:"message"`
			Label      string `json:"label"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if req.Message == "" {
			return nil, fmt.Errorf("message is required")
		}

		// Resolve session
		sessionID := req.SessionID
		sessionKey := req.SessionKey
		if sessionKey != "" && sessionID == "" {
			agentID, channelType, peerID := gateway.SessionKeyToQuery(sessionKey)
			session, err := s.GetOrCreateSession(context.Background(), agentID, "dashboard", channelType, peerID, "", "dm")
			if err == nil && session != nil {
				sessionID = session.ID
			}
		}
		if sessionID == "" {
			return nil, fmt.Errorf("sessionKey or sessionId is required")
		}

		content := req.Message
		if req.Label != "" {
			content = "[" + req.Label + "] " + content
		}

		// Store as assistant message with stopReason "injected"
		assistantMsg := &types.Message{
			SessionID: sessionID,
			AgentID:   "main",
			Role:      types.RoleAssistant,
			Content:   content,
		}
		if err := s.CreateMessage(context.Background(), assistantMsg); err != nil {
			return nil, err
		}

		// Broadcast OpenClaw-format final event
		if sessionKey == "" {
			sessionKey = "agent:main:main"
		}
		runID := generateID()
		hub.Broadcast("chat", map[string]any{
			"runId": runID, "sessionKey": sessionKey,
			"seq": 1, "state": "final",
			"message": map[string]any{
				"role":       "assistant",
				"content":    []any{map[string]string{"type": "text", "text": content}},
				"timestamp":  assistantMsg.CreatedAt.UnixMilli(),
				"stopReason": "injected",
				"usage":      map[string]int{"input": 0, "output": 0, "totalTokens": 0},
			},
		})
		// Legacy compat
		hub.Broadcast("chat.message", map[string]any{
			"sessionId": sessionID,
			"message": map[string]any{
				"id":      assistantMsg.ID,
				"role":    assistantMsg.Role,
				"content": assistantMsg.Content,
			},
		})

		return map[string]any{"ok": true, "messageId": assistantMsg.ID}, nil
	})

	// chat.new — Create a new chat session (convenience method).
	hub.Register("chat.new", func(params json.RawMessage) (any, error) {
		var req struct {
			AgentID string `json:"agentId"`
		}
		if params != nil {
			_ = json.Unmarshal(params, &req)
		}
		if req.AgentID == "" {
			req.AgentID = "main"
		}
		// Expire any active dashboard session for this agent.
		sessions, err := s.ListSessions(context.Background())
		if err != nil {
			return nil, err
		}
		for i := range sessions {
			sess := &sessions[i]
			if sess.AgentID == req.AgentID &&
				sess.ChannelType == "webhook" &&
				sess.PeerID == "dashboard-user" &&
				sess.Status == "active" {
				sess.Status = "expired"
				_ = s.UpdateSession(context.Background(), sess)
			}
		}
		hub.Broadcast("session.updated", nil)
		return map[string]any{"ok": true}, nil
	})
}

// --- Memory Methods ---

func registerMemoryMethods(hub *dashboard.Hub, gw *gateway.Service) {
	hub.Register("memory.search", func(params json.RawMessage) (any, error) {
		var req struct {
			Query        string `json:"query"`
			WorkspaceDir string `json:"workspaceDir"`
			Limit        int    `json:"limit"`
		}
		if err := json.Unmarshal(params, &req); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		if req.Query == "" {
			return nil, fmt.Errorf("query is required")
		}

		// Use default workspace if not specified
		wsDir := req.WorkspaceDir
		if wsDir == "" {
			wsDir = os.Getenv("OPENBOT_WORKSPACE")
			if wsDir == "" {
				home, _ := os.UserHomeDir()
				wsDir = filepath.Join(home, ".openbot", "workspace")
			}
		}

		results, err := gw.SearchMemory(context.Background(), wsDir, req.Query, req.Limit)
		if err != nil {
			return nil, err
		}

		type memResult struct {
			Path      string  `json:"path"`
			Source    string  `json:"source"`
			StartLine int     `json:"startLine"`
			EndLine   int     `json:"endLine"`
			Score     float64 `json:"score"`
			Snippet   string  `json:"snippet"`
		}
		var out []memResult
		for _, r := range results {
			src := r.Source
			if src == "" {
				src = "memory"
			}
			out = append(out, memResult{
				Path:      r.Path,
				Source:    src,
				StartLine: r.StartLine,
				EndLine:   r.EndLine,
				Score:     r.Score,
				Snippet:   r.Snippet,
			})
		}
		if out == nil {
			out = []memResult{}
		}
		return map[string]any{"results": out}, nil
	})

	hub.Register("memory.stats", func(params json.RawMessage) (any, error) {
		var req struct {
			WorkspaceDir string `json:"workspaceDir"`
		}
		if params != nil {
			_ = json.Unmarshal(params, &req)
		}

		wsDir := req.WorkspaceDir
		if wsDir == "" {
			wsDir = os.Getenv("OPENBOT_WORKSPACE")
			if wsDir == "" {
				home, _ := os.UserHomeDir()
				wsDir = filepath.Join(home, ".openbot", "workspace")
			}
		}

		files, chunks, err := gw.MemoryStats(wsDir)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"files":  files,
			"chunks": chunks,
		}, nil
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
