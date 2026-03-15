package facebook

import "testing"

func TestNormalizeURL(t *testing.T) {
	t.Run("slug to mbasic", func(t *testing.T) {
		got := NormalizeURL("openai", true)
		want := "https://mbasic.facebook.com/openai"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("canonical host", func(t *testing.T) {
		got := CanonicalURL("https://mbasic.facebook.com/groups/123/posts/456")
		want := "https://www.facebook.com/groups/123/posts/456"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
}

func TestInferEntityType(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"https://www.facebook.com/groups/123", EntityGroup},
		{"https://www.facebook.com/groups/123/posts/456", EntityPost},
		{"https://www.facebook.com/profile.php?id=42", EntityProfile},
		{"https://www.facebook.com/somepage", EntityPage},
		{"https://www.facebook.com/search/top/?q=openai", EntitySearch},
	}

	for _, tc := range cases {
		if got := InferEntityType(tc.url); got != tc.want {
			t.Fatalf("url=%q got=%q want=%q", tc.url, got, tc.want)
		}
	}
}
