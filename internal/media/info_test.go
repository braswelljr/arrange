package media

import (
	"path/filepath"
	"testing"
)

func TestDestDir(t *testing.T) {
	cases := []struct {
		name    string
		info    MediaInfo
		creator string
		want    string
	}{
		{
			name: "tv series with season",
			info: MediaInfo{Type: TypeTVSeries, Title: "Breaking Bad", Season: 1},
			want: filepath.Join("Breaking Bad", "Season 01"),
		},
		{
			name: "tv series without season",
			info: MediaInfo{Type: TypeTVSeries, Title: "Miniseries"},
			want: "Miniseries",
		},
		{
			name: "movie with year",
			info: MediaInfo{Type: TypeMovie, Title: "Inception", Year: 2010},
			want: "Inception (2010)",
		},
		{
			name: "movie without year",
			info: MediaInfo{Type: TypeMovie, Title: "Avengers"},
			want: "Avengers",
		},
		{
			name: "multi-part movie",
			info: MediaInfo{Type: TypeMoviePart, Title: "Lord Of The Rings", Year: 2001, Part: 1},
			want: filepath.Join("Lord Of The Rings (2001)", "Part 1"),
		},
		{
			name:    "creator grouping with possessive",
			info:    MediaInfo{Type: TypeMovie, Title: "Tyler Perry's Madea", Year: 2009},
			creator: "Tyler Perry",
			want:    filepath.Join("Tyler Perry", "Madea (2009)"),
		},
		{
			name:    "creator grouping without apostrophe",
			info:    MediaInfo{Type: TypeMovie, Title: "Tyler Perrys Madea", Year: 2009},
			creator: "Tyler Perry",
			want:    filepath.Join("Tyler Perry", "Madea (2009)"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.info.DestDir(c.creator)
			if got != c.want {
				t.Errorf("DestDir = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDestName(t *testing.T) {
	cases := []struct {
		name string
		info MediaInfo
		want string
	}{
		{
			name: "tv series with season and episode",
			info: MediaInfo{Type: TypeTVSeries, Title: "Breaking Bad", Season: 1, Episode: 5, Ext: "mkv"},
			want: "Breaking Bad S01E05.mkv",
		},
		{
			name: "tv series with quality",
			info: MediaInfo{Type: TypeTVSeries, Title: "Breaking Bad", Season: 1, Episode: 5, Quality: "720p", Ext: "mkv"},
			want: "Breaking Bad S01E05 [720p].mkv",
		},
		{
			name: "multi-episode",
			info: MediaInfo{Type: TypeTVSeries, Title: "Show", Season: 1, Episode: 1, EpisodeEnd: 3, Ext: "mkv"},
			want: "Show S01E01-E03.mkv",
		},
		{
			name: "movie with year and quality",
			info: MediaInfo{Type: TypeMovie, Title: "Inception", Year: 2010, Quality: "1080p", Ext: "mkv"},
			want: "Inception (2010) [1080p].mkv",
		},
		{
			name: "movie without year",
			info: MediaInfo{Type: TypeMovie, Title: "Avengers", Ext: "mkv"},
			want: "Avengers.mkv",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.info.DestName()
			if got != c.want {
				t.Errorf("DestName = %q, want %q", got, c.want)
			}
		})
	}
}

func TestDestName_SeasonPack(t *testing.T) {
	m := MediaInfo{Type: TypeTVSeries, Title: "Silo", Season: 2, Ext: "mkv"}
	got := m.DestName()
	want := "Silo S02.mkv"
	if got != want {
		t.Errorf("DestName = %q, want %q", got, want)
	}
}

func TestDestName_EpisodeOnly(t *testing.T) {
	m := MediaInfo{Type: TypeTVSeries, Title: "Show", Episode: 5, Ext: "mp4"}
	got := m.DestName()
	want := "Show E05.mp4"
	if got != want {
		t.Errorf("DestName = %q, want %q", got, want)
	}
}

func TestMediaInfo_String(t *testing.T) {
	cases := []struct {
		info MediaInfo
		want string
	}{
		{
			MediaInfo{Type: TypeTVSeries, Title: "Breaking Bad", Season: 1, Episode: 5},
			"[Series] Breaking Bad S01E05",
		},
		{
			MediaInfo{Type: TypeMovie, Title: "Inception", Year: 2010},
			"[Movie] Inception (2010)",
		},
		{
			MediaInfo{Type: TypeMoviePart, Title: "The Lord of the Rings", Year: 2001, Part: 2},
			"[Movie/Part 2] The Lord of the Rings (2001)",
		},
	}
	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			got := c.info.String()
			if got != c.want {
				t.Errorf("String() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestCreatorMatch(t *testing.T) {
	creators := []string{"Tyler Perry", "Christopher Nolan"}

	cases := []struct {
		title string
		want  string
	}{
		{"Tyler Perry's Madea Goes to Jail", "Tyler Perry"},
		{"Tyler Perrys Madea Goes to Jail", "Tyler Perry"}, // dot-separated, no apostrophe
		{"Tyler Perry Boo", "Tyler Perry"},                 // plain prefix
		{"Christopher Nolan's Inception", "Christopher Nolan"},
		{"Inception 2010", ""}, // no match
		{"", ""},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			m := &MediaInfo{Title: c.title}
			got := m.CreatorMatch(creators)
			if got != c.want {
				t.Errorf("CreatorMatch(%q) = %q, want %q", c.title, got, c.want)
			}
		})
	}
}
