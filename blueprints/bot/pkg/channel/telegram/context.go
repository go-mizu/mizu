package telegram

import (
	"strconv"
	"strings"
)

// ReplyContext holds information about a replied-to message.
type ReplyContext struct {
	ID     string // message_id of the replied-to message
	Sender string // display name of the sender
	Body   string // text/caption or media placeholder
	Kind   string // "reply" or "quote"
}

// extractReplyContext extracts reply context from a message.
// It checks the Quote field first (Telegram 7.0+), then falls back to
// ReplyToMessage for older-style replies.
func extractReplyContext(msg *TelegramMessage) *ReplyContext {
	if msg == nil {
		return nil
	}

	// 1. Check msg.Quote first (Telegram 7.0+).
	if msg.Quote != nil && msg.Quote.Text != "" {
		rc := &ReplyContext{
			Body: msg.Quote.Text,
			Kind: "quote",
		}
		// If there is also a ReplyToMessage, extract the sender and ID from it.
		if msg.ReplyToMessage != nil {
			rc.ID = strconv.FormatInt(msg.ReplyToMessage.MessageID, 10)
			if msg.ReplyToMessage.From != nil {
				rc.Sender = buildSenderName(msg.ReplyToMessage.From)
			}
		}
		return rc
	}

	// 2. Check ReplyToMessage.
	if msg.ReplyToMessage == nil {
		return nil
	}

	reply := msg.ReplyToMessage
	rc := &ReplyContext{
		ID:   strconv.FormatInt(reply.MessageID, 10),
		Kind: "reply",
	}

	// Build sender name from reply_to_message.from.
	if reply.From != nil {
		rc.Sender = buildSenderName(reply.From)
	}

	// Get text or caption.
	switch {
	case reply.Text != "":
		rc.Body = reply.Text
	case reply.Caption != "":
		rc.Body = reply.Caption
	case len(reply.Photo) > 0:
		rc.Body = "<media:image>"
	case reply.Video != nil || reply.VideoNote != nil:
		rc.Body = "<media:video>"
	case reply.Audio != nil || reply.Voice != nil:
		rc.Body = "<media:audio>"
	case reply.Document != nil:
		rc.Body = "<media:document>"
	case reply.Location != nil:
		name, address := "", ""
		if reply.Venue != nil {
			name = reply.Venue.Title
			address = reply.Venue.Address
		}
		rc.Body = formatLocationText(reply.Location.Latitude, reply.Location.Longitude, name, address)
	case reply.Sticker != nil:
		rc.Body = "<media:sticker>"
	}

	return rc
}

// ForwardContext holds information about a forwarded message.
type ForwardContext struct {
	From          string // display name of original sender
	Date          int64  // original send date
	FromType      string // user, hidden_user, chat, channel, legacy_user, legacy_chat, legacy_hidden
	FromID        string // original sender ID
	FromUsername  string
	FromTitle     string
	FromSignature string
}

// extractForwardContext extracts forward origin from a message.
// It checks the ForwardOrigin field first (new API), then falls back to
// legacy fields (forward_from, forward_from_chat, forward_sender_name).
func extractForwardContext(msg *TelegramMessage) *ForwardContext {
	if msg == nil {
		return nil
	}

	// 1. Check ForwardOrigin (new API).
	if msg.ForwardOrigin != nil {
		fo := msg.ForwardOrigin
		fc := &ForwardContext{
			Date:     fo.Date,
			FromType: fo.Type,
		}

		switch fo.Type {
		case "user":
			if fo.SenderUser != nil {
				fc.From = buildSenderName(fo.SenderUser)
				fc.FromID = strconv.FormatInt(fo.SenderUser.ID, 10)
				fc.FromUsername = fo.SenderUser.Username
			}
		case "hidden_user":
			fc.From = fo.SenderUserName
		case "chat":
			if fo.SenderChat != nil {
				fc.From = fo.SenderChat.Title
				fc.FromID = strconv.FormatInt(fo.SenderChat.ID, 10)
				fc.FromUsername = fo.SenderChat.Username
				fc.FromTitle = fo.SenderChat.Title
			}
		case "channel":
			if fo.Chat != nil {
				fc.From = fo.Chat.Title
				fc.FromID = strconv.FormatInt(fo.Chat.ID, 10)
				fc.FromUsername = fo.Chat.Username
				fc.FromTitle = fo.Chat.Title
			}
		}

		return fc
	}

	// 2. Fallback to legacy fields.
	if msg.ForwardFromChat != nil {
		chatType := "legacy_chat"
		if msg.ForwardFromChat.Type == "channel" {
			chatType = "legacy_channel"
		}
		return &ForwardContext{
			From:          msg.ForwardFromChat.Title,
			Date:          msg.ForwardDate,
			FromType:      chatType,
			FromID:        strconv.FormatInt(msg.ForwardFromChat.ID, 10),
			FromUsername:   msg.ForwardFromChat.Username,
			FromTitle:     msg.ForwardFromChat.Title,
			FromSignature: msg.ForwardSignature,
		}
	}

	if msg.ForwardFrom != nil {
		return &ForwardContext{
			From:     buildSenderName(msg.ForwardFrom),
			Date:     msg.ForwardDate,
			FromType: "legacy_user",
			FromID:   strconv.FormatInt(msg.ForwardFrom.ID, 10),
			FromUsername: msg.ForwardFrom.Username,
		}
	}

	if msg.ForwardSenderName != "" {
		return &ForwardContext{
			From:     msg.ForwardSenderName,
			Date:     msg.ForwardDate,
			FromType: "legacy_hidden",
		}
	}

	// 3. Not forwarded.
	return nil
}

