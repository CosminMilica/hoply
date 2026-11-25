package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"

	"profile/internal/data/repositories"
	"profile/internal/identity"
	"profile/internal/models"
	"profile/tests/testutil"
)

const mePath = "/api/v1/profiles/me"

func ptr[T any](value T) *T { return &value }

func TestProfilePatchAgainstPostgres(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgProfileRepository(pool).WithScope(bindTenant(pool))
	owner := identity.New(uuid.New(), uuid.NewString())

	created, err := repo.Ensure(ctx, owner)
	if err != nil || created.Name != "" || created.NoticePeriodWeeks != nil || created.SalaryMin != nil || created.OpenToRelocation ||
		created.WorkModes == nil || created.TargetRoles == nil || created.AdditionalInfo == nil || created.Skills == nil ||
		created.Experience == nil || created.Projects == nil || created.Education == nil || created.Certifications == nil ||
		created.Languages == nil {
		t.Fatalf("empty profile: %+v %v", created, err)
	}
	if again, err := repo.Ensure(ctx, owner); err != nil || again.ID != created.ID {
		t.Fatalf("ensure is idempotent: %v %v", again, err)
	}

	patched, err := repo.Patch(ctx, owner, models.ProfilePatch{
		Name: ptr("Ana Pop"), Headline: ptr("Engineer"), About: ptr("About"), Email: ptr("a@b.co"), Phone: ptr("+40"),
		GitHub: ptr("gh"), LinkedIn: ptr("li"), City: ptr("Cluj"), Country: ptr("Romania"),
		WorkModes: []string{"hybrid", "remote"}, OpenToRelocation: ptr(true), WorkAuthorization: ptr("citizen"),
		WorkAuthorizationRegion: ptr("EU"), NoticePeriodWeeks: models.Nullable{Set: true, Value: ptr(4)},
		AvailableFrom: ptr("2027-01"), TargetRoles: []string{"Backend", "SRE"}, TargetSeniority: []string{"senior"},
		EmploymentTypes: []string{"contract", "full_time"}, SalaryMin: models.Nullable{Set: true, Value: ptr(90000)},
		SalaryCurrency: ptr("EUR"), SalaryPeriod: ptr("year"), AdditionalInfo: []string{"Driving licence"},
	})
	if err != nil || patched.Name != "Ana Pop" || patched.About != "About" || patched.Country != "Romania" ||
		!slices.Equal(patched.WorkModes, []string{"hybrid", "remote"}) || !patched.OpenToRelocation ||
		patched.WorkAuthorization != "citizen" || patched.WorkAuthorizationRegion != "EU" || *patched.NoticePeriodWeeks != 4 ||
		patched.AvailableFrom != "2027-01" || !slices.Equal(patched.TargetRoles, []string{"Backend", "SRE"}) ||
		!slices.Equal(patched.EmploymentTypes, []string{"contract", "full_time"}) || *patched.SalaryMin != 90000 ||
		patched.SalaryCurrency != "EUR" || patched.SalaryPeriod != "year" || patched.AdditionalInfo[0] != "Driving licence" ||
		!patched.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("patch: %+v %v", patched, err)
	}
	patched, err = repo.Patch(ctx, owner, models.ProfilePatch{Headline: ptr(""), WorkModes: []string{},
		NoticePeriodWeeks: models.Nullable{Set: true}, OpenToRelocation: ptr(false)})
	if err != nil || patched.Headline != "" || len(patched.WorkModes) != 0 || patched.NoticePeriodWeeks != nil ||
		patched.OpenToRelocation || *patched.SalaryMin != 90000 || patched.Name != "Ana Pop" ||
		!slices.Equal(patched.TargetRoles, []string{"Backend", "SRE"}) {
		t.Fatalf("patch changes only the sent fields: %+v %v", patched, err)
	}

	skilled, err := repo.SetSkills(ctx, owner, []string{"Vue", "Go", "Postgres"})
	if err != nil || !slices.Equal(skilled.Skills, []string{"Vue", "Go", "Postgres"}) {
		t.Fatalf("skills keep their order: %v %v", skilled, err)
	}
	if cleared, err := repo.SetSkills(ctx, owner, nil); err != nil || cleared.Skills == nil || len(cleared.Skills) != 0 {
		t.Fatalf("skills clear: %v %v", cleared, err)
	}
}

