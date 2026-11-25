package unit_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"jobs/internal/extract"
)

// exampleAd and linkDemoAd are the prototype's ads (docs/design/prototype.dc.html).
var exampleAd = strings.Join([]string{"DevOps Engineer", "Firma: Carpathia Systems, Brașov (hibrid)",
	"Căutăm un coleg care să ne ajute să automatizăm infrastructura și să avem lansări mai sigure.",
	"Responsabilități", "- Administrezi servere Linux și containere Docker.", "- Automatizezi infrastructura cu Ansible și Terraform.",
	"- Configurezi monitorizare și alerte cu Prometheus.", "- Scrii documentație pentru echipa de gardă.",
	"Cerințe", "- Minimum 3 ani experiență în DevOps sau administrare de sisteme.", "- Bash sau Python pentru automatizări.", "- Experiență cu CI/CD în GitLab.",
	"Constituie un avantaj", "- Kubernetes", "- AWS"}, "\n")

var linkDemoAd = strings.Join([]string{"Platform Engineer", "Northwind Cloud · Cluj-Napoca, hybrid",
	"We run the infrastructure behind Northwind payment products and we want releases to be boring.",
	"Responsibilities", "- Run and evolve our Kubernetes clusters on AWS.", "- Build CI/CD pipelines with GitHub Actions and Terraform.",
	"- Improve monitoring and on-call with Prometheus and Grafana.", "- Write runbooks and help developers ship safely.",
	"Requirements", "- 4+ years in DevOps or platform engineering.", "- Strong Linux and Docker skills.", "- Kubernetes and Terraform in production.", "- Python or Go for automation.",
	"Nice to have", "- Helm and Kafka.", "- PostgreSQL."}, "\n")

func years(n int) *int { return &n }

type expected struct {
	title, company, city, country, workMode, level string
	years                                          *int
	must, nice, responsibilities                   []string
	about                                          string
}

func checkResult(t *testing.T, name string, got extract.Result, want expected) {
	t.Helper()
	gotYears, wantYears := "nil", "nil"
	if got.YearsRequired != nil {
		gotYears = fmt.Sprint(*got.YearsRequired)
	}
	if want.years != nil {
		wantYears = fmt.Sprint(*want.years)
	}
	checks := []struct {
		field     string
		got, want any
	}{
		{"title", got.Title, want.title}, {"company", got.Company, want.company}, {"city", got.City, want.city},
		{"country", got.Country, want.country}, {"work_mode", got.WorkMode, want.workMode}, {"level", got.Level, want.level},
		{"years", gotYears, wantYears}, {"about", got.About, want.about},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: %s = %q, want %q", name, c.field, c.got, c.want)
		}
	}
	for _, list := range []struct {
		field     string
		got, want []string
	}{{"must_have", got.MustHave, want.must}, {"nice_to_have", got.NiceToHave, want.nice},
		{"responsibilities", got.Responsibilities, want.responsibilities}} {
		if list.got == nil || !slices.Equal(list.got, list.want) {
			t.Errorf("%s: %s = %q, want %q", name, list.field, list.got, list.want)
		}
	}
}

