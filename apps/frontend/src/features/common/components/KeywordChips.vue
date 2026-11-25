<script setup lang="ts">
import type { RelatedSkill } from '@/features/jobs/match'

withDefaults(defineProps<{ matched?: string[], related?: RelatedSkill[], missing?: string[], size?: 'sm' | 'md' }>(), {
  matched: () => [],
  related: () => [],
  missing: () => [],
  size: 'sm',
})
</script>

<template>
  <ul class="flex flex-wrap gap-1.5" :class="size === 'md' ? 'text-sm' : 'text-[13px]'">
    <li
      v-for="keyword in matched"
      :key="`m-${keyword}`"
      class="rounded-lg bg-hoply-900 px-3 py-1.5 font-bold text-hoply-300"
    >
      {{ keyword }}
    </li>
    <li
      v-for="item in related"
      :key="`r-${item.skill}`"
      class="rounded-lg border-[1.5px] border-dashed border-hoply-700 px-2.5 py-1 font-bold text-hoply-300"
      :title="`${item.category}: you use ${item.via.join(', ')}`"
    >
      {{ item.skill }}<span class="sr-only"> ({{ item.category }}: you use {{ item.via.join(', ') }})</span>
    </li>
    <li
      v-for="keyword in missing"
      :key="`x-${keyword}`"
      class="rounded-lg border-[1.5px] border-ink-400 px-2.5 py-1 font-bold text-toned"
    >
      <span class="sr-only">Missing: </span>{{ keyword }}
    </li>
  </ul>
</template>
