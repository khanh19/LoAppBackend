package places

import "testing"

func TestStoredCoverURL(t *testing.T) {
	t.Parallel()

	hosted := "https://cdn.example.com/place.jpg"
	google := "https://lh3.googleusercontent.com/abc"
	blank := "  "

	cases := []struct {
		name  string
		cover *string
		want  string
	}{
		{name: "nil", cover: nil, want: ""},
		{name: "blank", cover: &blank, want: ""},
		{name: "google cdn", cover: &google, want: ""},
		{name: "hosted", cover: &hosted, want: hosted},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := storedCoverURL(tc.cover); got != tc.want {
				t.Fatalf("storedCoverURL() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSamePhotoName(t *testing.T) {
	t.Parallel()

	if !samePhotoName([]string{" places/a/photos/1 "}, []string{"places/a/photos/1"}, 0) {
		t.Fatal("expected matching photo names")
	}
	if samePhotoName([]string{"places/a/photos/1"}, []string{"places/a/photos/2"}, 0) {
		t.Fatal("expected different photo names")
	}
	if samePhotoName(nil, []string{"places/a/photos/1"}, 0) {
		t.Fatal("expected missing name to differ")
	}
}
