package dcrawler

import "testing"

func TestAutoBrowserPages(t *testing.T) {
	cases := []struct {
		availMB int
		want    int
	}{
		{500, 8},    // clamp at min 8
		{1000, 10},  // 1000/100=10
		{3500, 35},  // server1
		{4000, 40},  // macOS fallback
		{9000, 80},  // server2 → clamp at max 80
		{20000, 80}, // large RAM → still capped
	}
	for _, tc := range cases {
		got := AutoBrowserPages(tc.availMB)
		if got != tc.want {
			t.Errorf("AutoBrowserPages(%d) = %d, want %d", tc.availMB, got, tc.want)
		}
	}
}
