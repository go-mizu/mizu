package dahlia

import (
	"math"
	"sort"
)

// fieldNormTable mirrors Tantivy's fieldnorm code table exactly.
// Source:
// https://raw.githubusercontent.com/quickwit-oss/tantivy/main/src/fieldnorm/code.rs
var fieldNormTable = [256]uint32{
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
	16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
	32, 33, 34, 35, 36, 37, 38, 39, 40, 42, 44, 46, 48, 50, 52, 54,
	56, 60, 64, 68, 72, 76, 80, 84, 88, 96, 104, 112, 120, 128, 136, 144,
	152, 168, 184, 200, 216, 232, 248, 264, 280, 312, 344, 376, 408, 440, 472, 504,
	536, 600, 664, 728, 792, 856, 920, 984, 1048, 1176, 1304, 1432, 1560, 1688, 1816, 1944,
	2072, 2328, 2584, 2840, 3096, 3352, 3608, 3864, 4120, 4632, 5144, 5656, 6168, 6680, 7192, 7704,
	8216, 9240, 10264, 11288, 12312, 13336, 14360, 15384, 16408, 18456, 20504, 22552, 24600, 26648, 28696, 30744,
	32792, 36888, 40984, 45080, 49176, 53272, 57368, 61464, 65560, 73752, 81944, 90136, 98328, 106520, 114712, 122904,
	131096, 147480, 163864, 180248, 196632, 213016, 229400, 245784, 262168, 294936, 327704, 360472, 393240, 426008, 458776, 491544,
	524312, 589848, 655384, 720920, 786456, 851992, 917528, 983064, 1048600, 1179672, 1310744, 1441816, 1572888, 1703960, 1835032, 1966104,
	2097176, 2359320, 2621464, 2883608, 3145752, 3407896, 3670040, 3932184, 4194328, 4718616, 5242904, 5767192, 6291480, 6815768, 7340056, 7864344,
	8388632, 9437208, 10485784, 11534360, 12582936, 13631512, 14680088, 15728664, 16777240, 18874392, 20971544, 23068696, 25165848, 27263000, 29360152, 31457304,
	33554456, 37748760, 41943064, 46137368, 50331672, 54525976, 58720280, 62914584, 67108888, 75497496, 83886104, 92274712, 100663320, 109051928, 117440536, 125829144,
	134217752, 150994968, 167772184, 184549400, 201326616, 218103832, 234881048, 251658264, 268435480, 301989912, 335544344, 369098776, 402653208, 436207640, 469762072, 503316504,
	536870936, 603979800, 671088664, 738197528, 805306392, 872415256, 939524120, 1006632984, 1073741848, 1207959576, 1342177304, 1476395032, 1610612760, 1744830488, 1879048216, 2013265944,
}

// encodeFieldNorm maps a raw field length to the largest fieldnorm id whose
// decoded value is <= dl, matching Tantivy's fieldnorm_to_id behavior.
func encodeFieldNorm(dl uint32) uint8 {
	idx := sort.Search(len(fieldNormTable), func(i int) bool {
		return fieldNormTable[i] > dl
	})
	if idx == 0 {
		return 0
	}
	if idx >= len(fieldNormTable) {
		return 255
	}
	return uint8(idx - 1)
}

// decodeFieldNorm maps a fieldnorm id back to its decoded field length.
func decodeFieldNorm(b uint8) uint32 {
	return fieldNormTable[int(b)]
}

// buildFieldNormBM25Table precomputes the BM25 denominator component
// k1 * (1 - b + b * dl / avgdl) for all 256 possible norm byte values.
func buildFieldNormBM25Table(avgDocLen float64) [256]float32 {
	var table [256]float32
	if avgDocLen <= 0 {
		avgDocLen = 1
	}
	for i := 0; i < 256; i++ {
		dl := float64(decodeFieldNorm(uint8(i)))
		table[i] = float32(bm25K1 * (1.0 - bm25B + bm25B*dl/avgDocLen))
	}
	return table
}

// fieldNormBM25Score computes BM25+ TF component using precomputed norm table.
func fieldNormBM25Score(tf float64, normComponent float32) float64 {
	return (tf * (bm25K1 + 1.0)) / (tf + float64(normComponent))
}

// fieldNormUpperBound returns the maximum possible BM25 TF score for a block,
// given the max TF and the shortest document norm in the block.
func fieldNormUpperBound(maxTF uint32, shortestNorm uint8, normTable [256]float32) float64 {
	tf := float64(maxTF)
	normComp := float64(normTable[shortestNorm])
	if normComp < 0 {
		normComp = 0
	}
	score := (tf*(bm25K1+1.0))/(tf+normComp) + bm25Delta
	if math.IsInf(score, 1) || math.IsNaN(score) {
		return 0
	}
	return score
}
