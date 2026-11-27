import { computed, toRef } from 'vue'
import { refDebounced } from '@vueuse/core'
import { useApplicationsQuery, useSkipsQuery } from '@/features/applications/composables/useApplications'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { knownSkills } from '@/features/profile/skills'
import type { JobCard } from '../api/jobs.schema'
import { matchJob, type JobMatch } from '../match'
import { useJobSearchStore } from '../stores/search'
import { useJobsQuery } from './useJobs'

export type ScoredJob = JobCard & JobMatch
export type Decision = 'save' | 'skip' | 'tailor'

/** A company posts one job once per city: those postings are one job in the deck. */
const sameJob = (job: Pick<JobCard, 'company' | 'title'>) => `${job.company.trim().toLowerCase()}\n${job.title.trim().toLowerCase()}`

/** How long typing in the search box waits before it searches. */
export const SEARCH_DELAY_MS = 250

/** The jobs the filters find, best match first, one per company and title, without the ones already saved or skipped. */
export function useDeck() {
  const search = useJobSearchStore()
  const query = refDebounced(toRef(search, 'query'), SEARCH_DELAY_MS)
  const jobs = useJobsQuery(() => ({ ...search.filters, q: query.value }))
  const profile = useProfileQuery()
  const applications = useApplicationsQuery()
  const skips = useSkipsQuery()

  // Scoring reads only the jobs and the skills, so saving or skipping a job does not score them all again.
  const scored = computed<ScoredJob[]>(() => {
    const skills = profile.data.value ? knownSkills(profile.data.value) : []
    return (jobs.data.value?.items ?? []).map(job => ({ ...job, ...matchJob(job, skills) })).sort((a, b) => b.score - a.score)
  })
  const deck = computed<ScoredJob[]>(() => {
    const tracked = new Set((applications.data.value ?? []).map(item => item.job_id))
    const skipped = new Set(skips.data.value ?? [])
    const decided = new Set(scored.value.filter(job => tracked.has(job.id) || skipped.has(job.id)).map(sameJob))
    const shown = new Set<string>()
    return scored.value.filter((job) => {
      const key = sameJob(job)
      if (decided.has(key) || shown.has(key)) return false
      shown.add(key)
      return true
    })
  })
  /** The jobs the service sent and how many the filters find, when it sent only the newest of them. */
  const beyond = computed(() => {
    const answer = jobs.data.value
    return answer && answer.total > answer.items.length ? { sent: answer.items.length, total: answer.total } : null
  })
  const loading = computed(() => jobs.isPending.value || applications.isPending.value || skips.isPending.value)

  return { deck, beyond, loading, error: computed(() => jobs.isError.value) }
}
