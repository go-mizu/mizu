package rose

import "math"

// BM25+ constants (Robertson variant).
//
// k1    — TF saturation: higher k1 means term frequency matters more before
//          saturating.  1.2 is the Robertson standard (also used by Lucene).
// b     — document-length normalisation weight.  0 = no normalisation;
//          1 = full normalisation.  0.75 is the standard recommended value.
// delta — additive floor (BM25+).  Prevents zero scores for any matching
//          term, improving recall on short queries with rare terms.
//          Value 1.0 from Liang & Croft (2012).
const (
	bm25K1    = 1.2
	bm25B     = 0.75
	bm25Delta = 1.0
)

// idf computes the Robertson IDF for a term with document frequency df in a
// corpus of N documents:
//
//	IDF(t) = log( (N - df + 0.5) / (df + 0.5) + 1 )
//
// The +1 inside the log ensures IDF >= 0 for all df, N values (including
// df > N/2).  Returns 0 for degenerate inputs (N == 0 or df == 0 — the
// latter should not normally occur during scoring but is handled gracefully).
func idf(df, N uint32) float64 {
	if N == 0 || df == 0 {
		return 0
	}
	// Cast to float64 before arithmetic to avoid uint32 underflow when df > N.
	n := float64(N)
	d := float64(df)
	return math.Log((n-d+0.5)/(d+0.5) + 1)
}

// bm25Plus computes the Robertson BM25+ score for a single (term, document)
// pair:
//
//	TF_norm = (k1+1) * tf  /  ( tf + k1 * (1 - b + b * dl/avgdl) )
//	BM25+   = IDF(df, N) * TF_norm + delta
//
// Parameters (all uint32):
//   - tf    — raw term frequency of the term in this document
//   - df    — document frequency of the term (number of docs containing it)
//   - dl    — document length (total tokens in this document)
//   - avgdl — average document length across the corpus
//   - N     — total number of documents in the corpus
//
// Returns 0 for any degenerate input: tf==0, df==0, N==0.
// avgdl==0 is safe: treated as 1 to avoid division by zero (no normalisation).
func bm25Plus(tf, df, dl, avgdl, N uint32) float64 {
	// Short-circuit on degenerate inputs.
	if tf == 0 || df == 0 || N == 0 {
		return 0
	}

	tfF := float64(tf)
	dlF := float64(dl)

	// Protect against avgdl == 0; treat as 1 (no length normalisation).
	avgdlF := float64(avgdl)
	if avgdlF == 0 {
		avgdlF = 1
	}

	// Length-normalised TF component.
	norm := tfF + bm25K1*(1-bm25B+bm25B*dlF/avgdlF)
	tfNorm := (bm25K1 + 1) * tfF / norm

	// BM25+ = IDF * TF_norm + delta
	return idf(df, N)*tfNorm + bm25Delta
}

// quantise converts a slice of raw BM25+ float64 scores into uint8 impact
// values using proportional scaling (BM25S paper, arXiv:2407.03618 §3):
//
//	impact = clamp( round(score / maxScore * 255), 1, 255 )
//
// The maximum score maps to 255.  Any score <= 0 (including exact zero) maps
// to 1, satisfying the BM25+ invariant that every matching term contributes a
// positive impact.  When all scores are equal (including the all-zero case),
// all outputs are 255.
//
// Returns nil for a nil input and an empty slice for an empty input.
func quantise(scores []float64) []uint8 {
	if len(scores) == 0 {
		return nil
	}

	// Find the maximum score.
	maxScore := scores[0]
	for _, s := range scores[1:] {
		if s > maxScore {
			maxScore = s
		}
	}

	// Guard: if maxScore is 0 (or negative), treat as 1 so all values map to 255.
	if maxScore <= 0 {
		maxScore = 1
	}

	out := make([]uint8, len(scores))
	for i, s := range scores {
		q := int(math.Round(s / maxScore * 255))
		if q < 1 {
			q = 1
		} else if q > 255 {
			q = 255
		}
		out[i] = uint8(q)
	}
	return out
}
