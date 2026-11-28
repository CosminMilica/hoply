import { z } from 'zod'
import { t } from '@/features/common/i18n'

const id = z.union([z.string(), z.number()]).transform(String)
const text = z.string().nullish().transform(value => value ?? '')
const list = z.array(z.string()).nullish().transform(value => value ?? [])
const count = z.number().int().nullish().transform(value => value ?? null)
const many = <T extends z.ZodType>(item: T) => z.array(item).nullish().transform(value => (value ?? []) as z.output<T>[])

export const LANGUAGE_LEVELS = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2', 'Native'] as const
export type LanguageLevel = (typeof LANGUAGE_LEVELS)[number]

/** The values the profile service accepts; their names in each language are in the feature's messages. */
export const WORK_MODES = ['remote', 'hybrid', 'onsite'] as const
export const WORK_AUTHORIZATIONS = ['citizen', 'permanent_resident', 'work_permit', 'needs_sponsorship'] as const
export const SENIORITIES = ['intern', 'junior', 'mid', 'senior', 'lead', 'principal'] as const
export const EMPLOYMENT_TYPES = ['full_time', 'part_time', 'contract', 'freelance', 'internship'] as const
export const SALARY_PERIODS = ['year', 'month', 'hour'] as const
export const EDUCATION_LEVELS = ['high_school', 'associate', 'bachelor', 'master', 'doctorate', 'other'] as const
export const DOMAIN_KEYS = ['it', 'banking', 'accounting', 'finance', 'insurance', 'hr'] as const

export type WorkMode = (typeof WORK_MODES)[number]
export type WorkAuthorization = (typeof WORK_AUTHORIZATIONS)[number]
export type Seniority = (typeof SENIORITIES)[number]
export type EmploymentType = (typeof EMPLOYMENT_TYPES)[number]
export type SalaryPeriod = (typeof SALARY_PERIODS)[number]
export type EducationLevel = (typeof EDUCATION_LEVELS)[number]
export type Domain = (typeof DOMAIN_KEYS)[number]

export const bulletSchema = z.object({ text, result: text, skills: list })
export type Bullet = z.infer<typeof bulletSchema>

/** What the person told Prepare your CV, in their own words: Claude writes CV bullets from it, never prints it as it is. */
export const noteSchema = z.object({ text, skills: list })
export type Note = z.infer<typeof noteSchema>

export const experienceSchema = z.object({
  id,
  role: text,
  company: text,
  location: text,
  employment_type: text,
  industry: text,
  team_size: count,
  people_managed: count,
  start: text,
  end: text,
  tools: list,
  bullets: many(bulletSchema),
  notes: many(noteSchema),
})

export const projectSchema = z.object({
  id, name: text, role: text, url: text, start: text, end: text, tools: list, bullets: many(bulletSchema), notes: many(noteSchema),
})

export const educationSchema = z.object({ id, level: text, field: text, degree: text, school: text, start_year: text, end_year: text })

export const certificationSchema = z.object({ id, name: text, issuer: text, issued: text, expires: text, credential_id: text, url: text })

export const languageSchema = z.object({ id, name: text, level: text })

export const profileSchema = z.object({
  id,
  name: text,
  headline: text,
  about: text,
  email: text,
  phone: text,
  github: text,
  linkedin: text,
  city: text,
  country: text,
  work_modes: list,
  open_to_relocation: z.boolean().nullish().transform(value => value ?? false),
  work_authorization: text,
  work_authorization_region: text,
  notice_period_weeks: count,
  available_from: text,
  target_roles: list,
  target_seniority: list,
  employment_types: list,
  salary_min: count,
  salary_currency: text,
  salary_period: text,
  additional_info: list,
  notes: many(noteSchema),
  app_language: text,
  domains: list,
  skills: list,
  experience: many(experienceSchema),
  projects: many(projectSchema),
  education: many(educationSchema),
  certifications: many(certificationSchema),
  languages: many(languageSchema),
  created_at: text,
  updated_at: text,
})

export type Profile = z.infer<typeof profileSchema>
export type Experience = z.infer<typeof experienceSchema>
export type Project = z.infer<typeof projectSchema>
export type Education = z.infer<typeof educationSchema>
export type Certification = z.infer<typeof certificationSchema>
export type Language = z.infer<typeof languageSchema>

/** A validation message in the language the app shows when the value is checked. */
const say = (message: () => string) => ({ error: message })
const at = (limit: number) => say(() => t.value.common.validation.atMost(limit))
const short = (limit: number) => z.string().trim().max(limit, at(limit))
const year = z.string().trim().regex(/^((19[5-9]\d)|(20\d\d)|2100)?$/, say(() => t.value.profile.validation.year))
export const month = z.string().trim().regex(/^((19[5-9]\d|20\d\d|2100)-(0[1-9]|1[0-2]))?$/, say(() => t.value.profile.validation.month))
const whole = (max: number) => z.number().int(say(() => t.value.profile.validation.wholeNumber)).min(0, say(() => t.value.profile.validation.zeroOrMore)).max(max, say(() => t.value.profile.validation.atMost(max))).nullable()
const tags = (limit: number, message: (limit: number) => string) => z.array(short(60)).max(limit, say(() => message(limit)))
const notBefore = (start: string, end: string) => !start || !end || end >= start

export const bulletInputSchema = z.object({
  text: short(300),
  result: short(200),
  skills: tags(10, max => t.value.profile.validation.skills(max)),
})

