package xs_test

import (
	"slices"
	"testing"

	"github.com/go-mizu/mizu/xs"
)

func TestChunk(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4, 5, 6})
	got := slices.Collect(xs.Chunk(in, 2))

	want := [][]int{{1, 2}, {3, 4}, {5, 6}}
	if !equalBatches(got, want) {
		t.Errorf("Chunk gave %v, want %v", got, want)
	}
}

func TestChunkWithAShortLastBatch(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4, 5})
	got := slices.Collect(xs.Chunk(in, 2))

	want := [][]int{{1, 2}, {3, 4}, {5}}
	if !equalBatches(got, want) {
		t.Errorf("Chunk gave %v, want %v", got, want)
	}
}

func TestChunkBiggerThanTheSequence(t *testing.T) {
	in := slices.Values([]int{1, 2})
	got := slices.Collect(xs.Chunk(in, 10))

	want := [][]int{{1, 2}}
	if !equalBatches(got, want) {
		t.Errorf("Chunk gave %v, want %v", got, want)
	}
}

func TestChunkOfNothing(t *testing.T) {
	if got := slices.Collect(xs.Chunk(slices.Values([]int(nil)), 3)); len(got) != 0 {
		t.Errorf("Chunk of nothing gave %v, want no batches at all", got)
	}
}

// TestChunkHandsOverASliceTheCallerOwns is the difference from slices.Chunk,
// which gives back windows into the original. Keeping a batch has to be safe,
// since batching exists so the batch can be handed somewhere.
func TestChunkHandsOverASliceTheCallerOwns(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4})

	var kept [][]int
	for batch := range xs.Chunk(in, 2) {
		kept = append(kept, batch)
	}

	want := [][]int{{1, 2}, {3, 4}}
	if !equalBatches(kept, want) {
		t.Errorf("the batches held on to became %v, want %v", kept, want)
	}
}

func TestChunkStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4, 5, 6})

	for range xs.Chunk(in, 2) {
		break
	}
	if *read != 2 {
		t.Errorf("it read %d elements to fill one batch of two, want 2", *read)
	}
}

// TestChunkStopsOnTheLastBatch covers the yield that is not checked, the short
// one at the end, where there is nothing left to stop reading.
func TestChunkStopsOnTheLastBatch(t *testing.T) {
	in := slices.Values([]int{1, 2, 3})

	var got [][]int
	for batch := range xs.Chunk(in, 2) {
		got = append(got, batch)
		if len(got) == 2 {
			break
		}
	}
	if want := [][]int{{1, 2}, {3}}; !equalBatches(got, want) {
		t.Errorf("Chunk gave %v, want %v", got, want)
	}
}

func TestChunkWithNoBatchSize(t *testing.T) {
	for _, n := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Chunk of %d was accepted", n)
				}
			}()
			xs.Chunk(slices.Values([]int{1}), n)
		}()
	}
}

func TestWindow(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4})
	got := slices.Collect(xs.Window(in, 2))

	want := [][]int{{1, 2}, {2, 3}, {3, 4}}
	if !equalBatches(got, want) {
		t.Errorf("Window gave %v, want %v", got, want)
	}
}

func TestWindowOfOne(t *testing.T) {
	in := slices.Values([]int{1, 2, 3})
	got := slices.Collect(xs.Window(in, 1))

	want := [][]int{{1}, {2}, {3}}
	if !equalBatches(got, want) {
		t.Errorf("Window gave %v, want %v", got, want)
	}
}

func TestWindowBiggerThanTheSequence(t *testing.T) {
	in := slices.Values([]int{1, 2})
	if got := slices.Collect(xs.Window(in, 3)); len(got) != 0 {
		t.Errorf("Window gave %v, want nothing: there is no run of three in two elements", got)
	}
}

// TestWindowHandsOverASliceTheCallerOwns matters more here than for Chunk,
// since the windows overlap and one buffer is reused to build them.
func TestWindowHandsOverASliceTheCallerOwns(t *testing.T) {
	in := slices.Values([]int{1, 2, 3, 4})

	var kept [][]int
	for w := range xs.Window(in, 2) {
		kept = append(kept, w)
	}

	want := [][]int{{1, 2}, {2, 3}, {3, 4}}
	if !equalBatches(kept, want) {
		t.Errorf("the windows held on to became %v, want %v", kept, want)
	}
}

func TestWindowStopsWhenTheCallerDoes(t *testing.T) {
	in, read := counted([]int{1, 2, 3, 4})

	for range xs.Window(in, 2) {
		break
	}
	if *read != 2 {
		t.Errorf("it read %d elements to fill one window of two, want 2", *read)
	}
}

func TestWindowWithNoSize(t *testing.T) {
	for _, n := range []int{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Window of %d was accepted", n)
				}
			}()
			xs.Window(slices.Values([]int{1}), n)
		}()
	}
}

func equalBatches[T comparable](got, want [][]T) bool {
	return slices.EqualFunc(got, want, slices.Equal)
}
