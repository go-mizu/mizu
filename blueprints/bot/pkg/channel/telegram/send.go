package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// sender handles outbound Telegram API calls.
type sender struct {
	token  string
	client *http.Client
}

// sendMessage sends a text message with optional parse_mode, reply_to, thread_id, and inline keyboard.
// If HTML parse mode fails, it retries without parse mode.
func (s *sender) sendMessage(ctx context.Context, chatID int64, text string, opts sendOpts) (int64, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}

	if opts.parseMode != "" {
		payload["parse_mode"] = opts.parseMode
	}
	if opts.replyToMessageID != 0 {
		payload["reply_parameters"] = map[string]any{
			"message_id": opts.replyToMessageID,
		}
	}
	if opts.threadID != 0 {
		payload["message_thread_id"] = opts.threadID
	}
	if opts.replyMarkup != "" {
		var markup json.RawMessage
		if err := json.Unmarshal([]byte(opts.replyMarkup), &markup); err == nil {
			payload["reply_markup"] = markup
		}
	}
	if !opts.linkPreview {
		payload["link_preview_options"] = map[string]any{"is_disabled": true}
	}

	msgID, err := s.callSendMessage(ctx, payload)
	if err != nil && opts.parseMode != "" {
		// Retry without parse mode if formatting failed.
		delete(payload, "parse_mode")
		return s.callSendMessage(ctx, payload)
	}
	return msgID, err
}

// callSendMessage performs the actual sendMessage API call and returns the sent message ID.
func (s *sender) callSendMessage(ctx context.Context, payload map[string]any) (int64, error) {
	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("telegram sendMessage: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("telegram sendMessage %d: %s", resp.StatusCode, respBody)
	}

	// Extract sent message ID from response.
	var apiResp struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err == nil && apiResp.OK {
		return apiResp.Result.MessageID, nil
	}

	return 0, nil
}

// sendOpts holds optional parameters for sending messages.
type sendOpts struct {
	parseMode        string
	replyToMessageID int64
	threadID         int64
	replyMarkup      string // JSON inline keyboard
	linkPreview      bool   // true to enable link preview
}

// editMessage edits an existing message's text.
func (s *sender) editMessage(ctx context.Context, chatID int64, messageID int64, text string, parseMode string) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}
	if parseMode != "" {
		payload["parse_mode"] = parseMode
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", s.token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram editMessageText: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram editMessageText %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

// deleteMessage deletes a message.
func (s *sender) deleteMessage(ctx context.Context, chatID int64, messageID int64) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage", s.token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram deleteMessage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram deleteMessage %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

// setReaction sets a reaction emoji on a message.
func (s *sender) setReaction(ctx context.Context, chatID int64, messageID int64, emoji string) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"reaction": []map[string]any{
			{"type": "emoji", "emoji": emoji},
		},
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/setMessageReaction", s.token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram setMessageReaction: %w", err)
	}
	defer resp.Body.Close()

	// Reactions may not be supported in all chats; log but don't fail.
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram setMessageReaction %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

// answerCallbackQuery acknowledges a callback query from an inline keyboard.
func (s *sender) answerCallbackQuery(ctx context.Context, callbackQueryID string, text string) error {
	payload := map[string]any{
		"callback_query_id": callbackQueryID,
	}
	if text != "" {
		payload["text"] = text
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", s.token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram answerCallbackQuery: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram answerCallbackQuery %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

// sendChatAction sends a "typing" indicator or other chat action.
func (s *sender) sendChatAction(ctx context.Context, chatID int64, action string) error {
	payload := map[string]any{
		"chat_id": chatID,
		"action":  action,
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendChatAction", s.token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram sendChatAction: %w", err)
	}
	resp.Body.Close()
	return nil
}

// getMe retrieves the bot's own user info (for bot username etc.).
func (s *sender) getMe(ctx context.Context) (*TelegramUser, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", s.token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram getMe: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var apiResp struct {
		OK     bool         `json:"ok"`
		Result TelegramUser `json:"result"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("parse getMe response: %w", err)
	}
	if !apiResp.OK {
		return nil, fmt.Errorf("getMe failed: %s", string(body))
	}

	return &apiResp.Result, nil
}

// splitAndSend splits content that exceeds Telegram's 4096-character limit
// and sends each chunk as a separate message. Returns the ID of the last sent message.
func (s *sender) splitAndSend(ctx context.Context, chatID int64, content string, opts sendOpts) (int64, error) {
	chunks := splitMessage(content)
	var lastID int64
	for i, chunk := range chunks {
		chunkOpts := opts
		// Only set reply_to on the first chunk.
		if i > 0 {
			chunkOpts.replyToMessageID = 0
		}
		// Only set inline keyboard on the last chunk.
		if i < len(chunks)-1 {
			chunkOpts.replyMarkup = ""
		}
		id, err := s.sendMessage(ctx, chatID, chunk, chunkOpts)
		if err != nil {
			return lastID, err
		}
		lastID = id
	}
	return lastID, nil
}
