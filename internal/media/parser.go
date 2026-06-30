package media

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/braswelljr/arrange/internal/fileops"
)

// ── Compiled patterns ────────────────────────────────────────────────────────
//
// Applied in the order listed below; the earliest match position in the
// normalised stem defines where the clean title ends.

var (
	// ── TV series ────────────────────────────────────────────────────────────

	// S01E05, S01E05-E07, S01E05-07, S01E05E07  (most common scene format)
	// Supports 1-3 digit season numbers (e.g. S013E17 zero-padded to 3 digits).
	reSxxExx = regexp.MustCompile(`(?i)\bS(\d{1,3})\s*E(\d{2,3})(?:[-–]\s*E?(\d{2,3}))?\b`)

	// S01 E05  (space between season and episode markers)
	reSxxSpaceExx = regexp.MustCompile(`(?i)\bS(\d{1,3})\s+E(\d{2,3})\b`)

	// 1x05, 2x12  (old-school NxNN)
	reNxNN = regexp.MustCompile(`(?i)\b(\d{1,2})x(\d{2,3})\b`)

	// Season 1 Episode 5  (fully written out, various separators)
	reSeasonEp = regexp.MustCompile(`(?i)\bSeason[. _-]?(\d{1,3})[. _-]Episode[. _-]?(\d{1,3})\b`)

	// Season 1  (season number without episode — marks type as series)
	reSeasonOnly = regexp.MustCompile(`(?i)\bSeason[. _-]?(\d{1,3})\b`)

	// S01  (season shorthand without episode — season packs, e.g. "Show.S02.Complete")
	reSxxOnly = regexp.MustCompile(`(?i)\bS(\d{1,3})\b`)

	// Episode 05 / Ep 05 / Ep.5  (episode without season, e.g. mini-series)
	reEpOnly = regexp.MustCompile(`(?i)\b(?:Ep(?:isode)?)[. _-]?(\d{2,3})\b`)

	// E05  (bare episode marker without season — short-run or daily series)
	reExxOnly = regexp.MustCompile(`(?i)\bE(\d{2,3})\b`)

	// Anime dash  "Show Name - 05 [720p]" or "Show Name - 05v2"
	// Only fires as a last resort when no other episode pattern matched.
	reAnimeDash = regexp.MustCompile(`(?i)\s[-–]\s(\d{2,3})(?:v\d+)?\b`)

	// ── Multi-part movies ────────────────────────────────────────────────────

	// Part 1, Part.2, Pt 3
	rePart = regexp.MustCompile(`(?i)\b(?:Part|Pt)[. _-]?(\d+)\b`)

	// CD1, CD2, Disc 1, Disk 2, Volume 3, Vol.4
	reCDDisc = regexp.MustCompile(`(?i)\b(?:CD|Disc|Disk|Volume|Vol)[. _-]?(\d+)\b`)

	// ── Movie year ───────────────────────────────────────────────────────────

	// (2008) or 2008 — parentheses accepted
	reYear = regexp.MustCompile(`[\(\[]?((?:19|20)\d{2})[\)\]]?`)

	// ── Quality tags ─────────────────────────────────────────────────────────
	// Split into resolution (preferred) and source (fallback) so that a file
	// tagged "BluRay 1080p" captures "1080p" rather than "BluRay".
	// Short streaming-service codes (NF, SHO…) are omitted — they match
	// inside common title words.

	// Resolution-based quality — always preferred when present.
	reQualRes = regexp.MustCompile(`(?i)\b(4[Kk]|UHD|2160[pi]|1080[pi]|720[pi]|480[pi]|360[pi])\b`)

	// Source-based quality — used when no resolution tag is found.
	reQualSrc = regexp.MustCompile(
		`(?i)\b(BluRay|Blu-Ray|BDRip|BRRip|DVDRip|DVDScr|` +
			`WEBRip|WEB-DL|WEBDL|WEB|HDTV|PDTV|AMZN|HULU|DSNP|ATVP)\b`,
	)

	// HDR modifier — appended to the base quality tag when present.
	// HDR10+ is listed first so the greedy match captures the '+'.
	// The trailing anchor uses (?:$|\s|\b) instead of \b because '+' is a
	// non-word character and \b cannot follow it.
	reQualHDR = regexp.MustCompile(`(?i)\b(HDR10\+|HDR10|HDR|DV|Dolby[ .]?Vision)(?:$|\s|\b)`)

	// Combined pattern used only for stripping quality tokens from titles.
	reQuality = regexp.MustCompile(
		`(?i)\b(4[Kk]|UHD|2160[pi]|1080[pi]|720[pi]|480[pi]|360[pi]|` +
			`BluRay|Blu-Ray|BDRip|BRRip|DVDRip|DVDScr|` +
			`WEBRip|WEB-DL|WEBDL|WEB|HDTV|PDTV|AMZN|HULU|DSNP|ATVP|` +
			`HDR10\+?|HDR|DV|Dolby\.?Vision)\b`,
	)

	// ── Codec / release-group junk ────────────────────────────────────────────
	// These tokens appear after the meaningful content and must be stripped.
	reJunk = regexp.MustCompile(
		`(?i)\b(` +
			// video codecs
			`x264|x265|h264|h265|H\.264|H\.265|HEVC|AVC|AVC1|VP8|VP9|AV1|` +
			// audio codecs
			`AAC|AC3|DTS(?:-HD)?|FLAC|DD5\.1|DD\+|TrueHD|Atmos|MP3|EAC3|` +
			// HDR / colour
			`HDR10\+?|DV|Dolby\.?Vision|SDR|` +
			// release modifiers
			`REMUX|PROPER|REPACK|INTERNAL|READNFO|` +
			`EXTENDED|UNRATED|THEATRICAL|DC|` +
			`REMASTERED|RESTORED|ANNIVERSARY|CRITERION|IMAX|` +
			// scene / p2p groups
			`YIFY|YTS|RARBG|FGT|NTb|SPARKS|FraMeSToR|TEPES|EtHD|` +
			`MkvCage|DEFLATE|SYNCOPY|CMRG|GalaxyRG|` +
			// anime groups (common in filenames)
			`SubsPlease|Erai-raws|HorribleSubs|Ohys-Raws|` +
			`[A-Z0-9]{2,10}-(?:raws|subs|enc)` +
			`)\b`,
	)

	// ── Pre-processing helpers ────────────────────────────────────────────────

	// Leading [GroupName] or (GroupName) — strip before parsing
	reLeadGroup = regexp.MustCompile(`^[\[\(][^\]\)]+[\]\)]\s*`)

	// Trailing [...] or (...) tags that contain only non-title noise
	reTrailNoise = regexp.MustCompile(`\s*[\[\(][^\]\)]*(?:` +
		`(?i)(?:\d{3,4}[pi]|BluRay|WEB|HDTV|x26[45]|HEVC|REMUX|PROPER)` +
		`)[^\]\)]*[\]\)]\s*$`)

	// Illegal filename characters — replaced in output names
	reIllegal = regexp.MustCompile(`[\\/:*?"<>|]`)
)

