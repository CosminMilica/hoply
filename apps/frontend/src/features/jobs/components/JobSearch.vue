<script setup lang="ts">
/**
 * Finding jobs: filters, then the swipe deck. Parked while jobs only come in through Add a job;
 * Find jobs renders it again once the service finds jobs.
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useApplicationMutations } from '@/features/applications/composables/useApplications'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import JobFilters from './JobFilters.vue'
import SwipeDeck from './SwipeDeck.vue'
import { useDeck, type Decision, type ScoredJob } from '../composables/useDeck'
import { useJobSearchStore } from '../stores/search'

const UNDO_MS = 3500
const router = useRouter()
const search = useJobSearchStore()
const { deck, loading, error } = useDeck()
const profile = useProfileQuery()
watch([() => profile.data.value, () => search.stage], ([person, stage]) => {
  if (!person) return
  search.country = person.country.trim()
  if (stage === 'filters') search.prefill(person)
}, { immediate: true })
const { setStatus, remove, skip, unskip, clearSkips } = useApplicationMutations()
const last = ref<{ job: ScoredJob, decision: 'save' | 'skip' } | null>(null)
let timer: number | undefined

const toast = computed(() => (last.value ? `${last.value.decision === 'save' ? 'Saved' : 'Skipped'} · ${last.value.job.title}` : ''))
const results = computed(() => `${deck.value.length} ${deck.value.length === 1 ? 'job' : 'jobs'} left`)

function remember(job: ScoredJob, decision: 'save' | 'skip') {
  window.clearTimeout(timer)
  last.value = { job, decision }
  timer = window.setTimeout(() => {
    if (last.value?.job.id === job.id) last.value = null
  }, UNDO_MS)
}

function onDecide(decision: Decision, job: ScoredJob) {
  if (decision === 'skip') {
    skip.mutate(job.id)
    remember(job, 'skip')
    return
  }
  setStatus.mutate({ jobId: job.id, status: 'saved' })
  if (decision === 'save') remember(job, 'save')
  else router.push({ name: 'hoply-prep', params: { id: job.id } })
}

function undo() {
  const decision = last.value
  if (!decision) return
  last.value = null
  if (decision.decision === 'save') remove.mutate(decision.job.id)
  else unskip.mutate(decision.job.id)
}

function show() {
  last.value = null
  search.stage = 'deck'
}

onBeforeUnmount(() => window.clearTimeout(timer))
</script>

<template>
  <JobFilters v-if="search.stage === 'filters'" :count="deck.length" @show="show" />
  <div v-else class="mx-auto flex h-full min-h-[560px] w-full max-w-[560px] flex-col gap-2.5">
    <div class="flex items-center justify-between gap-2.5">
      <div class="flex min-w-0 flex-col gap-px">
        <h1 class="m-0 truncate text-[15px] font-extrabold">{{ search.summary }}</h1>
        <span class="text-[13px] font-medium text-muted">{{ results }}</span>
      </div>
      <UButton label="Filters" color="neutral" variant="outline" class="shrink-0" @click="search.stage = 'filters'" />
    </div>
    <UAlert v-if="error" color="error" variant="subtle" title="We could not load jobs" />
    <USkeleton v-else-if="loading" class="min-h-[360px] flex-1 rounded-[18px]" />
    <SwipeDeck
      v-else
      :deck="deck"
      :toast="toast"
      @decide="onDecide"
      @details="job => router.push({ name: 'hoply-job', params: { id: job.id } })"
      @undo="undo"
      @change-filters="search.stage = 'filters'"
      @reset-skipped="clearSkips.mutate()"
    />
  </div>
</template>