func TestProfileCollectionsAgainstPostgres(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgProfileRepository(pool).WithScope(bindTenant(pool))
	owner := identity.New(uuid.New(), uuid.NewString())

	bullet := []models.Bullet{{Text: "b2", Result: "r", Skills: []string{"Go"}}, {Text: "b1", Skills: []string{}}}
	var roles []*models.Experience
	for _, role := range []string{"first", "second", "third"} {
		added, err := repo.AddExperience(ctx, owner, models.ExperienceInput{Role: role, Start: "2020-01", TeamSize: ptr(5),
			EmploymentType: "full_time", Tools: []string{"Go", "SQL"}, Bullets: bullet})
		if err != nil || added.Role != role || *added.TeamSize != 5 || added.PeopleManaged != nil ||
			!slices.Equal(added.Tools, []string{"Go", "SQL"}) || len(added.Bullets) != 2 || added.Bullets[0].Result != "r" {
			t.Fatalf("add experience: %+v %v", added, err)
		}
		roles = append(roles, added)
	}
	updated, err := repo.UpdateExperience(ctx, owner, roles[0].ID, models.ExperienceInput{Role: "first, renamed", End: "2024-06",
		PeopleManaged: ptr(2)})
	if err != nil || updated.Role != "first, renamed" || updated.End != "2024-06" || updated.Start != "" || updated.TeamSize != nil ||
		*updated.PeopleManaged != 2 || updated.Tools == nil || len(updated.Tools) != 0 || updated.Bullets == nil {
		t.Fatalf("update experience replaces it: %+v %v", updated, err)
	}

	var projects []*models.Project
	for _, name := range []string{"alpha", "beta"} {
		added, err := repo.AddProject(ctx, owner, models.ProjectInput{Name: name, URL: "https://x.dev", Bullets: bullet})
		if err != nil || added.Name != name || added.Bullets[0].Skills[0] != "Go" {
			t.Fatalf("add project: %+v %v", added, err)
		}
		projects = append(projects, added)
	}
	if edited, err := repo.UpdateProject(ctx, owner, projects[0].ID, models.ProjectInput{Name: "alpha 2", Start: "2025-01"}); err != nil ||
		edited.Name != "alpha 2" || edited.URL != "" || edited.Start != "2025-01" {
		t.Fatalf("update project: %+v %v", edited, err)
	}

	var degrees []*models.Education
	for _, degree := range []string{"BSc", "MSc"} {
		added, err := repo.AddEducation(ctx, owner, models.EducationInput{Level: "bachelor", Degree: degree, StartYear: "2010"})
		if err != nil {
			t.Fatal(err)
		}
		degrees = append(degrees, added)
	}
	if edited, err := repo.UpdateEducation(ctx, owner, degrees[0].ID, models.EducationInput{Level: "other", Field: "CS", EndYear: "2013"}); err != nil ||
		edited.Degree != "" || edited.Field != "CS" || edited.EndYear != "2013" || edited.StartYear != "" {
		t.Fatalf("update education replaces it: %+v %v", edited, err)
	}

	var certificates []*models.Certification
	for _, name := range []string{"CKA", "CKAD"} {
		added, err := repo.AddCertification(ctx, owner, models.CertificationInput{Name: name, Issuer: "CNCF", Issued: "2025-01"})
		if err != nil {
			t.Fatal(err)
		}
		certificates = append(certificates, added)
	}
	if edited, err := repo.UpdateCertification(ctx, owner, certificates[0].ID, models.CertificationInput{Name: "CKA", Expires: "2028-01",
		CredentialID: "LF-1", URL: "https://cncf.io"}); err != nil || edited.Issuer != "" || edited.CredentialID != "LF-1" {
		t.Fatalf("update certification: %+v %v", edited, err)
	}
	for _, name := range []string{"Romanian", "English", "German"} {
		if _, err := repo.AddLanguage(ctx, owner, models.LanguageInput{Name: name, Level: "B2"}); err != nil {
			t.Fatal(err)
		}
	}

	profile, err := repo.Ensure(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	names := func(n int, name func(int) string) []string {
		out := make([]string, n)
		for i := range n {
			out[i] = name(i)
		}
		return out
	}
	experience := names(len(profile.Experience), func(i int) string { return profile.Experience[i].Role })
	projectNames := names(len(profile.Projects), func(i int) string { return profile.Projects[i].Name })
	education := names(len(profile.Education), func(i int) string { return profile.Education[i].Field + profile.Education[i].Degree })
	certifications := names(len(profile.Certifications), func(i int) string { return profile.Certifications[i].Name })
	languages := names(len(profile.Languages), func(i int) string { return profile.Languages[i].Name })
	if !slices.Equal(experience, []string{"third", "second", "first, renamed"}) || !slices.Equal(projectNames, []string{"beta", "alpha 2"}) ||
		!slices.Equal(education, []string{"CS", "MSc"}) || !slices.Equal(certifications, []string{"CKA", "CKAD"}) ||
		!slices.Equal(languages, []string{"Romanian", "English", "German"}) {
		t.Fatalf("order: %v %v %v %v %v", experience, projectNames, education, certifications, languages)
	}
	if b := profile.Experience[0].Bullets; len(b) != 2 || b[0].Text != "b2" || b[1].Text != "b1" || b[1].Skills == nil {
		t.Fatalf("bullets keep their order: %+v", b)
	}

	for name, remove := range map[string]func() (bool, error){
		"experience":    func() (bool, error) { return repo.DeleteExperience(ctx, owner, roles[1].ID) },
		"project":       func() (bool, error) { return repo.DeleteProject(ctx, owner, projects[1].ID) },
		"education":     func() (bool, error) { return repo.DeleteEducation(ctx, owner, degrees[0].ID) },
		"certification": func() (bool, error) { return repo.DeleteCertification(ctx, owner, certificates[1].ID) },
		"language":      func() (bool, error) { return repo.DeleteLanguage(ctx, owner, profile.Languages[1].ID) },
	} {
		if removed, err := remove(); err != nil || !removed {
			t.Fatalf("delete %s: %v %v", name, removed, err)
		}
		if removed, err := remove(); err != nil || removed {
			t.Fatalf("delete %s twice: %v %v", name, removed, err)
		}
	}
	profile, _ = repo.Ensure(ctx, owner)
	if len(profile.Experience) != 2 || len(profile.Projects) != 1 || len(profile.Education) != 1 || profile.Education[0].Degree != "MSc" ||
		len(profile.Certifications) != 1 || len(profile.Languages) != 2 || profile.Languages[1].Name != "German" {
		t.Fatalf("after deletes: %+v", profile)
	}
}

func TestProfileRepositoryIsOwnerScoped(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgProfileRepository(pool)
	owner := identity.New(uuid.New(), uuid.NewString())
	sameSubjectOtherTenant := identity.New(uuid.New(), owner.Subject)
	other := identity.New(owner.TenantID, uuid.NewString())

	experience, err := repo.AddExperience(ctx, owner, models.ExperienceInput{Role: "private"})
	if err != nil {
		t.Fatal(err)
	}
	project, _ := repo.AddProject(ctx, owner, models.ProjectInput{Name: "private"})
	education, _ := repo.AddEducation(ctx, owner, models.EducationInput{Level: "other", Degree: "private"})
	certification, _ := repo.AddCertification(ctx, owner, models.CertificationInput{Name: "private"})
	language, _ := repo.AddLanguage(ctx, owner, models.LanguageInput{Name: "private", Level: "A1"})
	for _, stranger := range []identity.IdentityContext{sameSubjectOtherTenant, other} {
		if found, err := repo.UpdateExperience(ctx, stranger, experience.ID, models.ExperienceInput{Role: "stolen"}); err != nil || found != nil {
			t.Fatalf("cross-owner experience update: %v %v", found, err)
		}
		if found, err := repo.UpdateProject(ctx, stranger, project.ID, models.ProjectInput{Name: "stolen"}); err != nil || found != nil {
			t.Fatalf("cross-owner project update: %v %v", found, err)
		}
		if found, err := repo.UpdateEducation(ctx, stranger, education.ID, models.EducationInput{Level: "other"}); err != nil || found != nil {
			t.Fatalf("cross-owner education update: %v %v", found, err)
		}
		if found, err := repo.UpdateCertification(ctx, stranger, certification.ID, models.CertificationInput{Name: "stolen"}); err != nil || found != nil {
			t.Fatalf("cross-owner certification update: %v %v", found, err)
		}
		for name, remove := range map[string]func() (bool, error){
			"experience":    func() (bool, error) { return repo.DeleteExperience(ctx, stranger, experience.ID) },
			"project":       func() (bool, error) { return repo.DeleteProject(ctx, stranger, project.ID) },
			"education":     func() (bool, error) { return repo.DeleteEducation(ctx, stranger, education.ID) },
			"certification": func() (bool, error) { return repo.DeleteCertification(ctx, stranger, certification.ID) },
			"language":      func() (bool, error) { return repo.DeleteLanguage(ctx, stranger, language.ID) },
		} {
			if removed, err := remove(); err != nil || removed {
				t.Fatalf("cross-owner delete of %s: %v %v", name, removed, err)
			}
		}
		profile, err := repo.Ensure(ctx, stranger)
		if err != nil || len(profile.Experience)+len(profile.Projects)+len(profile.Education)+len(profile.Certifications)+len(profile.Languages) != 0 {
			t.Fatalf("a stranger sees an empty profile: %+v %v", profile, err)
		}
	}
	profile, _ := repo.Ensure(ctx, owner)
	if len(profile.Experience) != 1 || profile.Experience[0].Role != "private" || profile.Projects[0].Name != "private" ||
		profile.Education[0].Degree != "private" || profile.Certifications[0].Name != "private" || len(profile.Languages) != 1 {
		t.Fatalf("owner's rows are untouched: %+v", profile)
	}
}

func TestProfileEnsureIsRaceSafe(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgProfileRepository(pool)
	ident := identity.New(uuid.New(), uuid.NewString())
	var wg sync.WaitGroup
	ids := make([]uuid.UUID, 16)
	failures := make([]error, len(ids))
	for i := range ids {
		wg.Go(func() {
			profile, err := repo.Ensure(ctx, ident)
			if err != nil {
				failures[i] = err
				return
			}
			ids[i] = profile.ID
		})
	}
	wg.Wait()
	for i, err := range failures {
		if err != nil || ids[i] != ids[0] {
			t.Fatalf("concurrent ensure %d: %v %v", i, ids[i], err)
		}
	}
	var rows int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM profiles WHERE subject = $1", ident.Subject).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("one profile row: %d %v", rows, err)
	}
}

