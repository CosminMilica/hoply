<script setup lang="ts">
import { reactive } from 'vue'
import { t } from '@/features/common/i18n'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { Prep } from '../composables/usePrep'

defineProps<{ prep: Prep }>()
const impacts = reactive<Record<string, string>>({})
</script>

<template>
  <section v-if="prep.impactRows.value.length" class="flex flex-col gap-2.5">
    <SectionLabel>{{ t.cv.impact.title }}</SectionLabel>
    <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ t.cv.impact.intro }}</p>
    <div v-for="row in prep.impactRows.value" :key="row.key" class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3">
      <p class="m-0 text-sm leading-snug font-bold">{{ row.bullet }}</p>
      <UInput
        v-model="impacts[row.key]"
        :aria-label="t.cv.impact.label"
        :placeholder="t.cv.impact.placeholder"
        maxlength="200"
        class="w-full"
        @change="prep.saveImpact(row.key, impacts[row.key] ?? '')"
      />
      <p v-if="prep.answers.state.errors[`i:${row.key}`]" role="alert" class="m-0 text-sm font-bold text-error">{{ prep.answers.state.errors[`i:${row.key}`] }}</p>
    </div>
  </section>
</template>
