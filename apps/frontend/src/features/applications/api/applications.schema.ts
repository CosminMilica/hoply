import { z } from 'zod'

export const APPLICATION_STATUSES = ['saved', 'cv_ready', 'applied'] as const
export type ApplicationStatus = (typeof APPLICATION_STATUSES)[number]

export const STATUS_LABELS: Record<ApplicationStatus, string> = {
  saved: 'Saved',
  cv_ready: 'CV ready',
  applied: 'Applied',
}

export const applicationSchema = z.object({
  job_id: z.string(),
  status: z.enum(APPLICATION_STATUSES),
  updated_at: z.string(),
})

export type Application = z.infer<typeof applicationSchema>

export const applicationListSchema = z.object({ items: z.array(applicationSchema) })

export const skipListSchema = z.object({ job_ids: z.array(z.string()) })