func TestReadPrototypeAds(t *testing.T) {
	checkResult(t, "exampleAd", extract.Read(exampleAd, extract.Hints{}), expected{
		title: "DevOps Engineer", company: "Carpathia Systems", city: "Brașov", country: "Romania", workMode: "hybrid",
		level: "mid", years: years(3),
		must: []string{"Linux", "Containers", "Docker", "Ansible", "Terraform", "Monitoring", "Prometheus", "Bash", "Python", "CI/CD", "GitLab CI"},
		nice: []string{"Kubernetes", "AWS"},
		responsibilities: []string{"Administrezi servere Linux și containere Docker.", "Automatizezi infrastructura cu Ansible și Terraform.",
			"Configurezi monitorizare și alerte cu Prometheus.", "Scrii documentație pentru echipa de gardă."},
		about: "Căutăm un coleg care să ne ajute să automatizăm infrastructura și să avem lansări mai sigure.",
	})
	checkResult(t, "linkDemoAd", extract.Read(linkDemoAd, extract.Hints{Title: "Platform Engineer", Company: "Northwind Cloud"}), expected{
		title: "Platform Engineer", company: "Northwind Cloud", city: "Cluj-Napoca", country: "Romania", workMode: "hybrid",
		level: "mid", years: years(4),
		must: []string{"Kubernetes", "AWS", "CI/CD", "GitHub Actions", "Terraform", "Monitoring", "Prometheus", "Grafana", "Linux", "Docker", "Python", "Go"},
		nice: []string{"Helm", "Kafka", "PostgreSQL"},
		responsibilities: []string{"Run and evolve our Kubernetes clusters on AWS.", "Build CI/CD pipelines with GitHub Actions and Terraform.",
			"Improve monitoring and on-call with Prometheus and Grafana.", "Write runbooks and help developers ship safely."},
		about: "We run the infrastructure behind Northwind payment products and we want releases to be boring.",
	})
}

func TestReadEnglishAndRomanianAds(t *testing.T) {
	english := strings.Join([]string{
		"Senior Backend Engineer (Remote)",
		"Company: Acme Payments Ltd",
		"Acme builds payment rails for small online shops across Europe and keeps them boring.",
		"Short line.",
		"## What you'll do:",
		"1. Design services in golang and Postgres.",
		"2) Keep our K8s fleet healthy with IaC.",
		"**Requirements**",
		"* 6+ years building backend systems",
		"* You will go fast with Go and TypeScript, not JavaScript or Java.",
		"Bonus",
		"- Kafka, React Native",
		"- Also: Go",
	}, "\n")
	checkResult(t, "english", extract.Read(english, extract.Hints{}), expected{
		title: "Senior Backend Engineer (Remote)", company: "Acme Payments Ltd", workMode: "remote", level: "senior",
		years:            years(6),
		must:             []string{"PostgreSQL", "Kubernetes", "Infrastructure as Code", "TypeScript", "JavaScript", "Java"},
		nice:             []string{"Kafka", "React", "React Native", "Go"},
		responsibilities: []string{"Design services in golang and Postgres.", "Keep our K8s fleet healthy with IaC."},
		about:            "Acme builds payment rails for small online shops across Europe and keeps them boring.",
	})

	romanian := strings.Join([]string{
		"Administrator de rețea Jr.",
		"Angajator: Rețele Vest SRL",
		"Lucrăm la birou în Timisoara, lângă Iulius Town, cu o echipă mică și prietenoasă.",
		"Cerințele postului:",
		"- Cunoștințe de rețele, DNS și DHCP",
		"- Experiență cu Windows Server și Active Directory",
		"Ar fi un plus",
		"- PowerShell",
		"Ce vei face",
		"- Administrezi rețelele și firewall-urile clienților",
	}, "\n")
	checkResult(t, "romanian", extract.Read(romanian, extract.Hints{}), expected{
		title: "Administrator de rețea Jr.", company: "Rețele Vest SRL", city: "Timișoara", country: "Romania",
		workMode: "onsite", level: "junior",
		must:             []string{"Networking", "DNS", "DHCP", "Windows Server", "Active Directory", "Firewalls"},
		nice:             []string{"PowerShell"},
		responsibilities: []string{"Administrezi rețelele și firewall-urile clienților"},
		about:            "Lucrăm la birou în Timisoara, lângă Iulius Town, cu o echipă mică și prietenoasă.",
	})

	unsectioned := "Data Engineer\nWe use Snowflake, dbt and Airflow every day, from our office in Bucuresti or de acasă.\n1 year of SQL is enough."
	checkResult(t, "no sections", extract.Read(unsectioned, extract.Hints{Title: "  Data Engineer ", Company: "Hint Co"}), expected{
		title: "Data Engineer", company: "Hint Co", city: "Bucharest", country: "Romania", workMode: "remote",
		level: "junior", years: years(1),
		must: []string{"Snowflake", "dbt", "Airflow", "SQL"}, nice: []string{}, responsibilities: []string{},
		about: "We use Snowflake, dbt and Airflow every day, from our office in Bucuresti or de acasă.",
	})
}

