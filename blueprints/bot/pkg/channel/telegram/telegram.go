package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/skill"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func init() {
	channel.Register(types.ChannelTelegram, func(config string, handler channel.MessageHandler) (channel.Driver, error) {
		return NewDriver(config, handler)
	})
}

// NewDriver creates a Telegram driver directly (for standalone use).
// It accepts both the legacy TelegramConfig and the enhanced TelegramDriverConfig
// JSON formats — missing fields get safe defaults.
func NewDriver(config string, handler channel.MessageHandler) (*Driver, error) {
	var cfg TelegramDriverConfig
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return nil, fmt.Errorf("parse telegram config: %w", err)
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("telegram config: botToken is required")
	}

	// Defaults.
	if cfg.DMPolicy == "" {
		cfg.DMPolicy = "pairing"
	}
	if cfg.GroupPolicy == "" {
		cfg.GroupPolicy = "open"
	}
	if cfg.StreamMode == "" {
		cfg.StreamMode = "chunked"
	}
	if cfg.ReplyToMode == "" {
		cfg.ReplyToMode = "always"
	}

	client := &http.Client{Timeout: 60 * time.Second}

	d := &Driver{
		config:  cfg,
		handler: handler,
		client:  client,
		sender:  sender{token: cfg.BotToken, client: client},
	}
	return d, nil
}

// Driver implements channel.Driver for Telegram via Bot API.
type Driver struct {
	config      TelegramDriverConfig
	handler     channel.MessageHandler
	client      *http.Client
	sender      sender
	status      string
	cancel      context.CancelFunc
	botUsername string // resolved on Connect via getMe
}

func (d *Driver) Type() types.ChannelType { return types.ChannelTelegram }
func (d *Driver) Status() string {
	if d.status == "" {
		return "disconnected"
	}
	return d.status
}

// Connect starts long-polling for updates from the Telegram Bot API.
// It calls getMe to resolve the bot's username for mention detection.
func (d *Driver) Connect(ctx context.Context) error {
	ctx, d.cancel = context.WithCancel(ctx)

	// Resolve bot username for mention detection.
	me, err := d.sender.getMe(ctx)
	if err != nil {
		log.Printf("telegram: getMe failed (mention detection disabled): %v", err)
	} else {
		d.botUsername = me.Username
		log.Printf("telegram: connected as @%s (id:%d)", me.Username, me.ID)
	}

	d.status = "connected"
	go d.poll(ctx)
	return nil
}

func (d *Driver) Disconnect(_ context.Context) error {
	if d.cancel != nil {
		d.cancel()
	}
	d.status = "disconnected"
	return nil
}

const telegramMaxLength = 4096

// Send sends a message via Telegram Bot API. It supports:
// - Text messages with HTML/Markdown parse mode
// - Message splitting for content exceeding 4096 chars
// - Reply threading (reply_to, thread_id)
// - Inline keyboards (reply_markup)
// - Message edit and delete operations
// - Reactions on messages
func (d *Driver) Send(ctx context.Context, msg *types.OutboundMessage) error {
	target := msg.ChannelID
	if target == "" {
		target = msg.PeerID
	}

	chatID, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid telegram chat ID %q: %w", target, err)
	}

	// Handle reaction.
	if msg.ReactionEmoji != "" && msg.ReactionMessageID != "" {
		reactMsgID, _ := strconv.ParseInt(msg.ReactionMessageID, 10, 64)
		if reactMsgID != 0 {
			if err := d.sender.setReaction(ctx, chatID, reactMsgID, msg.ReactionEmoji); err != nil {
				log.Printf("telegram: setReaction failed: %v", err)
			}
		}
		// If this is reaction-only (no content), return.
		if msg.Content == "" && msg.EditMessageID == "" && msg.DeleteMessageID == "" {
			return nil
		}
	}

	// Handle delete.
	if msg.DeleteMessageID != "" {
		delMsgID, _ := strconv.ParseInt(msg.DeleteMessageID, 10, 64)
		if delMsgID != 0 {
			return d.sender.deleteMessage(ctx, chatID, delMsgID)
		}
	}

	// Handle edit.
	if msg.EditMessageID != "" {
		editMsgID, _ := strconv.ParseInt(msg.EditMessageID, 10, 64)
		if editMsgID != 0 {
			parseMode := resolveParseMode(msg.ParseMode, msg.Content)
			content := msg.Content
			if parseMode == "HTML" {
				content = markdownToHTML(content)
			}
			return d.sender.editMessage(ctx, chatID, editMsgID, content, parseMode)
		}
	}

	// Determine parse mode and convert content.
	parseMode := resolveParseMode(msg.ParseMode, msg.Content)
	content := msg.Content
	if parseMode == "HTML" {
		content = markdownToHTML(content)
	}

	// Build send options.
	opts := sendOpts{
		parseMode:   parseMode,
		linkPreview: d.config.LinkPreview,
		replyMarkup: msg.ReplyMarkup,
	}

	if msg.ReplyTo != "" {
		opts.replyToMessageID, _ = strconv.ParseInt(msg.ReplyTo, 10, 64)
	}
	if msg.ThreadID != "" {
		opts.threadID, _ = strconv.ParseInt(msg.ThreadID, 10, 64)
	}

	_, err = d.sender.splitAndSend(ctx, chatID, content, opts)
	return err
}

