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
		want:  parseWant{title: "Game of Thrones", typ: TypeTVSeries, season: 8, episode: 6, quality: "1080p"},
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
	// ── Title case ────────────────────────────────────────────────────────────
	{
		name:  "UK abbreviation uppercased",
		input: "Love.Island.Uk.S013E17.720p.YouthTrendx.mkv",
		want:  parseWant{title: "Love Island UK", typ: TypeTVSeries, season: 13, episode: 17, quality: "720p"},
	},
	{
		name:  "US abbreviation uppercased",
		input: "Love.Island.Us.S08E21.1080p.YouthTrendx.mkv",
		want:  parseWant{title: "Love Island US", typ: TypeTVSeries, season: 8, episode: 21, quality: "1080p"},
	},
	{
		name:  "small word 'of' lowercased in the middle",
		input: "Game.of.Thrones.S01E01.mkv",
		want:  parseWant{title: "Game of Thrones", typ: TypeTVSeries, season: 1, episode: 1},
	},
	{
		name:  "article 'the' stays capitalised at start",
		input: "The.Boys.S01E01.mkv",
		want:  parseWant{title: "The Boys", typ: TypeTVSeries, season: 1, episode: 1},
	},
	{
		name:  "FBI abbreviation uppercased",
		input: "Fbi.S04E01.WEB-DL.mkv",
		want:  parseWant{title: "FBI", typ: TypeTVSeries, season: 4, episode: 1, quality: "WEB-DL"},
	},
	// ── Season packs and bare episode markers ────────────────────────────────
	{
		name:  "season pack S02 shorthand",
		input: "Breaking.Bad.S02.Complete.720p.mkv",
		want:  parseWant{title: "Breaking Bad", typ: TypeTVSeries, season: 2, quality: "720p"},
	},
	{
		name:  "bare E episode marker (mini-series)",
		input: "The.Office.E05.720p.mkv",
		want:  parseWant{title: "The Office", typ: TypeTVSeries, episode: 5, quality: "720p"},
	},
	// ── HDR quality modifier ──────────────────────────────────────────────────
	{
		name:  "HDR modifier appended to resolution",
		input: "Inception.2010.1080p.HDR.mkv",
		want:  parseWant{title: "Inception", typ: TypeMovie, year: 2010, quality: "1080p HDR"},
	},
	{
		name:  "HDR10+ modifier appended to 2160p",
		input: "Dune.2021.2160p.HDR10+.BluRay.mkv",
		want:  parseWant{title: "Dune", typ: TypeMovie, year: 2021, quality: "2160p HDR10+"},
	},
	{
		name:  "DV (Dolby Vision) modifier appended",
		input: "The.Batman.2022.2160p.DV.WEB-DL.mkv",
		want:  parseWant{title: "The Batman", typ: TypeMovie, year: 2022, quality: "2160p DV"},
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
