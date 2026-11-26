package extract

import "regexp"

var (
	devopsTitle   = regexp.MustCompile(`devops|dev ops|\bsre\b|site reliability|reliability engineer|platform engineer|infrastructure|\bcloud\b|kubernetes|release engineer|build engineer|mlops`)
	sysadminTitle = regexp.MustCompile(`administrator|sysadmin|\blinux\b|\bnetwork\b|systems engineer|system engineer|\bnoc\b`)
	dataTitle     = regexp.MustCompile(`\bdata\b|analytics|machine learning|\bml\b|\bai engineer|\bbi\b|\betl\b`)
	backendTitle  = regexp.MustCompile(`back-?end|back end|full-?stack|full stack|software (?:engineer|developer)|\bdeveloper\b|programmer|\b(?:java|golang|go|python|node\.?js|nodejs|\.net|c\+\+|php|ruby|rust|scala)\b.*\b(?:engineer|developer)\b|\bapi\b`)
	clientTitle   = regexp.MustCompile(`front-?end|front end|\bui\b|\bios\b|android|mobile|\bqa\b|test|game`)
	serverTitle   = regexp.MustCompile(`back-?end|back end|full-?stack|full stack`)
)

// Areas returns the areas Find jobs filters on (DevOps, Backend, Data, Sysadmin) that a job title names, in that
// order; none for a title in another field. A front-end, mobile or test title is Backend only when it also says
// back end or full stack.
func Areas(title string) []string {
	folded := Fold(title)
	areas := []string{}
	if devopsTitle.MatchString(folded) {
		areas = append(areas, "DevOps")
	}
	if backendTitle.MatchString(folded) && (!clientTitle.MatchString(folded) || serverTitle.MatchString(folded)) {
		areas = append(areas, "Backend")
	}
	if dataTitle.MatchString(folded) {
		areas = append(areas, "Data")
	}
	if sysadminTitle.MatchString(folded) {
		areas = append(areas, "Sysadmin")
	}
	return areas
}