func TestReadEdgeCases(t *testing.T) {
	empty := extract.Read("   \n\n", extract.Hints{})
	checkResult(t, "empty", empty, expected{must: []string{}, nice: []string{}, responsibilities: []string{}})

	for _, tc := range []struct{ title, level string }{
		{"Sr. Data Engineer", "senior"}, {"Jr Developer", "junior"}, {"Entry-level QA", "junior"},
		{"Tech Lead", "lead"}, {"Principal Engineer", "lead"}, {"Staff SRE", "lead"}, {"Head of Platform", "lead"},
		{"Engineer", ""}, {"Leader of the pack", ""},
	} {
		if got := extract.Read(tc.title, extract.Hints{}).Level; got != tc.level {
			t.Errorf("level of %q = %q, want %q", tc.title, got, tc.level)
		}
	}
	for _, tc := range []struct {
		text  string
		years *int
		level string
	}{
		{"Engineer\n5 years", years(5), "senior"}, {"Engineer\n2 yrs", years(2), "mid"},
		{"Engineer\n0 years", years(0), "junior"}, {"Engineer\nMinimum 3 ani", years(3), "mid"},
		{"Engineer\nfounded 2024 years ago", nil, ""}, {"Engineer\n41 years", nil, ""},
		{"Engineer\n10+ YEARS of Go", years(10), "senior"},
	} {
		got := extract.Read(tc.text, extract.Hints{})
		if (got.YearsRequired == nil) != (tc.years == nil) || (got.YearsRequired != nil && *got.YearsRequired != *tc.years) || got.Level != tc.level {
			t.Errorf("years of %q = %v %q", tc.text, got.YearsRequired, got.Level)
		}
	}
	for _, tc := range []struct{ text, mode string }{
		{"Remote first, hybrid possible", "hybrid"}, {"Office-based in Arad", "onsite"}, {"On site", "onsite"},
		{"Lucrezi de acasă", "remote"}, {"Work remotely", "remote"}, {"Hybridization of crops", "hybrid"}, {"No mode", ""},
	} {
		if got := extract.Read("Title\n"+tc.text, extract.Hints{}).WorkMode; got != tc.mode {
			t.Errorf("work mode of %q = %q, want %q", tc.text, got, tc.mode)
		}
	}

	long := strings.Repeat("Responsibility text that is long enough. ", 20)
	var many []string
	for i := range 35 {
		many = append(many, fmt.Sprintf("- %d %s", i, long))
	}
	clipped := extract.Read(strings.Repeat("T", 200)+"\n"+strings.Repeat("About line with many words. ", 200)+
		"\nResponsibilities\n"+strings.Join(many, "\n"), extract.Hints{Company: strings.Repeat("C", 130)})
	if len([]rune(clipped.Title)) != extract.MaxTitle || len([]rune(clipped.Company)) != extract.MaxCompany ||
		len(clipped.Responsibilities) != extract.MaxResponsibilities || len([]rune(clipped.Responsibilities[0])) > extract.MaxResponsibility ||
		len([]rune(clipped.About)) > extract.MaxAbout {
		t.Fatalf("draft limits: title %d company %d responsibilities %d about %d", len([]rune(clipped.Title)),
			len([]rune(clipped.Company)), len(clipped.Responsibilities), len([]rune(clipped.About)))
	}
	headingTooLong := "Title\nRequirements for this role are listed below in detail, please read them all\n- Kubernetes"
	if got := extract.Read(headingTooLong, extract.Hints{}); len(got.MustHave) != 1 || len(got.Responsibilities) != 0 {
		t.Fatalf("a long line is not a heading: %+v", got)
	}
	bulletHeading := "Title\n- Requirements\n- Kafka\nNice to have\n- Redis"
	if got := extract.Read(bulletHeading, extract.Hints{}); !slices.Equal(got.MustHave, []string{}) || !slices.Equal(got.NiceToHave, []string{"Redis"}) {
		t.Fatalf("a list item is not a heading, and body skills are not read once sections exist: %+v", got)
	}
}

