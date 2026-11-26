import { computed } from 'vue'
import { useApplicationsQuery, useSkipsQuery } from '@/features/applications/composables/useApplications'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { knownSkills } from '@/features/profile/skills'
import type { Job } from '../api/jobs.schema'
import { matchJob, type JobMatch } from '../match'
import { useJobSearchStore } from '../stores/search'
import { useJobsQuery } from './useJobs'

export type ScoredJob = Job & JobMatch
export type Decision = 'save' | 'skip' | 'tailor'

/** A company posts one job once per city: those postings are one job in the deck. */
const sameJob = (job: Pick<Job, 'company' | 'title'>) => `${job.company.trim().toLowerCase()}\n${job.title.trim().toLowerCase()}`

/** The jobs the filters find, best match first, one per company and title, without the ones already saved or skipped. */
export function useDeck() {
  const search = useJobSearchStore()
  const jobs = useJobsQuery(() => search.filters)
  const profile = useProfileQuery()
  const applications = useApplicationsQuery()
  const skips = useSkipsQuery()

  const deck = computed<ScoredJob[]>(() => {
    const tracked = new Set((applications.data.value ?? []).map(item => item.job_id))
    const skipped = new Set(skips.data.value ?? [])
    const skills = profile.data.value ? knownSkills(profile.data.value) : []
    const items = jobs.data.value?.items ?? []
    const decided = new Set(items.filter(job => tracked.has(job.id) || skipped.has(job.id)).map(sameJob))
    const shown = new Set<string>()
    return items
      .filter(job => !decided.has(sameJob(job)))
      .map(job => ({ ...job, ...matchJob(job, skills) }))
      .sort((a, b) => b.score - a.score)
      .filter((job) => {
        const key = sameJob(job)
        if (shown.has(key)) return false
        shown.add(key)
        return true
      })
  })
  const loading = computed(() => jobs.isPending.value || applications.isPending.value || skips.isPending.value)

  return { deck, loading, error: computed(() => jobs.isError.value) }
}
