package unit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"profile/internal/apperrors"
	"profile/internal/identity"
	"profile/internal/models"
	"profile/internal/routes"
	"profile/internal/services"
	"profile/tests/testutil"
)

const mePath = "/api/v1/profiles/me"

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

// memoryProfiles is an in-memory ProfileRepository keyed by tenant and subject.
type memoryProfiles struct {
	mu       sync.Mutex
	profiles map[string]*models.Profile
	fail     error
}

func newMemoryProfiles() *memoryProfiles {
	return &memoryProfiles{profiles: map[string]*models.Profile{}}
}

func key(ident identity.IdentityContext) string { return ident.TenantID.String() + "/" + ident.Subject }

func (m *memoryProfiles) own(ident identity.IdentityContext) *models.Profile {
	if p, ok := m.profiles[key(ident)]; ok {
		return p
	}
	now := time.Now()
	p := &models.Profile{ID: uuid.New(), WorkModes: []string{}, TargetRoles: []string{}, TargetSeniority: []string{},
		EmploymentTypes: []string{}, AdditionalInfo: []string{}, Skills: []string{}, Experience: []models.Experience{},
		Projects: []models.Project{}, Education: []models.Education{}, Certifications: []models.Certification{},
		Languages: []models.Language{}, CreatedAt: now, UpdatedAt: now}
	m.profiles[key(ident)] = p
	return p
}

func (m *memoryProfiles) Ensure(_ context.Context, ident identity.IdentityContext) (*models.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	copied := *m.own(ident)
	return &copied, nil
}

func (m *memoryProfiles) Patch(ctx context.Context, ident identity.IdentityContext, patch models.ProfilePatch) (*models.Profile, error) {
	m.mu.Lock()
	p := m.own(ident)
	for _, field := range []struct {
		target *string
		value  *string
	}{{&p.Name, patch.Name}, {&p.Headline, patch.Headline}, {&p.About, patch.About}, {&p.Email, patch.Email},
		{&p.Phone, patch.Phone}, {&p.GitHub, patch.GitHub}, {&p.LinkedIn, patch.LinkedIn}, {&p.City, patch.City},
		{&p.Country, patch.Country}, {&p.WorkAuthorization, patch.WorkAuthorization},
		{&p.WorkAuthorizationRegion, patch.WorkAuthorizationRegion}, {&p.AvailableFrom, patch.AvailableFrom},
		{&p.SalaryCurrency, patch.SalaryCurrency}, {&p.SalaryPeriod, patch.SalaryPeriod}} {
		if field.value != nil {
			*field.target = *field.value
		}
	}
	for _, field := range []struct {
		target *[]string
		value  []string
	}{{&p.WorkModes, patch.WorkModes}, {&p.TargetRoles, patch.TargetRoles}, {&p.TargetSeniority, patch.TargetSeniority},
		{&p.EmploymentTypes, patch.EmploymentTypes}, {&p.AdditionalInfo, patch.AdditionalInfo}} {
		if field.value != nil {
			*field.target = field.value
		}
	}
	if patch.OpenToRelocation != nil {
		p.OpenToRelocation = *patch.OpenToRelocation
	}
	if patch.NoticePeriodWeeks.Set {
		p.NoticePeriodWeeks = patch.NoticePeriodWeeks.Value
	}
	if patch.SalaryMin.Set {
		p.SalaryMin = patch.SalaryMin.Value
	}
	m.mu.Unlock()
	return m.Ensure(ctx, ident)
}

func (m *memoryProfiles) SetSkills(ctx context.Context, ident identity.IdentityContext, skills []string) (*models.Profile, error) {
	m.mu.Lock()
	m.own(ident).Skills = skills
	m.mu.Unlock()
	return m.Ensure(ctx, ident)
}

// collection reaches one child list of a profile.
type collection[T any] struct {
	list    func(*models.Profile) *[]T
	id      func(T) uuid.UUID
	newest  bool
	compose func(uuid.UUID, T) T
}

func addTo[T any](m *memoryProfiles, ident identity.IdentityContext, c collection[T], item T) (*T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	item = c.compose(uuid.New(), item)
	list := c.list(m.own(ident))
	if c.newest {
		*list = append([]T{item}, *list...)
	} else {
		*list = append(*list, item)
	}
	return &item, nil
}

func replaceIn[T any](m *memoryProfiles, ident identity.IdentityContext, c collection[T], id uuid.UUID, item T) (*T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return nil, m.fail
	}
	p, ok := m.profiles[key(ident)]
	if !ok {
		return nil, nil
	}
	list := c.list(p)
	for i := range *list {
		if c.id((*list)[i]) == id {
			(*list)[i] = c.compose(id, item)
			updated := (*list)[i]
			return &updated, nil
		}
	}
	return nil, nil
}

func deleteFrom[T any](m *memoryProfiles, ident identity.IdentityContext, c collection[T], id uuid.UUID) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fail != nil {
		return false, m.fail
	}
	p, ok := m.profiles[key(ident)]
	if !ok {
		return false, nil
	}
	list := c.list(p)
	before := len(*list)
	*list = slices.DeleteFunc(*list, func(item T) bool { return c.id(item) == id })
	return len(*list) < before, nil
}

