package extract

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
)

//go:embed skills.json
var skillsJSON []byte

// skillEntry is one dictionary entry: the canonical name, aliases matched
// case-insensitively and exact spellings matched case-sensitively. The name is
// an alias too unless it is listed as exact.
type skillEntry struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
	Exact   []string `json:"exact"`
}

type skill struct {
	name  string
	any   *regexp.Regexp
	exact *regexp.Regexp
}

// Dictionary finds skills in text by their names and aliases.
type Dictionary struct {
	skills []skill
}

// boundary is a character that may surround a skill: anything but a letter, a
// digit, "+" or "#", so "C" never matches inside "C++" and "Java" never inside
// "JavaScript".
const boundary = `[^\p{L}\p{N}+#]`

var loadSkills = sync.OnceValue(func() *Dictionary {
	var entries []skillEntry
	if err := json.Unmarshal(skillsJSON, &entries); err != nil {
		panic("extract: invalid skills.json: " + err.Error())
	}
	return newDictionary(entries)
})

// Skills returns the embedded skills dictionary.
func Skills() *Dictionary { return loadSkills() }

// newDictionary compiles entries; aliases are folded like the text they match.
func newDictionary(entries []skillEntry) *Dictionary {
	d := &Dictionary{}
	for _, e := range entries {
		aliases := slices.Clone(e.Aliases)
		if !slices.Contains(e.Exact, e.Name) {
			aliases = append(aliases, e.Name)
		}
		s := skill{name: e.Name, any: pattern(aliases, Fold), exact: pattern(e.Exact, foldMarks)}
		d.skills = append(d.skills, s)
	}
	return d
}

func pattern(aliases []string, fold func(string) string) *regexp.Regexp {
	if len(aliases) == 0 {
		return nil
	}
	quoted := make([]string, len(aliases))
	for i, alias := range aliases {
		quoted[i] = regexp.QuoteMeta(fold(alias))
	}
	// Longer aliases first so "react native" is tried before "react".
	sort.Slice(quoted, func(i, j int) bool { return len(quoted[i]) > len(quoted[j]) })
	return regexp.MustCompile(`(?:^|` + boundary + `)(` + strings.Join(quoted, "|") + `)(?:$|` + boundary + `)`)
}

// Find returns the canonical skills mentioned in text, in order of first
// appearance.
func (d *Dictionary) Find(text string) []string {
	folded, marks := Fold(text), foldMarks(text)
	type hit struct {
		name string
		at   int
	}
	var hits []hit
	for _, s := range d.skills {
		at := -1
		if s.any != nil {
			if m := s.any.FindStringSubmatchIndex(folded); m != nil {
				at = m[2]
			}
		}
		if s.exact != nil {
			if m := s.exact.FindStringSubmatchIndex(marks); m != nil && (at < 0 || m[2] < at) {
				at = m[2]
			}
		}
		if at >= 0 {
			hits = append(hits, hit{s.name, at})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].at < hits[j].at })
	names := make([]string, len(hits))
	for i, h := range hits {
		names[i] = h.name
	}
	return names
}

var romanian = strings.NewReplacer(
	"ă", "a", "â", "a", "î", "i", "ș", "s", "ş", "s", "ț", "t", "ţ", "t",
	"Ă", "A", "Â", "A", "Î", "I", "Ș", "S", "Ş", "S", "Ț", "T", "Ţ", "T",
)

// foldMarks removes Romanian diacritics and keeps the case.
func foldMarks(value string) string { return romanian.Replace(value) }

// Fold lower-cases value and removes Romanian diacritics, so "Iași" and
// "IASI" both become "iasi".
func Fold(value string) string { return strings.ToLower(foldMarks(value)) }
