import { z } from 'zod'
import { t } from '@/features/common/i18n'
import type { Domain } from '@/features/profile/api/profile.schema'

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
  /** The fields of work the title names (DOMAIN_AREAS' keys); a job the person added in none is in every one. */
  domains: list,
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

/** A job as the list sends it: what the cards, the match and the filters read; `jobsApi.get` has the whole ad. */
export const jobCardSchema = jobSchema.pick({
  id: true, title: true, company: true, city: true, country: true, work_mode: true, level: true, years_required: true,
  must_have: true, nice_to_have: true, responsibilities: true, about: true, areas: true, domains: true, url: true, source: true,
  requirements_from: true, sample: true, added_by_me: true, created_at: true, origin: true, posted_at: true, closed_at: true,
})

export type JobCard = z.infer<typeof jobCardSchema>

export const jobListSchema = z.object({ items: z.array(jobCardSchema), total: z.number() })

// The messages are read when a value is checked, so they come out in the app's language at that moment.
const short = (limit: number) => z.string().trim().max(limit, { error: () => t.value.common.validation.atMost(limit) })
const skills = z.array(short(60)).max(40, { error: () => t.value.jobs.validation.requirements(40) })

export const jobInputSchema = z.object({
  title: z.string().trim().min(1, { error: () => t.value.jobs.validation.title }).max(160, { error: () => t.value.common.validation.atMost(160) }),
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
  url: z.string().trim().max(2000).regex(/^(https?:\/\/\S+)?$/i, { error: () => t.value.jobs.validation.link }),
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
  /** Jobs in any of these fields of work, and the person's added jobs in none. */
  domains?: string[]
  /** One of the filter's levels, `Any` for none. */
  level?: string
  /** These jobs only, closed ones included. */
  ids?: string[]
}

/** Each field of work's areas, as the jobs service reads them from a job's title (extract.DomainAreas), in its order. */
export const DOMAIN_AREAS = {
  it: ['DevOps', 'Backend', 'Frontend', 'Full stack', 'Mobile', 'Data', 'AI', 'QA', 'Security', 'Embedded', 'Sysadmin', 'Product', 'Design'],
  banking: ['Front office', 'Lending', 'Bank operations', 'Corporate banking'],
  accounting: ['General accounting', 'Payables', 'Receivables', 'Payroll', 'Tax'],
  finance: ['FP&A & controlling', 'Treasury', 'Risk & compliance', 'Audit'],
  insurance: ['Underwriting', 'Claims', 'Insurance sales', 'Actuarial'],
  hr: ['Recruitment', 'HR generalist', 'Compensation & benefits', 'Training & development', 'HR administration'],
} as const satisfies Record<Domain, readonly string[]>
export const AREAS = Object.values(DOMAIN_AREAS).flat()
/** Each field of work's icon. */
export const DOMAIN_ICONS: Record<Domain, string> = {
  it: 'i-lucide-code-xml',
  banking: 'i-lucide-landmark',
  accounting: 'i-lucide-calculator',
  finance: 'i-lucide-trending-up',
  insurance: 'i-lucide-shield-check',
  hr: 'i-lucide-users',
}
export const LEVELS = ['Any', 'Junior', 'Mid', 'Senior'] as const
export const CITIES = ['Iași', 'Bucharest', 'Cluj-Napoca'] as const

export const UNREADABLE_REASONS = ['login_required', 'no_job_data', 'fetch_failed', 'too_large', 'blocked_address'] as const
export type UnreadableReason = (typeof UNREADABLE_REASONS)[number]
