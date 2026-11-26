import { z } from 'zod'

const text = z.string().nullish().transform(value => value ?? '')
const list = z.array(z.string()).nullish().transform(value => value ?? [])

export const JOB_WORK_MODES = { remote: 'Remote', hybrid: 'Hybrid', onsite: 'On-site' } as const
export const JOB_LEVELS = { junior: 'Junior', mid: 'Mid', senior: 'Senior', lead: 'Lead' } as const
export type JobWorkMode = '' | keyof typeof JOB_WORK_MODES
export type JobLevel = '' | keyof typeof JOB_LEVELS
/** How the service read a job; a value added later still parses, as plain text. */
export const READ_FROM = ['', 'json_ld', 'page_text', 'ats_api', 'text'] as const
const readFrom = z.union([z.enum(READ_FROM), z.string()]).nullish().transform(value => value ?? '')

export const jobSchema = z.object({
  id: z.union([z.string(), z.number()]).transform(String),
  title: text,
  company: text,
  city: text,
  country: text,
  work_mode: text,
  level: text,
  years_required: z.number().int().nullish().transform(value => value ?? null),
  must_have: list,
  nice_to_have: list,
  responsibilities: list,
  /** The ad's own lines for what it asks and what is a plus. */
  qualifications: list,
  bonuses: list,
  about: text,
  about_company: text,
  product: text,
  perks: list,
  areas: list,
  url: text,
  source: text,
  read_from: readFrom,
  /** Whether the requirements are the rules' reading of the ad or Claude's. */
  requirements_from: z.enum(['rules', 'analysis']).nullish().catch('rules').transform(value => value ?? 'rules'),
  sample: z.boolean().nullish().transform(value => value ?? false),
  added_by_me: z.boolean().nullish().transform(value => value ?? false),
  created_at: text,
  /** The job board platform an imported job comes from. */
  origin: text,
  posted_at: z.string().nullish().transform(value => value || null),
  closed_at: z.string().nullish().transform(value => value || null),
})

export type Job = z.infer<typeof jobSchema>

export const jobListSchema = z.object({ items: z.array(jobSchema), total: z.number() })

const short = (limit: number) => z.string().trim().max(limit, `At most ${limit} characters`)
const skills = z.array(short(60)).max(40, 'At most 40 requirements')

export const jobInputSchema = z.object({
  title: z.string().trim().min(1, 'Add the job title.').max(160, 'At most 160 characters'),
  company: short(120),
  city: short(120),
  country: short(120),
  work_mode: z.enum(['', 'remote', 'hybrid', 'onsite']),
  level: z.enum(['', 'junior', 'mid', 'senior', 'lead']),
  years_required: z.number().int().min(0).max(40).nullable(),
  must_have: skills,
  nice_to_have: skills,
  responsibilities: z.array(short(400)).max(30),
  qualifications: z.array(short(400)).max(30).default([]),
  bonuses: z.array(short(400)).max(30).default([]),
  about_company: short(4000).default(''),
  perks: z.array(short(400)).max(30).default([]),
  about: short(4000),
  url: z.string().trim().max(2000).regex(/^(https?:\/\/\S+)?$/i, 'Use a link starting with http:// or https://'),
  source: z.enum(['link', 'text']),
  read_from: readFrom,
})

export type JobInput = z.infer<typeof jobInputSchema>

export const jobDraftSchema = jobSchema
  .pick({ title: true, company: true, city: true, country: true, work_mode: true, level: true, years_required: true, must_have: true, nice_to_have: true, responsibilities: true, qualifications: true, bonuses: true, about: true, about_company: true, perks: true, url: true, source: true, read_from: true })
  .extend({ host: text })

export type JobDraft = z.infer<typeof jobDraftSchema>

/** The board a job link belongs to, and whether Hoply imports its jobs already. */
export const boardOfSchema = z.object({ platform: z.string(), board: z.string(), company: text, imported: z.boolean() })
export type BoardOf = z.infer<typeof boardOfSchema>

export const boardSchema = z.object({ company: z.string(), platform: z.string(), board: z.string(), followed_at: z.string().nullish() })

export type ReadRequest = { url: string } | { text: string, title?: string, company?: string, url?: string }

export interface JobFilters {
  q?: string
  city?: string
  country?: string
  includeRemote?: boolean
  areas?: string[]
  /** One of the filter's levels, `Any` for none. */
  level?: string
  /** These jobs only, closed ones included. */
  ids?: string[]
}

/** The tech areas the jobs service reads from a job's title (extract.Areas). */
export const AREAS = ['DevOps', 'Backend', 'Frontend', 'Full stack', 'Mobile', 'Data', 'AI', 'QA', 'Security', 'Embedded', 'Sysadmin', 'Product', 'Design'] as const
export const LEVELS = ['Any', 'Junior', 'Mid', 'Senior'] as const
export const CITIES = ['Iași', 'Bucharest', 'Cluj-Napoca'] as const

export const UNREADABLE_REASONS = ['login_required', 'no_job_data', 'fetch_failed', 'too_large', 'blocked_address'] as const
export type UnreadableReason = (typeof UNREADABLE_REASONS)[number]