func TestUpdatesAndDeletesDoNotCreateTheProfile(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgProfileRepository(pool)
	ident := identity.New(uuid.New(), uuid.NewString())
	id := uuid.New()
	for name, call := range map[string]func() (bool, error){
		"update experience": func() (bool, error) {
			f, err := repo.UpdateExperience(ctx, ident, id, models.ExperienceInput{Role: "r"})
			return f != nil, err
		},
		"update project": func() (bool, error) {
			f, err := repo.UpdateProject(ctx, ident, id, models.ProjectInput{Name: "p"})
			return f != nil, err
		},
		"update education": func() (bool, error) {
			f, err := repo.UpdateEducation(ctx, ident, id, models.EducationInput{Level: "other"})
			return f != nil, err
		},
		"update certification": func() (bool, error) {
			f, err := repo.UpdateCertification(ctx, ident, id, models.CertificationInput{Name: "c"})
			return f != nil, err
		},
		"delete experience":    func() (bool, error) { return repo.DeleteExperience(ctx, ident, id) },
		"delete project":       func() (bool, error) { return repo.DeleteProject(ctx, ident, id) },
		"delete education":     func() (bool, error) { return repo.DeleteEducation(ctx, ident, id) },
		"delete certification": func() (bool, error) { return repo.DeleteCertification(ctx, ident, id) },
		"delete language":      func() (bool, error) { return repo.DeleteLanguage(ctx, ident, id) },
	} {
		if found, err := call(); err != nil || found {
			t.Fatalf("%s without a profile: %v %v", name, found, err)
		}
	}
	var rows int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM profiles WHERE subject = $1", ident.Subject).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("updates and deletes must not create the profile: %d %v", rows, err)
	}
}

