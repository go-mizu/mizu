// Package session provides an OpenClaw-compatible file-based session store.
// It maintains a sessions.json index and per-session JSONL transcript files.
package session

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FileStore manages OpenClaw-compatible file-based sessions.
// It maintains a sessions.json index and per-session JSONL transcript files.
type FileStore struct {
	baseDir string
	mu      sync.Mutex
}

// NewFileStore creates a file store. baseDir is the sessions directory.
// It creates the directory if it doesn't exist.
func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create sessions dir: %w", err)
	}
	return &FileStore{baseDir: baseDir}, nil
}

// SessionKey builds an OpenClaw-compatible session key.
// DM:    "agent:{agentId}:{peerId}"
// Group: "agent:{agentId}:{channel}:group:{groupId}"
func SessionKey(agentID, channelType, peerID, groupID string) string {
	if groupID != "" {
		return fmt.Sprintf("agent:%s:%s:group:%s", agentID, channelType, groupID)
	}
	return fmt.Sprintf("agent:%s:%s", agentID, peerID)
}

// Entry represents a session in sessions.json (OpenClaw-compatible).
type Entry struct {
	SessionID       string         `json:"sessionId"`
	UpdatedAt       int64          `json:"updatedAt"`                 // ms since epoch
	SystemSent      bool           `json:"systemSent,omitempty"`      // whether system prompt was sent
	AbortedLastRun  bool           `json:"abortedLastRun,omitempty"`  // whether last run was aborted
	ChatType        string         `json:"chatType,omitempty"`        // direct, group
	Channel         string         `json:"channel,omitempty"`
	DisplayName     string         `json:"displayName,omitempty"`
	DeliveryContext *DeliveryCtx   `json:"deliveryContext,omitempty"` // channel/to/accountId for delivery
	LastChannel     string         `json:"lastChannel,omitempty"`     // last channel used
	Origin          *SessionOrigin `json:"origin,omitempty"`
	SessionFile     string         `json:"sessionFile,omitempty"`     // path to JSONL transcript
	CompactionCount int            `json:"compactionCount,omitempty"`
	SkillsSnapshot  *SkillsSnap   `json:"skillsSnapshot,omitempty"`  // skills at session creation

	// Auth profile tracking.
	AuthProfileOverride                string `json:"authProfileOverride,omitempty"`
	AuthProfileOverrideSource          string `json:"authProfileOverrideSource,omitempty"`
	AuthProfileOverrideCompactionCount int    `json:"authProfileOverrideCompactionCount,omitempty"`

	// Delivery tracking.
	LastTo        string `json:"lastTo,omitempty"`
	LastAccountId string `json:"lastAccountId,omitempty"`

	// Token usage.
	InputTokens  int `json:"inputTokens,omitempty"`
	OutputTokens int `json:"outputTokens,omitempty"`
	TotalTokens  int `json:"totalTokens,omitempty"`

	// Model info.
	ModelProvider string `json:"modelProvider,omitempty"`
	Model         string `json:"model,omitempty"`
	ContextTokens int    `json:"contextTokens,omitempty"`

	// Diagnostic report.
	SystemPromptReport *SystemPromptReport `json:"systemPromptReport,omitempty"`

	// Display.
	Status string `json:"status,omitempty"` // active, expired
	Label  string `json:"label,omitempty"`
}

// DeliveryCtx describes how messages are delivered for a session.
type DeliveryCtx struct {
	Channel   string `json:"channel,omitempty"`
	To        string `json:"to,omitempty"`
	AccountId string `json:"accountId,omitempty"`
}

// SessionOrigin describes how a session was created.
type SessionOrigin struct {
	Label     string `json:"label,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Surface   string `json:"surface,omitempty"`
	ChatType  string `json:"chatType,omitempty"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	AccountId string `json:"accountId,omitempty"`
}

// SkillsSnap is a snapshot of available skills at session creation.
type SkillsSnap struct {
	Prompt   string       `json:"prompt,omitempty"`
	Skills   []SkillRef   `json:"skills,omitempty"`
	Version  int          `json:"version,omitempty"`
}

// SkillRef is a reference to a skill in the snapshot.
type SkillRef struct {
	Name       string `json:"name"`
	PrimaryEnv string `json:"primaryEnv,omitempty"`
}

