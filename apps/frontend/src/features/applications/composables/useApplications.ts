import { computed } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { applicationsApi } from '../api/applicationsApi'
import type { Application, ApplicationStatus } from '../api/applications.schema'

export const applicationKeys = {
  list: ['applications', 'list'] as const,
  skips: ['applications', 'skips'] as const,
}

export function useApplicationsQuery() {
  return useQuery({ queryKey: applicationKeys.list, queryFn: applicationsApi.list })
}

/** The status of each tracked job, by job id. */
export function useApplicationStatuses() {
  const query = useApplicationsQuery()
  const statuses = computed(() => new Map((query.data.value ?? []).map(item => [item.job_id, item.status])))
  return { query, statuses }
}

export function useSkipsQuery() {
  return useQuery({ queryKey: applicationKeys.skips, queryFn: applicationsApi.skips })
}

/**
 * The applications' and skips' mutations. They change the cache first, so the deck moves on at
 * once, and put it back when the service refuses.
 */
export function useApplicationMutations() {
  const client = useQueryClient()

  function optimistic<T>(key: readonly unknown[], change: (current: T) => T) {
    return async () => {
      await client.cancelQueries({ queryKey: key })
      const previous = client.getQueryData<T>(key)
      if (previous !== undefined) client.setQueryData<T>(key, change(previous))
      return { previous }
    }
  }
  function rollback<T>(key: readonly unknown[]) {
    return (_error: unknown, _variables: unknown, context: { previous?: T } | undefined) => {
      if (context?.previous !== undefined) client.setQueryData(key, context.previous)
    }
  }
  const settle = (key: readonly unknown[]) => () => client.invalidateQueries({ queryKey: key })

  const setStatus = useMutation({
    mutationFn: ({ jobId, status }: { jobId: string, status: ApplicationStatus }) => applicationsApi.setStatus(jobId, status),
    onMutate: ({ jobId, status }) => optimistic<Application[]>(applicationKeys.list, list => [
      { job_id: jobId, status, updated_at: new Date().toISOString() },
      ...list.filter(item => item.job_id !== jobId),
    ])(),
    onError: rollback<Application[]>(applicationKeys.list),
    onSettled: settle(applicationKeys.list),
  })
  const remove = useMutation({
    mutationFn: (jobId: string) => applicationsApi.remove(jobId),
    onMutate: jobId => optimistic<Application[]>(applicationKeys.list, list => list.filter(item => item.job_id !== jobId))(),
    onError: rollback<Application[]>(applicationKeys.list),
    onSettled: settle(applicationKeys.list),
  })
  const skip = useMutation({
    mutationFn: (jobId: string) => applicationsApi.skip(jobId),
    onMutate: jobId => optimistic<string[]>(applicationKeys.skips, ids => [...ids.filter(id => id !== jobId), jobId])(),
    onError: rollback<string[]>(applicationKeys.skips),
    onSettled: settle(applicationKeys.skips),
  })
  const unskip = useMutation({
    mutationFn: (jobId: string) => applicationsApi.unskip(jobId),
    onMutate: jobId => optimistic<string[]>(applicationKeys.skips, ids => ids.filter(id => id !== jobId))(),
    onError: rollback<string[]>(applicationKeys.skips),
    onSettled: settle(applicationKeys.skips),
  })
  const clearSkips = useMutation({
    mutationFn: () => applicationsApi.clearSkips(),
    onMutate: () => optimistic<string[]>(applicationKeys.skips, () => [])(),
    onError: rollback<string[]>(applicationKeys.skips),
    onSettled: settle(applicationKeys.skips),
  })

  return { setStatus, remove, skip, unskip, clearSkips }
}
