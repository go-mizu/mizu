package telegram

import "strings"

// accessResult is the result of an access control check.
type accessResult struct {
	allowed bool   // whether access is granted
	reason  string // reason for denial (empty if allowed)
	pairing bool   // true if pairing flow should be triggered
}

// checkDMAccess checks if a DM sender is allowed based on the DM policy.
//
// Supported dmPolicy modes:
//   - "open"      - allow all senders
//   - "disabled"  - deny all DMs
//   - "allowlist" - only allow senders whose ID is in cfg.AllowFrom
//   - "pairing"   - if sender is in AllowFrom, allow; otherwise trigger pairing flow
//
// The default policy (empty string) is "pairing".
func checkDMAccess(cfg *TelegramDriverConfig, senderID string) accessResult {
	policy := cfg.DMPolicy
	if policy == "" {
		policy = "pairing"
	}

	switch policy {
	case "open":
		return accessResult{allowed: true}

	case "disabled":
		return accessResult{
			allowed: false,
			reason:  "DMs are disabled",
		}

	case "allowlist":
		if containsID(cfg.AllowFrom, senderID) {
			return accessResult{allowed: true}
		}
		return accessResult{
			allowed: false,
			reason:  "sender not in DM allowlist",
		}

	case "pairing":
		if containsID(cfg.AllowFrom, senderID) {
			return accessResult{allowed: true}
		}
		// Not in allowlist: trigger pairing flow.
		return accessResult{
			allowed: false,
			pairing: true,
			reason:  "pairing required",
		}

	default:
		// Unknown policy, deny by default.
		return accessResult{
			allowed: false,
			reason:  "unknown DM policy: " + policy,
		}
	}
}

// checkGroupAccess checks if a group message should be processed.
//
// Supported groupPolicy modes:
//   - "open" (default) - allow all groups
//   - "disabled"       - deny all group messages
//   - "allowlist"      - only if chatID is in cfg.GroupAllowFrom
//
// Additionally, per-group configuration is checked:
//   - cfg.Groups[chatID].Enabled: if explicitly set to false, deny
//   - cfg.Groups[chatID].AllowFrom: if non-empty, senderID must be present
func checkGroupAccess(cfg *TelegramDriverConfig, chatID string, senderID string) accessResult {
	policy := cfg.GroupPolicy
	if policy == "" {
		policy = "open"
	}

	switch policy {
	case "disabled":
		return accessResult{
			allowed: false,
			reason:  "group messages are disabled",
		}

	case "allowlist":
		if !containsID(cfg.GroupAllowFrom, chatID) {
			return accessResult{
				allowed: false,
				reason:  "group not in allowlist",
			}
		}
		// Fall through to per-group checks.

	case "open":
		// Fall through to per-group checks.

	default:
		return accessResult{
			allowed: false,
			reason:  "unknown group policy: " + policy,
		}
	}

	// Per-group configuration checks.
	if cfg.Groups != nil {
		groupCfg, hasGroupCfg := cfg.Groups[chatID]
		if hasGroupCfg {
			// Check if group is explicitly disabled.
			if groupCfg.Enabled != nil && !*groupCfg.Enabled {
				return accessResult{
					allowed: false,
					reason:  "group is disabled",
				}
			}

			// Check per-group sender allowlist.
			if len(groupCfg.AllowFrom) > 0 && !containsID(groupCfg.AllowFrom, senderID) {
				return accessResult{
					allowed: false,
					reason:  "sender not in group allowlist",
				}
			}
		}
	}

	return accessResult{allowed: true}
}

// checkTopicAccess checks if a specific forum topic (thread) is enabled.
// It looks up the topic configuration in cfg.Groups[chatID].Topics[threadID].
// If no topic configuration exists, access is allowed by default.
// An empty threadID always returns allowed (not a topic message).
func checkTopicAccess(cfg *TelegramDriverConfig, chatID string, threadID string) accessResult {
	if threadID == "" {
		return accessResult{allowed: true}
	}

	if cfg.Groups == nil {
		return accessResult{allowed: true}
	}

	groupCfg, hasGroupCfg := cfg.Groups[chatID]
	if !hasGroupCfg || groupCfg.Topics == nil {
		return accessResult{allowed: true}
	}

	topicCfg, hasTopicCfg := groupCfg.Topics[threadID]
	if !hasTopicCfg {
		return accessResult{allowed: true}
	}

	if topicCfg.Enabled != nil && !*topicCfg.Enabled {
		return accessResult{
			allowed: false,
			reason:  "topic is disabled",
		}
	}

	return accessResult{allowed: true}
}

// checkMention checks if the bot was mentioned in a group message.
// It looks for @botUsername in the message entities (type "mention") and
// also checks for the bot username as a plain text substring.
func checkMention(text string, entities []TelegramEntity, botUsername string) bool {
	if botUsername == "" {
		return false
	}

	// Normalize the bot username for comparison (strip leading @, lowercase).
	normalizedBot := strings.TrimPrefix(strings.ToLower(botUsername), "@")

	// Check entities for explicit mention.
	for _, e := range entities {
		if e.Type == "mention" && e.Offset >= 0 && e.Offset+e.Length <= len(text) {
			mentioned := text[e.Offset : e.Offset+e.Length]
			// Telegram mentions include the @ prefix.
			mentioned = strings.TrimPrefix(strings.ToLower(mentioned), "@")
			if mentioned == normalizedBot {
				return true
			}
		}
	}

	// Fallback: check for @botUsername as plain text (handles messages without
	// entities, e.g. in edited messages or captions).
	if strings.Contains(strings.ToLower(text), "@"+normalizedBot) {
		return true
	}

	return false
}

// containsID checks whether a string slice contains the given ID.
func containsID(list []string, id string) bool {
	for _, item := range list {
		if item == id {
			return true
		}
	}
	return false
}