var (
	experienceList = collection[models.Experience]{
		list: func(p *models.Profile) *[]models.Experience { return &p.Experience },
		id:   func(e models.Experience) uuid.UUID { return e.ID }, newest: true,
		compose: func(id uuid.UUID, e models.Experience) models.Experience { e.ID = id; return e },
	}
	projectList = collection[models.Project]{
		list: func(p *models.Profile) *[]models.Project { return &p.Projects },
		id:   func(e models.Project) uuid.UUID { return e.ID }, newest: true,
		compose: func(id uuid.UUID, e models.Project) models.Project { e.ID = id; return e },
	}
	educationList = collection[models.Education]{
		list:    func(p *models.Profile) *[]models.Education { return &p.Education },
		id:      func(e models.Education) uuid.UUID { return e.ID },
		compose: func(id uuid.UUID, e models.Education) models.Education { e.ID = id; return e },
	}
	certificationList = collection[models.Certification]{
		list:    func(p *models.Profile) *[]models.Certification { return &p.Certifications },
		id:      func(e models.Certification) uuid.UUID { return e.ID },
		compose: func(id uuid.UUID, e models.Certification) models.Certification { e.ID = id; return e },
	}
	languageList = collection[models.Language]{
		list:    func(p *models.Profile) *[]models.Language { return &p.Languages },
		id:      func(e models.Language) uuid.UUID { return e.ID },
		compose: func(id uuid.UUID, e models.Language) models.Language { e.ID = id; return e },
	}
)

func (m *memoryProfiles) AddExperience(_ context.Context, ident identity.IdentityContext, d models.ExperienceInput) (*models.Experience, error) {
	return addTo(m, ident, experienceList, models.Experience{ExperienceInput: d})
}

func (m *memoryProfiles) UpdateExperience(_ context.Context, ident identity.IdentityContext, id uuid.UUID, d models.ExperienceInput) (*models.Experience, error) {
	return replaceIn(m, ident, experienceList, id, models.Experience{ExperienceInput: d})
}

func (m *memoryProfiles) DeleteExperience(_ context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return deleteFrom(m, ident, experienceList, id)
}

func (m *memoryProfiles) AddProject(_ context.Context, ident identity.IdentityContext, d models.ProjectInput) (*models.Project, error) {
	return addTo(m, ident, projectList, models.Project{ProjectInput: d})
}

func (m *memoryProfiles) UpdateProject(_ context.Context, ident identity.IdentityContext, id uuid.UUID, d models.ProjectInput) (*models.Project, error) {
	return replaceIn(m, ident, projectList, id, models.Project{ProjectInput: d})
}

func (m *memoryProfiles) DeleteProject(_ context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return deleteFrom(m, ident, projectList, id)
}

func (m *memoryProfiles) AddEducation(_ context.Context, ident identity.IdentityContext, d models.EducationInput) (*models.Education, error) {
	return addTo(m, ident, educationList, models.Education{EducationInput: d})
}

func (m *memoryProfiles) UpdateEducation(_ context.Context, ident identity.IdentityContext, id uuid.UUID, d models.EducationInput) (*models.Education, error) {
	return replaceIn(m, ident, educationList, id, models.Education{EducationInput: d})
}

func (m *memoryProfiles) DeleteEducation(_ context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return deleteFrom(m, ident, educationList, id)
}

func (m *memoryProfiles) AddCertification(_ context.Context, ident identity.IdentityContext, d models.CertificationInput) (*models.Certification, error) {
	return addTo(m, ident, certificationList, models.Certification{CertificationInput: d})
}

func (m *memoryProfiles) UpdateCertification(_ context.Context, ident identity.IdentityContext, id uuid.UUID, d models.CertificationInput) (*models.Certification, error) {
	return replaceIn(m, ident, certificationList, id, models.Certification{CertificationInput: d})
}

func (m *memoryProfiles) DeleteCertification(_ context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return deleteFrom(m, ident, certificationList, id)
}

func (m *memoryProfiles) AddLanguage(_ context.Context, ident identity.IdentityContext, d models.LanguageInput) (*models.Language, error) {
	return addTo(m, ident, languageList, models.Language{LanguageInput: d})
}

func (m *memoryProfiles) DeleteLanguage(_ context.Context, ident identity.IdentityContext, id uuid.UUID) (bool, error) {
	return deleteFrom(m, ident, languageList, id)
}

// profilesHandler serves the routes as ident, or without an identity when
// ident is nil.
func profilesHandler(repo *memoryProfiles, ident *identity.IdentityContext) http.Handler {
	router := chi.NewRouter()
	routes.Register(router, routes.Deps{DB: okPinger{}, Profiles: services.NewProfileService(repo)})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ident != nil {
			r = r.WithContext(identity.WithContext(r.Context(), *ident))
		}
		router.ServeHTTP(w, r)
	})
}

func devHandler(repo *memoryProfiles) http.Handler {
	dev := identity.Dev()
	return profilesHandler(repo, &dev)
}

func errorField(body map[string]any) string {
	envelope, _ := body["error"].(map[string]any)
	details, _ := envelope["context"].(map[string]any)
	field, _ := details["field"].(string)
	return field
}

func strs(values ...string) []string { return values }

func repeat(n int) string { return strings.Repeat("x", n) }

func ptr[T any](value T) *T { return &value }

// field asserts err is a 422 naming field.
func field(t *testing.T, label string, err error, want string) {
	t.Helper()
	if appErr := apperrors.As(err); appErr.Code != apperrors.ValidationFailed || appErr.Context["field"] != want {
		t.Fatalf("%s: want a 422 on %q, got %v (%v)", label, want, err, apperrors.As(err).Context)
	}
}

