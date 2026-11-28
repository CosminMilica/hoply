package extract

import (
	"regexp"
	"slices"
)

// Domains are the fields of work a person picks when they sign up, in the order the app shows them.
var Domains = []string{"it", "banking", "accounting", "finance", "insurance", "hr"}

// DomainAreas are each domain's areas, in the order Find jobs shows them. An area belongs to one domain.
var DomainAreas = map[string][]string{
	"it":         {"DevOps", "Backend", "Frontend", "Full stack", "Mobile", "Data", "AI", "QA", "Security", "Embedded", "Sysadmin", "Product", "Design"},
	"banking":    {"Front office", "Lending", "Bank operations", "Corporate banking"},
	"accounting": {"General accounting", "Payables", "Receivables", "Payroll", "Tax"},
	"finance":    {"FP&A & controlling", "Treasury", "Risk & compliance", "Audit"},
	"insurance":  {"Underwriting", "Claims", "Insurance sales", "Actuarial"},
	"hr":         {"Recruitment", "HR generalist", "Compensation & benefits", "Training & development", "HR administration"},
}

// Areas are every domain's areas, in the order of Domains.
var Areas = func() []string {
	areas := []string{}
	for _, domain := range Domains {
		areas = append(areas, DomainAreas[domain]...)
	}
	return areas
}()

// fallback is the area of a job in a domain whose title names none of the domain's areas, for the domains with a
// general one.
var fallback = map[string]string{"accounting": "General accounting", "hr": "HR generalist"}

