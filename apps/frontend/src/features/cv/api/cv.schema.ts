import { z } from 'zod'

export const cvRequestSchema = z.object({
  job_id: z.string(),
  /** The roles left out of this CV. */
  skip_roles: z.array(z.string()),
  /** The matched skills to put up front, at most 3; empty takes the first three. */
  focus: z.array(z.string()).max(3),
  /** The key responsibilities the person confirmed for this job (indexes into the job's responsibilities). */
  confirmed: z.array(z.number().int().nonnegative()),
})

export type CvRequest = z.infer<typeof cvRequestSchema>

const section = z.object({ meta: z.string(), bullets: z.array(z.string()) })

export const cvDocumentSchema = z.object({
  name: z.string(),
  headline: z.string(),
  contact: z.array(z.string()),
  links: z.array(z.string()),
  summary: z.string(),
  skills: z.array(z.string()),
  experience: z.array(section.extend({ role: z.string() })),
  projects: z.array(section.extend({ name: z.string() })),
  education: z.array(z.object({ degree: z.string(), meta: z.string() })),
  certifications: z.array(z.string()),
  languages: z.string(),
  additional: z.array(z.string()),
})

export type CvDocument = z.infer<typeof cvDocumentSchema>

export const atsCheckSchema = z.object({ ok: z.boolean(), label: z.string(), hint: z.string() })

export type AtsCheck = z.infer<typeof atsCheckSchema>

export const CV_STATUSES = ['queued', 'writing', 'checking', 'ready', 'failed'] as const
export type CvStatus = (typeof CV_STATUSES)[number]

const atsSchema = z.object({ passed: z.number(), total: z.number(), checks: z.array(atsCheckSchema) })
const previewSchema = z.object({ from: z.string(), to: z.string() })

/** A CV as the cv service returns it: written asynchronously, so the written parts are null until it is ready. */
export const cvSchema = z.object({
  id: z.string(),
  job_id: z.string(),
  status: z.enum(CV_STATUSES),
  created_at: z.string(),
  updated_at: z.string(),
  job_title: z.string(),
  company: z.string(),
  file_name: z.string(),
  highlights: z.array(z.string()).nullable(),
  profile: cvDocumentSchema.nullable(),
  tailored: cvDocumentSchema.nullable(),
  ats: atsSchema.nullable(),
  /** The first highlight before and after the rewrite, from the checking step on. */
  preview: previewSchema.nullable(),
  /** How Claude tailored it: the angle, the job's priorities and how strongly the person's facts prove each, the review's changes. */
  tailoring: z.object({
    positioning: z.string(),
    priorities: z.array(z.object({ theme: z.string(), strength: z.enum(['direct', 'partial', 'transferable', 'none']) })),
    close: z.array(z.object({ skill: z.string(), via: z.array(z.string()), category: z.string().default('') })).default([]),
    changes: z.array(z.string()),
  }).nullable().default(null),
  /** Who wrote it, once it is ready: Claude, or the rules engine (and then `rules_reason` says why). */
  written_by: z.enum(['claude', 'rules']).nullable(),
  model: z.string(),
  /** How many of Claude's sentences the checks replaced. */
  refused: z.number().int(),
  rules_reason: z.enum(['', 'off', 'no_token', 'login_needed', 'usage_limit', 'claude_failed']),
  limit_until: z.string().nullable(),
})

export type Cv = z.infer<typeof cvSchema>

/** A CV whose writing has finished. */
export type ReadyCv = Cv & { status: 'ready', highlights: string[], profile: CvDocument, tailored: CvDocument, ats: NonNullable<Cv['ats']> }

export const isReady = (cv: Cv | null | undefined): cv is ReadyCv => cv?.status === 'ready'

export const cvListSchema = z.object({ items: z.array(cvSchema) })

export const prepQuestionsSchema = z.object({
  job_id: z.string(),
  base_score: z.number(),
  covered: z.array(z.string()),
  gaps: z.array(z.object({ skill: z.string(), ask: z.string(), example: z.string() })),
  /** Requirements the person has not used but knows a close equivalent of: the CV mentions them as transferable. */
  related: z.array(z.object({ skill: z.string(), via: z.array(z.string()), note: z.string() })).default([]),
  responsibilities: z.array(z.object({
    index: z.number().int().nonnegative(),
    text: z.string(),
    question: z.string(),
    prefill: z.string(),
  })),
})

export type PrepQuestions = z.infer<typeof prepQuestionsSchema>

export const probeQuestionSchema = z.object({
  theme: z.string(),
  question: z.string(),
  why: z.string(),
  example: z.string(),
  /** The fact or the role the question is about, as the person wrote it; empty for the person as a whole. */
  about: z.string(),
  /** Where the answer goes: `experience:<id>`, `projects:<id>` or `other`. */
  place: z.string(),
})

export type ProbeQuestion = z.infer<typeof probeQuestionSchema>

/** What Claude asks to make the CV stronger for one job: ready, still being asked for, or unavailable. */
export const probeSchema = z.object({
  status: z.enum(['ready', 'pending', 'unavailable']),
  questions: z.array(probeQuestionSchema),
})

export type Probe = z.infer<typeof probeSchema>
