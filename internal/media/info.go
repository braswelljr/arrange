package media

import (
	"fmt"
	"path/filepath"
	"strings"
)

// MediaType classifies the kind of media file.
type MediaType uint8

// Media type constants for classifying parsed files.
const (
	TypeUnknown   MediaType = iota
	TypeTVSeries            // has season / episode markers
	TypeMovie               // standalone movie, may carry a year
	TypeMoviePart           // multi-part movie (CD1/CD2, Part 1/2, Disc N)
)

// MediaInfo holds parsed information about a single media file.
type MediaInfo struct {
	Type       MediaType
	Title      string // cleaned title
	Year       int    // 0 if unknown
	Season     int    // 0 if not a series
	Episode    int    // 0 if not a series
	EpisodeEnd int    // >0 for multi-episode files, e.g. S01E01-E03
	Part       int    // >0 for multi-part movies
	Date       string // "YYYY-MM-DD" for date-based (daily) episodes, else ""
	Quality    string // "1080p", "720p", "BluRay", …
	OrigPath   string // absolute original file path
	Ext        string // lower-cased extension without leading dot
}

// DestDir returns the relative destination directory for this file.
// creator is the matched creator prefix (e.g. "Tyler Perry"); pass "" if none.
func (m *MediaInfo) DestDir(creator string) string {
	sub := titleWithoutCreator(m.Title, creator)
	base := folderName(sub, m.Year, m.Type)

	if creator != "" {
		base = filepath.Join(creator, base)
	}

	switch m.Type {
	case TypeTVSeries:
		if m.Season > 0 {
			return filepath.Join(base, fmt.Sprintf("Season %02d", m.Season))
		}
		return base
	case TypeMoviePart:
		if m.Part > 0 {
			return filepath.Join(base, fmt.Sprintf("Part %d", m.Part))
		}
		return base
	default:
		return base
	}
}

// DestName returns the cleaned, normalised, filesystem-safe filename (with extension).
func (m *MediaInfo) DestName() string {
	var name string

	switch m.Type {
	case TypeTVSeries:
		name = m.Title
		switch {
		case m.Date != "":
			// Date-based daily episode: "<Title> YYYY-MM-DD".
			name = fmt.Sprintf("%s %s", m.Title, m.Date)
		case m.Season > 0 && m.Episode > 0:
			ep := fmt.Sprintf("S%02dE%02d", m.Season, m.Episode)
			if m.EpisodeEnd > 0 {
				ep += fmt.Sprintf("-E%02d", m.EpisodeEnd)
			}
			name = fmt.Sprintf("%s %s", m.Title, ep)
		case m.Season > 0:
			// Season pack: no episode info available.
			name = fmt.Sprintf("%s S%02d", m.Title, m.Season)
		case m.Episode > 0:
			name = fmt.Sprintf("%s E%02d", m.Title, m.Episode)
		}
	default:
		name = folderName(m.Title, m.Year, m.Type)
	}

	if m.Quality != "" {
		name += " [" + m.Quality + "]"
	}

	return SanitizeFilename(name) + "." + m.Ext
}

// CreatorMatch returns the first creator from the list whose name appears as a
// possessive prefix of the title, e.g. "Tyler Perry" matches
// "Tyler Perry's …" and "Tyler Perrys …". Returns "" if no match.
func (m *MediaInfo) CreatorMatch(creators []string) string {
	norm := normaliseCreator(m.Title)
	for _, c := range creators {
		cn := normaliseCreator(c)
		if strings.HasPrefix(norm, cn+" ") || strings.HasPrefix(norm, cn+"s ") {
			return c
		}
	}
	return ""
}

// String returns a short human-readable description.
func (m *MediaInfo) String() string {
	switch m.Type {
	case TypeTVSeries:
		return fmt.Sprintf("[Series] %s S%02dE%02d", m.Title, m.Season, m.Episode)
	case TypeMoviePart:
		return fmt.Sprintf("[Movie/Part %d] %s", m.Part, folderName(m.Title, m.Year, m.Type))
	default:
		return fmt.Sprintf("[Movie] %s", folderName(m.Title, m.Year, m.Type))
	}
}

// folderName builds "<Title> (YYYY)" for movies or just "<Title>" for series.
func folderName(title string, year int, t MediaType) string {
	if year > 0 && t != TypeTVSeries {
		return fmt.Sprintf("%s (%d)", title, year)
	}
	return title
}

// titleWithoutCreator strips the creator prefix (possessive or plain) from title.
func titleWithoutCreator(title, creator string) string {
	if creator == "" {
		return title
	}
	for _, pfx := range []string{
		creator + "'s ",
		creator + "s ",
		creator + " ",
	} {
		if strings.HasPrefix(strings.ToLower(title), strings.ToLower(pfx)) {
			return strings.TrimSpace(title[len(pfx):])
		}
	}
	return title
}

// normaliseCreator lowercases and removes apostrophes for prefix comparison.
func normaliseCreator(s string) string {
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "’", "") // right single quotation mark
	return strings.ToLower(s)
}
