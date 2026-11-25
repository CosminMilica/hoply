<script setup lang="ts">
import ProgressBar from '@/features/common/components/ProgressBar.vue'
import type { ReadyCv } from '../api/cv.schema'

defineProps<{ ats: ReadyCv['ats'] }>()
defineEmits<{ improve: [] }>()
</script>

<template>
  <section class="flex flex-col gap-2.5 rounded-[14px] bg-accented p-3.5" aria-labelledby="ats-title">
    <div class="flex items-baseline justify-between">
      <h2 id="ats-title" class="m-0 text-xs font-extrabold tracking-wider text-toned uppercase">ATS check</h2>
      <span class="text-xl font-extrabold">{{ ats.passed }} of {{ ats.total }}</span>
    </div>
    <ProgressBar :value="Math.round((ats.passed / ats.total) * 100)" label="ATS check" />
    <ul class="flex flex-col gap-2.5">
      <li v-for="check in ats.checks" :key="check.label" class="flex items-start gap-2.5">
        <span
          class="mt-px flex size-5 shrink-0 items-center justify-center rounded-full text-xs font-extrabold"
          :class="check.ok ? 'bg-hoply-600 text-white' : 'bg-ember-500 text-ink-950'"
        >
          <span aria-hidden="true">{{ check.ok ? '✓' : '!' }}</span>
          <span class="sr-only">{{ check.ok ? 'Passed:' : 'To fix:' }}</span>
        </span>
        <span class="flex flex-col gap-px">
          <span class="text-[13px] leading-snug font-bold">{{ check.label }}</span>
          <span v-if="!check.ok && check.hint" class="text-xs leading-snug font-medium text-toned">{{ check.hint }}</span>
        </span>
      </li>
    </ul>
    <UButton label="Improve this CV" color="neutral" variant="outline" block class="justify-center bg-elevated" @click="$emit('improve')" />
  </section>
</template>
