package view

import (
	"testing"
	"time"
)

func TestParseInfoboxes(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantLen  int
		wantErr  bool
	}{
		{
			name:    "empty string",
			json:    "",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "empty array",
			json:    "[]",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "single infobox",
			json:    `[{"name":"country","items":[{"label":"Capital","value":"Hanoi"}]}]`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "multiple infoboxes",
			json:    `[{"name":"country","items":[]},{"name":"geography","items":[]}]`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Page{InfoboxesJSON: tt.json}
			err := p.ParseInfoboxes()

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInfoboxes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(p.Infoboxes) != tt.wantLen {
				t.Errorf("ParseInfoboxes() got %d infoboxes, want %d", len(p.Infoboxes), tt.wantLen)
			}
		})
	}
}

func TestParseInfoboxesContent(t *testing.T) {
	json := `[{"name":"country","items":[{"label":"Capital","value":"Hanoi"},{"label":"Population","value":"100M"}]}]`
	p := &Page{InfoboxesJSON: json}

	if err := p.ParseInfoboxes(); err != nil {
		t.Fatalf("ParseInfoboxes() error = %v", err)
	}

	if len(p.Infoboxes) != 1 {
		t.Fatalf("expected 1 infobox, got %d", len(p.Infoboxes))
	}

	infobox := p.Infoboxes[0]
	if infobox.Name != "country" {
		t.Errorf("expected name 'country', got '%s'", infobox.Name)
	}
	if len(infobox.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(infobox.Items))
	}
	if infobox.Items[0].Label != "Capital" {
		t.Errorf("expected label 'Capital', got '%s'", infobox.Items[0].Label)
	}
	if infobox.Items[0].Value != "Hanoi" {
		t.Errorf("expected value 'Hanoi', got '%s'", infobox.Items[0].Value)
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"1 minute ago", now.Add(-1 * time.Minute), "1 minute ago"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1 hour ago"},
		{"3 hours ago", now.Add(-3 * time.Hour), "3 hours ago"},
		{"yesterday", now.Add(-30 * time.Hour), "yesterday"},
		{"3 days ago", now.Add(-3 * 24 * time.Hour), "3 days ago"},
		{"1 week ago", now.Add(-7 * 24 * time.Hour), "1 week ago"},
		{"2 weeks ago", now.Add(-14 * 24 * time.Hour), "2 weeks ago"},
		{"1 month ago", now.Add(-30 * 24 * time.Hour), "1 month ago"},
		{"6 months ago", now.Add(-180 * 24 * time.Hour), "6 months ago"},
		{"1 year ago", now.Add(-365 * 24 * time.Hour), "1 year ago"},
		{"2 years ago", now.Add(-730 * 24 * time.Hour), "2 years ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRelativeTime(tt.time)
			if got != tt.want {
				t.Errorf("FormatRelativeTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	tests := []struct {
		time time.Time
		want string
	}{
		{time.Date(2024, 12, 18, 0, 0, 0, 0, time.UTC), "Dec 18, 2024"},
		{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "Jan 1, 2024"},
		{time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC), "Jun 15, 2023"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDate(tt.time)
			if got != tt.want {
				t.Errorf("FormatDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatDates(t *testing.T) {
	tests := []struct {
		name         string
		dateModified string
		wantRel      bool
		wantFmt      bool
	}{
		{"empty", "", false, false},
		{"RFC3339", "2024-12-18T10:30:00Z", true, true},
		{"ISO format", "2024-12-18T10:30:00Z", true, true},
		{"date only", "2024-12-18", true, true},
		{"invalid", "not-a-date", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Page{DateModified: tt.dateModified}
			p.FormatDates()

			gotRel := p.DateModifiedRel != ""
			gotFmt := p.DateModifiedFmt != ""

			if gotRel != tt.wantRel {
				t.Errorf("FormatDates() DateModifiedRel populated = %v, want %v", gotRel, tt.wantRel)
			}
			if gotFmt != tt.wantFmt {
				t.Errorf("FormatDates() DateModifiedFmt populated = %v, want %v", gotFmt, tt.wantFmt)
			}
		})
	}
}
