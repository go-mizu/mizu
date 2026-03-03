package rose

import (
	"math"
	"testing"
)

func TestBM25Plus_IncreasingTF(t *testing.T) {
	s1 := bm25Plus(1, 100, 400, 400, 10000)
	s2 := bm25Plus(5, 100, 400, 400, 10000)
	s3 := bm25Plus(20, 100, 400, 400, 10000)
	if !(s1 < s2 && s2 < s3) {
		t.Errorf("scores not increasing: %f %f %f", s1, s2, s3)
	}
}

func TestBM25Plus_LongerDocLower(t *testing.T) {
	short := bm25Plus(3, 100, 200, 400, 10000)
	avg := bm25Plus(3, 100, 400, 400, 10000)
	long := bm25Plus(3, 100, 800, 400, 10000)
	if !(long < avg && avg < short) {
		t.Errorf("length normalisation wrong: %f %f %f", short, avg, long)
	}
}

func TestBM25Plus_Delta(t *testing.T) {
	score := bm25Plus(1, 1, 1000, 400, 10000)
	if score <= 0 {
		t.Errorf("BM25+ score must be > 0, got %f", score)
	}
}

func TestBM25Plus_IDF(t *testing.T) {
	common := bm25Plus(1, 5000, 400, 400, 10000)
	rare := bm25Plus(1, 10, 400, 400, 10000)
	if common >= rare {
		t.Errorf("common %f should be < rare %f", common, rare)
	}
}

func TestBM25Plus_ZeroInputs(t *testing.T) {
	// tf=0 → 0; df=0 → 0; N=0 → 0
	if bm25Plus(0, 100, 400, 400, 10000) != 0 {
		t.Error("tf=0 should return 0")
	}
	if bm25Plus(1, 0, 400, 400, 10000) != 0 {
		t.Error("df=0 should return 0")
	}
	if bm25Plus(1, 100, 400, 400, 0) != 0 {
		t.Error("N=0 should return 0")
	}
}

func TestIDF_NeverNegative(t *testing.T) {
	for df := uint32(1); df <= 10000; df += 100 {
		v := idf(df, 10000)
		if v < 0 {
			t.Errorf("idf(%d, 10000) = %f, want >= 0", df, v)
		}
	}
}

func TestIDF_MonotoneDecreasing(t *testing.T) {
	if idf(1, 10000) <= idf(5000, 10000) {
		t.Error("rarer term should have higher IDF")
	}
}

func TestIDF_NeverNaN(t *testing.T) {
	if math.IsNaN(idf(1, 1)) {
		t.Error("idf(1,1) is NaN")
	}
	if math.IsNaN(idf(0, 0)) {
		t.Error("idf(0,0) is NaN")
	}
}

func TestQuantise_Basic(t *testing.T) {
	scores := []float64{0.0, 1.5, 3.0, 6.0, 10.0}
	q := quantise(scores)
	if q[0] != 1 {
		t.Errorf("zero → 1, got %d", q[0])
	}
	if q[4] != 255 {
		t.Errorf("max → 255, got %d", q[4])
	}
	for i := 1; i < len(q); i++ {
		if q[i] < q[i-1] {
			t.Errorf("not monotone at %d", i)
		}
	}
}

func TestQuantise_AllEqual(t *testing.T) {
	q := quantise([]float64{5.0, 5.0, 5.0})
	for _, v := range q {
		if v != 255 {
			t.Errorf("all-equal → all 255, got %d", v)
		}
	}
}

func TestQuantise_Empty(t *testing.T) {
	if len(quantise(nil)) != 0 {
		t.Error("nil → empty")
	}
	if len(quantise([]float64{})) != 0 {
		t.Error("[] → empty")
	}
}

func TestQuantise_NeverZero(t *testing.T) {
	// Even a score of 0.0 must produce uint8 >= 1
	q := quantise([]float64{0.0, 0.0, 5.0})
	for i, v := range q {
		if v == 0 {
			t.Errorf("quantise[%d] = 0, must be >= 1", i)
		}
	}
}

func TestBM25Plus_ZeroAvgdl(t *testing.T) {
	// avgdl=0 should not panic (div-by-zero guard) and return a valid score
	score := bm25Plus(1, 10, 100, 0, 10000)
	if score <= 0 {
		t.Errorf("expected positive score with avgdl=0 guard, got %f", score)
	}
}

func TestBM25Plus_DFExceedsN(t *testing.T) {
	// df > N is an invariant violation, but should not produce negative scores
	score := bm25Plus(1, 20000, 400, 400, 10000)
	if score < 0 {
		t.Errorf("df>N should not produce negative score, got %f", score)
	}
}
