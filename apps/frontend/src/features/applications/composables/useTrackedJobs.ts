import { computed, watch } from 'vue'
import { keepPreviousData, useQuery } from '@tanstack/vue-query'
import { cvApi } from '@/features/cv/api/cvApi'
import { isReady } from '@/features/cv/api/cv.schema'
import { cvKeys } from '@/features/cv/composables/useCv'
import { jobsApi } from '@/features/jobs/api/jobsApi'
import { jobKeys } from '@/features/jobs/composables/useJobs'
import { matchJob } from '@/features/jobs/match'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { knownSkills } from '@/features/profile/skills'
import { statusSince } from '../api/applications.schema'
import { useApplicationMutations, useApplicationsQuery } from './useApplications'

/** The most job ids the jobs service takes in one request (services/jobs MaxIDs). */
const MAX_IDS = 200

/** The jobs the person tracks, each with its status and how it matches the profile's skills. */
export function useTrackedJobs() {
  const applications = useApplicationsQuery()
  const ids = computed(() => (applications.data.value ?? []).map(application => application.job_id).sort())
  const jobs = useQuery({
    queryKey: computed(() => jobKeys.list({ ids: ids.value })),
    queryFn: async () => {
      const batches = []
      for (let at = 0; at < ids.value.length; at += MAX_IDS) batches.push(ids.value.slice(at, at + MAX_IDS))
      const answers = await Promise.all(batches.map(batch => jobsApi.list({ ids: batch })))
      return { items: answers.flatMap(answer => answer.items), total: answers.reduce((sum, answer) => sum + answer.total, 0) }
    },
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

  // A CV finishes in the background: a saved job whose CV is ready moves on to CV ready, even when the person left
  // Prepare your CV before it was written.
  const cvs = useQuery({ queryKey: cvKeys.all, queryFn: cvApi.all, meta: { suppressToast: true } })
  const { setStatus } = useApplicationMutations()
  const moved = new Set<string>()
  watch([() => applications.data.value, () => cvs.data.value], ([tracked, written]) => {
    const ready = new Set((written ?? []).filter(isReady).map(cv => cv.job_id))
    for (const application of tracked ?? []) {
      if (application.status === 'saved' && ready.has(application.job_id) && !moved.has(application.job_id)) {
        moved.add(application.job_id)
        setStatus.mutate({ jobId: application.job_id, status: 'cv_ready' })
      }
    }
  }, { immediate: true })
  const error = computed(() => applications.isError.value || jobs.isError.value)
  return { tracked, skills, loading, error }
}
