import { z } from 'zod'
import { formatDate, t } from '@/features/common/i18n'

export const APPLICATION_STATUSES = ['saved', 'cv_ready', 'applied'] as const
export type ApplicationStatus = (typeof APPLICATION_STATUSES)[number]

/** A status's name in the app's language. */
export const statusLabel = (status: ApplicationStatus) => t.value.applications.status[status]

/** The statuses' names by key, read in the app's language each time. */
export const STATUS_LABELS: Readonly<Record<ApplicationStatus, string>> = {
  get saved() { return statusLabel('saved') },
  get cv_ready() { return statusLabel('cv_ready') },
  get applied() { return statusLabel('applied') },
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
  return t.value.applications.since[application.status](formatDate(date, { day: 'numeric', month: 'short' }))
}

export const skipListSchema = z.object({ job_ids: z.array(z.string()) })
