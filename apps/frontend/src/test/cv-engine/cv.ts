import type { EngineJob as Job } from './job'
import { matchJob } from '@/features/jobs/match'
import type { Bullet, Profile } from '@/features/profile/api/profile.schema'
import { educationTitle, monthLabel, yearRange } from '@/features/profile/format'
import { bulletLine, knownSkills } from '@/features/profile/skills'
import type { AtsCheck, Cv, CvDocument, CvRequest } from '@/features/cv/api/cv.schema'
import { countKeywordHits } from '@/features/cv/highlight'
import { cleanFact, rewriteBullet } from './rewrite'
import { hasWord, joinList, lc } from './text'

interface Built {
  document: CvDocument
  /** The keywords the tailored CV is written around. */
  matched: string[]
}

const MONTH = /^\d{4}-\d{2}$/

/** `03/2022 – Present`; nothing without a start. */
export function cvDates(start: string, end: string): string {
  return start ? `${monthLabel(start)} – ${end ? monthLabel(end) : 'Present'}` : ''
}

const monthIndex = (value: string) => {
  const [year, month] = value.split('-').map(Number)
  return year! * 12 + (month! - 1)
}

/** Whole years from the earliest role's start month to `now`. */
export function yearsOfExperience(profile: Pick<Profile, 'experience'>, now: Date): number {
  const starts = profile.experience.map(role => role.start).filter(start => MONTH.test(start)).map(monthIndex)
  if (!starts.length) return 0
  return Math.max(0, Math.floor((now.getFullYear() * 12 + now.getMonth() - Math.min(...starts)) / 12))
}

/**
 * One CV from a profile snapshot: as the profile reads (`tailored: false`), or rewritten in the
 * job's words around the skills it asks for.
 */
export function buildDocument(profile: Profile, job: Job, request: CvRequest, tailored: boolean, now: Date): Built {
  const { matched } = matchJob(job, knownSkills(profile))
  const keywords = tailored ? matched : []
  const roles = profile.experience.filter(role => !tailored || !request.skip_roles.includes(role.id))
  const rewrite = (bullets: Bullet[], tools: string[]) => {
    const used = new Set<string>()
    return bullets.map((bullet, index) => {
      const line = cleanFact(bulletLine(bullet), null)
      const text = tailored ? rewriteBullet(line, index, { job, matched, tools, used }) : line
      return { text, hits: countKeywordHits(text, keywords), index }
    })
  }

  const top = request.focus.length ? request.focus.filter(skill => matched.includes(skill)) : matched.slice(0, 3)
  const years = yearsOfExperience(profile, now)
  const sentences = profile.about.split('.').map(sentence => sentence.trim()).filter(Boolean)
  let summary = profile.about
  if (tailored) {
    const experience = years ? `${years} years of experience` : 'hands-on experience'
    const strongest = top.length ? `, strongest in ${joinList(top)}` : ''
    summary = [
      `${profile.headline || 'Engineer'} with ${experience}${strongest}.`,
      sentences.length > 1 ? `${sentences[1]}.` : '',
      `Looking to join ${job.company} as ${job.title} and work on its ${job.product}.`,
    ].filter(Boolean).join(' ')
  }
  const headline = profile.headline
  const matchedLow = matched.map(lc)
  const skills = tailored ? [...matched, ...profile.skills.filter(skill => !matchedLow.includes(lc(skill)))] : profile.skills

  const experience = roles.map((role, roleIndex) => {
    let bullets = rewrite(role.bullets, role.tools)
    if (tailored) {
      const digits = (text: string) => (/\d/.test(text) ? 1 : 0)
      bullets.sort((a, b) => (b.hits - a.hits) || (digits(b.text) - digits(a.text)) || (a.index - b.index))
      bullets = bullets.slice(0, roleIndex === 0 ? 7 : 3)
    }
    return {
      role: role.role,
      meta: [role.company, role.location, cvDates(role.start, role.end)].filter(Boolean).join(' | '),
      bullets: bullets.map(bullet => bullet.text),
    }
  })
  const projects = profile.projects.map(project => ({
    name: project.name,
    meta: [project.role, project.url, cvDates(project.start, project.end)].filter(Boolean).join(' | '),
    bullets: rewrite(project.bullets, project.tools).map(bullet => bullet.text),
  }))
  const certifications = profile.certifications.map(item => [
    item.name,
    item.issuer,
    item.issued ? `Issued ${monthLabel(item.issued)}` : '',
    item.expires ? `Expires ${monthLabel(item.expires)}` : '',
    item.credential_id ? `ID ${item.credential_id}` : '',
    item.url,
  ].filter(Boolean).join(' | '))

  return {
    matched,
    document: {
      name: profile.name.trim(),
      headline,
      contact: [profile.email, profile.phone, [profile.city, profile.country].filter(part => part.trim()).join(', ')].filter(part => part.trim()),
      links: [profile.github, profile.linkedin].filter(part => part.trim()),
      summary,
      skills,
      experience,
      projects,
      education: profile.education.map(item => ({
        degree: educationTitle(item),
        meta: [item.school, yearRange(item.start_year, item.end_year).replace('now', 'Present')].filter(Boolean).join(' | '),
      })),
      certifications,
      languages: profile.languages.map(language => `${language.name} (${language.level})`).join(', '),
      additional: profile.additional_info.map(line => cleanFact(line, null)).filter(Boolean),
    },
  }
}

