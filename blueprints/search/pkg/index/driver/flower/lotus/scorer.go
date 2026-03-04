package lotus

import "math"

const (
	bm25K1    = 1.2
	bm25B     = 0.75
	bm25Delta = 1.0
)

func bm25Score(tf, df, dl uint32, avgdl float64, n uint32) float64 {
	idf := math.Log((float64(n-df)+0.5)/(float64(df)+0.5) + 1)
	tfNorm := float64(tf) * (bm25K1 + 1) /
		(float64(tf) + bm25K1*(1-bm25B+bm25B*float64(dl)/avgdl))
	return idf*tfNorm + bm25Delta
}

func bm25IDF(df, n uint32) float64 {
	return math.Log((float64(n-df) + 0.5) / (float64(df) + 0.5) + 1)
}

func quantizeBM25(score, maxScore float64) uint8 {
	if maxScore <= 0 {
		return 1
	}
	q := int(math.Round(score / maxScore * 255))
	if q < 1 {
		return 1
	}
	if q > 255 {
		return 255
	}
	return uint8(q)
}

func dequantizeBM25(impact uint8, maxScore float64) float64 {
	return float64(impact) / 255.0 * maxScore
}