func TestProfileRepositoryFailsOnClosedPool(t *testing.T) {
	_, pool := application(t, isolatedSchema(t))
	ctx := context.Background()
	repo := repositories.NewPgProfileRepository(pool)
	ident := identity.New(uuid.New(), uuid.NewString())
	pool.Close()
	id := uuid.New()
	calls := map[string]func() error{
		"ensure":            func() error { _, err := repo.Ensure(ctx, ident); return err },
		"patch":             func() error { _, err := repo.Patch(ctx, ident, models.ProfilePatch{}); return err },
		"skills":            func() error { _, err := repo.SetSkills(ctx, ident, nil); return err },
		"add experience":    func() error { _, err := repo.AddExperience(ctx, ident, models.ExperienceInput{Role: "r"}); return err },
		"update experience": func() error { _, err := repo.UpdateExperience(ctx, ident, id, models.ExperienceInput{}); return err },
		"delete experience": func() error { _, err := repo.DeleteExperience(ctx, ident, id); return err },
		"add project":       func() error { _, err := repo.AddProject(ctx, ident, models.ProjectInput{}); return err },
		"update project":    func() error { _, err := repo.UpdateProject(ctx, ident, id, models.ProjectInput{}); return err },
		"add education":     func() error { _, err := repo.AddEducation(ctx, ident, models.EducationInput{}); return err },
		"update education":  func() error { _, err := repo.UpdateEducation(ctx, ident, id, models.EducationInput{}); return err },
		"add certification": func() error { _, err := repo.AddCertification(ctx, ident, models.CertificationInput{}); return err },
		"update certification": func() error {
			_, err := repo.UpdateCertification(ctx, ident, id, models.CertificationInput{})
			return err
		},
		"add language": func() error { _, err := repo.AddLanguage(ctx, ident, models.LanguageInput{}); return err },
	}
	for name, call := range calls {
		if err := call(); err == nil {
			t.Fatalf("%s on a closed pool must fail", name)
		}
	}
}