const ACTION_VERBS = ['built', 'wrote', 'made', 'ran', 'led', 'kept', 'set', 'took']

/** What an applicant tracking system looks for, checked on the final text and the profile it came from. */
export function atsChecks(document: CvDocument, job: Job, profile: Profile, request: CvRequest): AtsCheck[] {
  const bullets = [...document.experience, ...document.projects].flatMap(entry => entry.bullets)
  const skillsLine = document.skills.join(', ')
  const body = [document.summary, skillsLine, ...bullets].join(' ')
  const inSentences = [document.summary, ...bullets].join(' ')
  const found = job.keywords.filter(keyword => hasWord(body, keyword))
  const inContext = job.keywords.filter(keyword => hasWord(inSentences, keyword))
  const firstWord = (bullet: string) => (bullet.match(/^(\w+)/)?.[1] ?? '').toLowerCase()
  const verbs = bullets.length > 0 && bullets.every(b => /ed$/.test(firstWord(b)) || ACTION_VERBS.includes(firstWord(b)))
  const withNumbers = bullets.filter(bullet => /\d/.test(bullet)).length
  const length = (document.summary + skillsLine + bullets.join(' ')).length
  const roles = profile.experience.filter(role => !request.skip_roles.includes(role.id))
  const dated = roles.length > 0 && roles.every(role => MONTH.test(role.start) && (!role.end || MONTH.test(role.end)))
  const mirrored = new Set(request.confirmed.filter(index => job.key_responsibilities.includes(index))).size
  const check = (ok: boolean, label: string, hint = '') => ({ ok, label, hint })
  return [
    check(true, 'Single column, plain text, no tables or images'),
    check(true, 'Standard headings: Summary, Skills, Experience, Education'),
    check(Boolean(profile.email.trim() && profile.phone.trim()), 'Email and phone in the body, not in a header', 'Add them in Profile.'),
    check(Boolean(profile.city.trim() || profile.country.trim()), 'Location in the header', 'Add your city and country in Profile.'),
    check(dated, 'Dates with month and year', 'Add the start month of each role in Profile.'),
    check(hasWord(document.summary, job.title), `Target job title “${job.title}” in the summary`),
    check(found.length === job.keywords.length, `Keywords from the job: ${found.length} of ${job.keywords.length}`,
      'The missing ones are not in your profile yet. Tell us about them in Prepare your CV.'),
    check(inContext.length >= Math.ceil(found.length * 0.6), `Keywords used in sentences, not only listed: ${inContext.length} of ${found.length}`,
      'List the tools you used for each role in Profile.'),
    check(mirrored > 0, `Job responsibilities mirrored: ${mirrored} of ${job.key_responsibilities.length}`,
      'Describe them in Prepare your CV.'),
    check(verbs, 'Every bullet starts with an action verb'),
    check(withNumbers > 0, `Results with numbers: ${withNumbers} of ${bullets.length}`,
      'Add a result to your highlights in Prepare your CV.'),
    check(length < 3600, 'Fits on one page'),
  ]
}

/** The sample shown while the CV is being made: the first highlight of the first role kept, rewritten for the job. */
export function rewritePreview(profile: Profile, job: Job, request: CvRequest): Cv['preview'] {
  const { matched } = matchJob(job, knownSkills(profile))
  const role = profile.experience.find(item => !request.skip_roles.includes(item.id) && item.bullets.length)
  if (!role) return null
  const from = cleanFact(bulletLine(role.bullets[0]!), null)
  return { from, to: rewriteBullet(from, 0, { job, matched, tools: role.tools, used: new Set() }) }
}

const fileSafe = (value: string) => value.trim().replace(/[^A-Za-z0-9]+/g, '_')

/** `<Name>_CV_<Company>.pdf`, or `CV_<Company>.pdf` without a name. */
export function cvFileName(name: string, company: string): string {
  return `${name.trim() ? `${fileSafe(name)}_` : ''}CV_${fileSafe(company)}.pdf`
}

/** Both versions of the CV from one snapshot of the profile, the tailored one checked for ATS: a ready CV but its id and times. */
export function generateCv(profile: Profile, job: Job, request: CvRequest, now: Date): Omit<Cv, 'id' | 'created_at' | 'updated_at'> {
  const snapshot = structuredClone(profile)
  const plain = buildDocument(snapshot, job, request, false, now)
  const tailored = buildDocument(snapshot, job, request, true, now)
  const checks = atsChecks(tailored.document, job, snapshot, request)
  return {
    job_id: job.id,
    status: 'ready',
    job_title: job.title,
    company: job.company,
    file_name: cvFileName(snapshot.name, job.company),
    highlights: tailored.matched,
    profile: plain.document,
    tailored: tailored.document,
    ats: { passed: checks.filter(item => item.ok).length, total: checks.length, checks },
    preview: rewritePreview(snapshot, job, request),
    tailoring: null,
    written_by: 'rules',
    model: '',
    refused: 0,
    rules_reason: 'off',
    limit_until: null,
  }
}