func TestSkillsDictionary(t *testing.T) {
	dictionary := extract.Skills()
	cases := []struct {
		text string
		want []string
	}{
		{"K8s, IaC and Postgres", []string{"Kubernetes", "Infrastructure as Code", "PostgreSQL"}},
		{"golang, Go, GO, go", []string{"Go"}},
		{"let's go", []string{}},
		{"GO-based services", []string{}},
		{"JavaScript only", []string{"JavaScript"}},
		{"C++, C# and .NET (ASP.NET Core)", []string{"C++", "C#", ".NET"}},
		{"GitHub Actions, not GitLab", []string{"GitHub Actions", "GitLab CI"}},
		{"git and GIT", []string{"Git"}},
		{"rețele, retele, REȚELE, reţele", []string{"Networking"}},
		{"monitorizare și virtualizare", []string{"Monitoring", "Virtualization"}},
		{"Swift and swift delivery", []string{"Swift"}},
		{"Node.js, nodejs, Vue", []string{"Node.js", "Vue.js"}},
		{"CI/CD and continuous delivery", []string{"CI/CD"}},
		{"", []string{}},
	}
	for _, tc := range cases {
		got := dictionary.Find(tc.text)
		if !slices.Equal(got, tc.want) && !(len(got) == 0 && len(tc.want) == 0) {
			t.Errorf("Find(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}

func TestPlaceAndFold(t *testing.T) {
	for _, tc := range []struct{ text, city, country string }{
		{"Iasi, on-site", "Iași", "Romania"}, {"IAȘI", "Iași", "Romania"}, {"București", "Bucharest", "Romania"},
		{"Bucharest or Cluj", "Bucharest", "Romania"}, {"Cluj then Bucharest", "Cluj-Napoca", "Romania"},
		{"Targu Mures", "Târgu Mureș", "Romania"}, {"Tg. Mureș", "Târgu Mureș", "Romania"}, {"Ploiești", "Ploiești", "Romania"},
		{"Constanța, Craiova, Galați", "Constanța", "Romania"}, {"Oradea", "Oradea", "Romania"}, {"Sibiu", "Sibiu", "Romania"},
		{"Remote in România", "", "Romania"}, {"Berlin, Germany", "", ""}, {"Charade", "", ""}, {"Romanian speakers", "", ""},
	} {
		if city, country := extract.Place(tc.text); city != tc.city || country != tc.country {
			t.Errorf("Place(%q) = %q %q, want %q %q", tc.text, city, country, tc.city, tc.country)
		}
	}
	for _, tc := range []struct{ value, name string }{
		{"iasi", "Iași"}, {" Bucuresti ", "Bucharest"}, {"cluj", "Cluj-Napoca"}, {"Paris", ""}, {"", ""},
	} {
		if got := extract.CityName(tc.value); got != tc.name {
			t.Errorf("CityName(%q) = %q, want %q", tc.value, got, tc.name)
		}
	}
	if got := extract.Fold("ĂÂÎȘŞȚŢ ăâîșşțţ Iași"); got != "aaissttaaisstt iasi" && got != "aaisstt aaisstt iasi" {
		t.Errorf("Fold = %q", got)
	}
}

func TestReadHeadingVocabularyAndIgnoredSections(t *testing.T) {
	ad := strings.Join([]string{
		"Backend Engineer",
		"Company: Northwind Cloud",
		"Northwind Cloud builds payment infrastructure for small online shops in Europe.",
		"Key Responsibilities",
		"- Design APIs in Go",
		"Who You’ll Work With",
		"- A small team using Redis",
		"What We’re Looking For:",
		"- 3+ years with PostgreSQL",
		"It’s a plus",
		"- Kafka",
		"Benefits",
		"- Remote-friendly, 25 years of history, Python lunches",
		"About Northwind Cloud",
		"- Founded by Java developers in Iași",
		"Equal Opportunity",
		"We welcome everyone; Docker is our mascot.",
	}, "\n")
	checkResult(t, "vocabulary", extract.Read(ad, extract.Hints{}), expected{
		title: "Backend Engineer", company: "Northwind Cloud", level: "mid", years: years(3),
		must:             []string{"Go", "Redis", "PostgreSQL"},
		nice:             []string{"Kafka"},
		responsibilities: []string{"Design APIs in Go", "Who You'll Work With", "A small team using Redis"},
		about:            "Northwind Cloud builds payment infrastructure for small online shops in Europe.",
	})
	romanian := "Inginer DevOps\nDespre noi\nSuntem în Cluj, lucrăm hibrid, folosim Java de 20 de ani.\n" +
		"Profilul candidatului\n- Linux și Ansible\nReprezintă un avantaj\n- Terraform\nAtribuții\n- Administrezi servere\nBeneficii\n- Abonament Python"
	checkResult(t, "romanian vocabulary", extract.Read(romanian, extract.Hints{}), expected{
		title: "Inginer DevOps", must: []string{"Linux", "Ansible"}, nice: []string{"Terraform"},
		responsibilities: []string{"Administrezi servere"},
	})
	for _, heading := range []string{"Minimum qualifications", "Qualifications:", "What we look for", "What you bring",
		"What you’ll bring", "What you'll need", "What you need", "What we require", "What makes you a great fit", "Who you are", "About you", "You have", "Skills and experience",
		"Must-have", "Cerințe", "Ce căutăm", "Ce aduci"} {
		if got := extract.Read("T\n"+heading+"\n- Docker\nPreferred qualifications\n- Helm", extract.Hints{}); !slices.Equal(got.MustHave, []string{"Docker"}) ||
			!slices.Equal(got.NiceToHave, []string{"Helm"}) {
			t.Errorf("%q opens the requirements: %+v", heading, got)
		}
	}
	for _, heading := range []string{"Responsibilities", "What you will do", "What you'll be doing", "What you will be doing",
		"Your role", "Your responsibilities", "In this role", "The role", "Ce vei face", "Responsabilități"} {
		if got := extract.Read("T\n"+heading+"\n- Ship Docker images", extract.Hints{}); !slices.Equal(got.Responsibilities, []string{"Ship Docker images"}) {
			t.Errorf("%q opens the responsibilities: %+v", heading, got)
		}
	}
	for _, heading := range []string{"Nice to have", "Bonus points", "Extra points", "Constituie un avantaj", "Ar fi un plus"} {
		if got := extract.Read("T\n"+heading+"\n- Helm", extract.Hints{}); !slices.Equal(got.NiceToHave, []string{"Helm"}) {
			t.Errorf("%q opens the nice to have: %+v", heading, got)
		}
	}
	for _, heading := range []string{"What we offer", "Perks", "Why join us", "About us", "Our culture", "Apply for this job",
		"How to apply", "Ce oferim", "Why you'll love it"} {
		if got := extract.Read("T\nRequirements\n- Go\n"+heading+"\n- Rust, remote, 9 years", extract.Hints{}); !slices.Equal(got.MustHave, []string{"Go"}) ||
			got.WorkMode != "" || got.YearsRequired != nil {
			t.Errorf("%q is ignored: %+v", heading, got)
		}
	}
}
