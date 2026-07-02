package media

import "strings"

// sidecarSeparators are the characters that may sit between a media file's stem
// and a sidecar's trailing language / flag tag, e.g. the dot in
// "Movie.2020.en.srt" or the space in "Movie 2020 en.srt".
const sidecarSeparators = " ._-"

// MatchSidecar reports whether subStem belongs to the media file mediaStem and,
// when it does, returns the trailing language / flag tag (already normalised to
// dot separators, e.g. "en", "en.forced", "pt.BR"). Both stems must be passed
// WITHOUT their extension.
//
// A sidecar matches when its stem either equals the media stem exactly
// ("Movie.2020.srt" beside "Movie.2020.mkv") or extends it after a separator
// ("Movie.2020.en.srt" → tag "en"). Matching is case-insensitive; the returned
// tag preserves the original case of the source so region codes such as "BR"
// survive.
//
//	MatchSidecar("Movie 2020", "Movie 2020")           → "",          true
//	MatchSidecar("Movie 2020", "Movie 2020.en")        → "en",        true
//	MatchSidecar("Movie 2020", "Movie 2020.en.forced") → "en.forced", true
//	MatchSidecar("Movie 2020", "Other Movie")          → "",          false
func MatchSidecar(mediaStem, subStem string) (string, bool) {
	m := strings.ToLower(mediaStem)
	s := strings.ToLower(subStem)

	if s == m {
		return "", true
	}
	if !strings.HasPrefix(s, m) {
		return "", false
	}

	// Preserve original case for the tag; the prefix length is identical in
	// bytes because ToLower does not change the length of ASCII stems.
	rest := subStem[len(mediaStem):]
	if rest == "" {
		return "", true
	}
	if !strings.ContainsRune(sidecarSeparators, rune(rest[0])) {
		return "", false // number/word ran on — not a real boundary
	}

	tag := strings.Trim(rest, sidecarSeparators)
	if tag == "" {
		return "", true
	}
	// Collapse any internal separators to dots so the emitted name is uniform:
	// "en forced" / "en-forced" → "en.forced".
	tag = strings.NewReplacer(" ", ".", "_", ".", "-", ".").Replace(tag)
	tag = SanitizeFilename(tag)
	return tag, true
}

// SidecarName builds the destination filename for a sidecar given the final
// (already-collision-resolved) stem of its media file, the language / flag tag
// returned by MatchSidecar, and the sidecar's own extension.
//
//	SidecarName("The Office S01E05 [720p]", "en",        "srt") → "The Office S01E05 [720p].en.srt"
//	SidecarName("The Office S01E05 [720p]", "en.forced", "srt") → "The Office S01E05 [720p].en.forced.srt"
//	SidecarName("The Office S01E05 [720p]", "",          "srt") → "The Office S01E05 [720p].srt"
func SidecarName(mediaStem, tag, ext string) string {
	if tag != "" {
		mediaStem += "." + tag
	}
	return mediaStem + "." + ext
}