// resolveParseMode determines the parse mode for outbound messages.
// If explicitly set, use that. Otherwise default to HTML (since we convert
// markdown to Telegram HTML internally).
func resolveParseMode(explicit string, _ string) string {
	if explicit != "" {
		return explicit
	}
	// Default: use HTML since markdownToHTML handles the conversion.
	return "HTML"
}

// RegisterSkillCommands sets bot commands from user-invocable skills via
// Telegram's setMyCommands API. Built-in commands are always included first,
// followed by eligible user-invocable skills (up to Telegram's 100-command limit).
func (d *Driver) RegisterSkillCommands(ctx context.Context, skills []*skill.Skill) error {
	type botCommand struct {
		Command     string `json:"command"`
		Description string `json:"description"`
	}

	// Built-in commands always present.
	commands := []botCommand{
		{Command: "new", Description: "Start a new conversation"},
		{Command: "status", Description: "Show bot status"},
		{Command: "help", Description: "Show available commands"},
	}

	// Add skill commands.
	for _, s := range skills {
		if !s.Ready || !s.UserInvocable || s.DisableModelInvocation {
			continue
		}
		desc := s.Description
		if desc == "" {
			desc = s.Name
		}
		if len(desc) > 256 {
			desc = desc[:253] + "..."
		}
		commands = append(commands, botCommand{
			Command:     s.Name,
			Description: desc,
		})
		if len(commands) >= 100 {
			break
		}
	}

	payload := map[string]any{
		"commands": commands,
	}
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/setMyCommands", d.config.BotToken)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build setMyCommands request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram setMyCommands: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram setMyCommands %d: %s", resp.StatusCode, respBody)
	}

	log.Printf("Registered %d Telegram commands (%d skills)", len(commands), len(commands)-3)
	return nil
}

// poll performs long-polling for Telegram updates and dispatches them.
// It requests all update types: message, edited_message, callback_query,
// message_reaction.
func (d *Driver) poll(ctx context.Context) {
	var offset int64

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, nextOffset, err := d.fetchUpdates(ctx, offset)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		offset = nextOffset

		for _, update := range updates {
			d.dispatchUpdate(ctx, &update)
		}
	}
}

