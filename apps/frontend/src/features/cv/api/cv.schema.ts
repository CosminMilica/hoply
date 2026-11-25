import { z } from 'zod'
import { profileSchema } from '@/features/profile/api/profile.schema'

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

export const cvSchema = z.object({
  id: z.string(),
  job_id: z.string(),
  created_at: z.string(),
  job_title: z.string(),
  company: z.string(),
  file_name: z.string(),
  highlights: z.array(z.string()),
  profile: cvDocumentSchema,
  tailored: cvDocumentSchema,
  ats: z.object({ passed: z.number(), total: z.number(), checks: z.array(atsCheckSchema) }),
  preview: z.object({ from: z.string(), to: z.string() }).nullable(),
  /** The profile the CV was made from, as it was then. */
  snapshot: profileSchema,
})

export type Cv = z.infer<typeof cvSchema>

export const cvListSchema = z.object({ items: z.array(cvSchema) })

export const prepQuestionsSchema = z.object({
  job_id: z.string(),
  base_score: z.number(),
  covered: z.array(z.string()),
  gaps: z.array(z.object({ skill: z.string(), ask: z.string(), example: z.string() })),
  responsibilities: z.array(z.object({
    index: z.number().int().nonnegative(),
    text: z.string(),
    question: z.string(),
    prefill: z.string(),
  })),
})

export type PrepQuestions = z.infer<typeof prepQuestionsSchema>
