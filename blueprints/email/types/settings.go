package types

// Settings represents user preferences for the email client.
type Settings struct {
	ID               int    `json:"id"`
	DisplayName      string `json:"display_name"`
	EmailAddress     string `json:"email_address"`
	Signature        string `json:"signature"`
	Theme            string `json:"theme"`
	Density          string `json:"density"`
	ConversationView bool   `json:"conversation_view"`
	AutoAdvance      string `json:"auto_advance"`
	UndoSendSeconds  int    `json:"undo_send_seconds"`
}
