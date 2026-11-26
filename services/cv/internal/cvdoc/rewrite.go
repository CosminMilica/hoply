package cvdoc

import (
	"regexp"
	"slices"
	"strings"
)

var verbGroups = [][]string{
	{"built", "developed", "created", "shipped"},
	{"fixed", "resolved"},
	{"maintained", "managed", "ran", "administered"},
	{"wrote", "authored"},
	{"monitored", "tracked"},
}

var terms = [][]string{
	{"deployment pipelines", "release pipelines", "CI/CD pipelines"},
	{"servers", "hosts", "instances"},
	{"incidents", "outages"},
	{"interface", "UI", "user interface"},
	{"platform", "web app", "application", "app"},
	{"websites", "web apps", "sites"},
}

// affinity lists the tools that are only added to a bullet whose wording fits them.
var affinity = map[string]*regexp.Regexp{
	"sql":        regexp.MustCompile(`(?i)report|quer|data|dashboard`),
	"figma":      regexp.MustCompile(`(?i)design|screen|page|flow`),
	"css":        regexp.MustCompile(`(?i)style|page|screen|design|website|interface`),
	"react":      regexp.MustCompile(`(?i)interface|ui|component|feature|app|screen|page|platform`),
	"typescript": regexp.MustCompile(`(?i)interface|ui|component|feature|app|screen|platform|api|service`),
	"javascript": regexp.MustCompile(`(?i)website|site|bug|feature|page|interface`),
	"node.js":    regexp.MustCompile(`(?i)api|service|server|endpoint|quer`),
	"graphql":    regexp.MustCompile(`(?i)api|endpoint`),
	"next.js":    regexp.MustCompile(`(?i)page|app|interface|site`),
	"storybook":  regexp.MustCompile(`(?i)component`),
	"testing":    regexp.MustCompile(`(?i)test|check|bug`),
	"docker":     regexp.MustCompile(`(?i)deploy|container|service|server`),
	"ci/cd":      regexp.MustCompile(`(?i)pipeline|deploy|release`),
	"linux":      regexp.MustCompile(`(?i)server|setup|config|host|account|system`),
	"python":     regexp.MustCompile(`(?i)script|automat|report|data|tool|setup`),
	"git":        regexp.MustCompile(`(?i)release|code|deploy|version|pipeline`),
	"bash":       regexp.MustCompile(`(?i)script|automat|setup|server|backup`),
	"monitoring": regexp.MustCompile(`(?i)monitor|incident|alert|uptime|call`),
	"terraform":  regexp.MustCompile(`(?i)provision|infrastructure|environment`),
	"kubernetes": regexp.MustCompile(`(?i)container|cluster|deploy`),
	"aws":        regexp.MustCompile(`(?i)cloud|infrastructure|deploy|service`),
}

var noInject = []string{"monitoring", "testing", "networking", "backup", "virtualization", "algorithms", "distributed systems"}

// jobText is the job's own words, lower case.
func jobText(job Job) string {
	return lc(job.Title + " " + job.About + " " + strings.Join(job.Responsibilities, " ") + " " + strings.Join(job.Requirements, " "))
}

// VerbPairs is the first verb of each responsibility and, after "and", the
// second one, in the past tense.
func VerbPairs(job Job) [][]string {
	pairs := make([][]string, 0, len(job.Responsibilities))
	for _, line := range job.Responsibilities {
		words := strings.Split(PastPhrase(line), " ")
		pair := []string{words[0]}
		if len(words) > 2 && words[1] == "and" && words[2] != "" {
			pair = append(pair, words[2])
		}
		pairs = append(pairs, pair)
	}
	return pairs
}

// RewriteContext is what a bullet is rewritten with.
type RewriteContext struct {
	Job Job
	// Matched are the job keywords the person has.
	Matched []string
	// Tools are the tools of the role the bullet belongs to.
	Tools []string
	// Used holds the tools already added to this role's bullets; nil adds none.
	Used map[string]bool
}

var leadVerb = regexp.MustCompile(`^(\w+)(\s[\s\S]*)$`)

// RewriteBullet rewrites a bullet in the job's words: its terms, its verbs,
// then the tools the person used in that role.
func RewriteBullet(text string, index int, ctx RewriteContext) string {
	jd := jobText(ctx.Job)
	out := text
	for _, group := range terms {
		jdTerm := ""
		for _, word := range group {
			if HasWord(jd, word) {
				jdTerm = word
				break
			}
		}
		if jdTerm == "" {
			continue
		}
		for _, word := range group {
			if word != jdTerm && HasWord(out, word) && !HasWord(out, jdTerm) {
				out = replaceFirst(wordPattern(word), out, func(m []string) string { return m[1] + jdTerm + m[3] })
			}
		}
	}
	if lead := leadVerb.FindStringSubmatch(out); lead != nil {
		verb := lc(lead[1])
		var group []string
		for _, candidate := range verbGroups {
			if slices.Contains(candidate, verb) {
				group = candidate
				break
			}
		}
		if group != nil {
			if pair := pairIn(VerbPairs(ctx.Job), group); pair != nil {
				verbs := pair
				if words := strings.Split(out, " "); len(words) > 1 && words[1] == "and" {
					verbs = pair[:1]
				}
				if verbs[0] != verb {
					out = Capitalize(JoinList(verbs)) + lead[2]
				}
			}
		}
	}
	if ctx.Used == nil {
		return out
	}
	roleTools := make([]string, len(ctx.Tools))
	for i, tool := range ctx.Tools {
		roleTools[i] = lc(tool)
	}
	fits := func(keyword string) bool {
		pattern, ok := affinity[lc(keyword)]
		return !ok || pattern.MatchString(text)
	}
	added := []string{}
	for _, keyword := range ctx.Matched {
		key := lc(keyword)
		if len(added) == 2 {
			break
		}
		if !slices.Contains(noInject, key) && slices.Contains(roleTools, key) && fits(keyword) &&
			!strings.Contains(lc(out), key) && !ctx.Used[key] {
			added = append(added, keyword)
		}
	}
	if len(added) == 0 {
		return out
	}
	for _, keyword := range added {
		ctx.Used[lc(keyword)] = true
	}
	joiner := " with "
	if strings.Contains(out, " with ") || index%2 == 0 {
		joiner = ", using "
	}
	return trailingDot.ReplaceAllString(out, "") + joiner + JoinList(added) + "."
}

// pairIn is the first verb pair with a verb of group, keeping only those verbs.
func pairIn(pairs [][]string, group []string) []string {
	for _, pair := range pairs {
		kept := []string{}
		for _, verb := range pair {
			if slices.Contains(group, verb) {
				kept = append(kept, verb)
			}
		}
		if len(kept) > 0 {
			return kept
		}
	}
	return nil
}

var (
	spaces       = regexp.MustCompile(`\s+`)
	anyDotsAtEnd = regexp.MustCompile(`[.]*$`)
)

// CleanFact is the person's own words as a CV bullet: capitalised, one full
// stop, and the skill named if they did not name it.
func CleanFact(text, keyword string) string {
	fact := spaces.ReplaceAllString(strings.TrimSpace(text), " ")
	if fact == "" {
		return ""
	}
	fact = anyDotsAtEnd.ReplaceAllString(Capitalize(fact), "")
	if keyword != "" && !HasWord(fact, keyword) {
		fact += ", using " + keyword
	}
	return fact + "."
}
