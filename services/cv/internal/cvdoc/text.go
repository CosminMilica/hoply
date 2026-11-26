// Package cvdoc is the rules engine: it reads a job against a profile, asks
// the questions of Prepare your CV, and writes both versions of a CV with the
// job's words, never inventing a fact (docs/api/cv.md).
package cvdoc

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

func lc(value string) string { return strings.ToLower(value) }

func Capitalize(value string) string {
	first, size := utf8.DecodeRuneInString(value)
	if size == 0 {
		return value
	}
	return string(unicode.ToUpper(first)) + value[size:]
}

func lowerFirst(value string) string {
	first, size := utf8.DecodeRuneInString(value)
	if size == 0 {
		return value
	}
	return string(unicode.ToLower(first)) + value[size:]
}

// JoinList joins items the way a sentence does: "a, b and c".
func JoinList(items []string) string {
	if len(items) <= 1 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// wordPattern matches word as a whole word, ignoring case: group 1 is the character before it, 2 the word, 3 the
// character after. A boundary is anything but a letter, a digit, "_", "+" or "#", as in the jobs service's
// skills: "C++", "C#" and ".NET" match, and "C" does not match inside "C++".
func wordPattern(word string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(^|[^\pL\pN_+#])(` + regexp.QuoteMeta(word) + `)($|[^\pL\pN_+#])`)
}

// HasWord reports whether text names word as a whole word, ignoring case.
func HasWord(text, word string) bool {
	return wordPattern(word).MatchString(text)
}

var past = map[string]string{
	"build": "built", "ship": "shipped", "work": "worked", "review": "reviewed", "document": "documented", "add": "added",
	"write": "wrote", "design": "designed", "manage": "managed", "run": "ran", "watch": "watched", "own": "owned", "join": "joined",
	"keep": "kept", "clean": "cleaned", "fix": "fixed", "share": "shared", "maintain": "maintained", "create": "created",
	"develop": "developed", "improve": "improved", "check": "checked", "test": "tested", "administer": "administered",
	"automate": "automated", "implement": "implemented", "monitor": "monitored", "troubleshoot": "troubleshot",
	"support": "supported", "collaborate": "collaborated", "respond": "responded", "participate": "participated",
	"analyze": "analyzed", "mentor": "mentored", "turn": "turned", "install": "installed", "plan": "planned", "apply": "applied",
	"escalate": "escalated", "lead": "led", "harden": "hardened",
}

func PastOf(word string) string {
	if verb, ok := past[lc(word)]; ok {
		return verb
	}
	return word
}

var (
	trailingDot = regexp.MustCompile(`[.]$`)
	ourWord     = regexp.MustCompile(`(?i)\bour\b`)
	yourWord    = regexp.MustCompile(`(?i)\byour\b`)
	firstWord   = regexp.MustCompile(`^(\w+)`)
	andWord     = regexp.MustCompile(` and (\w+)`)
)

// PastPhrase turns a responsibility into a past-tense phrase: "Build and ship
// features in our React web app." becomes "built and shipped features in a
// React web app".
func PastPhrase(line string) string {
	text := trailingDot.ReplaceAllString(line, "")
	text = ourWord.ReplaceAllString(text, "a")
	text = yourWord.ReplaceAllString(text, "the")
	text = replaceFirst(firstWord, text, func(match []string) string { return PastOf(match[1]) })
	text = replaceFirst(andWord, text, func(match []string) string { return " and " + PastOf(match[1]) })
	return lowerFirst(text)
}

// replaceFirst replaces the first match of pattern in text with what fn makes
// of its submatches.
func replaceFirst(pattern *regexp.Regexp, text string, fn func([]string) string) string {
	at := pattern.FindStringSubmatchIndex(text)
	if at == nil {
		return text
	}
	groups := make([]string, len(at)/2)
	for i := range groups {
		if at[2*i] >= 0 {
			groups[i] = text[at[2*i]:at[2*i+1]]
		}
	}
	return text[:at[0]] + fn(groups) + text[at[1]:]
}

// countKeywordHits counts the places text names one of the keywords, ignoring
// case.
func countKeywordHits(text string, keywords []string) int {
	if len(keywords) == 0 {
		return 0
	}
	quoted := make([]string, len(keywords))
	for i, keyword := range keywords {
		quoted[i] = regexp.QuoteMeta(keyword)
	}
	return len(regexp.MustCompile(`(?i)(`+strings.Join(quoted, "|")+`)`).FindAllStringIndex(text, -1))
}

// keepNonEmpty drops the empty strings, as a JavaScript filter(Boolean) does.
func keepNonEmpty(values ...string) []string {
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			kept = append(kept, value)
		}
	}
	return kept
}

// keepNonBlank drops the strings that are empty once trimmed.
func keepNonBlank(values ...string) []string {
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			kept = append(kept, value)
		}
	}
	return kept
}

func containsFold(items []string, value string) bool {
	for _, item := range items {
		if lc(item) == lc(value) {
			return true
		}
	}
	return false
}