export const profilePatchSchema = z.object({
  name: short(120),
  headline: short(120),
  about: short(280),
  email: z.union([z.literal(''), z.email(say(() => t.value.profile.validation.email))]),
  phone: short(200),
  github: short(200),
  linkedin: short(200),
  city: short(60),
  country: short(60),
  work_modes: z.array(z.enum(WORK_MODES)),
  open_to_relocation: z.boolean(),
  work_authorization: z.union([z.literal(''), z.enum(WORK_AUTHORIZATIONS)]),
  work_authorization_region: short(60),
  notice_period_weeks: whole(52),
  available_from: month,
  target_roles: z.array(short(120)).max(10, say(() => t.value.profile.validation.targetRoles(10))),
  target_seniority: z.array(z.enum(SENIORITIES)),
  employment_types: z.array(z.enum(EMPLOYMENT_TYPES)),
  salary_min: whole(100000000),
  salary_currency: z.string().trim().regex(/^([A-Za-z]{3})?$/, say(() => t.value.profile.validation.currency)).transform(value => value.toUpperCase()),
  salary_period: z.union([z.literal(''), z.enum(SALARY_PERIODS)]),
  additional_info: z.array(short(300)).max(10, say(() => t.value.profile.validation.lines(10))),
  notes: z.array(z.object({ text: z.string().trim().max(1000, at(1000)), skills: z.array(z.string()) })).max(30, say(() => t.value.profile.validation.notes(30))),
  app_language: z.union([z.literal(''), z.enum(['en', 'ro'])]),
  domains: z.array(z.enum(DOMAIN_KEYS)),
}).partial()

export type ProfilePatch = z.infer<typeof profilePatchSchema>

const employmentType = z.union([z.literal(''), z.enum(EMPLOYMENT_TYPES)])

export const experienceInputSchema = z.object({
  role: z.string().trim().min(1, say(() => t.value.profile.validation.addRole)).max(120, at(120)),
  company: short(120),
  location: short(120),
  employment_type: employmentType,
  industry: short(60),
  team_size: whole(100000),
  people_managed: whole(100000),
  start: month,
  end: month,
  tools: tags(30, max => t.value.profile.validation.tools(max)),
  bullets: z.array(bulletInputSchema).max(10, say(() => t.value.profile.validation.highlights(10))),
  /** Left out, the service keeps the role's notes as they are. */
  notes: z.array(noteSchema).optional(),
}).refine(value => notBefore(value.start, value.end), { ...say(() => t.value.profile.validation.endBeforeStart), path: ['end'] })

export type ExperienceInput = z.infer<typeof experienceInputSchema>

export const projectInputSchema = z.object({
  name: z.string().trim().min(1, say(() => t.value.profile.validation.addProject)).max(120, at(120)),
  role: short(120),
  url: short(200),
  start: month,
  end: month,
  tools: tags(30, max => t.value.profile.validation.tools(max)),
  bullets: z.array(bulletInputSchema).max(10, say(() => t.value.profile.validation.highlights(10))),
  /** Left out, the service keeps the project's notes as they are. */
  notes: z.array(noteSchema).optional(),
}).refine(value => notBefore(value.start, value.end), { ...say(() => t.value.profile.validation.endBeforeStart), path: ['end'] })

export type ProjectInput = z.infer<typeof projectInputSchema>

export const educationInputSchema = z.object({
  level: z.enum(EDUCATION_LEVELS, say(() => t.value.profile.validation.chooseLevel)),
  field: short(120),
  degree: short(120),
  school: short(120),
  start_year: year,
  end_year: year,
}).refine(value => notBefore(value.start_year, value.end_year), { ...say(() => t.value.profile.validation.endYearBeforeStart), path: ['end_year'] })

export type EducationInput = z.infer<typeof educationInputSchema>

export const certificationInputSchema = z.object({
  name: z.string().trim().min(1, say(() => t.value.profile.validation.addCertification)).max(120, at(120)),
  issuer: short(120),
  issued: month,
  expires: month,
  credential_id: short(120),
  url: short(200),
}).refine(value => notBefore(value.issued, value.expires), { ...say(() => t.value.profile.validation.expiresBeforeIssued), path: ['expires'] })

export type CertificationInput = z.infer<typeof certificationInputSchema>

export const languageInputSchema = z.object({
  name: z.string().trim().min(1, say(() => t.value.profile.validation.addLanguage)).max(120, at(120)),
  level: z.enum(LANGUAGE_LEVELS),
})

export type LanguageInput = z.infer<typeof languageInputSchema>

/** Cleans a list the way the service does: trimmed, no empty entries, no case-insensitive duplicates. */
export function cleanList(values: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const raw of values) {
    const value = raw.trim()
    if (!value || seen.has(value.toLowerCase())) continue
    seen.add(value.toLowerCase())
    out.push(value)
  }
  return out
}

/** An entry as its PUT body: everything but the id. */
export function withoutId<T extends { id: string }>(entry: T): Omit<T, 'id'> {
  const { id: _id, ...rest } = entry
  return rest
}

type EntryChange = Partial<Pick<ExperienceInput, 'bullets' | 'notes'>>

/** A role as the service takes it back, with a change; an employment type it no longer knows is left empty. */
export function experienceInput(role: Experience, change: EntryChange = {}): ExperienceInput {
  const known = (EMPLOYMENT_TYPES as readonly string[]).includes(role.employment_type)
  return { ...withoutId(role), employment_type: known ? role.employment_type as ExperienceInput['employment_type'] : '', ...change }
}

/** A project as the service takes it back, with a change. */
export function projectInput(project: Project, change: EntryChange = {}): ProjectInput {
  return { ...withoutId(project), ...change }
}