func nullable(raw string) models.Nullable {
	var n models.Nullable
	if err := json.Unmarshal([]byte(raw), &n); err != nil {
		panic(err)
	}
	return n
}

func TestCleanSkills(t *testing.T) {
	got, err := services.CleanSkills(strs("  Go ", "", "   ", "go", "Vue", "GO", "vue ", "Rust"))
	if err != nil || !slices.Equal(got, strs("Go", "Vue", "Rust")) {
		t.Fatalf("cleaned skills: %q %v", got, err)
	}
	if got, err := services.CleanSkills(nil); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("nil skills become an empty list: %#v %v", got, err)
	}
	fifty := make([]string, 0, 51)
	for i := range 50 {
		fifty = append(fifty, fmt.Sprintf("skill-%d", i))
	}
	if got, err := services.CleanSkills(append(slices.Clone(fifty), "Skill-0", " ")); err != nil || len(got) != 50 {
		t.Fatalf("50 skills after cleaning are allowed: %d %v", len(got), err)
	}
	_, err = services.CleanSkills(append(fifty, "one-more"))
	field(t, "51 skills", err, "skills")
	if _, err := services.CleanSkills(strs(repeat(60), "ü"+repeat(59))); err != nil {
		t.Fatalf("60 characters are allowed: %v", err)
	}
	_, err = services.CleanSkills(strs(repeat(61)))
	field(t, "61 characters", err, "skills")
}

func TestCleanPatchTextFields(t *testing.T) {
	cleaned, err := services.CleanPatch(models.ProfilePatch{Name: ptr("  Ana Pop "), Headline: ptr("  Engineer  "),
		Email: ptr(" a@b.co "), City: ptr(" Cluj "), SalaryCurrency: ptr(" eur ")})
	if err != nil || *cleaned.Name != "Ana Pop" || *cleaned.Headline != "Engineer" || *cleaned.Email != "a@b.co" ||
		*cleaned.City != "Cluj" || *cleaned.SalaryCurrency != "EUR" || cleaned.About != nil {
		t.Fatalf("trimmed patch: %+v %v", cleaned, err)
	}
	if cleaned, err := services.CleanPatch(models.ProfilePatch{Email: ptr(""), SalaryCurrency: ptr("")}); err != nil ||
		*cleaned.Email != "" || *cleaned.SalaryCurrency != "" {
		t.Fatalf("empty values clear: %v", err)
	}
	limits := []struct {
		field string
		max   int
		set   func(*models.ProfilePatch, *string)
	}{
		{"name", 120, func(p *models.ProfilePatch, v *string) { p.Name = v }},
		{"headline", 120, func(p *models.ProfilePatch, v *string) { p.Headline = v }},
		{"about", 280, func(p *models.ProfilePatch, v *string) { p.About = v }},
		{"phone", 200, func(p *models.ProfilePatch, v *string) { p.Phone = v }},
		{"github", 200, func(p *models.ProfilePatch, v *string) { p.GitHub = v }},
		{"linkedin", 200, func(p *models.ProfilePatch, v *string) { p.LinkedIn = v }},
		{"city", 60, func(p *models.ProfilePatch, v *string) { p.City = v }},
		{"country", 60, func(p *models.ProfilePatch, v *string) { p.Country = v }},
		{"work_authorization_region", 60, func(p *models.ProfilePatch, v *string) { p.WorkAuthorizationRegion = v }},
	}
	for _, limit := range limits {
		var ok, tooLong models.ProfilePatch
		limit.set(&ok, ptr(repeat(limit.max)))
		limit.set(&tooLong, ptr(repeat(limit.max+1)))
		if _, err := services.CleanPatch(ok); err != nil {
			t.Fatalf("%s at %d: %v", limit.field, limit.max, err)
		}
		_, err := services.CleanPatch(tooLong)
		field(t, limit.field, err, limit.field)
	}
	for _, email := range []string{"ana@example.com", "first.last+tag@sub.example.org"} {
		if _, err := services.CleanPatch(models.ProfilePatch{Email: ptr(email)}); err != nil {
			t.Fatalf("valid email %q: %v", email, err)
		}
	}
	for _, email := range []string{"plain", "a@b", "@example.com", "a@", "Ana <ana@example.com>", "a b@c.com", "a@.com"} {
		_, err := services.CleanPatch(models.ProfilePatch{Email: ptr(email)})
		field(t, "email "+email, err, "email")
	}
	for _, currency := range []string{"EU", "EURO", "E1R", "€UR"} {
		_, err := services.CleanPatch(models.ProfilePatch{SalaryCurrency: ptr(currency)})
		field(t, "currency "+currency, err, "salary_currency")
	}
}