// fetchUpdates calls getUpdates and returns parsed updates.
func (d *Driver) fetchUpdates(ctx context.Context, offset int64) ([]TelegramUpdate, int64, error) {
	url := fmt.Sprintf(
		"https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30&allowed_updates=%s",
		d.config.BotToken, offset,
		"[\"message\",\"edited_message\",\"callback_query\",\"message_reaction\"]",
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, offset, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, offset, err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool             `json:"ok"`
		Result []TelegramUpdate `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, offset, err
	}

	nextOffset := offset
	for _, u := range result.Result {
		if u.UpdateID >= nextOffset {
			nextOffset = u.UpdateID + 1
		}
	}

	return result.Result, nextOffset, nil
}

// dispatchUpdate routes an update to the appropriate handler.
func (d *Driver) dispatchUpdate(ctx context.Context, update *TelegramUpdate) {
	switch {
	case update.Message != nil:
		d.handleMessage(ctx, update.Message, false)

	case update.EditedMessage != nil:
		d.handleMessage(ctx, update.EditedMessage, true)

	case update.CallbackQuery != nil:
		d.handleCallbackQuery(ctx, update.CallbackQuery)

	case update.MessageReaction != nil:
		// Log reactions but don't dispatch as inbound messages for now.
		log.Printf("telegram: reaction on message %d in chat %d",
			update.MessageReaction.MessageID, update.MessageReaction.Chat.ID)
	}
}

// handleMessage processes a regular or edited message and dispatches it
// through the message handler.
func (d *Driver) handleMessage(ctx context.Context, tgMsg *TelegramMessage, _ bool) {
	if tgMsg.From == nil {
		return // Channel posts without a "from" sender.
	}

	chatID := strconv.FormatInt(tgMsg.Chat.ID, 10)
	senderID := strconv.FormatInt(tgMsg.From.ID, 10)

	// Determine origin.
	origin := "dm"
	if tgMsg.Chat.Type == "group" || tgMsg.Chat.Type == "supergroup" {
		origin = "group"
	}

	// Access control.
	if origin == "dm" {
		result := checkDMAccess(&d.config, senderID)
		if !result.allowed {
			if result.pairing {
				log.Printf("telegram: DM from %s requires pairing", buildSenderLabel(tgMsg))
			} else {
				log.Printf("telegram: DM from %s denied: %s", buildSenderLabel(tgMsg), result.reason)
			}
			return
		}
	} else {
		result := checkGroupAccess(&d.config, chatID, senderID)
		if !result.allowed {
			log.Printf("telegram: group %s message from %s denied: %s", chatID, senderID, result.reason)
			return
		}

		// Check topic access for forums.
		threadID := strconv.FormatInt(tgMsg.MessageThreadID, 10)
		if tgMsg.Chat.IsForum && tgMsg.MessageThreadID != 0 {
			result := checkTopicAccess(&d.config, chatID, threadID)
			if !result.allowed {
				log.Printf("telegram: topic %s in group %s denied: %s", threadID, chatID, result.reason)
				return
			}
		}

		// In groups, check if the bot is mentioned (if required).
		if d.config.RequireMention && d.botUsername != "" {
			mentioned := hasBotMention(tgMsg, d.botUsername)
			if !mentioned {
				return // Silently ignore non-mentioned messages.
			}
		}
	}

	// Build content: prefer text, then caption, then media placeholder.
	content := tgMsg.Text
	if content == "" {
		content = tgMsg.Caption
	}

	// Expand text_link entities to markdown links.
	if content == tgMsg.Text && len(tgMsg.Entities) > 0 {
		content = expandTextLinks(content, tgMsg.Entities)
	} else if content == tgMsg.Caption && len(tgMsg.CaptionEntities) > 0 {
		content = expandTextLinks(content, tgMsg.CaptionEntities)
	}

	// Extract media.
	mediaType, mediaFileID, _ := extractMedia(tgMsg)

	// If no text content and has media, use a media placeholder.
	if content == "" && mediaType != "" {
		content = "<media:" + mediaType + ">"
	}

	// Extract location (if present and no other content).
	if content == "" {
		if lat, lon, name, address, ok := extractLocation(tgMsg); ok {
			content = formatLocationText(lat, lon, name, address)
			mediaType = "location"
		}
	}

	// Nothing to process.
	if content == "" {
		return
	}

	// Build peer name.
	peerName := buildSenderName(tgMsg.From)

	// Build group ID.
	var groupID string
	if origin == "group" {
		groupID = chatID
	}

	// Session key (OpenClaw compatible).
	sessKey := sessionKey(tgMsg.Chat.ID, tgMsg.Chat.Type, tgMsg.Chat.IsForum, tgMsg.MessageThreadID)

	// Reply context.
	var replyCtxJSON string
	if rc := extractReplyContext(tgMsg); rc != nil {
		if data, err := json.Marshal(rc); err == nil {
			replyCtxJSON = string(data)
		}
	}

	// Forward context.
	var fwdCtxJSON string
	if fc := extractForwardContext(tgMsg); fc != nil {
		if data, err := json.Marshal(fc); err == nil {
			fwdCtxJSON = string(data)
		}
	}

	// Bot mention check (for stripping from content).
	mentionsBot := false
	if d.botUsername != "" {
		mentionsBot = hasBotMention(tgMsg, d.botUsername)
	}

	// Strip bot mention from content if present.
	if mentionsBot && d.botUsername != "" {
		content = stripBotMention(content, d.botUsername)
	}

	// Threading.
	var threadIDStr string
	if tgMsg.MessageThreadID != 0 {
		threadIDStr = strconv.FormatInt(tgMsg.MessageThreadID, 10)
	}

	msg := &types.InboundMessage{
		ChannelType:    types.ChannelTelegram,
		ChannelID:      chatID,
		PeerID:         senderID,
		PeerName:       peerName,
		Content:        content,
		Origin:         origin,
		GroupID:        groupID,
		ReplyTo:        strconv.FormatInt(tgMsg.MessageID, 10),
		SessionKey:     sessKey,
		MediaType:      mediaType,
		MediaFileID:    mediaFileID,
		MediaCaption:   tgMsg.Caption,
		ThreadID:       threadIDStr,
		IsForum:        tgMsg.Chat.IsForum,
		ReplyContext:   replyCtxJSON,
		ForwardContext: fwdCtxJSON,
		MentionsBot:    mentionsBot,
		BotUsername:     d.botUsername,
	}

	// Send ack reaction if configured.
	if d.config.AckEmoji != "" {
		_ = d.sender.setReaction(ctx, tgMsg.Chat.ID, tgMsg.MessageID, d.config.AckEmoji)
	}

	if err := d.handler(ctx, msg); err != nil {
		log.Printf("telegram: handler error for message %d: %v", tgMsg.MessageID, err)
	}
}

// handleCallbackQuery processes an inline keyboard button press.
func (d *Driver) handleCallbackQuery(ctx context.Context, cq *TelegramCallbackQuery) {
	// Always acknowledge the callback to remove the loading indicator.
	_ = d.sender.answerCallbackQuery(ctx, cq.ID, "")

	if cq.Data == "" {
		return
	}

	// Parse callback data.
	command, args := parseCallbackData(cq.Data)
	content := "/" + command
	if len(args) > 0 {
		content += " " + strings.Join(args, " ")
	}

	// Determine chat context from the original message.
	var chatID int64
	var chatType string
	var isForum bool
	var threadID int64
	if cq.Message != nil {
		chatID = cq.Message.Chat.ID
		chatType = cq.Message.Chat.Type
		isForum = cq.Message.Chat.IsForum
		threadID = cq.Message.MessageThreadID
	}

	origin := "dm"
	if chatType == "group" || chatType == "supergroup" {
		origin = "group"
	}

	senderID := strconv.FormatInt(cq.From.ID, 10)
	chatIDStr := strconv.FormatInt(chatID, 10)

	var groupID string
	if origin == "group" {
		groupID = chatIDStr
	}

	sessKey := sessionKey(chatID, chatType, isForum, threadID)

	// Serialize raw callback data for platform-specific handling.
	rawData, _ := json.Marshal(cq)

	msg := &types.InboundMessage{
		ChannelType: types.ChannelTelegram,
		ChannelID:   chatIDStr,
		PeerID:      senderID,
		PeerName:    buildSenderName(&cq.From),
		Content:     content,
		Origin:      origin,
		GroupID:     groupID,
		SessionKey:  sessKey,
		RawMessage:  string(rawData),
	}

	if err := d.handler(ctx, msg); err != nil {
		log.Printf("telegram: handler error for callback %s: %v", cq.ID, err)
	}
}

// stripBotMention removes @botUsername from message content, cleaning up
// whitespace left behind.
func stripBotMention(content string, botUsername string) string {
	target := "@" + botUsername
	// Case-insensitive removal.
	lower := strings.ToLower(content)
	lowerTarget := strings.ToLower(target)
	idx := strings.Index(lower, lowerTarget)
	if idx == -1 {
		return content
	}
	result := content[:idx] + content[idx+len(target):]
	return strings.TrimSpace(result)
}

// splitMessage splits content into chunks that fit within Telegram's limit.
func splitMessage(content string) []string {
	if len(content) <= telegramMaxLength {
		return []string{content}
	}

	var chunks []string
	for len(content) > 0 {
		if len(content) <= telegramMaxLength {
			chunks = append(chunks, content)
			break
		}
		// Find a good split point (newline, space, or hard limit).
		splitAt := telegramMaxLength
		if idx := strings.LastIndex(content[:splitAt], "\n"); idx > telegramMaxLength/2 {
			splitAt = idx + 1
		} else if idx := strings.LastIndex(content[:splitAt], " "); idx > telegramMaxLength/2 {
			splitAt = idx + 1
		}
		chunks = append(chunks, content[:splitAt])
		content = content[splitAt:]
	}
	return chunks
}
