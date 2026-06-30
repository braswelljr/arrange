package media

import (
	"testing"
)

type parseWant struct {
	title      string
	typ        MediaType
	season     int
	episode    int
	episodeEnd int
	year       int
	quality    string
	part       int
}

var parseCases = []struct {
	name  string
	input string
	want  parseWant
}{
	{
		name:  "S01E05 canonical dot-separated",
		input: "Breaking.Bad.S01E05.720p.BluRay.mkv",
		want:  parseWant{title: "Breaking Bad", typ: TypeTVSeries, season: 1, episode: 5, quality: "720p"},
	},
	{
		name:  "resolution preferred over source tag",
		input: "Game.of.Thrones.S08E06.1080p.BluRay.mkv",
		want:  parseWant{title: "Game Of Thrones", typ: TypeTVSeries, season: 8, episode: 6, quality: "1080p"},
	},
	{
		name:  "4K movie with year",
		input: "The.Dark.Knight.2008.4K.BluRay.mkv",
		want:  parseWant{title: "The Dark Knight", typ: TypeMovie, year: 2008, quality: "4K"},
	},
	{
		name:  "NxNN format",
		input: "The.Wire.1x05.HDTV.avi",
		want:  parseWant{title: "The Wire", typ: TypeTVSeries, season: 1, episode: 5, quality: "HDTV"},
	},
	{
		name:  "multi-episode S01E01-E03",
		input: "Show.S01E01-E03.mkv",
		want:  parseWant{title: "Show", typ: TypeTVSeries, season: 1, episode: 1, episodeEnd: 3},
	},
	{
		name:  "quality captured from trailing bracket before noise strip",
		input: "Inception.2010.WEBRip [1080p].mkv",
		want:  parseWant{title: "Inception", typ: TypeMovie, year: 2010, quality: "1080p"},
	},
	{
		name:  "browser duplicate suffix stripped",
		input: "Inception (2010) (1).mkv",
		want:  parseWant{title: "Inception", typ: TypeMovie, year: 2010},
	},
	{
		name:  "leading release group stripped",
		input: "[YTS] Interstellar.2014.1080p.mkv",
		want:  parseWant{title: "Interstellar", typ: TypeMovie, year: 2014, quality: "1080p"},
	},
	{
		name:  "multi-part movie",
		input: "Kill.Bill.Part.1.2003.mkv",
		want:  parseWant{title: "Kill Bill", typ: TypeMoviePart, year: 2003, part: 1},
	},
	{
		name:  "season-only marker (no episode)",
		input: "Stranger.Things.Season.3.mkv",
		want:  parseWant{title: "Stranger Things", typ: TypeTVSeries, season: 3},
	},
	{
		name:  "plain movie with no year",
		input: "Avengers.mkv",
		want:  parseWant{title: "Avengers", typ: TypeMovie},
	},
	{
		name:  "anime dash episode with lead group",
		input: "[HorribleSubs] Demon Slayer - 26 [1080p].mkv",
		want:  parseWant{title: "Demon Slayer", typ: TypeTVSeries, episode: 26, quality: "1080p"},
	},
	{
		name:  "SHO not matched inside word Show",
		input: "The.Show.S02E05.mkv",
		want:  parseWant{title: "The Show", typ: TypeTVSeries, season: 2, episode: 5, quality: ""},
	},
	{
		name:  "WEB-DL source tag",
		input: "Better.Call.Saul.S05E10.WEB-DL.mkv",
		want:  parseWant{title: "Better Call Saul", typ: TypeTVSeries, season: 5, episode: 10, quality: "WEB-DL"},
	},
	{
		name:  "space between S and E markers",
		input: "Lost S03 E05.mkv",
		want:  parseWant{title: "Lost", typ: TypeTVSeries, season: 3, episode: 5},
	},
}

func TestParseName(t *testing.T) {
	for _, c := range parseCases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseName(c.input)
			check := func(field string, got, want interface{}) {
				t.Helper()
				if got != want {
					t.Errorf("%s: got %v, want %v", field, got, want)
				}
			}
			check("Title", got.Title, c.want.title)
			check("Type", got.Type, c.want.typ)
			check("Season", got.Season, c.want.season)
			check("Episode", got.Episode, c.want.episode)
			check("EpisodeEnd", got.EpisodeEnd, c.want.episodeEnd)
			check("Year", got.Year, c.want.year)
			check("Quality", got.Quality, c.want.quality)
			check("Part", got.Part, c.want.part)
		})
	}
}

func TestParseNameOrigPathEmpty(t *testing.T) {
	got := ParseName("movie.mkv")
	if got.OrigPath != "" {
		t.Errorf("OrigPath should be empty from ParseName, got %q", got.OrigPath)
	}
}

func TestParseOrigPathSet(t *testing.T) {
	path := "/downloads/Breaking.Bad.S01E05.mkv"
	got := Parse(path)
	if got.OrigPath != path {
		t.Errorf("OrigPath = %q, want %q", got.OrigPath, path)
	}
	if got.Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", got.Title, "Breaking Bad")
	}
}
