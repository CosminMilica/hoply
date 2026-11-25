import { jobSchema, type Job } from '@/features/jobs/api/jobs.schema'
import { toEngineJob, type EngineJob } from '@/test/cv-engine/job'
import { CATALOG_JOBS } from './jobs-seed'

/** A catalog job (0: Snowflake at Capgemini, 1: DevOps at Bitdefender, 2: Amazon, 3: Carpathia) as the app reads it. */
export const catalogJob = (index: number, overrides: Partial<Job> = {}): Job => ({ ...jobSchema.parse(CATALOG_JOBS[index]), ...overrides })

export const engineJob = (index: number, overrides: Partial<Job> = {}): EngineJob => toEngineJob(catalogJob(index, overrides))