// ── Title-case tables ────────────────────────────────────────────────────────

// knownUppercase maps lowercase tokens to their all-caps form so abbreviations
// in titles are always rendered correctly (e.g. "Uk" → "UK").
var knownUppercase = map[string]string{
	"uk": "UK", "us": "US", "usa": "USA",
	"bbc": "BBC", "nbc": "NBC", "cbs": "CBS", "abc": "ABC", "hbo": "HBO",
	"cnn": "CNN", "mtv": "MTV", "pbs": "PBS",
	"nyc": "NYC",
	"fbi": "FBI", "cia": "CIA", "dea": "DEA", "nsa": "NSA",
	"tv": "TV",
	"dj": "DJ",
}

// smallWords are lowercased in the middle of a title (standard English title case).
// They are still capitalised when they are the first word.
var smallWords = map[string]bool{
	"a": true, "an": true, "the": true,
	"of": true, "in": true, "on": true, "at": true, "to": true,
	"and": true, "but": true, "or": true, "nor": true,
	"for": true, "with": true, "by": true, "as": true, "vs": true,
}

// ── Public API ───────────────────────────────────────────────────────────────

// ParseName extracts MediaInfo from a bare filename (base name + extension).
// It recognises every common naming convention used for scene, P2P, and
// streaming releases, as well as anime and hand-named files.
// OrigPath is left empty; set it yourself if you have the full path.
func ParseName(filename string) *MediaInfo {
	info := &MediaInfo{
		Ext: strings.ToLower(strings.TrimLeft(filepath.Ext(filename), ".")),
	}

	base := filepath.Base(filename)
	stem := strings.TrimSuffix(base, filepath.Ext(base))

	// ── Pre-process stem ─────────────────────────────────────────────────────

	// Strip browser-generated duplicate suffixes: "movie (1)" → "movie".
	stem = fileops.StripBrowserSuffix(stem)

	// Strip leading [GroupName] / (GroupName) release tags.
	stem = reLeadGroup.ReplaceAllString(stem, "")

	// Normalise word separators.
	norm := normaliseSeparators(stem)

	// ── Extract quality BEFORE stripping trailing noise ───────────────────────
	// Trailing brackets like "[1080p x265]" are stripped next; capture quality
	// now so it is not lost.  Resolution takes priority over source; HDR
	// modifier is appended when present ("1080p HDR", "2160p HDR10+").
	if ms := reQualRes.FindStringSubmatch(norm); len(ms) > 1 {
		q := canonicalQuality(ms[1])
		if mh := reQualHDR.FindStringSubmatch(norm); len(mh) > 1 {
			q += " " + canonicalHDR(mh[1])
		}
		info.Quality = q
	} else if ms := reQualSrc.FindStringSubmatch(norm); len(ms) > 1 {
		info.Quality = canonicalQuality(ms[1])
	}

	// Strip trailing noise brackets like "[1080p x265]" at the very end.
	norm = strings.TrimSpace(reTrailNoise.ReplaceAllString(norm, ""))

	stopIdx := len(norm)

	// ── Episode / series patterns (highest priority) ──────────────────────────

	// S01E05 — canonical scene format
	if m := reSxxExx.FindStringSubmatchIndex(norm); m != nil {
		info.Type = TypeTVSeries
		info.Season = mustAtoi(norm[m[2]:m[3]])
		info.Episode = mustAtoi(norm[m[4]:m[5]])
		if m[6] >= 0 {
			info.EpisodeEnd = mustAtoi(norm[m[6]:m[7]])
		}
		stopIdx = minOf(stopIdx, m[0])
	}

	// S01 E05  (space variant)
	if info.Type == TypeUnknown {
		if m := reSxxSpaceExx.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeTVSeries
			info.Season = mustAtoi(norm[m[2]:m[3]])
			info.Episode = mustAtoi(norm[m[4]:m[5]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	// 1x05
	if info.Type == TypeUnknown {
		if m := reNxNN.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeTVSeries
			info.Season = mustAtoi(norm[m[2]:m[3]])
			info.Episode = mustAtoi(norm[m[4]:m[5]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	// Season 1 Episode 5  (written out)
	if info.Type == TypeUnknown {
		if m := reSeasonEp.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeTVSeries
			info.Season = mustAtoi(norm[m[2]:m[3]])
			info.Episode = mustAtoi(norm[m[4]:m[5]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	// Season 1  (series without an explicit episode — written-out form)
	if info.Type == TypeUnknown {
		if m := reSeasonOnly.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeTVSeries
			info.Season = mustAtoi(norm[m[2]:m[3]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	// S01  (season shorthand, no episode — season packs like "Show.S02.Complete")
	if info.Type == TypeUnknown {
		if m := reSxxOnly.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeTVSeries
			info.Season = mustAtoi(norm[m[2]:m[3]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	// Episode 05 / Ep 05
	if info.Type == TypeUnknown {
		if m := reEpOnly.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeTVSeries
			info.Episode = mustAtoi(norm[m[2]:m[3]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	// E05  (bare episode marker, no season — short-run or daily series)
	if info.Type == TypeUnknown {
		if m := reExxOnly.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeTVSeries
			info.Episode = mustAtoi(norm[m[2]:m[3]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	// ── Multi-part movies ─────────────────────────────────────────────────────

	if info.Type == TypeUnknown {
		if m := rePart.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeMoviePart
			info.Part = mustAtoi(norm[m[2]:m[3]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	if info.Type == TypeUnknown {
		if m := reCDDisc.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeMoviePart
			info.Part = mustAtoi(norm[m[2]:m[3]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	// ── Year (valid for all types) ─────────────────────────────────────────────
	// Find the FIRST year occurrence — it marks the end of the title.
	if m := reYear.FindStringSubmatchIndex(norm); m != nil {
		info.Year = mustAtoi(norm[m[2]:m[3]])
		if info.Type == TypeUnknown {
			info.Type = TypeMovie
		}
		stopIdx = minOf(stopIdx, m[0])
	}

	// ── Anime dash fallback ───────────────────────────────────────────────────
	// Only applied when nothing else identified a series or a year, to avoid
	// false positives on legitimate title numbers.
	if info.Type == TypeUnknown {
		if m := reAnimeDash.FindStringSubmatchIndex(norm); m != nil {
			info.Type = TypeTVSeries
			info.Episode = mustAtoi(norm[m[2]:m[3]])
			stopIdx = minOf(stopIdx, m[0])
		}
	}

	// Default to movie when no pattern matched.
	if info.Type == TypeUnknown {
		info.Type = TypeMovie
	}

	info.Title = cleanTitle(strings.TrimSpace(norm[:stopIdx]))
	return info
}

// Parse extracts MediaInfo from a full file path.
// It sets OrigPath and delegates all parsing to ParseName.
func Parse(path string) *MediaInfo {
	info := ParseName(filepath.Base(path))
	info.OrigPath = path
	return info
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// normaliseSeparators replaces dots or underscores with spaces when they
// act as word separators (dots outnumber spaces).
func normaliseSeparators(s string) string {
	if strings.Count(s, ".") > strings.Count(s, " ") {
		s = strings.ReplaceAll(s, ".", " ")
	}
	s = strings.ReplaceAll(s, "_", " ")
	return strings.Join(strings.Fields(s), " ")
}

// cleanTitle strips noise tokens and applies English title case:
//   - Known abbreviations (UK, US, FBI…) are always uppercased.
//   - Articles and short prepositions (the, of, in…) are lowercased except
//     when they are the first word of the title.
//   - Everything else is capitalised on the first letter only.
func cleanTitle(s string) string {
	s = reJunk.ReplaceAllString(s, " ")
	s = reQuality.ReplaceAllString(s, " ")
	s = strings.TrimRight(s, " -–—.,([{")
	words := strings.Fields(s)
	if len(words) == 0 {
		return "Unknown"
	}
	for i, w := range words {
		lower := strings.ToLower(w)
		if up, ok := knownUppercase[lower]; ok {
			words[i] = up
		} else if i > 0 && smallWords[lower] {
			words[i] = lower
		} else {
			words[i] = capitaliseFirst(w)
		}
	}
	return strings.Join(words, " ")
}

// SanitizeFilename removes characters that are illegal in filenames on
// Windows, macOS, and Linux so the output name is universally safe.
func SanitizeFilename(s string) string {
	s = reIllegal.ReplaceAllStringFunc(s, func(r string) string {
		switch r {
		case ":":
			return " -"
		case "/", "\\":
			return "-"
		case "|":
			return "-"
		default:
			return ""
		}
	})
	return strings.TrimSpace(s)
}

func capitaliseFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func canonicalQuality(q string) string {
	switch strings.ToLower(q) {
	case "bluray", "blu-ray", "brrip":
		return "BluRay"
	case "bdrip":
		return "BDRip"
	case "dvdrip", "dvdscr":
		return "DVDRip"
	case "webrip":
		return "WEBRip"
	case "web-dl", "webdl":
		return "WEB-DL"
	case "web":
		return "WEB"
	case "4k", "uhd":
		return "4K"
	case "hdtv", "pdtv":
		return "HDTV"
	}
	return q
}

func canonicalHDR(h string) string {
	switch strings.ToLower(strings.NewReplacer(" ", ".", "-", ".").Replace(h)) {
	case "dolby.vision", "dv":
		return "DV"
	case "hdr10+":
		return "HDR10+"
	case "hdr10":
		return "HDR10"
	default:
		return "HDR"
	}
}

func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func minOf(a, b int) int {
	if b < a {
		return b
	}
	return a
}
