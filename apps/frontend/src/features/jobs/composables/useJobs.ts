import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { jobsApi } from '../api/jobsApi'
import type { BoardOf, Job, JobFilters } from '../api/jobs.schema'
import { cvApi } from '@/features/cv/api/cvApi'

export const jobKeys = {
  all: ['jobs'] as const,
  list: (filters: JobFilters) => ['jobs', 'list', filters] as const,
  detail: (id: string) => ['jobs', 'detail', id] as const,
  boardOf: (url: string) => ['jobs', 'board-of', url] as const,
}

/** The board a job link belongs to; null for a link on no board crashr reads, and a failure stays quiet. */
export function useBoardOfQuery(url: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => jobKeys.boardOf(toValue(url))),
    queryFn: () => jobsApi.boardOf(toValue(url)),
    enabled: computed(() => Boolean(toValue(url))),
    retry: false,
    meta: { suppressToast: true },
  })
}

/** Follows a job link's board: crashr then imports its jobs in Romania. */
export function useFollowBoard() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: jobsApi.follow,
    onError: () => {},
    onSuccess: (board, { url }) => client.setQueryData<BoardOf | null>(jobKeys.boardOf(url), current => current && { ...current, company: board.company, imported: true }),
  })
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

/**
 * Asks Claude to read an imported job's ad when the person saves it, so its requirements are the analysis's
 * by the time they look at it again; the rules read it on import.
 */
export function analyzeImported(job: Pick<Job, 'id' | 'source' | 'requirements_from'>) {
  if (job.source === 'import' && job.requirements_from === 'rules') cvApi.analyze(job.id).catch(() => {})
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
