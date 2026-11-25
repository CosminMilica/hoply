import { z } from 'zod'

export const APPLICATION_STATUSES = ['saved', 'cv_ready', 'applied'] as const
export type ApplicationStatus = (typeof APPLICATION_STATUSES)[number]

export const STATUS_LABELS: Record<ApplicationStatus, string> = {
  saved: 'Saved',
  cv_ready: 'CV ready',
  applied: 'Applied',
}

const when = z.string().nullish().transform(value => value || null)

export const applicationSchema = z.object({
  job_id: z.string(),
  status: z.enum(APPLICATION_STATUSES),
  saved_at: when,
  cv_ready_at: when,
  applied_at: when,
  created_at: when,
  updated_at: when,
})

export type Application = z.infer<typeof applicationSchema>

export const applicationListSchema = z.object({ items: z.array(applicationSchema) })

/** When the application entered its current status: "Saved 25 Nov". Empty without that time. */
export function statusSince(application: Pick<Application, 'status' | 'saved_at' | 'cv_ready_at' | 'applied_at'>): string {
  const at = { saved: application.saved_at, cv_ready: application.cv_ready_at, applied: application.applied_at }[application.status]
  const date = at ? new Date(at) : null
  if (!date || Number.isNaN(date.getTime())) return ''
  return `${STATUS_LABELS[application.status]} ${date.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })}`
}

export const skipListSchema = z.object({ job_ids: z.array(z.string()) })