func TestCleanPatchEnumsListsAndNumbers(t *testing.T) {
	cleaned, err := services.CleanPatch(models.ProfilePatch{
		WorkModes:         strs(" remote", "hybrid", "remote"),
		TargetSeniority:   strs("senior", "lead", "senior"),
		EmploymentTypes:   strs("contract", "full_time"),
		TargetRoles:       strs(" Backend engineer ", "backend ENGINEER", "", "SRE"),
		AdditionalInfo:    strs(" Has a car ", "", "Has a car"),
		WorkAuthorization: ptr(" work_permit "),
		SalaryPeriod:      ptr("month"),
		AvailableFrom:     ptr(" 2027-03 "),
		NoticePeriodWeeks: nullable("4"),
		SalaryMin:         nullable("null"),
	})
	if err != nil || !slices.Equal(cleaned.WorkModes, strs("remote", "hybrid")) ||
		!slices.Equal(cleaned.TargetSeniority, strs("senior", "lead")) ||
		!slices.Equal(cleaned.TargetRoles, strs("Backend engineer", "SRE")) ||
		!slices.Equal(cleaned.AdditionalInfo, strs("Has a car", "Has a car")) ||
		*cleaned.WorkAuthorization != "work_permit" || *cleaned.AvailableFrom != "2027-03" ||
		*cleaned.NoticePeriodWeeks.Value != 4 || !cleaned.SalaryMin.Set || cleaned.SalaryMin.Value != nil {
		t.Fatalf("cleaned patch: %+v %v", cleaned, err)
	}
	if cleaned, err := services.CleanPatch(models.ProfilePatch{WorkModes: []string{}}); err != nil || cleaned.WorkModes == nil || cleaned.Empty() {
		t.Fatalf("an empty array clears and is a change: %+v %v", cleaned, err)
	}
	if cleaned, err := services.CleanPatch(models.ProfilePatch{}); err != nil || !cleaned.Empty() || cleaned.WorkModes != nil {
		t.Fatalf("absent arrays stay absent: %+v %v", cleaned, err)
	}
	for _, edge := range []models.ProfilePatch{
		{NoticePeriodWeeks: nullable("0")}, {NoticePeriodWeeks: nullable("52")},
		{SalaryMin: nullable("100000000")}, {AvailableFrom: ptr("1950-01")}, {AvailableFrom: ptr("2100-12")},
		{WorkAuthorization: ptr("")}, {SalaryPeriod: ptr("")}, {OpenToRelocation: ptr(true)},
	} {
		if _, err := services.CleanPatch(edge); err != nil {
			t.Fatalf("edge %+v: %v", edge, err)
		}
	}
	tenRoles, elevenInfo := make([]string, 11), make([]string, 11)
	for i := range 11 {
		tenRoles[i], elevenInfo[i] = fmt.Sprintf("role %d", i), "fact"
	}
	cases := []struct {
		patch models.ProfilePatch
		field string
	}{
		{models.ProfilePatch{WorkModes: strs("office")}, "work_modes"},
		{models.ProfilePatch{WorkModes: strs("")}, "work_modes"},
		{models.ProfilePatch{TargetSeniority: strs("staff")}, "target_seniority"},
		{models.ProfilePatch{EmploymentTypes: strs("Full_time")}, "employment_types"},
		{models.ProfilePatch{TargetRoles: tenRoles}, "target_roles"},
		{models.ProfilePatch{TargetRoles: strs(repeat(121))}, "target_roles"},
		{models.ProfilePatch{AdditionalInfo: elevenInfo}, "additional_info"},
		{models.ProfilePatch{AdditionalInfo: strs(repeat(301))}, "additional_info"},
		{models.ProfilePatch{WorkAuthorization: ptr("visa")}, "work_authorization"},
		{models.ProfilePatch{SalaryPeriod: ptr("week")}, "salary_period"},
		{models.ProfilePatch{NoticePeriodWeeks: nullable("53")}, "notice_period_weeks"},
		{models.ProfilePatch{NoticePeriodWeeks: nullable("-1")}, "notice_period_weeks"},
		{models.ProfilePatch{NoticePeriodWeeks: nullable("2.5")}, "notice_period_weeks"},
		{models.ProfilePatch{NoticePeriodWeeks: nullable(`"4"`)}, "notice_period_weeks"},
		{models.ProfilePatch{SalaryMin: nullable("100000001")}, "salary_min"},
		{models.ProfilePatch{AvailableFrom: ptr("2027-13")}, "available_from"},
		{models.ProfilePatch{AvailableFrom: ptr("2027-3")}, "available_from"},
		{models.ProfilePatch{AvailableFrom: ptr("1949-12")}, "available_from"},
		{models.ProfilePatch{AvailableFrom: ptr("2027/03")}, "available_from"},
	}
	for _, tc := range cases {
		_, err := services.CleanPatch(tc.patch)
		field(t, fmt.Sprintf("%+v", tc.patch), err, tc.field)
	}
}

func validBullets(n int) []models.Bullet {
	bullets := make([]models.Bullet, n)
	for i := range bullets {
		bullets[i] = models.Bullet{Text: fmt.Sprintf("did %d", i)}
	}
	return bullets
}

