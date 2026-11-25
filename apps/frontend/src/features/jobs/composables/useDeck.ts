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

/** The jobs the filters find, best match first, without the ones already saved or skipped. */
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
    return (jobs.data.value?.items ?? [])
      .filter(job => !tracked.has(job.id) && !skipped.has(job.id))
      .map(job => ({ ...job, ...matchJob(job, skills) }))
      .sort((a, b) => b.score - a.score)
  })
  const loading = computed(() => jobs.isPending.value || applications.isPending.value || skips.isPending.value)

  return { deck, loading, error: computed(() => jobs.isError.value) }
}
