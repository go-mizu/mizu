package dahlia

import "math"

const (
	bm25K1    = 1.2
	bm25B     = 0.75
	bm25Delta = 1.0
)

// bm25IDF computes the inverse document frequency component.
// df = document frequency of term, n = total document count.
func bm25IDF(df, n uint64) float64 {
	return math.Log(1.0 + float64(n-df+1)/(float64(df)+0.5))
}

// bm25Score computes the full BM25+ score for a single term occurrence.
func bm25Score(tf float64, df uint64, dl float64, avgdl float64, n uint64) float64 {
	idf := bm25IDF(df, n)
	tfNorm := (tf * (bm25K1 + 1.0)) / (tf + bm25K1*(1.0-bm25B+bm25B*dl/avgdl))
	return idf * (tfNorm + bm25Delta)
}

// bm25ScoreWithNormTable uses precomputed norm table for fast scoring.
func bm25ScoreWithNormTable(tf float64, idf float64, normComponent float32) float64 {
	tfNorm := (tf * (bm25K1 + 1.0)) / (tf + float64(normComponent))
	return idf * (tfNorm + bm25Delta)
}

// quantizeBM25 maps a BM25 score to uint8 (0-255).
// Assumes scores are in range [0, maxScore].
func quantizeBM25(score, maxScore float64) uint8 {
	if maxScore <= 0 || score <= 0 {
		return 0
	}
	v := score / maxScore * 255.0
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// dequantizeBM25 converts a quantized uint8 back to approximate score.
func dequantizeBM25(q uint8, maxScore float64) float64 {
	return float64(q) / 255.0 * maxScore
}
