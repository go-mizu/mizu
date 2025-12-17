package live

import (
	"bytes"
	"testing"
)

func TestJSONCodec_Encode(t *testing.T) {
	codec := JSONCodec{}

	tests := []struct {
		name    string
		msg     Message
		want    string
		wantErr bool
	}{
		{
			name: "basic message",
			msg: Message{
				Type:  "message",
				Topic: "room:1",
				Ref:   "123",
				Body:  []byte("hello"),
			},
			want: `{"type":"message","topic":"room:1","ref":"123","body":"aGVsbG8="}`,
		},
		{
			name: "type only",
			msg: Message{
				Type: "ping",
			},
			want: `{"type":"ping"}`,
		},
		{
			name: "with empty topic",
			msg: Message{
				Type: "ack",
				Ref:  "abc",
			},
			want: `{"type":"ack","ref":"abc"}`,
		},
		{
			name: "nil body",
			msg: Message{
				Type:  "subscribe",
				Topic: "room:2",
			},
			want: `{"type":"subscribe","topic":"room:2"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := codec.Encode(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Encode() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestJSONCodec_Decode(t *testing.T) {
	codec := JSONCodec{}

	tests := []struct {
		name    string
		data    string
		want    Message
		wantErr bool
	}{
		{
			name: "basic message",
			data: `{"type":"message","topic":"room:1","ref":"123","body":"aGVsbG8="}`,
			want: Message{
				Type:  "message",
				Topic: "room:1",
				Ref:   "123",
				Body:  []byte("hello"),
			},
		},
		{
			name: "type only",
			data: `{"type":"ping"}`,
			want: Message{
				Type: "ping",
			},
		},
		{
			name:    "invalid json",
			data:    `{invalid`,
			wantErr: true,
		},
		{
			name: "empty json",
			data: `{}`,
			want: Message{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := codec.Decode([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %s, want %s", got.Type, tt.want.Type)
			}
			if got.Topic != tt.want.Topic {
				t.Errorf("Topic = %s, want %s", got.Topic, tt.want.Topic)
			}
			if got.Ref != tt.want.Ref {
				t.Errorf("Ref = %s, want %s", got.Ref, tt.want.Ref)
			}
			if !bytes.Equal(got.Body, tt.want.Body) {
				t.Errorf("Body = %v, want %v", got.Body, tt.want.Body)
			}
		})
	}
}

func TestJSONCodec_RoundTrip(t *testing.T) {
	codec := JSONCodec{}

	original := Message{
		Type:  "message",
		Topic: "chat:room:123",
		Ref:   "ref-456",
		Body:  []byte(`{"text":"Hello, World!"}`),
	}

	encoded, err := codec.Encode(original)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	decoded, err := codec.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, original.Type)
	}
	if decoded.Topic != original.Topic {
		t.Errorf("Topic mismatch: got %s, want %s", decoded.Topic, original.Topic)
	}
	if decoded.Ref != original.Ref {
		t.Errorf("Ref mismatch: got %s, want %s", decoded.Ref, original.Ref)
	}
	if !bytes.Equal(decoded.Body, original.Body) {
		t.Errorf("Body mismatch: got %s, want %s", decoded.Body, original.Body)
	}
}
