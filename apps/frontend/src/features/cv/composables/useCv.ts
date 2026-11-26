import { computed, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { cvApi } from '../api/cvApi'
import { isReady, type Cv } from '../api/cv.schema'

export const cvKeys = {
  questions: (jobId: string) => ['cv', 'questions', jobId] as const,
  all: ['cv', 'all'] as const,
  forJob: (jobId: string) => ['cv', 'job', jobId] as const,
  one: (id: string) => ['cv', 'one', id] as const,
  probe: (jobId: string) => ['cv', 'probe', jobId] as const,
  letter: (cvId: string) => ['cv', 'letter', cvId] as const,
  profileProbe: ['cv', 'probe', 'profile'] as const,
}

/** How often a letter being written is asked for again. */
export const LETTER_POLL_MS = 2000

/** The CV's cover letter and messages, followed while they are being written. */
export function useLetterQuery(cvId: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => cvKeys.letter(toValue(cvId))),
    queryFn: () => cvApi.letter(toValue(cvId)),
    enabled: computed(() => Boolean(toValue(cvId))),
    refetchInterval: current => (current.state.data?.status === 'writing' ? LETTER_POLL_MS : false),
  })
}

/** Asks for the CV's cover letter and messages, addressed to the recruiter when named. */
export function useWriteLetter(cvId: MaybeRefOrGetter<string>) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (recruiter: string) => cvApi.writeLetter(toValue(cvId), recruiter),
    onSuccess: letter => client.setQueryData(cvKeys.letter(toValue(cvId)), letter),
  })
}

/** How often questions still being asked for are asked for again. */
export const PROBE_POLL_MS = 2000

/** Claude's questions for the job, followed while they are being asked for. */
export function useCvProbeQuery(jobId: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => cvKeys.probe(toValue(jobId))),
    queryFn: () => cvApi.probe(toValue(jobId)),
    enabled: computed(() => Boolean(toValue(jobId))),
    staleTime: Infinity,
    refetchOnMount: 'always',
    refetchInterval: current => (current.state.data?.status === 'pending' ? PROBE_POLL_MS : false),
  })
}

/** The questions that make every CV stronger, followed while they are being asked for; a failure stays quiet. */
export function useProfileProbeQuery() {
  return useQuery({
    queryKey: cvKeys.profileProbe,
    queryFn: () => cvApi.profileProbe(),
    staleTime: Infinity,
    refetchOnMount: 'always',
    retry: false,
    meta: { suppressToast: true },
    refetchInterval: current => (current.state.data?.status === 'pending' ? PROBE_POLL_MS : false),
  })
}

/** How often a CV being written is asked for again. */
export const CV_POLL_MS = 1000

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

/** The newest CV for a job that is ready to show. */
export function useLatestCvQuery(jobId: MaybeRefOrGetter<string>) {
  return useQuery({
    queryKey: computed(() => cvKeys.forJob(toValue(jobId))),
    queryFn: () => cvApi.forJob(toValue(jobId)),
    enabled: computed(() => Boolean(toValue(jobId))),
    select: (items: Cv[]) => items.find(isReady) ?? null,
  })
}

/**
 * One CV, asked for again until it is ready or failed. Once it is, the job's list of CVs holds the
 * finished one, so the CV page shows it at once.
 */
export function useCvProgress(id: MaybeRefOrGetter<string | null>) {
  const client = useQueryClient()
  const query = useQuery({
    queryKey: computed(() => cvKeys.one(toValue(id) ?? '')),
    queryFn: () => cvApi.get(toValue(id)!),
    enabled: computed(() => Boolean(toValue(id))),
    refetchInterval: (current) => {
      const status = current.state.data?.status
      return status === 'ready' || status === 'failed' ? false : CV_POLL_MS
    },
  })
  watch(() => query.data.value, (cv) => {
    if (!cv || cv.status !== 'ready') return
    client.setQueryData<Cv[]>(cvKeys.forJob(cv.job_id), current => [cv, ...(current ?? []).filter(item => item.id !== cv.id)])
  })
  return query
}

export function useGenerateCv() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: cvApi.generate,
    onSuccess: (cv) => {
      client.setQueryData(cvKeys.one(cv.id), cv)
      client.setQueryData<Cv[]>(cvKeys.forJob(cv.job_id), current => [cv, ...(current ?? [])])
    },
  })
}
