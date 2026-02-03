package telegram

import "encoding/json"

// TelegramUpdate represents a Telegram Bot API update.
type TelegramUpdate struct {
	UpdateID        int64                    `json:"update_id"`
	Message         *TelegramMessage         `json:"message,omitempty"`
	EditedMessage   *TelegramMessage         `json:"edited_message,omitempty"`
	CallbackQuery   *TelegramCallbackQuery   `json:"callback_query,omitempty"`
	MessageReaction *TelegramMessageReaction  `json:"message_reaction,omitempty"`
}

// TelegramMessage represents a complete Telegram message with all supported fields.
type TelegramMessage struct {
	MessageID       int64                  `json:"message_id"`
	From            *TelegramUser          `json:"from,omitempty"`
	Chat            TelegramChat           `json:"chat"`
	Date            int64                  `json:"date"`
	Text            string                 `json:"text,omitempty"`
	Caption         string                 `json:"caption,omitempty"`
	Entities        []TelegramEntity       `json:"entities,omitempty"`
	CaptionEntities []TelegramEntity       `json:"caption_entities,omitempty"`
	ReplyToMessage  *TelegramMessage       `json:"reply_to_message,omitempty"`
	MessageThreadID int64                  `json:"message_thread_id,omitempty"`
	IsTopicMessage  bool                   `json:"is_topic_message,omitempty"`
	ForwardOrigin   *TelegramForwardOrigin `json:"forward_origin,omitempty"`
	ForwardFrom     *TelegramUser          `json:"forward_from,omitempty"`
	ForwardFromChat *TelegramChat          `json:"forward_from_chat,omitempty"`
	ForwardDate     int64                  `json:"forward_date,omitempty"`
	ForwardSenderName string              `json:"forward_sender_name,omitempty"`
	ForwardSignature  string              `json:"forward_signature,omitempty"`
	Quote           *TelegramQuote         `json:"quote,omitempty"`
	Photo           []TelegramPhotoSize    `json:"photo,omitempty"`
	Video           *TelegramVideo         `json:"video,omitempty"`
	VideoNote       *TelegramVideoNote     `json:"video_note,omitempty"`
	Audio           *TelegramAudio         `json:"audio,omitempty"`
	Voice           *TelegramVoice         `json:"voice,omitempty"`
	Document        *TelegramDocument      `json:"document,omitempty"`
	Sticker         *TelegramSticker       `json:"sticker,omitempty"`
	Location        *TelegramLocation      `json:"location,omitempty"`
	Venue           *TelegramVenue         `json:"venue,omitempty"`
	MediaGroupID    string                 `json:"media_group_id,omitempty"`
	MigrateToChatID int64                  `json:"migrate_to_chat_id,omitempty"`
}

// TelegramUser represents a Telegram user or bot account.
type TelegramUser struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// TelegramChat represents a Telegram chat (private, group, supergroup, or channel).
type TelegramChat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"` // private, group, supergroup, channel
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
	IsForum  bool   `json:"is_forum,omitempty"`
}

// TelegramPhotoSize represents one size variant of a photo or thumbnail.
type TelegramPhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// TelegramVideo represents a video file.
type TelegramVideo struct {
	FileID       string             `json:"file_id"`
	FileUniqueID string             `json:"file_unique_id"`
	Width        int                `json:"width"`
	Height       int                `json:"height"`
	Duration     int                `json:"duration"`
	MimeType     string             `json:"mime_type,omitempty"`
	FileSize     int64              `json:"file_size,omitempty"`
	Thumbnail    *TelegramPhotoSize `json:"thumbnail,omitempty"`
}

// TelegramVideoNote represents a video message (round video).
type TelegramVideoNote struct {
	FileID       string             `json:"file_id"`
	FileUniqueID string             `json:"file_unique_id"`
	Length       int                `json:"length"`
	Duration     int                `json:"duration"`
	Thumbnail    *TelegramPhotoSize `json:"thumbnail,omitempty"`
}

// TelegramAudio represents an audio file.
type TelegramAudio struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	Performer    string `json:"performer,omitempty"`
	Title        string `json:"title,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// TelegramVoice represents a voice message.
type TelegramVoice struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Duration     int    `json:"duration"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// TelegramDocument represents a general file (not photo, audio, video, or voice).
type TelegramDocument struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileName     string `json:"file_name,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
}

// TelegramSticker represents a sticker.
type TelegramSticker struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Type         string `json:"type"` // regular, mask, custom_emoji
	IsAnimated   bool   `json:"is_animated"`
	IsVideo      bool   `json:"is_video"`
	Emoji        string `json:"emoji,omitempty"`
	SetName      string `json:"set_name,omitempty"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

// TelegramLocation represents a geographic location.
type TelegramLocation struct {
	Latitude             float64 `json:"latitude"`
	Longitude            float64 `json:"longitude"`
	HorizontalAccuracy   float64 `json:"horizontal_accuracy,omitempty"`
	LivePeriod           int     `json:"live_period,omitempty"`
	Heading              int     `json:"heading,omitempty"`
	ProximityAlertRadius int     `json:"proximity_alert_radius,omitempty"`
}

