package regexutil

import (
	"regexp"
	"strings"
)

var quranRefPattern = regexp.MustCompile(`\(QS\.\s*([^:]+):\s*(\d+)\)`)

type QuranReference struct {
	SurahName string
	Ayah      int
	Raw       string
}

func ExtractQuranReferences(text string) []QuranReference {
	matches := quranRefPattern.FindAllStringSubmatch(text, -1)
	refs := make([]QuranReference, 0, len(matches))

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		refs = append(refs, QuranReference{
			SurahName: strings.TrimSpace(match[1]),
			Ayah:      parseInt(match[2]),
			Raw:       match[0],
		})
	}

	return refs
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			continue
		}
		n = n*10 + int(c-'0')
	}
	return n
}