// SystemPromptReport is diagnostic metadata about the system prompt.
type SystemPromptReport struct {
	Source                string             `json:"source,omitempty"`
	GeneratedAt          int64              `json:"generatedAt,omitempty"`
	SessionID            string             `json:"sessionId,omitempty"`
	SessionKey           string             `json:"sessionKey,omitempty"`
	Provider             string             `json:"provider,omitempty"`
	Model                string             `json:"model,omitempty"`
	WorkspaceDir         string             `json:"workspaceDir,omitempty"`
	BootstrapMaxChars    int                `json:"bootstrapMaxChars,omitempty"`
	Sandbox              *SandboxInfo       `json:"sandbox,omitempty"`
	SystemPrompt         *PromptStats       `json:"systemPrompt,omitempty"`
	InjectedWorkspaceFiles []WorkspaceFileInfo `json:"injectedWorkspaceFiles,omitempty"`
}

// SandboxInfo describes sandbox state.
type SandboxInfo struct {
	Mode      string `json:"mode"`
	Sandboxed bool   `json:"sandboxed"`
}

// PromptStats holds system prompt size statistics.
type PromptStats struct {
	Chars                 int `json:"chars,omitempty"`
	ProjectContextChars   int `json:"projectContextChars,omitempty"`
	NonProjectContextChars int `json:"nonProjectContextChars,omitempty"`
}

// WorkspaceFileInfo describes an injected workspace file.
type WorkspaceFileInfo struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Missing       bool   `json:"missing"`
	RawChars      int    `json:"rawChars,omitempty"`
	InjectedChars int    `json:"injectedChars,omitempty"`
	Truncated     bool   `json:"truncated,omitempty"`
}

// TranscriptEntry is a single line in a JSONL transcript file.
type TranscriptEntry struct {
	Type      string `json:"type"`                // session, message, model_change, custom
	ID        string `json:"id,omitempty"`
	Timestamp string `json:"timestamp"`

	// For session header:
	Version int    `json:"version,omitempty"`
	Cwd     string `json:"cwd,omitempty"`

	// For message:
	Message *TranscriptMessage `json:"message,omitempty"`

	// For model_change:
	Model string `json:"model,omitempty"`

	// For custom:
	Key   string `json:"key,omitempty"`
	Value any    `json:"value,omitempty"`

	// For message usage:
	Usage *TokenUsage `json:"usage,omitempty"`
}

// TranscriptMessage holds a chat message within a transcript.
type TranscriptMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []ContentBlock
}

// TokenUsage tracks input/output token counts for a message.
type TokenUsage struct {
	Input  int `json:"input,omitempty"`
	Output int `json:"output,omitempty"`
}

// SessionInfo pairs a session key with its index entry.
type SessionInfo struct {
	Key   string
	Entry *Entry
}

// indexFile is the name of the sessions index file.
const indexFile = "sessions.json"

// LoadIndex loads sessions.json. Returns empty map if file doesn't exist.
func (fs *FileStore) LoadIndex() (map[string]*Entry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.loadIndexLocked()
}

// loadIndexLocked loads sessions.json without acquiring the mutex.
// Caller must hold fs.mu.
func (fs *FileStore) loadIndexLocked() (map[string]*Entry, error) {
	path := filepath.Join(fs.baseDir, indexFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*Entry), nil
		}
		return nil, fmt.Errorf("read sessions index: %w", err)
	}

	index := make(map[string]*Entry)
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("parse sessions index: %w", err)
	}
	return index, nil
}

// SaveIndex atomically writes sessions.json.
func (fs *FileStore) SaveIndex(index map[string]*Entry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.saveIndexLocked(index)
}

// saveIndexLocked atomically writes sessions.json without acquiring the mutex.
// Caller must hold fs.mu.
func (fs *FileStore) saveIndexLocked(index map[string]*Entry) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sessions index: %w", err)
	}

	path := filepath.Join(fs.baseDir, indexFile)
	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp index: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // best effort cleanup
		return fmt.Errorf("rename temp index: %w", err)
	}
	return nil
}

// GetOrCreate gets existing session entry by key, or creates new one with generated UUID.
// Returns the entry and whether it was newly created.
func (fs *FileStore) GetOrCreate(key string, displayName, chatType, channel string) (*Entry, bool, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	index, err := fs.loadIndexLocked()
	if err != nil {
		return nil, false, err
	}

	if entry, ok := index[key]; ok {
		entry.UpdatedAt = nowMillis()
		if err := fs.saveIndexLocked(index); err != nil {
			return nil, false, err
		}
		return entry, false, nil
	}

	entry := &Entry{
		SessionID:   generateUUID(),
		UpdatedAt:   nowMillis(),
		ChatType:    chatType,
		Channel:     channel,
		DisplayName: displayName,
		Status:      "active",
	}
	index[key] = entry

	if err := fs.saveIndexLocked(index); err != nil {
		return nil, false, err
	}
	return entry, true, nil
}

