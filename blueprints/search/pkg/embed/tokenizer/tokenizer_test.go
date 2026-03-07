package tokenizer

import (
	"testing"
)

func testVocab() map[string]int32 {
	return map[string]int32{
		"[PAD]":   0,
		"[UNK]":   100,
		"[CLS]":   101,
		"[SEP]":   102,
		"hello":   7592,
		"world":   2088,
		"the":     1996,
		"quick":   4248,
		"brown":   2829,
		"fox":     4419,
		"jumps":   14523,
		"over":    2058,
		"lazy":    13971,
		"dog":     3899,
		"un":      4895,
		"##believ": 15588,
		"##able":  4014,
		",":       1010,
		".":       1012,
		"!":       999,
		"a":       1037,
	}
}

func TestBasicTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"hello world", []string{"hello", "world"}},
		{"hello, world!", []string{"hello", ",", "world", "!"}},
		{"  spaced  out  ", []string{"spaced", "out"}},
		{"", nil},
	}

	for _, tt := range tests {
		got := basicTokenize(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("basicTokenize(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("basicTokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestWordPieceTokenize(t *testing.T) {
	tok := NewFromVocab(testVocab(), 128)

	tests := []struct {
		input string
		want  []string
	}{
		{"hello", []string{"hello"}},
		{"unbelievable", []string{"un", "##believ", "##able"}},
		{"xyz", []string{"[UNK]"}}, // not in vocab
	}

	for _, tt := range tests {
		got := tok.wordPieceTokenize(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("wordPieceTokenize(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("wordPieceTokenize(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestEncode(t *testing.T) {
	tok := NewFromVocab(testVocab(), 8)

	enc := tok.Encode("hello world")

	// [CLS] hello world [SEP] [PAD] [PAD] [PAD] [PAD]
	if enc.InputIDs[0] != 101 { // [CLS]
		t.Errorf("expected [CLS]=101 at position 0, got %d", enc.InputIDs[0])
	}
	if enc.InputIDs[1] != 7592 { // hello
		t.Errorf("expected hello=7592 at position 1, got %d", enc.InputIDs[1])
	}
	if enc.InputIDs[2] != 2088 { // world
		t.Errorf("expected world=2088 at position 2, got %d", enc.InputIDs[2])
	}
	if enc.InputIDs[3] != 102 { // [SEP]
		t.Errorf("expected [SEP]=102 at position 3, got %d", enc.InputIDs[3])
	}
	// Padding
	for i := 4; i < 8; i++ {
		if enc.InputIDs[i] != 0 {
			t.Errorf("expected padding=0 at position %d, got %d", i, enc.InputIDs[i])
		}
	}

	// Attention mask: 1 for real tokens, 0 for padding
	expectedMask := []int64{1, 1, 1, 1, 0, 0, 0, 0}
	for i, want := range expectedMask {
		if enc.AttentionMask[i] != want {
			t.Errorf("AttentionMask[%d] = %d, want %d", i, enc.AttentionMask[i], want)
		}
	}

	// Token type IDs should all be 0 (single sentence)
	for i := 0; i < 8; i++ {
		if enc.TokenTypeIDs[i] != 0 {
			t.Errorf("TokenTypeIDs[%d] = %d, want 0", i, enc.TokenTypeIDs[i])
		}
	}
}

func TestEncodeTruncation(t *testing.T) {
	tok := NewFromVocab(testVocab(), 4) // maxLen=4 → only 2 tokens fit

	enc := tok.Encode("hello world the quick")

	// Should be: [CLS] hello world [SEP] (truncated after 2 tokens)
	if enc.InputIDs[0] != 101 {
		t.Errorf("expected [CLS] at 0, got %d", enc.InputIDs[0])
	}
	if enc.InputIDs[3] != 102 {
		t.Errorf("expected [SEP] at 3, got %d", enc.InputIDs[3])
	}
}

func TestEncodeEmpty(t *testing.T) {
	tok := NewFromVocab(testVocab(), 8)
	enc := tok.Encode("")

	// [CLS] [SEP] [PAD] ...
	if enc.InputIDs[0] != 101 {
		t.Errorf("expected [CLS] at 0, got %d", enc.InputIDs[0])
	}
	if enc.InputIDs[1] != 102 {
		t.Errorf("expected [SEP] at 1, got %d", enc.InputIDs[1])
	}
	if enc.AttentionMask[0] != 1 || enc.AttentionMask[1] != 1 {
		t.Error("expected attention mask 1 for CLS and SEP")
	}
	if enc.AttentionMask[2] != 0 {
		t.Error("expected attention mask 0 for padding")
	}
}
