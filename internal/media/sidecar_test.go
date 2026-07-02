package media

import "testing"

func TestMatchSidecar(t *testing.T) {
	tests := []struct {
		name      string
		mediaStem string
		subStem   string
		wantTag   string
		wantOK    bool
	}{
		{"exact match", "Movie.2020.1080p", "Movie.2020.1080p", "", true},
		{"dot language", "Movie.2020", "Movie.2020.en", "en", true},
		{"dot two-part tag", "Movie.2020", "Movie.2020.en.forced", "en.forced", true},
		{"space separator", "Movie 2020", "Movie 2020 en", "en", true},
		{"hyphen tag normalised", "Show S01E05", "Show S01E05-en", "en", true},
		{"case insensitive stem", "Movie 2020", "MOVIE 2020.EN", "EN", true},
		{"region code preserved", "Movie 2020", "Movie 2020.pt.BR", "pt.BR", true},
		{"different title", "Movie 2020", "Other Movie 2020", "", false},
		{"prefix without boundary", "Movie", "Movie2", "", false},
		{"longer title is not a sidecar of shorter", "Movie", "Movie 2 2021", "2.2021", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, ok := MatchSidecar(tt.mediaStem, tt.subStem)
			if ok != tt.wantOK {
				t.Fatalf("MatchSidecar(%q, %q) ok = %v, want %v", tt.mediaStem, tt.subStem, ok, tt.wantOK)
			}
			if ok && tag != tt.wantTag {
				t.Errorf("MatchSidecar(%q, %q) tag = %q, want %q", tt.mediaStem, tt.subStem, tag, tt.wantTag)
			}
		})
	}
}

func TestSidecarName(t *testing.T) {
	tests := []struct {
		stem, tag, ext, want string
	}{
		{"The Office S01E05 [720p]", "en", "srt", "The Office S01E05 [720p].en.srt"},
		{"The Office S01E05 [720p]", "en.forced", "srt", "The Office S01E05 [720p].en.forced.srt"},
		{"The Office S01E05 [720p]", "", "srt", "The Office S01E05 [720p].srt"},
	}
	for _, tt := range tests {
		if got := SidecarName(tt.stem, tt.tag, tt.ext); got != tt.want {
			t.Errorf("SidecarName(%q, %q, %q) = %q, want %q", tt.stem, tt.tag, tt.ext, got, tt.want)
		}
	}
}
