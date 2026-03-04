package dahlia

import (
	"math"
	"testing"
)

func TestBM25IDF(t *testing.T) {
	// Single doc corpus, term appears in 1 doc
	idf := bm25IDF(1, 1)
	if idf < 0 {
		t.Fatalf("IDF should be non-negative, got %f", idf)
	}

	// Rare term in large corpus
	idfRare := bm25IDF(1, 1000000)
	idfCommon := bm25IDF(500000, 1000000)
	if idfRare <= idfCommon {
		t.Fatalf("rare IDF (%f) should exceed common IDF (%f)", idfRare, idfCommon)
	}
}

func TestBM25Score(t *testing.T) {
	// Higher TF should give higher score
	s1 := bm25Score(1, 10, 100, 100, 1000)
	s2 := bm25Score(5, 10, 100, 100, 1000)
	if s2 <= s1 {
		t.Fatalf("higher TF should score higher: tf=1 → %f, tf=5 → %f", s1, s2)
	}

	// Shorter doc should score higher
	sShort := bm25Score(3, 10, 50, 100, 1000)
	sLong := bm25Score(3, 10, 200, 100, 1000)
	if sShort <= sLong {
		t.Fatalf("shorter doc should score higher: dl=50 → %f, dl=200 → %f", sShort, sLong)
	}

	// Score should be positive (BM25+)
	s := bm25Score(1, 999, 500, 100, 1000)
	if s <= 0 {
		t.Fatalf("BM25+ score should be positive, got %f", s)
	}
}

func TestBM25ScoreWithNormTable(t *testing.T) {
	idf := bm25IDF(10, 1000)
	table := buildFieldNormBM25Table(100.0)
	norm := encodeFieldNorm(100) // avgdl
	s := bm25ScoreWithNormTable(3.0, idf, table[norm])
	if s <= 0 || math.IsNaN(s) {
		t.Fatalf("score should be positive, got %f", s)
	}
}

func TestQuantizeRoundTrip(t *testing.T) {
	maxScore := 25.0
	tests := []float64{0, 1.5, 5.0, 12.5, 25.0}
	for _, score := range tests {
		q := quantizeBM25(score, maxScore)
		deq := dequantizeBM25(q, maxScore)
		// Allow 1% error
		if math.Abs(deq-score) > maxScore*0.01+0.1 {
			t.Fatalf("quantize(%f) → %d → %f, too much error", score, q, deq)
		}
	}
}

func TestQuantizeEdgeCases(t *testing.T) {
	if q := quantizeBM25(0, 10); q != 0 {
		t.Fatalf("quantize(0) = %d, want 0", q)
	}
	if q := quantizeBM25(-1, 10); q != 0 {
		t.Fatalf("quantize(-1) = %d, want 0", q)
	}
	if q := quantizeBM25(10, 10); q != 255 {
		t.Fatalf("quantize(max) = %d, want 255", q)
	}
	if q := quantizeBM25(100, 10); q != 255 {
		t.Fatalf("quantize(>max) = %d, want 255", q)
	}
}
