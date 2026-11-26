<script setup lang="ts">
/** Finding jobs: filters, then the swipe deck. */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useApplicationMutations } from '@/features/applications/composables/useApplications'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import JobFilters from './JobFilters.vue'
import JobFilterBar from './JobFilterBar.vue'
import JobGrid from './JobGrid.vue'
import SwipeDeck from './SwipeDeck.vue'
import { useDeck, type Decision, type ScoredJob } from '../composables/useDeck'
import { analyzeImported } from '../composables/useJobs'
import { useJobSearchStore } from '../stores/search'

const UNDO_MS = 3500
const VIEWS = [{ id: 'grid', label: 'Grid', icon: 'i-lucide-layout-grid' }, { id: 'swipe', label: 'Swipe', icon: 'i-lucide-layers' }] as const
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
// A view this device remembered from before the grid (the list) shows as the grid.
const shownView = computed(() => (search.view === 'swipe' ? 'swipe' : 'grid'))
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
  analyzeImported(job)
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

// Switching views keeps the filters in use and goes on to the jobs.
function setView(view: 'grid' | 'swipe') {
  search.view = view
  search.stage = 'deck'
}

function show() {
  last.value = null
  search.stage = 'deck'
}

onBeforeUnmount(() => window.clearTimeout(timer))
</script>

<template>
  <JobFilters v-if="shownView === 'swipe' && search.stage === 'filters'" :count="deck.length" @show="show" />
  <div v-else class="mx-auto flex h-full w-full flex-col gap-2.5" :class="search.view === 'swipe' ? 'min-h-[560px] max-w-[560px]' : 'max-w-[1080px]'">
    <div class="flex items-center justify-between gap-2.5">
      <div class="flex min-w-0 flex-col gap-px">
        <h1 v-if="shownView === 'grid'" class="m-0 truncate text-[28px] leading-tight font-extrabold tracking-tight">Find jobs</h1>
        <h1 v-else class="m-0 truncate text-[15px] font-extrabold">{{ search.summary }}</h1>
        <span class="text-[13px] font-medium text-muted">{{ results }}</span>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <div role="group" aria-label="View" class="flex gap-0.5 rounded-[10px] bg-accented p-0.5">
          <UButton
            v-for="option in VIEWS"
            :key="option.id"
            :icon="option.icon"
            :aria-label="option.label"
            :aria-pressed="shownView === option.id"
            :title="option.label"
            size="sm"
            color="neutral"
            :variant="shownView === option.id ? 'solid' : 'ghost'"
            @click="setView(option.id)"
          />
        </div>
        <UButton v-if="shownView === 'swipe'" label="Filters" color="neutral" variant="outline" @click="search.stage = 'filters'" />
        <UButton v-else :to="{ name: 'hoply-job-add' }" icon="i-lucide-plus" label="Add a job" color="neutral" variant="outline" />
      </div>
    </div>
    <JobFilterBar v-if="shownView === 'grid'" />
    <UAlert v-if="error" color="error" variant="subtle" title="We could not load jobs" />
    <USkeleton v-else-if="loading" class="min-h-[360px] flex-1 rounded-[18px]" />
    <JobGrid
      v-else-if="shownView === 'grid'"
      :deck="deck"
      :toast="toast"
      @decide="onDecide"
      @details="job => router.push({ name: 'hoply-job', params: { id: job.id } })"
      @undo="undo"
      @change-filters="search.reset()"
      @reset-skipped="clearSkips.mutate()"
    />
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
