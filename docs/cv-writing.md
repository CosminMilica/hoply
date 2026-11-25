# How Hoply writes a CV

The rules behind the prompt (`services/cv/internal/grounding/prompts/cv-system.md`), the checks
(`internal/grounding/check.go`, `internal/cvdoc/document.go`) and the export (`internal/render`).

| Rule | Where it lives | Source |
| --- | --- | --- |
| Write only from the person's facts; every sentence cites them; numbers, skills and companies are checked | checks | Hoply's own rule: a CV with invented skills fails the first interview |
| Notes are raw material, rewritten and combined with highlights, never printed as they are | prompt | the owner, 2026-11-25 |
| A tool proves its umbrella term (GitHub Actions proves CI/CD). A tool the person has not used is never named; the CV shows the kind of work beside the person's own tool instead ("GitOps delivery with FluxCD" for a job that asks for Argo CD) | `skillmap.json`, prompt, checks | the owner, 2026-11-25 |
| One idea per bullet, in the person's own words adapted where it fits; no glued clauses, tool lists or forced terms | prompt | the owner, 2026-11-25 |
| Each bullet: what was done, how, and so what (result, scale, who it served) | prompt | [5] |
| Active, specific verbs; no pronouns; no "responsible for"; "express, not impress" | prompt | [1] |
| At most two tools in a bullet; the rest live in the skills list | prompt, `internal/style`, review | the eval, 2026-11-25 |
| No participle tacked onto a sentence (", delivering…"), no stock words, varied bullet shapes and first words | prompt, `internal/style`, review | the eval, 2026-11-25 |
| The summary is an angle, not "<title> with N years of experience" and not the bullets again | prompt, `internal/style`, review | the eval, 2026-11-25 |
| Each skill named once, whatever its spelling | `skillmap.json`, code | the eval, 2026-11-25 |
| Quantify with the person's own numbers, never added ones | prompt, checks | [1], [5] |
| The job's terms, in the job's spelling, woven into real sentences; no keyword lists in bullets | prompt | [1], [2], [5] |
| Every requirement the person has appears in at least one sentence | prompt, ATS check | [2], [4] |
| No generic AI filler or unproven praise | prompt | [5] |
| One column, standard headings, no tables, images, headers or footers; contact in the body | export | [1], [2] |
| Arial or Liberation Sans, 10 pt | export | [2] (10–12 pt) |
| One page under eight years of experience, two from then on | `cvdoc.PageLimit` | [5] retires the one-page rule for experienced people; one page stays the norm early on |
| DOCX and PDF both offered | export | [2] prefers DOCX; [1] says check that a PDF reads as text |

Code lints every draft for these patterns (`internal/style`) and the review must fix each finding. The evaluation in
`services/cv/eval` measured the change on five cases in three fields: findings per sentence 0.80 → 0.15, the judge's
mean 3.60 → 3.88 (human 3.40 → 3.80), and a blind comparison preferred the new CVs in 4 cases, with 1 tie.

The articles disagree on keyword use: [4] says screeners reward job-description terms, [5] that mechanical
mirroring no longer works. Hoply uses the job's terms only where the person's facts support them.

## Cover letters and messages to recruiters

| Rule | Where | Source |
| --- | --- | --- |
| Open with the job's top priority and the strongest proof of it; never "I am writing to apply" | prompt, checks | [6], [7] |
| 220–320 words in 3 or 4 short paragraphs; one or two stories, not the CV again | prompt | [6], [8] |
| One sentence on why this company, only from what the ad says; no flattery | prompt, `J` facts | [6] |
| No clichés or generic AI phrasing ("passionate", "team player", "perfect fit") | prompt, checks | [6], [7] |
| A message after applying: under 120 words, the role, two proofs, a short ask for a call | prompt | [9], [10] |
| A connection note within LinkedIn's 300 characters | prompt, checks | the platform's limit |
| One follow-up, about a week after applying: the role, one reason they fit, a question about next steps | prompt, page hint | [11], [12] |
| The same facts, checks and naming rules as the CV | checks | Hoply's own rule |

Sources, read 2026-11-25:
1. Harvard FAS Mignone Center, [Create a strong resume](https://careerservices.fas.harvard.edu/resources/create-a-strong-resume/)
2. Empire Group, [How AI is changing the resume game](https://www.empiregroup.com.au/how-ai-is-changing-the-resume-game/)
3. NJEA, [How artificial intelligence is changing the job hunt](https://www.njea.org/how-artificial-intelligence-is-changing-the-job-hunt/)
4. Findem, [AI for resume screening](https://www.findem.ai/blog/ai-for-resume-screening)
5. Career Impressions, [Executive resume trends for 2026](https://careerimpressions.ca/executive-resume-trends-for-2026-what-works-and-what-to-retire/)
6. Built In, [How to write a cover letter](https://builtin.com/articles/how-to-write-a-cover-letter)
7. Toptal, [How to write a cover letter that gets you an interview](https://www.toptal.com/resume/career-center/how-to-write-a-cover-letter-that-gets-you-an-interview)
8. Enhancv, [Cover letter tips](https://enhancv.com/blog/cover-letter-tips/)
9. Duke Career Hub, [How to message recruiters to connect on LinkedIn](https://careerhub.students.duke.edu/blog/2024/11/01/how-to-message-recruiters-to-connect-on-linkedin-with-examples/)
10. Resume Worded, [How to contact recruiters on LinkedIn](https://resumeworded.com/contact-recruiters-on-linkedin-key-advice)
11. The Muse, [How to follow up on a job application](https://www.themuse.com/advice/how-to-follow-up-on-a-job-application-an-email-template)
12. Jobscan, [How to follow up on a job application](https://www.jobscan.co/blog/how-to-follow-up-on-a-job-application)