// buildSenderName builds a display name from a TelegramUser.
// It prefers "FirstName LastName", then falls back to "@username", then "id:123".
func buildSenderName(user *TelegramUser) string {
	if user == nil {
		return ""
	}

	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if name != "" {
		return name
	}
	if user.Username != "" {
		return "@" + user.Username
	}
	return "id:" + strconv.FormatInt(user.ID, 10)
}

// buildSenderLabel builds a label in "Name @username id:123" format.
// It combines all available identity parts for logging and display.
func buildSenderLabel(msg *TelegramMessage) string {
	if msg == nil || msg.From == nil {
		return ""
	}

	var parts []string

	name := strings.TrimSpace(msg.From.FirstName + " " + msg.From.LastName)
	if name != "" {
		parts = append(parts, name)
	}
	if msg.From.Username != "" {
		parts = append(parts, "@"+msg.From.Username)
	}
	parts = append(parts, "id:"+strconv.FormatInt(msg.From.ID, 10))

	return strings.Join(parts, " ")
}

// expandTextLinks converts text_link entities to markdown-style links.
// For each entity with type="text_link" and a URL set, it replaces the
// original text range with [linkText](url). Entities are processed in
// reverse offset order so that earlier positions remain valid.
func expandTextLinks(text string, entities []TelegramEntity) string {
	if len(entities) == 0 || text == "" {
		return text
	}

	// Convert to runes for correct Unicode offset handling.
	runes := []rune(text)

	// Collect text_link entities with URLs.
	type linkEntity struct {
		offset int
		length int
		url    string
	}
	var links []linkEntity
	for _, e := range entities {
		if e.Type == "text_link" && e.URL != "" {
			links = append(links, linkEntity{
				offset: e.Offset,
				length: e.Length,
				url:    e.URL,
			})
		}
	}

	if len(links) == 0 {
		return text
	}

	// Sort in reverse offset order to preserve positions when replacing.
	for i := 0; i < len(links)-1; i++ {
		for j := i + 1; j < len(links); j++ {
			if links[j].offset > links[i].offset {
				links[i], links[j] = links[j], links[i]
			}
		}
	}

	// Apply replacements in reverse order.
	for _, lk := range links {
		end := lk.offset + lk.length
		if lk.offset < 0 || end > len(runes) {
			continue
		}
		linkText := string(runes[lk.offset:end])
		replacement := []rune("[" + linkText + "](" + lk.url + ")")
		runes = append(runes[:lk.offset], append(replacement, runes[end:]...)...)
	}

	return string(runes)
}

// hasBotMention checks if @botUsername appears in the message text or entities.
// The comparison is case-insensitive.
func hasBotMention(msg *TelegramMessage, botUsername string) bool {
	if msg == nil || botUsername == "" {
		return false
	}

	target := "@" + botUsername

	// Check if text contains @username (case insensitive).
	if msg.Text != "" && strings.Contains(strings.ToLower(msg.Text), strings.ToLower(target)) {
		return true
	}
	if msg.Caption != "" && strings.Contains(strings.ToLower(msg.Caption), strings.ToLower(target)) {
		return true
	}

	// Check entities for type="mention" matching @username.
	textRunes := []rune(msg.Text)
	for _, e := range msg.Entities {
		if e.Type == "mention" {
			end := e.Offset + e.Length
			if e.Offset >= 0 && end <= len(textRunes) {
				mention := string(textRunes[e.Offset:end])
				if strings.EqualFold(mention, target) {
					return true
				}
			}
		}
	}

	// Also check caption entities.
	captionRunes := []rune(msg.Caption)
	for _, e := range msg.CaptionEntities {
		if e.Type == "mention" {
			end := e.Offset + e.Length
			if e.Offset >= 0 && end <= len(captionRunes) {
				mention := string(captionRunes[e.Offset:end])
				if strings.EqualFold(mention, target) {
					return true
				}
			}
		}
	}

	return false
}

// sessionKey generates an OpenClaw-compatible session key based on the chat
// context. The key format depends on the chat type:
//   - private chat: "telegram:{chatID}"
//   - group/supergroup without forum: "telegram:group:{chatID}"
//   - forum with topic thread: "telegram:group:{chatID}:topic:{threadID}"
func sessionKey(chatID int64, chatType string, isForum bool, threadID int64) string {
	id := strconv.FormatInt(chatID, 10)

	switch chatType {
	case "private":
		return "telegram:" + id
	case "group", "supergroup":
		base := "telegram:group:" + id
		if isForum && threadID != 0 {
			return base + ":topic:" + strconv.FormatInt(threadID, 10)
		}
		return base
	default:
		// channel or unknown type
		return "telegram:" + id
	}
}