func TestCleanExperienceAndProject(t *testing.T) {
	cleaned, err := services.CleanExperience(models.ExperienceInput{
		Role: "  Engineer ", Company: " Acme ", Location: " Remote ", EmploymentType: " full_time ", Industry: " Fintech ",
		TeamSize: ptr(8), PeopleManaged: ptr(0), Start: " 2019-04 ", End: "",
		Tools: strs("Go", " go ", "", "Postgres"),
		Bullets: []models.Bullet{
			{Text: " Shipped billing ", Result: " -30% churn ", Skills: strs("Go", "go", " ")},
			{Text: "  ", Result: "dropped"},
			{Text: "Shipped billing"},
		},
	})
	if err != nil || cleaned.Role != "Engineer" || cleaned.Company != "Acme" || cleaned.EmploymentType != "full_time" ||
		cleaned.Industry != "Fintech" || cleaned.Start != "2019-04" || !slices.Equal(cleaned.Tools, strs("Go", "Postgres")) ||
		len(cleaned.Bullets) != 2 || cleaned.Bullets[0].Text != "Shipped billing" || cleaned.Bullets[0].Result != "-30% churn" ||
		!slices.Equal(cleaned.Bullets[0].Skills, strs("Go")) || cleaned.Bullets[1].Skills == nil {
		t.Fatalf("cleaned experience: %+v %v", cleaned, err)
	}
	if cleaned, _ := services.CleanExperience(models.ExperienceInput{Role: "r"}); cleaned.Tools == nil || cleaned.Bullets == nil {
		t.Fatal("missing lists become empty lists")
	}
	tools := make([]string, 31)
	for i := range tools {
		tools[i] = fmt.Sprintf("tool-%d", i)
	}
	elevenSkills := make([]string, 11)
	for i := range elevenSkills {
		elevenSkills[i] = fmt.Sprintf("s%d", i)
	}
	cases := []struct {
		name  string
		input models.ExperienceInput
		field string
	}{
		{"missing role", models.ExperienceInput{}, "role"},
		{"blank role", models.ExperienceInput{Role: "   "}, "role"},
		{"long role", models.ExperienceInput{Role: repeat(121)}, "role"},
		{"long company", models.ExperienceInput{Role: "r", Company: repeat(121)}, "company"},
		{"long location", models.ExperienceInput{Role: "r", Location: repeat(121)}, "location"},
		{"employment type", models.ExperienceInput{Role: "r", EmploymentType: "gig"}, "employment_type"},
		{"long industry", models.ExperienceInput{Role: "r", Industry: repeat(61)}, "industry"},
		{"team size", models.ExperienceInput{Role: "r", TeamSize: ptr(100_001)}, "team_size"},
		{"people managed", models.ExperienceInput{Role: "r", PeopleManaged: ptr(-1)}, "people_managed"},
		{"bad start", models.ExperienceInput{Role: "r", Start: "2020"}, "start"},
		{"bad end", models.ExperienceInput{Role: "r", End: "now"}, "end"},
		{"far end", models.ExperienceInput{Role: "r", End: "2101-01"}, "end"},
		{"end before start", models.ExperienceInput{Role: "r", Start: "2020-05", End: "2020-04"}, "end"},
		{"too many tools", models.ExperienceInput{Role: "r", Tools: tools}, "tools"},
		{"long tool", models.ExperienceInput{Role: "r", Tools: strs(repeat(61))}, "tools"},
		{"too many bullets", models.ExperienceInput{Role: "r", Bullets: validBullets(11)}, "bullets"},
		{"long bullet", models.ExperienceInput{Role: "r", Bullets: []models.Bullet{{Text: repeat(301)}}}, "bullets.text"},
		{"long result", models.ExperienceInput{Role: "r", Bullets: []models.Bullet{{Text: "t", Result: repeat(201)}}}, "bullets.result"},
		{"many bullet skills", models.ExperienceInput{Role: "r", Bullets: []models.Bullet{{Text: "t", Skills: elevenSkills}}}, "bullets.skills"},
	}
	for _, tc := range cases {
		_, err := services.CleanExperience(tc.input)
		field(t, tc.name, err, tc.field)
	}
	for _, input := range []models.ExperienceInput{
		{Role: repeat(120), Start: "2020-05", End: "2020-05", TeamSize: ptr(100_000)},
		{Role: "r", End: "2020-01"},
		{Role: "r", Tools: tools[:30], Bullets: append(validBullets(10), models.Bullet{Text: " ", Result: repeat(300)})},
	} {
		if _, err := services.CleanExperience(input); err != nil {
			t.Fatalf("valid experience %+v: %v", input, err)
		}
	}

	project, err := services.CleanProject(models.ProjectInput{Name: " Hoply ", Role: " Author ", URL: " https://hoply.dev ",
		Start: "2026-01", Tools: strs("Go", "GO"), Bullets: []models.Bullet{{Text: " Built it "}}})
	if err != nil || project.Name != "Hoply" || project.Role != "Author" || project.URL != "https://hoply.dev" ||
		!slices.Equal(project.Tools, strs("Go")) || project.Bullets[0].Text != "Built it" {
		t.Fatalf("cleaned project: %+v %v", project, err)
	}
	for _, tc := range []struct {
		input models.ProjectInput
		field string
	}{
		{models.ProjectInput{}, "name"},
		{models.ProjectInput{Name: repeat(121)}, "name"},
		{models.ProjectInput{Name: "p", Role: repeat(121)}, "role"},
		{models.ProjectInput{Name: "p", URL: repeat(201)}, "url"},
		{models.ProjectInput{Name: "p", Start: "2026-00"}, "start"},
		{models.ProjectInput{Name: "p", Start: "2026-02", End: "2026-01"}, "end"},
		{models.ProjectInput{Name: "p", Tools: tools}, "tools"},
		{models.ProjectInput{Name: "p", Bullets: validBullets(11)}, "bullets"},
	} {
		_, err := services.CleanProject(tc.input)
		field(t, fmt.Sprintf("project %+v", tc.input), err, tc.field)
	}
}

