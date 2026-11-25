# How Hoply writes a CV

The rules behind the prompt (`services/cv/internal/grounding/prompts/cv-system.md`), the checks
(`internal/grounding/check.go`, `internal/cvdoc/document.go`) and the export (`internal/render`).

| Rule | Where it lives | Source |
| --- | --- | --- |
| Write only from the person's facts; every sentence cites them; numbers, skills and companies are checked | checks | Hoply's own rule: a CV with invented skills fails the first interview |
| Notes are raw material, rewritten and combined with highlights, never printed as they are | prompt | the owner, 2026-11-25 |
| A tool proves its umbrella term (GitHub Actions proves CI/CD); a close tool is named only beside the one the person has, as transferable ("FluxCD, a GitOps approach that transfers to Argo CD"), never in the skills list | `skillmap.json`, prompt, checks | the owner, 2026-11-25 |
| Each bullet: what was done, how, and so what (result, scale, who it served) | prompt | [5] |
| Active, specific verbs; no pronouns; no "responsible for"; "express, not impress" | prompt | [1] |
| Quantify with the person's own numbers, never added ones | prompt, checks | [1], [5] |
| The job's terms, in the job's spelling, woven into real sentences; no keyword lists in bullets | prompt | [1], [2], [5] |
| Every requirement the person has appears in at least one sentence | prompt, ATS check | [2], [4] |
| No generic AI filler or unproven praise | prompt | [5] |
| One column, standard headings, no tables, images, headers or footers; contact in the body | export | [1], [2] |
| Arial or Liberation Sans, 10 pt | export | [2] (10–12 pt) |
| One page under eight years of experience, two from then on | `cvdoc.PageLimit` | [5] retires the one-page rule for experienced people; one page stays the norm early on |
| DOCX and PDF both offered | export | [2] prefers DOCX; [1] says check that a PDF reads as text |

The articles disagree on keyword use: [4] says screeners reward job-description terms, [5] that mechanical
mirroring no longer works. Hoply uses the job's terms only where the person's facts support them.

Sources, read 2026-11-25:
1. Harvard FAS Mignone Center, [Create a strong resume](https://careerservices.fas.harvard.edu/resources/create-a-strong-resume/)
2. Empire Group, [How AI is changing the resume game](https://www.empiregroup.com.au/how-ai-is-changing-the-resume-game/)
3. NJEA, [How artificial intelligence is changing the job hunt](https://www.njea.org/how-artificial-intelligence-is-changing-the-job-hunt/)
4. Findem, [AI for resume screening](https://www.findem.ai/blog/ai-for-resume-screening)
5. Career Impressions, [Executive resume trends for 2026](https://careerimpressions.ca/executive-resume-trends-for-2026-what-works-and-what-to-retire/)
