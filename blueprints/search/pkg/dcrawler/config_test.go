package dcrawler

import "testing"

func TestAutoBrowserPages(t *testing.T) {
	cases := []struct {
		availMB int
		want    int
	}{
		{500, 20},    // clamp at min 20
		{1000, 20},   // 1000/50=20
		{3500, 70},   // server1
		{9000, 150},  // server2 → clamp at max 150
		{20000, 150}, // large RAM → still capped
	}
	for _, tc := range cases {
		got := AutoBrowserPages(tc.availMB)
		if got != tc.want {
			t.Errorf("AutoBrowserPages(%d) = %d, want %d", tc.availMB, got, tc.want)
		}
	}
}
