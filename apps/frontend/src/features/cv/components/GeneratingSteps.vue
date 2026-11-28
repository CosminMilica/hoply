<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import ProgressBar from '@/features/common/components/ProgressBar.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { Cv } from '../api/cv.schema'
import HighlightedText from './HighlightedText.vue'

const props = defineProps<{ step: number, title: string, company: string, keywords: string[], matched: string[], preview: Cv['preview'] | undefined, highlights: string[] }>()

const steps = computed(() => t.value.cv.generating.steps.map((label, index) => ({ label, done: props.step > index, now: props.step === index })))
const have = computed(() => new Set(props.matched.map(item => item.toLowerCase())))
</script>

<template>
  <div class="mx-auto flex w-full max-w-[620px] flex-col gap-4" aria-live="polite">
    <div class="flex flex-col gap-1.5">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">{{ t.cv.generating.title }}</h1>
      <p class="m-0 text-base font-medium text-muted">{{ title }} · {{ company }}</p>
    </div>
    <ProgressBar :value="Math.min(100, Math.round((step / 4) * 100))" :label="t.cv.generating.progress" />
    <ol class="flex flex-col gap-3">
      <li v-for="item in steps" :key="item.label" class="flex items-center gap-3 text-base font-bold" :class="item.done || item.now ? 'text-highlighted' : 'text-ink-500'">
        <span v-if="item.done" class="flex size-[26px] shrink-0 items-center justify-center rounded-full bg-hoply-600 text-white">
          <UIcon name="i-lucide-check" class="size-3.5" /><span class="sr-only">{{ t.cv.generating.done }}</span>
        </span>
        <span v-else-if="item.now" class="size-[26px] shrink-0 animate-pulse rounded-full bg-hoply-500"><span class="sr-only">{{ t.cv.generating.now }}</span></span>
        <span v-else class="size-[26px] shrink-0 rounded-full border-2 border-ink-700" />
        {{ item.label }}
      </li>
    </ol>
    <div v-if="step >= 1" class="flex flex-col gap-2 rounded-[14px] border border-default bg-elevated p-3.5">
      <SectionLabel as="h2">{{ t.cv.generating.keywords }}</SectionLabel>
      <ul class="flex flex-wrap gap-1.5">
        <li
          v-for="keyword in keywords"
          :key="keyword"
          class="rounded-lg border-[1.5px] px-2.5 py-1 text-[13px] font-bold"
          :class="have.has(keyword.toLowerCase()) ? 'border-hoply-900 bg-hoply-900 text-hoply-300' : 'border-ink-700 text-toned'"
        >
          {{ keyword }}
        </li>
      </ul>
    </div>
    <div v-if="step >= 3 && preview" class="flex flex-col gap-1.5 rounded-[14px] border border-default bg-elevated p-3.5">
      <SectionLabel as="h2">{{ t.cv.generating.fromProfile }}</SectionLabel>
      <p class="m-0 text-sm leading-normal font-medium text-toned">{{ preview.from }}</p>
      <SectionLabel as="h2" class="mt-1.5">{{ t.cv.generating.rewritten }}</SectionLabel>
      <p class="m-0 text-sm leading-normal"><HighlightedText :text="preview.to" :keywords="highlights" /></p>
    </div>
  </div>
</template>