func TestCleanEducationCertificationAndLanguage(t *testing.T) {
	education, err := services.CleanEducation(models.EducationInput{Level: " bachelor ", Field: " CS ", Degree: " BSc ",
		School: " Uni ", StartYear: " 2010 ", EndYear: "2014"})
	if err != nil || education != (models.EducationInput{Level: "bachelor", Field: "CS", Degree: "BSc", School: "Uni", StartYear: "2010", EndYear: "2014"}) {
		t.Fatalf("education: %+v %v", education, err)
	}
	if _, err := services.CleanEducation(models.EducationInput{Level: "other"}); err != nil {
		t.Fatalf("degree is optional: %v", err)
	}
	for _, tc := range []struct {
		input models.EducationInput
		field string
	}{
		{models.EducationInput{}, "level"},
		{models.EducationInput{Level: "Bachelor"}, "level"},
		{models.EducationInput{Level: "master", Field: repeat(121)}, "field"},
		{models.EducationInput{Level: "master", Degree: repeat(121)}, "degree"},
		{models.EducationInput{Level: "master", School: repeat(121)}, "school"},
		{models.EducationInput{Level: "master", StartYear: "1949"}, "start_year"},
		{models.EducationInput{Level: "master", StartYear: "２０２０"}, "start_year"},
		{models.EducationInput{Level: "master", EndYear: "20a0"}, "end_year"},
		{models.EducationInput{Level: "master", StartYear: "2020", EndYear: "2019"}, "end_year"},
	} {
		_, err := services.CleanEducation(tc.input)
		field(t, fmt.Sprintf("education %+v", tc.input), err, tc.field)
	}

	certification, err := services.CleanCertification(models.CertificationInput{Name: " CKA ", Issuer: " CNCF ",
		Issued: "2025-06", Expires: "2028-06", CredentialID: " LF-1 ", URL: " https://cncf.io "})
	if err != nil || certification.Name != "CKA" || certification.Issuer != "CNCF" || certification.CredentialID != "LF-1" ||
		certification.URL != "https://cncf.io" {
		t.Fatalf("certification: %+v %v", certification, err)
	}
	for _, tc := range []struct {
		input models.CertificationInput
		field string
	}{
		{models.CertificationInput{}, "name"},
		{models.CertificationInput{Name: "c", Issuer: repeat(121)}, "issuer"},
		{models.CertificationInput{Name: "c", Issued: "June"}, "issued"},
		{models.CertificationInput{Name: "c", Expires: "2028"}, "expires"},
		{models.CertificationInput{Name: "c", Issued: "2025-06", Expires: "2025-05"}, "expires"},
		{models.CertificationInput{Name: "c", CredentialID: repeat(121)}, "credential_id"},
		{models.CertificationInput{Name: "c", URL: repeat(201)}, "url"},
	} {
		_, err := services.CleanCertification(tc.input)
		field(t, fmt.Sprintf("certification %+v", tc.input), err, tc.field)
	}

	for _, level := range services.Levels {
		if _, err := services.CleanLanguage(models.LanguageInput{Name: "English", Level: level}); err != nil {
			t.Fatalf("level %s: %v", level, err)
		}
	}
	for _, input := range []models.LanguageInput{{Level: "B2"}, {Name: repeat(121), Level: "B2"}} {
		_, err := services.CleanLanguage(input)
		field(t, "language name", err, "name")
	}
	for _, level := range []string{"", "b2", "native", "C3", "Fluent"} {
		_, err := services.CleanLanguage(models.LanguageInput{Name: "English", Level: level})
		field(t, "language level "+level, err, "level")
	}
}

func TestProfilePatchThroughRoutes(t *testing.T) {
	handler := devHandler(newMemoryProfiles())
	status, profile := testutil.Do(t, handler, http.MethodGet, mePath, nil)
	if status != http.StatusOK || profile["name"] != "" || profile["notice_period_weeks"] != nil ||
		profile["open_to_relocation"] != false || fmt.Sprint(profile["work_modes"]) != "[]" ||
		fmt.Sprint(profile["projects"]) != "[]" || fmt.Sprint(profile["certifications"]) != "[]" || profile["id"] == nil {
		t.Fatalf("empty profile: %d %v", status, profile)
	}
	status, patched := testutil.Do(t, handler, http.MethodPatch, mePath, map[string]any{
		"name": " Ana Pop ", "headline": " Go developer ", "email": "ana@example.com", "work_modes": strs("remote", "remote"),
		"open_to_relocation": true, "notice_period_weeks": 4, "salary_min": 5000, "salary_currency": "ron",
		"target_roles": strs("Backend", "backend"),
	})
	if status != http.StatusOK || patched["name"] != "Ana Pop" || patched["headline"] != "Go developer" ||
		fmt.Sprint(patched["work_modes"]) != "[remote]" || patched["open_to_relocation"] != true ||
		patched["notice_period_weeks"] != float64(4) || patched["salary_currency"] != "RON" ||
		fmt.Sprint(patched["target_roles"]) != "[Backend]" {
		t.Fatalf("patch: %d %v", status, patched)
	}
	status, kept := testutil.Do(t, handler, http.MethodPatch, mePath, map[string]any{"city": "Cluj"})
	if status != http.StatusOK || kept["notice_period_weeks"] != float64(4) || kept["salary_min"] != float64(5000) || kept["name"] != "Ana Pop" {
		t.Fatalf("absent fields stay: %d %v", status, kept)
	}
	status, cleared := testutil.Do(t, handler, http.MethodPatch, mePath, `{"notice_period_weeks": null, "work_modes": []}`)
	if status != http.StatusOK || cleared["notice_period_weeks"] != nil || cleared["salary_min"] != float64(5000) ||
		fmt.Sprint(cleared["work_modes"]) != "[]" {
		t.Fatalf("null clears only the sent field: %d %v", status, cleared)
	}
	if status, same := testutil.Do(t, handler, http.MethodPatch, mePath, map[string]any{}); status != http.StatusOK || same["city"] != "Cluj" {
		t.Fatalf("empty patch: %d %v", status, same)
	}
	status, skilled := testutil.Do(t, handler, http.MethodPut, mePath+"/skills", map[string]any{"skills": strs("Vue", "go", "Go", " ")})
	if status != http.StatusOK || fmt.Sprint(skilled["skills"]) != "[Vue go]" {
		t.Fatalf("skills: %d %v", status, skilled)
	}
}

