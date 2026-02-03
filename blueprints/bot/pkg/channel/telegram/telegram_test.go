package telegram

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. markdownToHTML (format.go)
// ---------------------------------------------------------------------------

func TestMarkdownToHTML_Bold(t *testing.T) {
	got := markdownToHTML("**text**")
	want := "<b>text</b>"
	if got != want {
		t.Errorf("bold: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_Italic(t *testing.T) {
	got := markdownToHTML("*text*")
	want := "<i>text</i>"
	if got != want {
		t.Errorf("italic: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_InlineCode(t *testing.T) {
	got := markdownToHTML("`code`")
	want := "<code>code</code>"
	if got != want {
		t.Errorf("inline code: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_CodeBlock(t *testing.T) {
	got := markdownToHTML("```code```")
	want := "<pre>code</pre>"
	if got != want {
		t.Errorf("code block: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_Link(t *testing.T) {
	got := markdownToHTML("[text](https://example.com)")
	want := `<a href="https://example.com">text</a>`
	if got != want {
		t.Errorf("link: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_HTMLEscaping(t *testing.T) {
	got := markdownToHTML("a < b > c & d")
	want := "a &lt; b &gt; c &amp; d"
	if got != want {
		t.Errorf("html escaping: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_Mixed(t *testing.T) {
	got := markdownToHTML("**bold** and *italic*")
	want := "<b>bold</b> and <i>italic</i>"
	if got != want {
		t.Errorf("mixed: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_CodeBlockNoEscapeEntities(t *testing.T) {
	// HTML entities inside code blocks should still be escaped for Telegram
	// HTML mode (so that <b> inside code doesn't become a tag), but the
	// escaping is applied to the code content itself, not treated as markdown.
	got := markdownToHTML("```<b>tag</b>```")
	want := "<pre>&lt;b&gt;tag&lt;/b&gt;</pre>"
	if got != want {
		t.Errorf("code block escaping: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_InlineCodeEscaping(t *testing.T) {
	got := markdownToHTML("`a < b`")
	want := "<code>a &lt; b</code>"
	if got != want {
		t.Errorf("inline code escaping: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_CodeBlockWithLanguage(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```"
	got := markdownToHTML(input)
	want := `<pre>fmt.Println("hello")</pre>`
	if got != want {
		t.Errorf("code block with lang: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_PlainText(t *testing.T) {
	got := markdownToHTML("hello world")
	want := "hello world"
	if got != want {
		t.Errorf("plain text: got %q, want %q", got, want)
	}
}

func TestMarkdownToHTML_Empty(t *testing.T) {
	got := markdownToHTML("")
	want := ""
	if got != want {
		t.Errorf("empty: got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// 2. splitMessage (telegram.go)
// ---------------------------------------------------------------------------

func TestSplitMessage_Short(t *testing.T) {
	msg := "Hello, world!"
	chunks := splitMessage(msg)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != msg {
		t.Errorf("chunk mismatch: got %q, want %q", chunks[0], msg)
	}
}

func TestSplitMessage_SplitsAtNewlines(t *testing.T) {
	// Build a message with lines separated by newlines, total > 4096.
	line := strings.Repeat("a", 100) + "\n"
	msg := strings.Repeat(line, 50) // 50 * 101 = 5050 > 4096
	chunks := splitMessage(msg)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	// Verify each chunk is within the limit.
	for i, c := range chunks {
		if len(c) > telegramMaxLength {
			t.Errorf("chunk %d exceeds limit: %d > %d", i, len(c), telegramMaxLength)
		}
	}
	// Verify that reassembling gives us the original.
	reassembled := strings.Join(chunks, "")
	if reassembled != msg {
		t.Error("reassembled message does not match original")
	}
}

func TestSplitMessage_SplitsAtSpaces(t *testing.T) {
	// Build a long message with spaces but no newlines.
	word := strings.Repeat("b", 50) + " "
	msg := strings.Repeat(word, 100) // 100 * 51 = 5100 > 4096
	chunks := splitMessage(msg)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	for i, c := range chunks {
		if len(c) > telegramMaxLength {
			t.Errorf("chunk %d exceeds limit: %d > %d", i, len(c), telegramMaxLength)
		}
	}
}

func TestSplitMessage_HardSplit(t *testing.T) {
	// Build a single line longer than 4096 with no spaces or newlines.
	msg := strings.Repeat("x", 5000)
	chunks := splitMessage(msg)
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	// First chunk should be exactly 4096.
	if len(chunks[0]) != telegramMaxLength {
		t.Errorf("first chunk length: got %d, want %d", len(chunks[0]), telegramMaxLength)
	}
	// Remainder should be 5000 - 4096 = 904.
	if len(chunks[1]) != 904 {
		t.Errorf("second chunk length: got %d, want 904", len(chunks[1]))
	}
}

func TestSplitMessage_ExactLimit(t *testing.T) {
	msg := strings.Repeat("y", telegramMaxLength)
	chunks := splitMessage(msg)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for exact limit, got %d", len(chunks))
	}
}

// ---------------------------------------------------------------------------
// 3. extractReplyContext (context.go)
// ---------------------------------------------------------------------------

func TestExtractReplyContext_NilMessage(t *testing.T) {
	rc := extractReplyContext(nil)
	if rc != nil {
		t.Errorf("expected nil, got %+v", rc)
	}
}

func TestExtractReplyContext_Quote(t *testing.T) {
	msg := &TelegramMessage{
		Quote: &TelegramQuote{Text: "quoted text"},
		ReplyToMessage: &TelegramMessage{
			MessageID: 42,
			From:      &TelegramUser{FirstName: "Alice"},
		},
	}
	rc := extractReplyContext(msg)
	if rc == nil {
		t.Fatal("expected non-nil reply context")
	}
	if rc.Kind != "quote" {
		t.Errorf("kind: got %q, want %q", rc.Kind, "quote")
	}
	if rc.Body != "quoted text" {
		t.Errorf("body: got %q, want %q", rc.Body, "quoted text")
	}
	if rc.ID != "42" {
		t.Errorf("id: got %q, want %q", rc.ID, "42")
	}
	if rc.Sender != "Alice" {
		t.Errorf("sender: got %q, want %q", rc.Sender, "Alice")
	}
}

func TestExtractReplyContext_ReplyToMessage(t *testing.T) {
	msg := &TelegramMessage{
		ReplyToMessage: &TelegramMessage{
			MessageID: 99,
			From:      &TelegramUser{FirstName: "Bob", LastName: "Smith"},
			Text:      "original message",
		},
	}
	rc := extractReplyContext(msg)
	if rc == nil {
		t.Fatal("expected non-nil reply context")
	}
	if rc.Kind != "reply" {
		t.Errorf("kind: got %q, want %q", rc.Kind, "reply")
	}
	if rc.Body != "original message" {
		t.Errorf("body: got %q, want %q", rc.Body, "original message")
	}
	if rc.ID != "99" {
		t.Errorf("id: got %q, want %q", rc.ID, "99")
	}
	if rc.Sender != "Bob Smith" {
		t.Errorf("sender: got %q, want %q", rc.Sender, "Bob Smith")
	}
}

func TestExtractReplyContext_MediaReply(t *testing.T) {
	msg := &TelegramMessage{
		ReplyToMessage: &TelegramMessage{
			MessageID: 55,
			From:      &TelegramUser{FirstName: "Carol"},
			Photo: []TelegramPhotoSize{
				{FileID: "small", FileUniqueID: "s1"},
				{FileID: "large", FileUniqueID: "l1"},
			},
		},
	}
	rc := extractReplyContext(msg)
	if rc == nil {
		t.Fatal("expected non-nil reply context")
	}
	if rc.Body != "<media:image>" {
		t.Errorf("body: got %q, want %q", rc.Body, "<media:image>")
	}
}

func TestExtractReplyContext_NoReply(t *testing.T) {
	msg := &TelegramMessage{
		MessageID: 10,
		Text:      "just a message",
	}
	rc := extractReplyContext(msg)
	if rc != nil {
		t.Errorf("expected nil for non-reply, got %+v", rc)
	}
}

func TestExtractReplyContext_VideoReply(t *testing.T) {
	msg := &TelegramMessage{
		ReplyToMessage: &TelegramMessage{
			MessageID: 60,
			From:      &TelegramUser{FirstName: "Dave"},
			Video:     &TelegramVideo{FileID: "vid1"},
		},
	}
	rc := extractReplyContext(msg)
	if rc == nil {
		t.Fatal("expected non-nil")
	}
	if rc.Body != "<media:video>" {
		t.Errorf("body: got %q, want %q", rc.Body, "<media:video>")
	}
}

func TestExtractReplyContext_StickerReply(t *testing.T) {
	msg := &TelegramMessage{
		ReplyToMessage: &TelegramMessage{
			MessageID: 70,
			From:      &TelegramUser{FirstName: "Eve"},
			Sticker:   &TelegramSticker{FileID: "stk1"},
		},
	}
	rc := extractReplyContext(msg)
	if rc == nil {
		t.Fatal("expected non-nil")
	}
	if rc.Body != "<media:sticker>" {
		t.Errorf("body: got %q, want %q", rc.Body, "<media:sticker>")
	}
}

// ---------------------------------------------------------------------------
// 4. extractForwardContext (context.go)
// ---------------------------------------------------------------------------

func TestExtractForwardContext_UserOrigin(t *testing.T) {
	msg := &TelegramMessage{
		ForwardOrigin: &TelegramForwardOrigin{
			Type: "user",
			Date: 1700000000,
			SenderUser: &TelegramUser{
				ID:        123,
				FirstName: "Alice",
				LastName:  "Wonder",
				Username:  "alicew",
			},
		},
	}
	fc := extractForwardContext(msg)
	if fc == nil {
		t.Fatal("expected non-nil")
	}
	if fc.FromType != "user" {
		t.Errorf("type: got %q, want %q", fc.FromType, "user")
	}
	if fc.From != "Alice Wonder" {
		t.Errorf("from: got %q, want %q", fc.From, "Alice Wonder")
	}
	if fc.FromID != "123" {
		t.Errorf("fromID: got %q, want %q", fc.FromID, "123")
	}
	if fc.FromUsername != "alicew" {
		t.Errorf("fromUsername: got %q, want %q", fc.FromUsername, "alicew")
	}
	if fc.Date != 1700000000 {
		t.Errorf("date: got %d, want 1700000000", fc.Date)
	}
}

func TestExtractForwardContext_HiddenUser(t *testing.T) {
	msg := &TelegramMessage{
		ForwardOrigin: &TelegramForwardOrigin{
			Type:           "hidden_user",
			Date:           1700000001,
			SenderUserName: "Hidden Person",
		},
	}
	fc := extractForwardContext(msg)
	if fc == nil {
		t.Fatal("expected non-nil")
	}
	if fc.FromType != "hidden_user" {
		t.Errorf("type: got %q, want %q", fc.FromType, "hidden_user")
	}
	if fc.From != "Hidden Person" {
		t.Errorf("from: got %q, want %q", fc.From, "Hidden Person")
	}
}

func TestExtractForwardContext_Channel(t *testing.T) {
	msg := &TelegramMessage{
		ForwardOrigin: &TelegramForwardOrigin{
			Type: "channel",
			Date: 1700000002,
			Chat: &TelegramChat{
				ID:       -100123,
				Title:    "News Channel",
				Username: "newschan",
			},
		},
	}
	fc := extractForwardContext(msg)
	if fc == nil {
		t.Fatal("expected non-nil")
	}
	if fc.FromType != "channel" {
		t.Errorf("type: got %q, want %q", fc.FromType, "channel")
	}
	if fc.From != "News Channel" {
		t.Errorf("from: got %q, want %q", fc.From, "News Channel")
	}
	if fc.FromID != "-100123" {
		t.Errorf("fromID: got %q, want %q", fc.FromID, "-100123")
	}
	if fc.FromTitle != "News Channel" {
		t.Errorf("fromTitle: got %q, want %q", fc.FromTitle, "News Channel")
	}
}

func TestExtractForwardContext_LegacyForwardFrom(t *testing.T) {
	msg := &TelegramMessage{
		ForwardFrom: &TelegramUser{
			ID:        456,
			FirstName: "Legacy",
			Username:  "leguser",
		},
		ForwardDate: 1700000003,
	}
	fc := extractForwardContext(msg)
	if fc == nil {
		t.Fatal("expected non-nil")
	}
	if fc.FromType != "legacy_user" {
		t.Errorf("type: got %q, want %q", fc.FromType, "legacy_user")
	}
	if fc.From != "Legacy" {
		t.Errorf("from: got %q, want %q", fc.From, "Legacy")
	}
	if fc.FromID != "456" {
		t.Errorf("fromID: got %q, want %q", fc.FromID, "456")
	}
	if fc.Date != 1700000003 {
		t.Errorf("date: got %d, want 1700000003", fc.Date)
	}
}

func TestExtractForwardContext_LegacySenderName(t *testing.T) {
	msg := &TelegramMessage{
		ForwardSenderName: "Anonymous Forwarder",
		ForwardDate:       1700000004,
	}
	fc := extractForwardContext(msg)
	if fc == nil {
		t.Fatal("expected non-nil")
	}
	if fc.FromType != "legacy_hidden" {
		t.Errorf("type: got %q, want %q", fc.FromType, "legacy_hidden")
	}
	if fc.From != "Anonymous Forwarder" {
		t.Errorf("from: got %q, want %q", fc.From, "Anonymous Forwarder")
	}
}

func TestExtractForwardContext_NotForwarded(t *testing.T) {
	msg := &TelegramMessage{
		MessageID: 10,
		Text:      "normal message",
	}
	fc := extractForwardContext(msg)
	if fc != nil {
		t.Errorf("expected nil, got %+v", fc)
	}
}

func TestExtractForwardContext_NilMessage(t *testing.T) {
	fc := extractForwardContext(nil)
	if fc != nil {
		t.Errorf("expected nil, got %+v", fc)
	}
}

// ---------------------------------------------------------------------------
// 5. buildSenderName (context.go)
// ---------------------------------------------------------------------------

func TestBuildSenderName_FirstAndLast(t *testing.T) {
	user := &TelegramUser{FirstName: "John", LastName: "Doe"}
	got := buildSenderName(user)
	if got != "John Doe" {
		t.Errorf("got %q, want %q", got, "John Doe")
	}
}

func TestBuildSenderName_FirstOnly(t *testing.T) {
	user := &TelegramUser{FirstName: "Jane"}
	got := buildSenderName(user)
	if got != "Jane" {
		t.Errorf("got %q, want %q", got, "Jane")
	}
}

func TestBuildSenderName_UsernameOnly(t *testing.T) {
	user := &TelegramUser{Username: "janedoe"}
	got := buildSenderName(user)
	if got != "@janedoe" {
		t.Errorf("got %q, want %q", got, "@janedoe")
	}
}

func TestBuildSenderName_IDOnly(t *testing.T) {
	user := &TelegramUser{ID: 123}
	got := buildSenderName(user)
	if got != "id:123" {
		t.Errorf("got %q, want %q", got, "id:123")
	}
}

func TestBuildSenderName_Nil(t *testing.T) {
	got := buildSenderName(nil)
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

// ---------------------------------------------------------------------------
// 6. expandTextLinks (context.go)
// ---------------------------------------------------------------------------

func TestExpandTextLinks_SingleLink(t *testing.T) {
	text := "Click here for info"
	entities := []TelegramEntity{
		{Type: "text_link", Offset: 6, Length: 4, URL: "https://example.com"},
	}
	got := expandTextLinks(text, entities)
	want := "Click [here](https://example.com) for info"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandTextLinks_MultipleLinks(t *testing.T) {
	text := "Visit Google or GitHub"
	entities := []TelegramEntity{
		{Type: "text_link", Offset: 6, Length: 6, URL: "https://google.com"},
		{Type: "text_link", Offset: 16, Length: 6, URL: "https://github.com"},
	}
	got := expandTextLinks(text, entities)
	want := "Visit [Google](https://google.com) or [GitHub](https://github.com)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandTextLinks_EmptyEntities(t *testing.T) {
	text := "no links here"
	got := expandTextLinks(text, nil)
	if got != text {
		t.Errorf("got %q, want %q", got, text)
	}
	got = expandTextLinks(text, []TelegramEntity{})
	if got != text {
		t.Errorf("got %q, want %q", got, text)
	}
}

func TestExpandTextLinks_NonTextLinkIgnored(t *testing.T) {
	text := "Hello @user"
	entities := []TelegramEntity{
		{Type: "mention", Offset: 6, Length: 5},
	}
	got := expandTextLinks(text, entities)
	if got != text {
		t.Errorf("got %q, want %q", got, text)
	}
}

// ---------------------------------------------------------------------------
// 7. hasBotMention (context.go)
// ---------------------------------------------------------------------------

func TestHasBotMention_InText(t *testing.T) {
	msg := &TelegramMessage{Text: "Hello @TestBot how are you"}
	if !hasBotMention(msg, "TestBot") {
		t.Error("expected mention to be detected in text")
	}
}

func TestHasBotMention_CaseInsensitive(t *testing.T) {
	msg := &TelegramMessage{Text: "Hello @testbot how are you"}
	if !hasBotMention(msg, "TestBot") {
		t.Error("expected case-insensitive mention detection")
	}
}

func TestHasBotMention_InEntities(t *testing.T) {
	msg := &TelegramMessage{
		Text: "Hello @TestBot",
		Entities: []TelegramEntity{
			{Type: "mention", Offset: 6, Length: 8},
		},
	}
	if !hasBotMention(msg, "TestBot") {
		t.Error("expected mention to be detected in entities")
	}
}

func TestHasBotMention_InCaption(t *testing.T) {
	msg := &TelegramMessage{
		Caption: "Photo for @TestBot",
	}
	if !hasBotMention(msg, "TestBot") {
		t.Error("expected mention to be detected in caption")
	}
}

func TestHasBotMention_NoMention(t *testing.T) {
	msg := &TelegramMessage{Text: "Hello everyone"}
	if hasBotMention(msg, "TestBot") {
		t.Error("expected no mention detected")
	}
}

func TestHasBotMention_EmptyBotUsername(t *testing.T) {
	msg := &TelegramMessage{Text: "Hello @TestBot"}
	if hasBotMention(msg, "") {
		t.Error("expected false for empty bot username")
	}
}

func TestHasBotMention_NilMessage(t *testing.T) {
	if hasBotMention(nil, "TestBot") {
		t.Error("expected false for nil message")
	}
}

// ---------------------------------------------------------------------------
// 8. sessionKey (context.go)
// ---------------------------------------------------------------------------

func TestSessionKey_Private(t *testing.T) {
	got := sessionKey(12345, "private", false, 0)
	want := "telegram:12345"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSessionKey_Group(t *testing.T) {
	got := sessionKey(-100999, "group", false, 0)
	want := "telegram:group:-100999"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSessionKey_Supergroup(t *testing.T) {
	got := sessionKey(-100888, "supergroup", false, 0)
	want := "telegram:group:-100888"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSessionKey_ForumWithTopic(t *testing.T) {
	got := sessionKey(-100777, "supergroup", true, 42)
	want := "telegram:group:-100777:topic:42"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSessionKey_ForumWithoutTopic(t *testing.T) {
	got := sessionKey(-100777, "supergroup", true, 0)
	want := "telegram:group:-100777"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSessionKey_Channel(t *testing.T) {
	got := sessionKey(-100666, "channel", false, 0)
	want := "telegram:-100666"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// 9. checkDMAccess (access.go)
// ---------------------------------------------------------------------------

func TestCheckDMAccess_Open(t *testing.T) {
	cfg := &TelegramDriverConfig{DMPolicy: "open"}
	r := checkDMAccess(cfg, "123")
	if !r.allowed {
		t.Error("expected open policy to allow")
	}
}

func TestCheckDMAccess_Disabled(t *testing.T) {
	cfg := &TelegramDriverConfig{DMPolicy: "disabled"}
	r := checkDMAccess(cfg, "123")
	if r.allowed {
		t.Error("expected disabled policy to deny")
	}
	if r.reason == "" {
		t.Error("expected a denial reason")
	}
}

func TestCheckDMAccess_AllowlistMatch(t *testing.T) {
	cfg := &TelegramDriverConfig{
		DMPolicy:  "allowlist",
		AllowFrom: []string{"100", "200", "300"},
	}
	r := checkDMAccess(cfg, "200")
	if !r.allowed {
		t.Error("expected allowlist match to allow")
	}
}

func TestCheckDMAccess_AllowlistNoMatch(t *testing.T) {
	cfg := &TelegramDriverConfig{
		DMPolicy:  "allowlist",
		AllowFrom: []string{"100", "200"},
	}
	r := checkDMAccess(cfg, "999")
	if r.allowed {
		t.Error("expected allowlist miss to deny")
	}
}

func TestCheckDMAccess_PairingKnownID(t *testing.T) {
	cfg := &TelegramDriverConfig{
		DMPolicy:  "pairing",
		AllowFrom: []string{"100", "200"},
	}
	r := checkDMAccess(cfg, "100")
	if !r.allowed {
		t.Error("expected pairing with known ID to allow")
	}
}

func TestCheckDMAccess_PairingUnknownID(t *testing.T) {
	cfg := &TelegramDriverConfig{
		DMPolicy:  "pairing",
		AllowFrom: []string{"100"},
	}
	r := checkDMAccess(cfg, "999")
	if r.allowed {
		t.Error("expected pairing with unknown ID to deny")
	}
	if !r.pairing {
		t.Error("expected pairing flag to be set")
	}
}

func TestCheckDMAccess_DefaultPolicy(t *testing.T) {
	// Empty DMPolicy defaults to "pairing".
	cfg := &TelegramDriverConfig{
		AllowFrom: []string{"100"},
	}
	r := checkDMAccess(cfg, "100")
	if !r.allowed {
		t.Error("expected default pairing policy with known ID to allow")
	}
	r = checkDMAccess(cfg, "999")
	if r.allowed {
		t.Error("expected default pairing policy with unknown ID to deny")
	}
	if !r.pairing {
		t.Error("expected pairing flag for unknown ID")
	}
}

// ---------------------------------------------------------------------------
// 10. checkGroupAccess (access.go)
// ---------------------------------------------------------------------------

func TestCheckGroupAccess_Open(t *testing.T) {
	cfg := &TelegramDriverConfig{GroupPolicy: "open"}
	r := checkGroupAccess(cfg, "-100123", "456")
	if !r.allowed {
		t.Error("expected open group policy to allow")
	}
}

func TestCheckGroupAccess_Disabled(t *testing.T) {
	cfg := &TelegramDriverConfig{GroupPolicy: "disabled"}
	r := checkGroupAccess(cfg, "-100123", "456")
	if r.allowed {
		t.Error("expected disabled group policy to deny")
	}
}

func TestCheckGroupAccess_AllowlistMatch(t *testing.T) {
	cfg := &TelegramDriverConfig{
		GroupPolicy:    "allowlist",
		GroupAllowFrom: []string{"-100123", "-100456"},
	}
	r := checkGroupAccess(cfg, "-100123", "789")
	if !r.allowed {
		t.Error("expected group allowlist match to allow")
	}
}

func TestCheckGroupAccess_AllowlistNoMatch(t *testing.T) {
	cfg := &TelegramDriverConfig{
		GroupPolicy:    "allowlist",
		GroupAllowFrom: []string{"-100123"},
	}
	r := checkGroupAccess(cfg, "-100999", "789")
	if r.allowed {
		t.Error("expected group allowlist miss to deny")
	}
}

func TestCheckGroupAccess_PerGroupDisabled(t *testing.T) {
	disabled := false
	cfg := &TelegramDriverConfig{
		GroupPolicy: "open",
		Groups: map[string]GroupConfig{
			"-100123": {Enabled: &disabled},
		},
	}
	r := checkGroupAccess(cfg, "-100123", "456")
	if r.allowed {
		t.Error("expected per-group disabled to deny")
	}
}

func TestCheckGroupAccess_PerGroupAllowFrom(t *testing.T) {
	cfg := &TelegramDriverConfig{
		GroupPolicy: "open",
		Groups: map[string]GroupConfig{
			"-100123": {AllowFrom: []string{"111", "222"}},
		},
	}

	r := checkGroupAccess(cfg, "-100123", "111")
	if !r.allowed {
		t.Error("expected per-group allowFrom match to allow")
	}

	r = checkGroupAccess(cfg, "-100123", "999")
	if r.allowed {
		t.Error("expected per-group allowFrom miss to deny")
	}
}

func TestCheckGroupAccess_DefaultPolicy(t *testing.T) {
	// Empty GroupPolicy defaults to "open".
	cfg := &TelegramDriverConfig{}
	r := checkGroupAccess(cfg, "-100123", "456")
	if !r.allowed {
		t.Error("expected default open group policy to allow")
	}
}

// ---------------------------------------------------------------------------
// 11. checkTopicAccess (access.go)
// ---------------------------------------------------------------------------

func TestCheckTopicAccess_EmptyThreadID(t *testing.T) {
	cfg := &TelegramDriverConfig{}
	r := checkTopicAccess(cfg, "-100123", "")
	if !r.allowed {
		t.Error("expected empty threadID to always allow")
	}
}

func TestCheckTopicAccess_NoGroupConfig(t *testing.T) {
	cfg := &TelegramDriverConfig{}
	r := checkTopicAccess(cfg, "-100123", "42")
	if !r.allowed {
		t.Error("expected no group config to allow")
	}
}

func TestCheckTopicAccess_NilGroups(t *testing.T) {
	cfg := &TelegramDriverConfig{Groups: nil}
	r := checkTopicAccess(cfg, "-100123", "42")
	if !r.allowed {
		t.Error("expected nil groups to allow")
	}
}

func TestCheckTopicAccess_TopicDisabled(t *testing.T) {
	disabled := false
	cfg := &TelegramDriverConfig{
		Groups: map[string]GroupConfig{
			"-100123": {
				Topics: map[string]TopicConfig{
					"42": {Enabled: &disabled},
				},
			},
		},
	}
	r := checkTopicAccess(cfg, "-100123", "42")
	if r.allowed {
		t.Error("expected disabled topic to deny")
	}
}

func TestCheckTopicAccess_UnconfiguredTopic(t *testing.T) {
	cfg := &TelegramDriverConfig{
		Groups: map[string]GroupConfig{
			"-100123": {
				Topics: map[string]TopicConfig{
					"42": {},
				},
			},
		},
	}
	r := checkTopicAccess(cfg, "-100123", "99")
	if !r.allowed {
		t.Error("expected unconfigured topic to allow")
	}
}

func TestCheckTopicAccess_TopicEnabled(t *testing.T) {
	enabled := true
	cfg := &TelegramDriverConfig{
		Groups: map[string]GroupConfig{
			"-100123": {
				Topics: map[string]TopicConfig{
					"42": {Enabled: &enabled},
				},
			},
		},
	}
	r := checkTopicAccess(cfg, "-100123", "42")
	if !r.allowed {
		t.Error("expected enabled topic to allow")
	}
}

// ---------------------------------------------------------------------------
// 12. extractMedia (media.go)
// ---------------------------------------------------------------------------

func TestExtractMedia_Photo(t *testing.T) {
	msg := &TelegramMessage{
		Photo: []TelegramPhotoSize{
			{FileID: "small_id", FileUniqueID: "su1"},
			{FileID: "medium_id", FileUniqueID: "mu1"},
			{FileID: "large_id", FileUniqueID: "lu1"},
		},
	}
	mediaType, fileID, _ := extractMedia(msg)
	if mediaType != "image" {
		t.Errorf("type: got %q, want %q", mediaType, "image")
	}
	if fileID != "large_id" {
		t.Errorf("fileID: got %q, want %q (last element)", fileID, "large_id")
	}
}

func TestExtractMedia_Video(t *testing.T) {
	msg := &TelegramMessage{
		Video: &TelegramVideo{FileID: "vid_id", FileUniqueID: "vu1"},
	}
	mediaType, fileID, _ := extractMedia(msg)
	if mediaType != "video" {
		t.Errorf("type: got %q, want %q", mediaType, "video")
	}
	if fileID != "vid_id" {
		t.Errorf("fileID: got %q, want %q", fileID, "vid_id")
	}
}

func TestExtractMedia_Audio(t *testing.T) {
	msg := &TelegramMessage{
		Audio: &TelegramAudio{FileID: "aud_id", FileUniqueID: "au1"},
	}
	mediaType, fileID, _ := extractMedia(msg)
	if mediaType != "audio" {
		t.Errorf("type: got %q, want %q", mediaType, "audio")
	}
	if fileID != "aud_id" {
		t.Errorf("fileID: got %q, want %q", fileID, "aud_id")
	}
}

func TestExtractMedia_Document(t *testing.T) {
	msg := &TelegramMessage{
		Document: &TelegramDocument{FileID: "doc_id", FileUniqueID: "du1", FileName: "report.pdf"},
	}
	mediaType, fileID, filename := extractMedia(msg)
	if mediaType != "document" {
		t.Errorf("type: got %q, want %q", mediaType, "document")
	}
	if fileID != "doc_id" {
		t.Errorf("fileID: got %q, want %q", fileID, "doc_id")
	}
	if filename != "report.pdf" {
		t.Errorf("filename: got %q, want %q", filename, "report.pdf")
	}
}

func TestExtractMedia_Voice(t *testing.T) {
	msg := &TelegramMessage{
		Voice: &TelegramVoice{FileID: "voc_id", FileUniqueID: "vu1", MimeType: "audio/ogg"},
	}
	mediaType, fileID, _ := extractMedia(msg)
	if mediaType != "audio" {
		t.Errorf("type: got %q, want %q", mediaType, "audio")
	}
	if fileID != "voc_id" {
		t.Errorf("fileID: got %q, want %q", fileID, "voc_id")
	}
}

func TestExtractMedia_Sticker(t *testing.T) {
	msg := &TelegramMessage{
		Sticker: &TelegramSticker{FileID: "stk_id", FileUniqueID: "sku1"},
	}
	mediaType, fileID, _ := extractMedia(msg)
	if mediaType != "sticker" {
		t.Errorf("type: got %q, want %q", mediaType, "sticker")
	}
	if fileID != "stk_id" {
		t.Errorf("fileID: got %q, want %q", fileID, "stk_id")
	}
}

func TestExtractMedia_NoMedia(t *testing.T) {
	msg := &TelegramMessage{Text: "just text"}
	mediaType, fileID, filename := extractMedia(msg)
	if mediaType != "" || fileID != "" || filename != "" {
		t.Errorf("expected all empty, got type=%q fileID=%q filename=%q", mediaType, fileID, filename)
	}
}

func TestExtractMedia_PhotoPriority(t *testing.T) {
	// When message has both photo and document, photo should take priority.
	msg := &TelegramMessage{
		Photo: []TelegramPhotoSize{
			{FileID: "photo_id", FileUniqueID: "pu1"},
		},
		Document: &TelegramDocument{FileID: "doc_id", FileUniqueID: "du1"},
	}
	mediaType, fileID, _ := extractMedia(msg)
	if mediaType != "image" {
		t.Errorf("expected photo priority, got type %q", mediaType)
	}
	if fileID != "photo_id" {
		t.Errorf("expected photo fileID, got %q", fileID)
	}
}

// ---------------------------------------------------------------------------
// 13. buildInlineKeyboard (keyboard.go)
// ---------------------------------------------------------------------------

func TestBuildInlineKeyboard_SingleRowCallback(t *testing.T) {
	buttons := [][]OutboundButton{
		{
			{Text: "Click me", CallbackData: "action:1"},
		},
	}
	result := buildInlineKeyboard(buttons)
	rows, ok := result["inline_keyboard"].([][]map[string]any)
	if !ok {
		t.Fatal("expected inline_keyboard key with rows")
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if len(rows[0]) != 1 {
		t.Fatalf("expected 1 button, got %d", len(rows[0]))
	}
	btn := rows[0][0]
	if btn["text"] != "Click me" {
		t.Errorf("text: got %q, want %q", btn["text"], "Click me")
	}
	if btn["callback_data"] != "action:1" {
		t.Errorf("callback_data: got %q, want %q", btn["callback_data"], "action:1")
	}
}

func TestBuildInlineKeyboard_URLButton(t *testing.T) {
	buttons := [][]OutboundButton{
		{
			{Text: "Visit", URL: "https://example.com"},
		},
	}
	result := buildInlineKeyboard(buttons)
	rows := result["inline_keyboard"].([][]map[string]any)
	btn := rows[0][0]
	if btn["url"] != "https://example.com" {
		t.Errorf("url: got %q, want %q", btn["url"], "https://example.com")
	}
	// callback_data should not be set for URL buttons.
	if _, exists := btn["callback_data"]; exists {
		t.Error("callback_data should not be set for URL-only button")
	}
}

func TestBuildInlineKeyboard_MultipleRows(t *testing.T) {
	buttons := [][]OutboundButton{
		{
			{Text: "A", CallbackData: "a"},
			{Text: "B", CallbackData: "b"},
		},
		{
			{Text: "C", URL: "https://c.com"},
		},
	}
	result := buildInlineKeyboard(buttons)
	rows := result["inline_keyboard"].([][]map[string]any)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if len(rows[0]) != 2 {
		t.Errorf("first row: expected 2 buttons, got %d", len(rows[0]))
	}
	if len(rows[1]) != 1 {
		t.Errorf("second row: expected 1 button, got %d", len(rows[1]))
	}
}

// ---------------------------------------------------------------------------
// 14. parseCallbackData (keyboard.go)
// ---------------------------------------------------------------------------

func TestParseCallbackData_CommandWithArgs(t *testing.T) {
	cmd, args := parseCallbackData("command:arg1:arg2")
	if cmd != "command" {
		t.Errorf("command: got %q, want %q", cmd, "command")
	}
	if len(args) != 2 || args[0] != "arg1" || args[1] != "arg2" {
		t.Errorf("args: got %v, want [arg1 arg2]", args)
	}
}

func TestParseCallbackData_CommandOnly(t *testing.T) {
	cmd, args := parseCallbackData("command")
	if cmd != "command" {
		t.Errorf("command: got %q, want %q", cmd, "command")
	}
	if args != nil {
		t.Errorf("args: got %v, want nil", args)
	}
}

func TestParseCallbackData_Empty(t *testing.T) {
	cmd, args := parseCallbackData("")
	if cmd != "" {
		t.Errorf("command: got %q, want empty", cmd)
	}
	if args != nil {
		t.Errorf("args: got %v, want nil", args)
	}
}

func TestParseCallbackData_SingleArg(t *testing.T) {
	cmd, args := parseCallbackData("select:option1")
	if cmd != "select" {
		t.Errorf("command: got %q, want %q", cmd, "select")
	}
	if len(args) != 1 || args[0] != "option1" {
		t.Errorf("args: got %v, want [option1]", args)
	}
}

// ---------------------------------------------------------------------------
// 15. stripBotMention (telegram.go)
// ---------------------------------------------------------------------------

func TestStripBotMention_FromStart(t *testing.T) {
	got := stripBotMention("@TestBot hello world", "TestBot")
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestStripBotMention_FromMiddle(t *testing.T) {
	got := stripBotMention("hello @TestBot world", "TestBot")
	if got != "hello  world" {
		t.Errorf("got %q, want %q", got, "hello  world")
	}
}

func TestStripBotMention_CaseInsensitive(t *testing.T) {
	got := stripBotMention("@testbot hello", "TestBot")
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestStripBotMention_NotPresent(t *testing.T) {
	got := stripBotMention("hello world", "TestBot")
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestStripBotMention_OnlyMention(t *testing.T) {
	got := stripBotMention("@TestBot", "TestBot")
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}
