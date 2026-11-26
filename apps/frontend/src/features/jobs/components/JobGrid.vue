<script setup lang="ts">
import { computed, ref } from 'vue'
import KeywordChips from '@/features/common/components/KeywordChips.vue'
import ScoreTile from '@/features/common/components/ScoreTile.vue'
import type { Decision, ScoredJob } from '../composables/useDeck'
import { placeOf, sourceLine, tagline } from '../match'
import CaughtUp from './CaughtUp.vue'

/** The jobs as a grid of cards, best match first, a page at a time; the same decisions as the swipe deck. */
const props = defineProps<{ deck: ScoredJob[], toast: string }>()
const emit = defineEmits<{
  decide: [decision: Decision, job: ScoredJob]
  details: [job: ScoredJob]
  undo: []
  changeFilters: []
  resetSkipped: []
}>()

const PAGE = 24
const shown = ref(PAGE)
const visible = computed(() => props.deck.slice(0, shown.value))
</script>

<template>
  <template v-if="deck.length">
    <ul class="m-0 grid grid-cols-1 gap-3.5 p-0 sm:grid-cols-2 min-[900px]:grid-cols-[repeat(auto-fill,minmax(300px,1fr))]">
      <li
        v-for="job in visible"
        :key="job.id"
        :aria-label="`${job.title} at ${job.company}`"
        data-test="job-tile"
        class="flex list-none flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-3.5"
      >
        <div class="flex items-start gap-3">
          <span class="flex min-w-0 flex-1 flex-col gap-0.5">
            <button type="button" class="cursor-pointer self-start text-left text-[17px] leading-tight font-extrabold hover:underline" @click="emit('details', job)">
              {{ job.title }}
            </button>
            <span class="text-sm font-medium text-muted">{{ [job.company, placeOf(job)].filter(Boolean).join(' · ') }}</span>
            <span class="text-xs font-bold text-toned">{{ [tagline(job), sourceLine(job)].filter(Boolean).join(' · ') }}</span>
          </span>
          <ScoreTile :score="job.score" size="sm" />
        </div>
        <KeywordChips :matched="job.matched.slice(0, 3)" :related="job.related.slice(0, 1)" :missing="job.missing.slice(0, 2)" />
        <div class="mt-auto flex flex-wrap items-center gap-2 pt-1">
          <UButton icon="i-lucide-heart" label="Save" size="sm" @click="emit('decide', 'save', job)" />
          <UButton label="Tailor CV" color="neutral" size="sm" @click="emit('decide', 'tailor', job)" />
          <UButton label="Details" color="neutral" variant="outline" size="sm" @click="emit('details', job)" />
          <UButton icon="i-lucide-x" aria-label="Skip" title="Skip" color="neutral" variant="ghost" size="sm" class="ms-auto" @click="emit('decide', 'skip', job)" />
        </div>
      </li>
    </ul>
    <UButton
      v-if="deck.length > shown"
      :label="`Show ${Math.min(PAGE, deck.length - shown)} more`"
      color="neutral"
      variant="outline"
      block
      class="justify-center"
      @click="shown += PAGE"
    />
    <div
      v-if="toast"
      role="status"
      class="sticky bottom-3 z-[3] flex min-h-11 items-center justify-between gap-3 rounded-[10px] bg-ink-800 ps-4 pe-1.5 text-sm font-bold"
    >
      <span class="truncate">{{ toast }}</span>
      <UButton variant="ghost" color="neutral" label="Undo" class="text-hoply-400" @click="emit('undo')" />
    </div>
  </template>
  <CaughtUp v-else @change-filters="emit('changeFilters')" @reset-skipped="emit('resetSkipped')" />
</template>
