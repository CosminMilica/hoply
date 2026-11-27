<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import ScoreTile from '@/features/common/components/ScoreTile.vue'
import { placeOf } from '@/features/jobs/match'
import { useApplicationMutations } from '../composables/useApplications'
import { useTrackedJobs } from '../composables/useTrackedJobs'

const SORTS = [{ id: 'match', label: 'Best match' }, { id: 'newest', label: 'Newest' }] as const

const router = useRouter()
const sort = ref<(typeof SORTS)[number]['id']>('match')
const { tracked, loading, error } = useTrackedJobs()
const { remove } = useApplicationMutations()

// The service lists the newest first; the best match puts the highest score first.
const saved = computed(() => {
  const entries = tracked.value.filter(entry => entry.status === 'saved')
  return sort.value === 'match' ? [...entries].sort((a, b) => b.score - a.score) : entries
})

/** "Missing: GraphQL · Close: Argo CD", or that every keyword is covered. */
function keywordLine(entry: { missing: string[], related: { skill: string }[] }) {
  const parts = [
    entry.missing.length ? `Missing: ${entry.missing.join(', ')}` : '',
    entry.related.length ? `Close: ${entry.related.map(item => item.skill).join(', ')}` : '',
  ].filter(Boolean)
  return parts.length ? parts.join(' · ') : 'Every keyword is covered'
}
</script>

<template>
  <div class="mx-auto flex w-full max-w-[1080px] flex-col gap-4">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div class="flex flex-col gap-1">
        <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">Saved jobs</h1>
        <p class="m-0 text-[15px] leading-snug font-medium text-muted">Your shortlist. Prepare a CV for the ones worth it, and they move to Applications.</p>
      </div>
      <div v-if="saved.length > 1" role="group" aria-label="Sort" class="flex gap-0.5 rounded-[10px] bg-accented p-0.5">
        <UButton
          v-for="option in SORTS"
          :key="option.id"
          :label="option.label"
          :aria-pressed="sort === option.id"
          size="sm"
          color="neutral"
          :variant="sort === option.id ? 'solid' : 'ghost'"
          @click="sort = option.id"
        />
      </div>
    </div>

    <USkeleton v-if="loading" class="h-40 w-full rounded-[14px]" />
    <UAlert v-else-if="error" color="error" variant="subtle" title="We could not load your saved jobs" description="Try again in a moment." />
    <ul v-else-if="saved.length" class="grid grid-cols-1 gap-3.5 sm:grid-cols-2 min-[900px]:grid-cols-[repeat(auto-fill,minmax(320px,1fr))]">
      <li v-for="entry in saved" :key="entry.job.id" class="flex flex-col gap-3 rounded-[14px] border border-default bg-elevated p-4" data-test="saved-job">
        <div class="flex items-start gap-3">
          <span class="flex min-w-0 flex-1 flex-col gap-0.5">
            <h2 class="m-0 text-[17px] leading-tight font-extrabold">{{ entry.job.title }}</h2>
            <span class="text-sm font-medium text-muted">{{ [entry.job.company, placeOf(entry.job)].filter(Boolean).join(' · ') }}</span>
            <span v-if="entry.since" class="text-xs font-medium text-dimmed" data-test="since">{{ entry.since }}</span>
          </span>
          <ScoreTile :score="entry.score" size="sm" />
        </div>
        <p class="m-0 text-[13px] font-medium text-muted">{{ keywordLine(entry) }}</p>
        <div class="mt-auto flex items-center gap-2">
          <UButton label="Prepare CV" block class="flex-1 justify-center" @click="router.push({ name: 'hoply-prep', params: { id: entry.job.id } })" />
          <UButton :to="{ name: 'hoply-job', params: { id: entry.job.id } }" label="Details" color="neutral" variant="outline" />
          <UButton icon="i-lucide-x" aria-label="Remove from saved" color="neutral" variant="outline" @click="remove.mutate(entry.job.id)" />
        </div>
      </li>
    </ul>
    <div v-else class="flex flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-[22px]">
      <p class="m-0 text-[15px] leading-normal font-medium text-muted">
        {{ tracked.length ? 'Every saved job has a CV now. Find more jobs, or add one you found.' : 'Nothing saved yet. Add a job you found, from its link or its text, and it lands here.' }}
      </p>
      <UButton :to="{ name: 'hoply-job-add', query: { from: 'saved' } }" icon="i-lucide-plus" label="Add a job" size="lg" block class="justify-center" />
    </div>
  </div>
</template>