// child names a collection's path and a valid body for it.
type child struct {
	path   string
	entity string
	valid  map[string]any
	edited map[string]any
	check  string
}

var children = []child{
	{"/experience", "Experience", map[string]any{"role": "Dev", "start": "2018-01",
		"bullets": []map[string]any{{"text": " Shipped ", "result": "fast", "skills": strs("Go")}}},
		map[string]any{"role": "Lead", "team_size": 4}, "role"},
	{"/projects", "Project", map[string]any{"name": "Hoply", "url": "https://hoply.dev"},
		map[string]any{"name": "Hoply 2"}, "name"},
	{"/education", "Education", map[string]any{"level": "bachelor", "degree": "BSc"},
		map[string]any{"level": "master", "degree": "MSc"}, "degree"},
	{"/certifications", "Certification", map[string]any{"name": "CKA", "issued": "2025-01"},
		map[string]any{"name": "CKAD"}, "name"},
}

func TestCollectionsLifecycleThroughRoutes(t *testing.T) {
	handler := devHandler(newMemoryProfiles())
	for _, c := range children {
		status, first := testutil.Do(t, handler, http.MethodPost, mePath+c.path, c.valid)
		if status != http.StatusCreated || first["id"] == nil {
			t.Fatalf("POST %s: %d %v", c.path, status, first)
		}
		_, second := testutil.Do(t, handler, http.MethodPost, mePath+c.path, c.valid)
		firstPath := mePath + c.path + "/" + first["id"].(string)
		status, edited := testutil.Do(t, handler, http.MethodPut, firstPath, c.edited)
		if status != http.StatusOK || edited["id"] != first["id"] || edited[c.check] != c.edited[c.check] {
			t.Fatalf("PUT %s: %d %v", c.path, status, edited)
		}
		_, profile := testutil.Do(t, handler, http.MethodGet, mePath, nil)
		key := strings.TrimPrefix(c.path, "/")
		list := profile[key].([]any)
		position := 0
		if key == "experience" || key == "projects" {
			position = 1
		}
		if len(list) != 2 || list[position].(map[string]any)["id"] != first["id"] || list[1-position].(map[string]any)["id"] != second["id"] {
			t.Fatalf("%s order: %v", key, list)
		}
		if status, _ := testutil.Do(t, handler, http.MethodDelete, firstPath, nil); status != http.StatusNoContent {
			t.Fatalf("DELETE %s: %d", c.path, status)
		}
		for _, method := range []string{http.MethodPut, http.MethodDelete} {
			for _, path := range []string{firstPath, mePath + c.path + "/not-a-uuid"} {
				status, body := testutil.Do(t, handler, method, path, c.valid)
				if status != http.StatusNotFound || testutil.ErrorCode(body) != "NOT_FOUND" {
					t.Fatalf("%s %s: %d %v", method, path, status, body)
				}
			}
		}
	}
	status, experience := testutil.Do(t, handler, http.MethodPost, mePath+"/experience", children[0].valid)
	bullets := experience["bullets"].([]any)
	if status != http.StatusCreated || bullets[0].(map[string]any)["text"] != "Shipped" || experience["team_size"] != nil {
		t.Fatalf("experience shape: %d %v", status, experience)
	}
	status, language := testutil.Do(t, handler, http.MethodPost, mePath+"/languages", map[string]any{"name": "English", "level": "C1"})
	if status != http.StatusCreated || language["level"] != "C1" {
		t.Fatalf("add language: %d %v", status, language)
	}
	path := mePath + "/languages/" + language["id"].(string)
	if status, _ := testutil.Do(t, handler, http.MethodDelete, path, nil); status != http.StatusNoContent {
		t.Fatalf("delete language: %d", status)
	}
	if status, _ := testutil.Do(t, handler, http.MethodDelete, path, nil); status != http.StatusNotFound {
		t.Fatalf("delete language again: %d", status)
	}
}

func TestProfileIdsBelongToTheirOwner(t *testing.T) {
	repo := newMemoryProfiles()
	owner := devHandler(repo)
	other := identity.New(uuid.New(), uuid.NewString())
	stranger := profilesHandler(repo, &other)
	for _, c := range children {
		_, created := testutil.Do(t, owner, http.MethodPost, mePath+c.path, c.valid)
		path := mePath + c.path + "/" + created["id"].(string)
		if status, _ := testutil.Do(t, stranger, http.MethodPut, path, c.edited); status != http.StatusNotFound {
			t.Fatalf("another profile's %s: %d", c.path, status)
		}
		if status, _ := testutil.Do(t, stranger, http.MethodDelete, path, nil); status != http.StatusNotFound {
			t.Fatalf("another profile's delete of %s: %d", c.path, status)
		}
		testutil.Do(t, stranger, http.MethodGet, mePath, nil)
		if status, _ := testutil.Do(t, stranger, http.MethodPut, path, c.edited); status != http.StatusNotFound {
			t.Fatalf("another existing profile's %s: %d", c.path, status)
		}
	}
	_, profile := testutil.Do(t, owner, http.MethodGet, mePath, nil)
	if len(profile["experience"].([]any)) != 1 || len(profile["certifications"].([]any)) != 1 {
		t.Fatalf("owner's rows are untouched: %v", profile)
	}
}

