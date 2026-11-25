<script setup lang="ts">
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { Cv } from '../api/cv.schema'

defineProps<{ tailoring: NonNullable<Cv['tailoring']> }>()

const STRENGTH: Record<string, { label: string, tone: string }> = {
  direct: { label: 'Proven', tone: 'bg-hoply-600 text-white' },
  partial: { label: 'Partly', tone: 'bg-hoply-900 text-hoply-300' },
  transferable: { label: 'Transferable', tone: 'border border-dashed border-hoply-500 text-hoply-300' },
  none: { label: 'Not shown', tone: 'bg-accented text-muted' },
}
</script>

<template>
  <section class="flex flex-col gap-2.5 rounded-2xl border border-default bg-elevated p-4" aria-label="How we tailored it" data-test="tailoring">
    <SectionLabel as="h2">How we tailored it</SectionLabel>
    <p class="m-0 text-sm leading-snug font-semibold">{{ tailoring.positioning }}</p>
    <ul v-if="tailoring.priorities.length" class="flex flex-col gap-1.5">
      <li v-for="priority in tailoring.priorities" :key="priority.theme" class="flex items-center justify-between gap-2 text-[13px] font-medium">
        <span>{{ priority.theme }}</span>
        <span class="shrink-0 rounded-md px-2 py-0.5 text-[11px] font-extrabold tracking-wider uppercase" :class="STRENGTH[priority.strength]?.tone">{{ STRENGTH[priority.strength]?.label }}</span>
      </li>
    </ul>
    <p v-if="tailoring.close.length" class="m-0 text-[13px] leading-snug font-medium text-muted">
      Not named in the CV, shown through the kind of work instead: {{ tailoring.close.map(item => `${item.skill} as ${item.category || 'related'} experience with ${item.via.join(', ')}`).join('; ') }}.
    </p>
    <details v-if="tailoring.changes.length" class="text-[13px] font-medium text-muted">
      <summary class="cursor-pointer font-bold text-toned">What the review changed</summary>
      <ul class="mt-1.5 flex list-disc flex-col gap-1 ps-4">
        <li v-for="change in tailoring.changes" :key="change">{{ change }}</li>
      </ul>
    </details>
  </section>
</template>
