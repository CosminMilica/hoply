import { z } from 'zod'

const id = z.union([z.string(), z.number()]).transform(String)
const text = z.string().nullish().transform(value => value ?? '')
const list = z.array(z.string()).nullish().transform(value => value ?? [])
const count = z.number().int().nullish().transform(value => value ?? null)
const many = <T extends z.ZodType>(item: T) => z.array(item).nullish().transform(value => (value ?? []) as z.output<T>[])

export const LANGUAGE_LEVELS = ['A1', 'A2', 'B1', 'B2', 'C1', 'C2', 'Native'] as const
export type LanguageLevel = (typeof LANGUAGE_LEVELS)[number]

export const WORK_MODES = { remote: 'Remote', hybrid: 'Hybrid', onsite: 'On-site' } as const
export const WORK_AUTHORIZATIONS = {
  citizen: 'Citizen',
  permanent_resident: 'Permanent resident',
  work_permit: 'Work permit',
  needs_sponsorship: 'Needs sponsorship',
} as const
export const SENIORITIES = { intern: 'Intern', junior: 'Junior', mid: 'Mid', senior: 'Senior', lead: 'Lead', principal: 'Principal' } as const
export const EMPLOYMENT_TYPES = {
  full_time: 'Full-time',
  part_time: 'Part-time',
  contract: 'Contract',
  freelance: 'Freelance',
  internship: 'Internship',
} as const
export const SALARY_PERIODS = { year: 'per year', month: 'per month', hour: 'per hour' } as const
export const EDUCATION_LEVELS = {
  high_school: 'High school',
  associate: 'Associate degree',
  bachelor: 'Bachelor’s degree',
  master: 'Master’s degree',
  doctorate: 'Doctorate',
  other: 'Other',
} as const

const keys = <T extends Record<string, string>>(labels: T) => Object.keys(labels) as [keyof T & string, ...(keyof T & string)[]]

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

const at = (limit: number) => `At most ${limit} characters`
const short = (limit: number) => z.string().trim().max(limit, at(limit))
const year = z.string().trim().regex(/^((19[5-9]\d)|(20\d\d)|2100)?$/, 'Use a year from 1950 to 2100, e.g. 2022')
export const month = z.string().trim().regex(/^((19[5-9]\d|20\d\d|2100)-(0[1-9]|1[0-2]))?$/, 'Use a month, e.g. 2022-03')
const whole = (max: number) => z.number().int('Use a whole number').min(0, 'Use 0 or more').max(max, `At most ${max}`).nullable()
const tags = (limit: number, label: string) => z.array(short(60)).max(limit, `At most ${limit} ${label}`)
const notBefore = (start: string, end: string) => !start || !end || end >= start

export const bulletInputSchema = z.object({
  text: short(300),
  result: short(200),
  skills: tags(10, 'skills'),
})

export const profilePatchSchema = z.object({
  name: short(120),
  headline: short(120),
  about: short(280),
  email: z.union([z.literal(''), z.email('Enter a valid email address')]),
  phone: short(200),
  github: short(200),
  linkedin: short(200),
  city: short(60),
  country: short(60),
  work_modes: z.array(z.enum(keys(WORK_MODES))),
  open_to_relocation: z.boolean(),
  work_authorization: z.union([z.literal(''), z.enum(keys(WORK_AUTHORIZATIONS))]),
  work_authorization_region: short(60),
  notice_period_weeks: whole(52),
  available_from: month,
  target_roles: z.array(short(120)).max(10, 'At most 10 target roles'),
  target_seniority: z.array(z.enum(keys(SENIORITIES))),
  employment_types: z.array(z.enum(keys(EMPLOYMENT_TYPES))),
  salary_min: whole(100000000),
  salary_currency: z.string().trim().regex(/^([A-Za-z]{3})?$/, 'Use a three-letter code, e.g. EUR').transform(value => value.toUpperCase()),
  salary_period: z.union([z.literal(''), z.enum(keys(SALARY_PERIODS))]),
  additional_info: z.array(short(300)).max(10, 'At most 10 lines'),
  notes: z.array(z.object({ text: z.string().trim().max(1000), skills: z.array(z.string()) })).max(30, 'At most 30 notes'),
}).partial()

export type ProfilePatch = z.infer<typeof profilePatchSchema>

const employmentType = z.union([z.literal(''), z.enum(keys(EMPLOYMENT_TYPES))])

export const experienceInputSchema = z.object({
  role: z.string().trim().min(1, 'Add the role').max(120, at(120)),
  company: short(120),
  location: short(120),
  employment_type: employmentType,
  industry: short(60),
  team_size: whole(100000),
  people_managed: whole(100000),
  start: month,
  end: month,
  tools: tags(30, 'tools'),
  bullets: z.array(bulletInputSchema).max(10, 'At most 10 highlights'),
}).refine(value => notBefore(value.start, value.end), { message: 'The end cannot be before the start', path: ['end'] })

export type ExperienceInput = z.infer<typeof experienceInputSchema>

export const projectInputSchema = z.object({
  name: z.string().trim().min(1, 'Add the project name').max(120, at(120)),
  role: short(120),
  url: short(200),
  start: month,
  end: month,
  tools: tags(30, 'tools'),
  bullets: z.array(bulletInputSchema).max(10, 'At most 10 highlights'),
}).refine(value => notBefore(value.start, value.end), { message: 'The end cannot be before the start', path: ['end'] })

export type ProjectInput = z.infer<typeof projectInputSchema>

export const educationInputSchema = z.object({
  level: z.enum(keys(EDUCATION_LEVELS), 'Choose the level'),
  field: short(120),
  degree: short(120),
  school: short(120),
  start_year: year,
  end_year: year,
}).refine(value => notBefore(value.start_year, value.end_year), { message: 'The end year cannot be before the start year', path: ['end_year'] })

export type EducationInput = z.infer<typeof educationInputSchema>

export const certificationInputSchema = z.object({
  name: z.string().trim().min(1, 'Add the certification').max(120, at(120)),
  issuer: short(120),
  issued: month,
  expires: month,
  credential_id: short(120),
  url: short(200),
}).refine(value => notBefore(value.issued, value.expires), { message: 'It cannot expire before it was issued', path: ['expires'] })

export type CertificationInput = z.infer<typeof certificationInputSchema>

export const languageInputSchema = z.object({
  name: z.string().trim().min(1, 'Add the language').max(120, at(120)),
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