// The rules read titles folded (lower case, no diacritics), in English and Romanian, and the process codes of
// shared-service centres: "Specialist - F&A - AP 4A" is accounts payable.
var (
	// domainTitles name a domain outside IT; a title in IT is one that names an IT area.
	domainTitles = map[string]*regexp.Regexp{
		"banking": regexp.MustCompile(`\bbank|bancar|\bbanca\b|\bbanker\b|relationship manager|` +
			`credit (?:officer|analyst|specialist|advisor|consultant)|\bloans?\b|mortgage|ipotecar|creditare|teller|casier (?:bancar|valutar)|` +
			`branch (?:manager|advisor)|consilier (?:bancar|financiar|clienti)|clienti (?:pf|imm|persoane)|specialist (?:pf|imm)|` +
			`(?:referent|ofiter) (?:credite|bancar)`),
		"accounting": regexp.MustCompile(`accountant|accounting|contabil|accounts (?:payable|receivable)|\b(?:ap|ar|i2c|p2p|r2r|o2c)\b|` +
			`record to report|invoice to cash|order to cash|procure to pay|general ledger|\bgl\b|bookkeep|payroll|salarizare|` +
			`billing|facturare|collections?\b|cash applications?|reconciliation|reconcilier|\bf&a\b|` +
			`\btax\b|fiscal|referent financiar|economist`),
		"finance": regexp.MustCompile(`financial (?:analyst|controller|planning|reporting|manager|specialist|officer|assistant)|` +
			`finance (?:manager|specialist|analyst|business partner|lead|director|officer|associate|assistant)|\bfp&a\b|` +
			`\bcontroller\b|controlling|controlor financiar|specialist finance|treasury|trezorerie|analist financiar|\bcfo\b|` +
			`reporting (?:specialist|analyst)|budget(?:ing)? analyst|management accountant|` +
			`\brisk\b|\brisc\b|compliance|conformitate|\baml\b|\bkyc\b|money laundering|spalare|fraud|\baudit|\bsox\b|` +
			`internal control|controllership|regulatory (?:compliance|reporting|affairs|analyst|specialist)|reglementar|sanctions screening`),
		"insurance": regexp.MustCompile(`insurance|insurer|reinsur|\basigurari\b|asigurator|reasigur|underwrit|subscriere|` +
			`\bclaims?\b|daune|loss adjust|actuar|bancassurance|(?:agent|broker|inspector|consultant|consilier|referent) (?:de )?asigurar[ei]`),
		"hr": regexp.MustCompile(`\bhr\b|hrbp|\bhris\b|human resources|resurse umane|recruit|recrut|talent (?:acquisition|sourc|partner)|sourcer|` +
			`headhunt|people (?:partner|operations|business partner)|learning (?:and|&) development|\bl&d\b|` +
			`training (?:specialist|coordinator|manager|consultant)|compensation|\bc&b\b|total rewards|payroll|salarizare|` +
			`benefits (?:coordinator|specialist|analyst|advisor|manager|partner)|` +
			`(?:administrare|inspector|referent) (?:de )?personal`),
	}
	areaTitles = map[string]*regexp.Regexp{
		"DevOps":     regexp.MustCompile(`devops|dev ops|\bsre\b|site reliability|reliability engineer|platform engineer|infrastructure|\bcloud\b|kubernetes|release engineer|build engineer|mlops`),
		"Backend":    regexp.MustCompile(`back-?end|back end|software (?:engineer|developer)|\bdeveloper\b|programmer|programator|dezvoltator (?:software|aplicatii|web)|inginer (?:de )?(?:sistem )?software|\b(?:java|golang|go|python|node\.?js|nodejs|\.net|c#|c\+\+|php|ruby|rust|scala|kotlin|salesforce|sap)\b.*\b(?:engineer|developer)\b|\bapi\b`),
		"Frontend":   regexp.MustCompile(`front-?end|front end|\bui (?:engineer|developer)|\bweb developer|\b(?:react|angular|vue)(?:\.?js)?\b.*\b(?:engineer|developer)\b`),
		"Full stack": regexp.MustCompile(`full-?stack|full stack`),
		"Mobile":     regexp.MustCompile(`\bmobile\b|\bios\b|android|react native|flutter (?:engineer|developer)`),
		"Data":       regexp.MustCompile(`\bdata\b|analist (?:de )?date|analytics|\bbi\b|\betl\b|big data|databricks|snowflake`),
		"AI":         regexp.MustCompile(`\bai\b|\bml\b|machine learning|\bllm|gen ?ai|deep learning|\bnlp\b|computer vision|data scientist|agentic`),
		"QA":         regexp.MustCompile(`\bqa\b|\btest|\bsdet\b|quality (?:assurance|engineer)`),
		"Security":   regexp.MustCompile(`security|securitate cibernetica|securitatea informatiei|secops|\bsoc\b|penetration|pentest|cyber|\biam\b|appsec`),
		"Embedded":   regexp.MustCompile(`embedded|firmware|autosar|\bfpga\b`),
		"Sysadmin": regexp.MustCompile(`sysadmin|\blinux\b|\bnetwork\b|systems? (?:engineer|administrator)|\bnoc\b|it support|help ?desk|` +
			`(?:database|cloud|windows|server|it|system) administrator|administrator (?:de )?(?:retea|retele|sistem|sisteme|baze de date)|` +
			`tehnician (?:retea|retele|echipamente de calcul|calculatoare)`),
		"Product": regexp.MustCompile(`product (?:manager|owner)|\bproduct lead|technical product`),
		"Design":  regexp.MustCompile(`\bux\b|\bui\b.*designer|product designer|ux/ui|ui/ux|interaction designer`),

		"Front office": regexp.MustCompile(`teller|casier|ghiseu|branch|sucursal|agenti[ea]|consilier (?:bancar|financiar|clienti)|` +
			`personal banker|clienti (?:pf|persoane)|specialist pf|retail|front office|customer (?:advisor|service)|relatii cu clientii`),
		"Lending":           regexp.MustCompile(`credit|\bloans?\b|lending|mortgage|ipotecar`),
		"Bank operations":   regexp.MustCompile(`operations|operatiuni|back office|payments?\b|plati|\bswift\b|trade finance|settlement|custody|card`),
		"Corporate banking": regexp.MustCompile(`corporate|\bimm\b|\bsme\b|relationship manager|business banking|persoane juridice|\bpj\b`),

		"General accounting": regexp.MustCompile(`general ledger|\bgl\b|record to report|\br2r\b|bookkeep|reconcil|month.?end|` +
			`fixed assets|intercompany|contabil (?:general|sef|primar)`),
		"Payables":    regexp.MustCompile(`accounts payable|\bap\b|\bp2p\b|procure to pay|purchase to pay|furnizori|invoice processing`),
		"Receivables": regexp.MustCompile(`accounts receivable|\bar\b|\bo2c\b|\bi2c\b|order to cash|invoice to cash|collections?\b|cash applications?|credit control|billing|facturare|incasari|creante`),
		"Payroll":     regexp.MustCompile(`payroll|salarizare`),
		"Tax":         regexp.MustCompile(`\btax\b|\btaxe\b|fiscal|\bvat\b|\btva\b|transfer pricing|impozit`),

		"FP&A & controlling": regexp.MustCompile(`\bfp&a\b|financial (?:planning|analyst|reporting)|\bcontroller\b|controlling|controlor|budget|forecast|` +
			`analist financiar|finance business partner|management accountant|cost (?:analyst|controller|accountant)|reporting (?:specialist|analyst)`),
		"Treasury": regexp.MustCompile(`treasury|trezorerie|cash management|liquidity|lichiditate`),
		"Risk & compliance": regexp.MustCompile(`\brisk\b|\brisc\b|compliance|conformitate|\baml\b|\bkyc\b|money laundering|spalare|fraud|` +
			`regulatory (?:compliance|reporting|affairs|analyst|specialist)|reglementar|sanctions screening`),
		"Audit": regexp.MustCompile(`\baudit|\bsox\b|internal control|controllership`),

		"Underwriting":    regexp.MustCompile(`underwrit|subscri(?:ere|itor)`),
		"Claims":          regexp.MustCompile(`\bclaims?\b|daune|loss adjust|adjuster|constatator`),
		"Insurance sales": regexp.MustCompile(`(?:insurance|asigurari) (?:sales|agent|advisor|consultant|broker)|(?:sales|agent|consultant|broker|vanzari) (?:de )?(?:insurance|asigurari)|bancassurance|account manager|vanzari|agent (?:de )?asigurar[ei]`),
		"Actuarial":       regexp.MustCompile(`actuar`),

		"Recruitment":             regexp.MustCompile(`recruit|recrut|talent (?:acquisition|sourc|partner)|sourcer|headhunt`),
		"HR generalist":           regexp.MustCompile(`generalist|business partner|hrbp|people partner|hr (?:manager|director|lead|consultant)|manager resurse umane`),
		"Compensation & benefits": regexp.MustCompile(`compensation|benefits?\b|rewards?\b|\bc&b\b|beneficii|remunerare|payroll|salarizare|\bpay\b`),
		"Training & development":  regexp.MustCompile(`learning|\bl&d\b|training|trainer|talent development|instructional design|formare`),
		"HR administration": regexp.MustCompile(`hr (?:administrator|admin|assistant|operations|ops|services|shared services|coordinator)|` +
			`personnel|people operations|\bhris\b|onboarding|employee (?:data|records|services)|(?:administrare|inspector|referent) (?:de )?(?:personal|resurse umane)`),
	}
	// sysadmin names a system administrator; "administrator" alone is also a title elsewhere (Payroll Administrator).
	sysadmin = regexp.MustCompile(`sysadmin|\blinux\b|\bnetwork\b|systems? (?:engineer|administrator)|\bnoc\b|it support|help ?desk|` +
		`(?:database|cloud|windows|server|it) administrator`)
	backEnd = regexp.MustCompile(`back-?end|back end`)
	// narrower names the areas a generic "developer" or "software engineer" title belongs to instead of Backend.
	narrower = []string{"Frontend", "Full stack", "Mobile", "QA", "Embedded", "Design"}
)

