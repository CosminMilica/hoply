import { computed } from 'vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { jobsApi } from '@/features/jobs/api/jobsApi'
import { jobKeys } from '@/features/jobs/composables/useJobs'
import { matchJob } from '@/features/jobs/match'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { knownSkills } from '@/features/profile/skills'
import { statusSince } from '../api/applications.schema'
import { useApplicationsQuery } from './useApplications'

/** The jobs the person tracks, each with its status and how it matches the profile's skills. */
export function useTrackedJobs() {
  const applications = useApplicationsQuery()
  const ids = computed(() => (applications.data.value ?? []).map(application => application.job_id).sort())
  const jobs = useQuery({
    queryKey: computed(() => jobKeys.list({ ids: ids.value })),
    queryFn: () => jobsApi.list({ ids: ids.value }),
    enabled: computed(() => ids.value.length > 0),
    placeholderData: keepPreviousData,
  })
  const profile = useProfileQuery()
  const skills = computed(() => (profile.data.value ? knownSkills(profile.data.value) : []))

  const tracked = computed(() => {
    const byId = new Map((jobs.data.value ?? { items: [] }).items.map(job => [job.id, job]))
    return (applications.data.value ?? []).flatMap((application) => {
      const job = byId.get(application.job_id)
      return job ? [{ job, status: application.status, since: statusSince(application), ...matchJob(job, skills.value) }] : []
    })
  })
  const loading = computed(() => applications.isPending.value || (ids.value.length > 0 && jobs.isPending.value) || profile.isPending.value)
  return { tracked, skills, loading }
}