// UpdateEntry updates a session entry in the index.
func (fs *FileStore) UpdateEntry(key string, entry *Entry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	index, err := fs.loadIndexLocked()
	if err != nil {
		return err
	}

	entry.UpdatedAt = nowMillis()
	index[key] = entry
	return fs.saveIndexLocked(index)
}

// DeleteEntry removes a session entry from the index.
func (fs *FileStore) DeleteEntry(key string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	index, err := fs.loadIndexLocked()
	if err != nil {
		return err
	}

	delete(index, key)
	return fs.saveIndexLocked(index)
}

// AppendTranscript appends an entry to the session's JSONL file.
// Creates the file with a session header if it doesn't exist.
func (fs *FileStore) AppendTranscript(sessionID string, entry *TranscriptEntry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := filepath.Join(fs.baseDir, sessionID+".jsonl")

	// If file doesn't exist, write session header first.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		header := TranscriptEntry{
			Type:      "session",
			Version:   2,
			ID:        sessionID,
			Timestamp: nowISO8601(),
		}
		if err := fs.writeJSONLLine(path, &header); err != nil {
			return err
		}
	}

	if entry.Timestamp == "" {
		entry.Timestamp = nowISO8601()
	}
	return fs.writeJSONLLine(path, entry)
}

// writeJSONLLine appends a single JSON line to a file. Caller must hold fs.mu.
func (fs *FileStore) writeJSONLLine(path string, entry *TranscriptEntry) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open transcript: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal transcript entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write transcript entry: %w", err)
	}
	return nil
}

// ReadTranscript reads all entries from a session's JSONL file.
func (fs *FileStore) ReadTranscript(sessionID string) ([]TranscriptEntry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	path := filepath.Join(fs.baseDir, sessionID+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	defer f.Close()

	var entries []TranscriptEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry TranscriptEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("parse transcript line: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan transcript: %w", err)
	}
	return entries, nil
}

// ListSessions returns all session entries from the index, sorted by updatedAt desc.
func (fs *FileStore) ListSessions() ([]SessionInfo, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	index, err := fs.loadIndexLocked()
	if err != nil {
		return nil, err
	}

	sessions := make([]SessionInfo, 0, len(index))
	for key, entry := range index {
		sessions = append(sessions, SessionInfo{Key: key, Entry: entry})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Entry.UpdatedAt > sessions[j].Entry.UpdatedAt
	})
	return sessions, nil
}

// UpdateTokenUsage increments token counts for a session.
func (fs *FileStore) UpdateTokenUsage(key string, inputTokens, outputTokens int) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	index, err := fs.loadIndexLocked()
	if err != nil {
		return err
	}

	entry, ok := index[key]
	if !ok {
		return fmt.Errorf("session not found: %s", key)
	}

	entry.InputTokens += inputTokens
	entry.OutputTokens += outputTokens
	entry.TotalTokens = entry.InputTokens + entry.OutputTokens
	entry.UpdatedAt = nowMillis()

	return fs.saveIndexLocked(index)
}

// ResetSession creates a new session ID for an existing key (like /new or /reset).
func (fs *FileStore) ResetSession(key string) (*Entry, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	index, err := fs.loadIndexLocked()
	if err != nil {
		return nil, err
	}

	entry, ok := index[key]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", key)
	}

	entry.SessionID = generateUUID()
	entry.UpdatedAt = nowMillis()
	entry.InputTokens = 0
	entry.OutputTokens = 0
	entry.TotalTokens = 0
	entry.ContextTokens = 0
	entry.CompactionCount = 0

	if err := fs.saveIndexLocked(index); err != nil {
		return nil, err
	}
	return entry, nil
}

// generateUUID produces a UUID v4 string using crypto/rand.
func generateUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	// Set version (4) and variant (RFC 4122).
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// nowMillis returns the current time as milliseconds since epoch.
func nowMillis() int64 {
	return time.Now().UnixMilli()
}

// nowISO8601 returns the current time in ISO 8601 format.
func nowISO8601() string {
	return time.Now().UTC().Format(time.RFC3339)
}
