import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { cvApi } from '../api/cvApi'
import type { Cv } from '../api/cv.schema'

export const cvKeys = {
  questions: (jobId: string) => ['cv', 'questions', jobId] as const,
  forJob: (jobId: string) => ['cv', 'job', jobId] as const,
}

/** Asked once per visit to Prepare your CV: the answers in progress keep the questions they had. */
export function useCvQuestionsQuery(jobId: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => cvKeys.questions(toValue(jobId))),
    queryFn: () => cvApi.questions(toValue(jobId)),
    enabled: computed(() => Boolean(toValue(jobId))),
    staleTime: Infinity,
    refetchOnMount: 'always',
  })
}

export function useLatestCvQuery(jobId: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => cvKeys.forJob(toValue(jobId))),
    queryFn: () => cvApi.forJob(toValue(jobId)),
    enabled: computed(() => Boolean(toValue(jobId))),
    select: (items: Cv[]) => items[0] ?? null,
  })
}

export function useGenerateCv() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: cvApi.generate,
    onSuccess: (cv) => {
      client.setQueryData<Cv[]>(cvKeys.forJob(cv.job_id), current => [cv, ...(current ?? [])])
    },
  })
}