func TestProfileRoutesUseRealPostgres(t *testing.T) {
	handler, _ := application(t, isolatedSchema(t))
	status, profile := testutil.Do(t, handler, http.MethodGet, mePath, nil)
	if status != http.StatusOK || profile["email"] != "" || fmt.Sprint(profile["skills"]) != "[]" || profile["salary_min"] != nil {
		t.Fatalf("first GET creates the profile: %d %v", status, profile)
	}
	if status, again := testutil.Do(t, handler, http.MethodGet, mePath, nil); status != http.StatusOK || again["id"] != profile["id"] {
		t.Fatalf("second GET returns the same profile: %d %v", status, again)
	}
	status, patched := testutil.Do(t, handler, http.MethodPatch, mePath, map[string]any{"name": " Ana ", "salary_min": 1000,
		"notice_period_weeks": 2, "work_modes": []string{"remote"}})
	if status != http.StatusOK || patched["name"] != "Ana" || patched["salary_min"] != float64(1000) {
		t.Fatalf("patch: %d %v", status, patched)
	}
	status, patched = testutil.Do(t, handler, http.MethodPatch, mePath, `{"salary_min": null}`)
	if status != http.StatusOK || patched["salary_min"] != nil || patched["notice_period_weeks"] != float64(2) ||
		fmt.Sprint(patched["work_modes"]) != "[remote]" {
		t.Fatalf("null clears one field: %d %v", status, patched)
	}
	status, experience := testutil.Do(t, handler, http.MethodPost, mePath+"/experience", map[string]any{"role": "Dev", "end": "",
		"bullets": []map[string]any{{"text": "Cut deploys", "result": "2h to 15m", "skills": []string{"CI", "ci"}}}})
	bullets, _ := experience["bullets"].([]any)
	if status != http.StatusCreated || experience["end"] != "" || fmt.Sprint(experience["tools"]) != "[]" || len(bullets) != 1 ||
		fmt.Sprint(bullets[0]) != "map[result:2h to 15m skills:[CI] text:Cut deploys]" {
		t.Fatalf("experience: %d %v", status, experience)
	}
	status, certification := testutil.Do(t, handler, http.MethodPost, mePath+"/certifications", map[string]any{"name": "CKA"})
	if status != http.StatusCreated || certification["expires"] != "" {
		t.Fatalf("certification: %d %v", status, certification)
	}
	path := mePath + "/experience/" + experience["id"].(string)
	if status, _ := testutil.Do(t, handler, http.MethodDelete, path, nil); status != http.StatusNoContent {
		t.Fatalf("delete: %d", status)
	}
	if status, body := testutil.Do(t, handler, http.MethodPut, path, map[string]any{"role": "x"}); status != http.StatusNotFound ||
		testutil.ErrorCode(body) != "NOT_FOUND" {
		t.Fatalf("update deleted: %d %v", status, body)
	}
}
