<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { Cv } from '../api/cv.schema'

const props = defineProps<{ tailoring: NonNullable<Cv['tailoring']> }>()

const TONE: Record<NonNullable<Cv['tailoring']>['priorities'][number]['strength'], string> = {
  direct: 'bg-hoply-600 text-white',
  partial: 'bg-hoply-900 text-hoply-300',
  transferable: 'border border-dashed border-hoply-500 text-hoply-300',
  none: 'bg-accented text-muted',
}
const close = computed(() => {
  const words = t.value.cv.tailoring
  return words.close(props.tailoring.close.map(item => words.closeItem(item.skill, item.category || words.related, item.via.join(', '))).join('; '))
})
</script>

<template>
  <section class="flex flex-col gap-2.5 rounded-2xl border border-default bg-elevated p-4" :aria-label="t.cv.tailoring.title" data-test="tailoring">
    <SectionLabel as="h2">{{ t.cv.tailoring.title }}</SectionLabel>
    <p class="m-0 text-sm leading-snug font-semibold">{{ tailoring.positioning }}</p>
    <ul v-if="tailoring.priorities.length" class="flex flex-col gap-1.5">
      <li v-for="priority in tailoring.priorities" :key="priority.theme" class="flex items-center justify-between gap-2 text-[13px] font-medium">
        <span>{{ priority.theme }}</span>
        <span class="shrink-0 rounded-md px-2 py-0.5 text-[11px] font-extrabold tracking-wider uppercase" :class="TONE[priority.strength]">{{ t.cv.tailoring.strength[priority.strength] }}</span>
      </li>
    </ul>
    <p v-if="tailoring.close.length" class="m-0 text-[13px] leading-snug font-medium text-muted">
      {{ close }}
    </p>
    <details v-if="tailoring.changes.length" class="text-[13px] font-medium text-muted">
      <summary class="cursor-pointer font-bold text-toned">{{ t.cv.tailoring.changes }}</summary>
      <ul class="mt-1.5 flex list-disc flex-col gap-1 ps-4">
        <li v-for="change in tailoring.changes" :key="change">{{ change }}</li>
      </ul>
    </details>
  </section>
</template>
