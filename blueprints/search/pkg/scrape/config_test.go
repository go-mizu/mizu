package scrape

import "testing"

func TestAutoBrowserPages(t *testing.T) {
	cases := []struct {
		availMB int
		want    int
	}{
		{500, 8},    // clamp at min 8
		{1000, 8},   // 1000/150=6 → clamp to 8
		{1500, 10},  // 1500/150=10
		{3600, 24},  // server1 (~3.6GB avail) → 3600/150=24
		{4000, 24},  // macOS fallback → 4000/150=26 → clamp at 24
		{9000, 24},  // server2 → clamp at max 24
		{20000, 24}, // large RAM → still capped at 24
	}
	for _, tc := range cases {
		got := AutoBrowserPages(tc.availMB)
		if got != tc.want {
			t.Errorf("AutoBrowserPages(%d) = %d, want %d", tc.availMB, got, tc.want)
		}
	}
}