// TelegramVenue represents a venue with location information.
type TelegramVenue struct {
	Location        TelegramLocation `json:"location"`
	Title           string           `json:"title"`
	Address         string           `json:"address"`
	FoursquareID    string           `json:"foursquare_id,omitempty"`
	FoursquareType  string           `json:"foursquare_type,omitempty"`
	GooglePlaceID   string           `json:"google_place_id,omitempty"`
	GooglePlaceType string           `json:"google_place_type,omitempty"`
}

// TelegramEntity represents a special entity in a text message (command, mention, URL, etc.).
type TelegramEntity struct {
	Type          string        `json:"type"`
	Offset        int           `json:"offset"`
	Length        int           `json:"length"`
	URL           string        `json:"url,omitempty"`
	User          *TelegramUser `json:"user,omitempty"`
	Language      string        `json:"language,omitempty"`
	CustomEmojiID string        `json:"custom_emoji_id,omitempty"`
}

// TelegramForwardOrigin represents the origin of a forwarded message.
type TelegramForwardOrigin struct {
	Type           string        `json:"type"` // user, hidden_user, chat, channel
	SenderUser     *TelegramUser `json:"sender_user,omitempty"`
	SenderUserName string        `json:"sender_user_name,omitempty"`
	SenderChat     *TelegramChat `json:"sender_chat,omitempty"`
	Chat           *TelegramChat `json:"chat,omitempty"`
	Date           int64         `json:"date,omitempty"`
}

// TelegramQuote represents a quoted part of a message being replied to.
type TelegramQuote struct {
	Text     string           `json:"text"`
	Entities []TelegramEntity `json:"entities,omitempty"`
	Position int              `json:"position"`
	IsManual bool             `json:"is_manual,omitempty"`
}

// TelegramCallbackQuery represents an incoming callback query from an inline keyboard button.
type TelegramCallbackQuery struct {
	ID              string           `json:"id"`
	From            TelegramUser     `json:"from"`
	Message         *TelegramMessage `json:"message,omitempty"`
	InlineMessageID string           `json:"inline_message_id,omitempty"`
	ChatInstance    string           `json:"chat_instance"`
	Data            string           `json:"data,omitempty"`
	GameShortName   string           `json:"game_short_name,omitempty"`
}

// TelegramMessageReaction represents a change in message reactions.
type TelegramMessageReaction struct {
	Chat        TelegramChat           `json:"chat"`
	MessageID   int64                  `json:"message_id"`
	User        *TelegramUser          `json:"user,omitempty"`
	ActorChat   *TelegramChat          `json:"actor_chat,omitempty"`
	Date        int64                  `json:"date"`
	OldReaction []TelegramReactionType `json:"old_reaction"`
	NewReaction []TelegramReactionType `json:"new_reaction"`
}

// TelegramReactionType represents a reaction type (emoji or custom emoji).
type TelegramReactionType struct {
	Type          string `json:"type"` // emoji, custom_emoji
	Emoji         string `json:"emoji,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

// TelegramInlineKeyboardMarkup represents an inline keyboard attached to a message.
type TelegramInlineKeyboardMarkup struct {
	InlineKeyboard [][]TelegramInlineKeyboardButton `json:"inline_keyboard"`
}

// TelegramInlineKeyboardButton represents a single button in an inline keyboard.
type TelegramInlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

// TelegramAPIResponse represents a generic Telegram Bot API response.
type TelegramAPIResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	Description string          `json:"description,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
}

// TelegramFile represents a file ready to be downloaded from Telegram.
type TelegramFile struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int64  `json:"file_size,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
}

// TelegramDriverConfig is the enhanced Telegram driver configuration
// that supports DM/group policies, media limits, mention handling, and
// per-group/topic overrides.
type TelegramDriverConfig struct {
	BotToken       string                 `json:"botToken"`
	DMPolicy       string                 `json:"dmPolicy"`       // open, disabled, allowlist, pairing
	AllowFrom      []string               `json:"allowFrom"`
	GroupPolicy    string                 `json:"groupPolicy"`    // open, disabled, allowlist
	GroupAllowFrom []string               `json:"groupAllowFrom"`
	HistoryLimit   int                    `json:"historyLimit"`
	StreamMode     string                 `json:"streamMode"`
	MediaMaxMB     int                    `json:"mediaMaxMb"`
	LinkPreview    bool                   `json:"linkPreview"`
	RequireMention bool                   `json:"requireMention"`
	ReplyToMode    string                 `json:"replyToMode"`
	AckReaction    string                 `json:"ackReaction"`
	AckEmoji       string                 `json:"ackEmoji"`
	WebhookURL     string                 `json:"webhookUrl,omitempty"`
	Groups         map[string]GroupConfig `json:"groups,omitempty"`
}

// GroupConfig holds per-group overrides for a Telegram group/supergroup.
type GroupConfig struct {
	Enabled      *bool                  `json:"enabled,omitempty"`
	AllowFrom    []string               `json:"allowFrom,omitempty"`
	SystemPrompt string                 `json:"systemPrompt,omitempty"`
	Skills       []string               `json:"skills,omitempty"`
	Topics       map[string]TopicConfig `json:"topics,omitempty"`
}

// TopicConfig holds per-topic (forum thread) overrides within a group.
type TopicConfig struct {
	Enabled      *bool    `json:"enabled,omitempty"`
	SystemPrompt string   `json:"systemPrompt,omitempty"`
	Skills       []string `json:"skills,omitempty"`
}
