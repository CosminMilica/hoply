import type { Job } from '@/features/jobs/api/jobs.schema'
import { lookFor, requirementsOf } from '@/features/jobs/match'

/** A job as the tailoring engine reads it. */
export type EngineJob = Job & {
  /** The requirements to match and write around: must-haves, then nice-to-haves. */
  keywords: string[]
  /** The requirements as sentences, for the job's own wording. */
  requirements: string[]
  /** The responsibilities Prepare your CV asks about: the first four. */
  key_responsibilities: number[]
}

export function toEngineJob(job: Job): EngineJob {
  return {
    ...job,
    product: job.product || 'products',
    keywords: requirementsOf(job),
    requirements: lookFor(job),
    key_responsibilities: job.responsibilities.slice(0, 4).map((_, index) => index),
  }
}
