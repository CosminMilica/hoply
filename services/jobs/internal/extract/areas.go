package extract

import "regexp"

// Areas are the areas a job's title can name, in the order Find jobs shows them: the tech areas, then the finance
// ones (TechAreas, FinanceAreas).
var Areas = append(append([]string{}, TechAreas...), FinanceAreas...)

var (
	TechAreas    = []string{"DevOps", "Backend", "Frontend", "Full stack", "Mobile", "Data", "AI", "QA", "Security", "Embedded", "Sysadmin", "Product", "Design"}
	FinanceAreas = []string{"Banking", "Accounting", "Finance & control", "Risk & compliance"}
)

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
		// The finance areas read English and Romanian titles (folded, without diacritics) and the process codes of
		// shared-service centres: "Specialist - F&A - AP 4A" is accounts payable.
		"Banking": regexp.MustCompile(`\bbank|bancar|\bbanca\b|bancassurance|\bbanker\b|relationship manager|` +
			`credit (?:officer|analyst|specialist|advisor|consultant)|\bloans?\b|mortgage|ipotecar|creditare|teller|casier|` +
			`branch (?:manager|advisor)|consilier (?:bancar|financiar|clienti)|clienti (?:pf|imm|persoane)|specialist (?:pf|imm)|` +
			`(?:referent|ofiter) (?:credite|bancar)`),
		"Accounting": regexp.MustCompile(`accountant|accounting|contabil|accounts (?:payable|receivable)|\b(?:ap|ar|i2c|p2p|r2r|o2c)\b|` +
			`record to report|invoice to cash|order to cash|procure to pay|general ledger|\bgl\b|bookkeep|payroll|salarizare|` +
			`billing|facturare|collections?\b|cash applications?|reconciliation|reconcilier|\bf&a\b|` +
			`tax (?:specialist|accountant|analyst|consultant|manager)|fiscal|referent financiar|economist`),
		"Finance & control": regexp.MustCompile(`financial (?:analyst|controller|planning|reporting|manager|specialist|officer|assistant)|` +
			`finance (?:manager|specialist|analyst|business partner|lead|director|officer|associate|assistant)|\bfp&a\b|` +
			`\bcontroller\b|controlling|treasury|trezorerie|analist financiar|\bcfo\b|` +
			`reporting (?:specialist|analyst)|budget(?:ing)? analyst|management accountant`),
		"Risk & compliance": regexp.MustCompile(`\brisk\b|\brisc\b|compliance|conformitate|\baml\b|\bkyc\b|money laundering|` +
			`spalare|fraud|\baudit|\bsox\b|internal control|controllership|regulatory (?:compliance|reporting|affairs|analyst|specialist)|reglementar|sanctions screening`),
	}
	// sysadmin names a system administrator; "administrator" alone is also a finance title (Payroll Administrator).
	sysadmin = regexp.MustCompile(`sysadmin|\blinux\b|\bnetwork\b|systems? (?:engineer|administrator)|\bnoc\b|it support|help ?desk|` +
		`(?:database|cloud|windows|server|it) administrator`)
	backEnd = regexp.MustCompile(`back-?end|back end`)
	// narrower names the areas a generic "developer" or "software engineer" title belongs to instead of Backend.
	narrower = []string{"Frontend", "Full stack", "Mobile", "QA", "Embedded", "Design"}
)

// AreasOf returns the areas a job title names, in the order of Areas; none for a title outside them. A title is
// Backend when it says back end, or when it names a developer or a server language and no narrower area; it is
// Sysadmin only for a system administrator when it also names a finance area.
func AreasOf(title string) []string {
	folded := Fold(title)
	found := map[string]bool{}
	for area, pattern := range areaTitles {
		found[area] = pattern.MatchString(folded)
	}
	if found["Sysadmin"] && !sysadmin.MatchString(folded) {
		for _, area := range FinanceAreas {
			found["Sysadmin"] = found["Sysadmin"] && !found[area]
		}
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
