import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { jobsApi } from '../api/jobsApi'
import type { Job, JobFilters } from '../api/jobs.schema'
import { cvApi } from '@/features/cv/api/cvApi'

export const jobKeys = {
  all: ['jobs'] as const,
  list: (filters: JobFilters) => ['jobs', 'list', filters] as const,
  detail: (id: string) => ['jobs', 'detail', id] as const,
}

export function useJobsQuery(filters: MaybeRefOrGetter<JobFilters> = {}) {
  return useQuery({
    queryKey: computed(() => jobKeys.list(toValue(filters))),
    queryFn: () => jobsApi.list(toValue(filters)),
    placeholderData: keepPreviousData,
  })
}

export function useJobQuery(id: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => jobKeys.detail(toValue(id))),
    queryFn: () => jobsApi.get(toValue(id)),
    enabled: computed(() => Boolean(toValue(id))),
  })
}

/** Reading an ad, and saving the job the person checked; a saved job joins the lists. */
export function useAddJob() {
  const client = useQueryClient()
  return {
    read: useMutation({ mutationFn: jobsApi.read, onError: () => {} }),
    create: useMutation({
      mutationFn: jobsApi.create,
      onError: () => {},
      onSuccess: (job: Job) => {
        cvApi.analyze(job.id).catch(() => {})
        client.setQueryData(jobKeys.detail(job.id), job)
        return client.invalidateQueries({ queryKey: ['jobs', 'list'] })
      },
    }),
  }
}
