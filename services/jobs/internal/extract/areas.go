package extract

import "regexp"

// Areas are the tech areas a job's title can name, in the order Find jobs shows them.
var Areas = []string{"DevOps", "Backend", "Frontend", "Full stack", "Mobile", "Data", "AI", "QA", "Security", "Embedded", "Sysadmin", "Product", "Design"}

var (
	areaTitles = map[string]*regexp.Regexp{
		"DevOps":     regexp.MustCompile(`devops|dev ops|\bsre\b|site reliability|reliability engineer|platform engineer|infrastructure|\bcloud\b|kubernetes|release engineer|build engineer|mlops`),
		"Backend":    regexp.MustCompile(`back-?end|back end|software (?:engineer|developer)|\bdeveloper\b|programmer|\b(?:java|golang|go|python|node\.?js|nodejs|\.net|c#|c\+\+|php|ruby|rust|scala|kotlin|salesforce|sap)\b.*\b(?:engineer|developer)\b|\bapi\b`),
		"Frontend":   regexp.MustCompile(`front-?end|front end|\bui (?:engineer|developer)|\bweb developer|\b(?:react|angular|vue)(?:\.?js)?\b.*\b(?:engineer|developer)\b`),
		"Full stack": regexp.MustCompile(`full-?stack|full stack`),
		"Mobile":     regexp.MustCompile(`\bmobile\b|\bios\b|android|react native|flutter (?:engineer|developer)`),
		"Data":       regexp.MustCompile(`\bdata\b|analytics|\bbi\b|\betl\b|big data|databricks|snowflake`),
		"AI":         regexp.MustCompile(`\bai\b|\bml\b|machine learning|\bllm|gen ?ai|deep learning|\bnlp\b|computer vision|data scientist|agentic`),
		"QA":         regexp.MustCompile(`\bqa\b|\btest|\bsdet\b|quality (?:assurance|engineer)`),
		"Security":   regexp.MustCompile(`security|secops|\bsoc\b|penetration|pentest|cyber|\biam\b|appsec`),
		"Embedded":   regexp.MustCompile(`embedded|firmware|autosar|\bfpga\b`),
		"Sysadmin":   regexp.MustCompile(`administrator|sysadmin|\blinux\b|\bnetwork\b|systems engineer|system engineer|\bnoc\b|it support|help ?desk`),
		"Product":    regexp.MustCompile(`product (?:manager|owner)|\bproduct lead|technical product`),
		"Design":     regexp.MustCompile(`\bux\b|\bui\b.*designer|product designer|ux/ui|ui/ux|interaction designer`),
	}
	backEnd = regexp.MustCompile(`back-?end|back end`)
	// narrower names the areas a generic "developer" or "software engineer" title belongs to instead of Backend.
	narrower = []string{"Frontend", "Full stack", "Mobile", "QA", "Embedded", "Design"}
)

// AreasOf returns the areas a job title names, in the order of Areas; none for a title outside tech. A title is
// Backend when it says back end, or when it names a developer or a server language and no narrower area.
func AreasOf(title string) []string {
	folded := Fold(title)
	found := map[string]bool{}
	for area, pattern := range areaTitles {
		found[area] = pattern.MatchString(folded)
	}
	if found["Backend"] && !backEnd.MatchString(folded) {
		for _, area := range narrower {
			found["Backend"] = found["Backend"] && !found[area]
		}
	}
	areas := []string{}
	for _, area := range Areas {
		if found[area] {
			areas = append(areas, area)
		}
	}
	return areas
}