func TestProfileRequestValidation(t *testing.T) {
	handler := devHandler(newMemoryProfiles())
	id := uuid.NewString()
	cases := []struct {
		method, path string
		payload      any
		field        string
	}{
		{http.MethodPatch, mePath, map[string]any{"name": repeat(121)}, "name"},
		{http.MethodPatch, mePath, map[string]any{"email": "nope"}, "email"},
		{http.MethodPatch, mePath, map[string]any{"notice_period_weeks": 2.5}, "notice_period_weeks"},
		{http.MethodPatch, mePath, map[string]any{"salary_min": "lots"}, "salary_min"},
		{http.MethodPatch, mePath, map[string]any{"work_modes": strs("office")}, "work_modes"},
		{http.MethodPatch, mePath, map[string]any{"open_to_relocation": "yes"}, "open_to_relocation"},
		{http.MethodPatch, mePath, map[string]any{"headline": 5}, "headline"},
		{http.MethodPut, mePath + "/skills", map[string]any{"skills": strs(repeat(61))}, "skills"},
		{http.MethodPut, mePath + "/skills", map[string]any{"skills": "Go"}, "skills"},
		{http.MethodPost, mePath + "/experience", map[string]any{"company": "Acme"}, "role"},
		{http.MethodPost, mePath + "/experience", map[string]any{"role": "r", "team_size": 1.5}, "team_size"},
		{http.MethodPost, mePath + "/experience", map[string]any{"role": "r", "bullets": strs("old shape")}, "bullets"},
		{http.MethodPost, mePath + "/experience", map[string]any{"role": "r", "bullets": []map[string]any{{"text": 1}}}, "bullets.text"},
		{http.MethodPut, mePath + "/experience/" + id, map[string]any{"role": "r", "start": "2021-01", "end": "2020-12"}, "end"},
		{http.MethodPost, mePath + "/projects", map[string]any{"url": "x"}, "name"},
		{http.MethodPut, mePath + "/projects/" + id, map[string]any{"name": "p", "end": "soon"}, "end"},
		{http.MethodPost, mePath + "/education", map[string]any{"school": "Uni"}, "level"},
		{http.MethodPut, mePath + "/education/" + id, map[string]any{"level": "master", "degree": repeat(121)}, "degree"},
		{http.MethodPost, mePath + "/certifications", map[string]any{"name": "c", "expires": "2020-01", "issued": "2021-01"}, "expires"},
		{http.MethodPut, mePath + "/certifications/" + id, map[string]any{}, "name"},
		{http.MethodPost, mePath + "/languages", map[string]any{"name": "English", "level": "fluent"}, "level"},
		{http.MethodPatch, mePath, "{not json", ""},
		{http.MethodPut, mePath + "/education/" + id, "{", ""},
		{http.MethodPost, mePath + "/languages", "", ""},
	}
	for _, tc := range cases {
		status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload)
		if status != http.StatusUnprocessableEntity || testutil.ErrorCode(body) != "VALIDATION_FAILED" || errorField(body) != tc.field {
			t.Fatalf("%s %s %v: %d %v", tc.method, tc.path, tc.payload, status, body)
		}
	}
	if status, _ := testutil.Do(t, handler, http.MethodGet, "/api/v1/profiles", nil); status != http.StatusNotFound {
		t.Fatalf("the collection is not served: %d", status)
	}
}

// everyRoute lists one request per route with a valid body.
func everyRoute() []struct {
	method, path string
	payload      any
} {
	id := uuid.NewString()
	routes := []struct {
		method, path string
		payload      any
	}{
		{http.MethodGet, mePath, nil},
		{http.MethodPatch, mePath, map[string]any{"headline": "h"}},
		{http.MethodPut, mePath + "/skills", map[string]any{"skills": strs("Go")}},
		{http.MethodPost, mePath + "/languages", map[string]any{"name": "n", "level": "A1"}},
		{http.MethodDelete, mePath + "/languages/" + id, nil},
	}
	for _, c := range children {
		routes = append(routes,
			struct {
				method, path string
				payload      any
			}{http.MethodPost, mePath + c.path, c.valid},
			struct {
				method, path string
				payload      any
			}{http.MethodPut, mePath + c.path + "/" + id, c.valid},
			struct {
				method, path string
				payload      any
			}{http.MethodDelete, mePath + c.path + "/" + id, nil})
	}
	return routes
}

func TestProfileRequiresAnIdentity(t *testing.T) {
	handler := profilesHandler(newMemoryProfiles(), nil)
	for _, tc := range everyRoute() {
		if status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload); status != http.StatusUnauthorized || testutil.ErrorCode(body) != "AUTH_REQUIRED" {
			t.Fatalf("%s %s without identity: %d %v", tc.method, tc.path, status, body)
		}
	}
}

func TestProfileRepositoryFailuresSurface(t *testing.T) {
	repo := newMemoryProfiles()
	repo.fail = apperrors.New(apperrors.DatabaseUnavailable, "down")
	handler := devHandler(repo)
	for _, tc := range everyRoute() {
		if status, body := testutil.Do(t, handler, tc.method, tc.path, tc.payload); status != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: %d %v", tc.method, tc.path, status, body)
		}
	}
}
