// Package tokenizer implements a BERT WordPiece tokenizer in pure Go.
//
// It supports the tokenization pipeline used by BERT-family models:
// lowercase → punctuation splitting → WordPiece subword tokenization.
// Designed for use with all-MiniLM-L6-v2 and similar models.
package tokenizer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

const (
	clsToken = "[CLS]"
	sepToken = "[SEP]"
	padToken = "[PAD]"
	unkToken = "[UNK]"
)

// Tokenizer is a BERT WordPiece tokenizer.
type Tokenizer struct {
	vocab    map[string]int32
	invVocab map[int32]string
	maxLen   int
	clsID    int32
	sepID    int32
	padID    int32
	unkID    int32
}

// Encoded holds a single tokenized input.
type Encoded struct {
	InputIDs      []int64
	AttentionMask []int64
	TokenTypeIDs  []int64
}

// New creates a tokenizer from a vocab.txt file (one token per line).
// maxLen is the maximum sequence length including [CLS] and [SEP].
func New(vocabPath string, maxLen int) (*Tokenizer, error) {
	f, err := os.Open(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("tokenizer: open vocab: %w", err)
	}
	defer f.Close()

	vocab := make(map[string]int32, 32000)
	invVocab := make(map[int32]string, 32000)

	scanner := bufio.NewScanner(f)
	var idx int32
	for scanner.Scan() {
		token := scanner.Text()
		vocab[token] = idx
		invVocab[idx] = token
		idx++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("tokenizer: read vocab: %w", err)
	}

	lookup := func(tok string) int32 {
		if id, ok := vocab[tok]; ok {
			return id
		}
		return 0 // fallback; overwritten below if [UNK] exists
	}

	t := &Tokenizer{
		vocab:    vocab,
		invVocab: invVocab,
		maxLen:   maxLen,
		clsID:    lookup(clsToken),
		sepID:    lookup(sepToken),
		padID:    lookup(padToken),
		unkID:    lookup(unkToken),
	}
	return t, nil
}

// NewFromVocab creates a tokenizer from an in-memory vocab map.
func NewFromVocab(vocab map[string]int32, maxLen int) *Tokenizer {
	invVocab := make(map[int32]string, len(vocab))
	for tok, id := range vocab {
		invVocab[id] = tok
	}
	lookup := func(tok string) int32 {
		if id, ok := vocab[tok]; ok {
			return id
		}
		return 0
	}
	return &Tokenizer{
		vocab:    vocab,
		invVocab: invVocab,
		maxLen:   maxLen,
		clsID:    lookup(clsToken),
		sepID:    lookup(sepToken),
		padID:    lookup(padToken),
		unkID:    lookup(unkToken),
	}
}

// VocabSize returns the number of tokens in the vocabulary.
func (t *Tokenizer) VocabSize() int { return len(t.vocab) }

// Encode tokenizes text into a padded Encoded struct.
func (t *Tokenizer) Encode(text string) *Encoded {
	tokens := t.tokenize(text)

	// Truncate to maxLen - 2 (reserve room for [CLS] and [SEP]).
	maxTokens := t.maxLen - 2
	if len(tokens) > maxTokens {
		tokens = tokens[:maxTokens]
	}

	ids := make([]int64, t.maxLen)
	mask := make([]int64, t.maxLen)
	typeIDs := make([]int64, t.maxLen)

	ids[0] = int64(t.clsID)
	mask[0] = 1
	for i, tok := range tokens {
		ids[i+1] = int64(t.lookupID(tok))
		mask[i+1] = 1
	}
	ids[len(tokens)+1] = int64(t.sepID)
	mask[len(tokens)+1] = 1
	// Remaining positions stay 0 (pad).

	return &Encoded{
		InputIDs:      ids,
		AttentionMask: mask,
		TokenTypeIDs:  typeIDs,
	}
}

// EncodeBatch tokenizes multiple texts.
func (t *Tokenizer) EncodeBatch(texts []string) []*Encoded {
	results := make([]*Encoded, len(texts))
	for i, text := range texts {
		results[i] = t.Encode(text)
	}
	return results
}

// tokenize runs the full BERT tokenization pipeline: lowercase →
// basic tokenize (whitespace + punctuation split) → WordPiece.
func (t *Tokenizer) tokenize(text string) []string {
	text = strings.ToLower(text)
	basicTokens := basicTokenize(text)

	var result []string
	for _, token := range basicTokens {
		subTokens := t.wordPieceTokenize(token)
		result = append(result, subTokens...)
	}
	return result
}

// basicTokenize splits on whitespace and punctuation.
func basicTokenize(text string) []string {
	var tokens []string
	var buf strings.Builder

	flush := func() {
		if buf.Len() > 0 {
			tokens = append(tokens, buf.String())
			buf.Reset()
		}
	}

	for _, r := range text {
		if unicode.IsSpace(r) {
			flush()
		} else if isPunctuation(r) {
			flush()
			tokens = append(tokens, string(r))
		} else {
			buf.WriteRune(r)
		}
	}
	flush()
	return tokens
}

// wordPieceTokenize splits a single token into WordPiece sub-tokens.
func (t *Tokenizer) wordPieceTokenize(token string) []string {
	if _, ok := t.vocab[token]; ok {
		return []string{token}
	}

	var subTokens []string
	start := 0
	runes := []rune(token)

	for start < len(runes) {
		end := len(runes)
		var matched string
		for end > start {
			substr := string(runes[start:end])
			if start > 0 {
				substr = "##" + substr
			}
			if _, ok := t.vocab[substr]; ok {
				matched = substr
				break
			}
			end--
		}
		if matched == "" {
			// Unknown sub-token; emit [UNK] for the whole token.
			return []string{unkToken}
		}
		subTokens = append(subTokens, matched)
		start = end
	}
	return subTokens
}

func (t *Tokenizer) lookupID(token string) int32 {
	if id, ok := t.vocab[token]; ok {
		return id
	}
	return t.unkID
}

func isPunctuation(r rune) bool {
	if (r >= 33 && r <= 47) || (r >= 58 && r <= 64) ||
		(r >= 91 && r <= 96) || (r >= 123 && r <= 126) {
		return true
	}
	return unicode.IsPunct(r)
}
