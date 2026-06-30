package media

import "testing"

func TestSanitizeFilename(t *testing.T) {
	cases := []struct{ input, want string }{
		{"hello world", "hello world"},
		// ':' → " -", so "file: name" → "file" + " -" + " name" = "file - name"
		{"file: name", "file - name"},
		{"file:name", "file -name"},
		{"path/to/file", "path-to-file"},
		{"a\\b", "a-b"},
		{"pipe|here", "pipe-here"},
		{"bad<>name", "badname"},
		{"a*b?c", "abc"},
		{`"quoted"`, "quoted"},
		{"  spaces  ", "spaces"},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := SanitizeFilename(c.input)
			if got != c.want {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestCanonicalQuality(t *testing.T) {
	cases := []struct{ input, want string }{
		{"bluray", "BluRay"},
		{"blu-ray", "BluRay"},
		{"brrip", "BluRay"},
		{"bdrip", "BDRip"},
		{"dvdrip", "DVDRip"},
		{"dvdscr", "DVDRip"},
		{"webrip", "WEBRip"},
		{"web-dl", "WEB-DL"},
		{"webdl", "WEB-DL"},
		{"web", "WEB"},
		{"4k", "4K"},
		{"uhd", "4K"},
		{"hdtv", "HDTV"},
		{"pdtv", "HDTV"},
		{"1080p", "1080p"},
		{"720p", "720p"},
		{"AMZN", "AMZN"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := canonicalQuality(c.input)
			if got != c.want {
				t.Errorf("canonicalQuality(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestCanonicalHDR(t *testing.T) {
	cases := []struct{ input, want string }{
		{"HDR10+", "HDR10+"},
		{"hdr10+", "HDR10+"},
		{"HDR10", "HDR10"},
		{"HDR", "HDR"},
		{"hdr", "HDR"},
		{"DV", "DV"},
		{"dv", "DV"},
		{"Dolby Vision", "DV"},
		{"Dolby.Vision", "DV"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := canonicalHDR(c.input)
			if got != c.want {
				t.Errorf("canonicalHDR(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}