// Classification is what a job's title says about its field of work: its domains, in the order of Domains, and its
// areas, in the order of Areas.
type Classification struct {
	Domains []string
	Areas   []string
}

// Classify reads a job title's domains and areas; none for a title outside them. A title is in a domain outside IT
// when it names the domain, and then in each of the domain's areas it names, or the domain's general area when it
// names none. A title is in IT when it names an IT area: Backend when it says back end, or when it names a
// developer or a server language and no narrower area; Sysadmin only for a system administrator when it is also
// in another domain.
func Classify(title string) Classification {
	folded := Fold(title)
	inDomain := map[string]bool{}
	for domain, pattern := range domainTitles {
		inDomain[domain] = pattern.MatchString(folded)
	}
	found := map[string]bool{}
	for _, area := range DomainAreas["it"] {
		found[area] = areaTitles[area].MatchString(folded)
	}
	if found["Sysadmin"] && !sysadmin.MatchString(folded) {
		for domain, in := range inDomain {
			found["Sysadmin"] = found["Sysadmin"] && !(in && domain != "it")
		}
	}
	if found["Backend"] && !backEnd.MatchString(folded) {
		for _, area := range narrower {
			found["Backend"] = found["Backend"] && !found[area]
		}
	}
	for _, area := range DomainAreas["it"] {
		inDomain["it"] = inDomain["it"] || found[area]
	}
	for domain, in := range inDomain {
		if !in || domain == "it" {
			continue
		}
		named := false
		for _, area := range DomainAreas[domain] {
			if pattern, ok := areaTitles[area]; ok && pattern.MatchString(folded) && area != fallback[domain] {
				found[area], named = true, true
			}
		}
		if general, ok := fallback[domain]; ok && (!named || areaTitles[general].MatchString(folded)) {
			found[general] = true
		}
	}
	c := Classification{Domains: []string{}, Areas: []string{}}
	for _, domain := range Domains {
		if inDomain[domain] {
			c.Domains = append(c.Domains, domain)
		}
	}
	for _, area := range Areas {
		if found[area] && slices.Contains(c.Domains, DomainOf(area)) {
			c.Areas = append(c.Areas, area)
		}
	}
	return c
}

// DomainOf is the domain an area belongs to, or "" for an unknown area.
func DomainOf(area string) string {
	for _, domain := range Domains {
		if slices.Contains(DomainAreas[domain], area) {
			return domain
		}
	}
	return ""
}
