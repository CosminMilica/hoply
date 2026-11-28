<script setup lang="ts">
import { t } from '@/features/common/i18n'

/** filtersLabel names the first way out: changing the filters (the default), or resetting them where the filters are in view. */
defineProps<{ filtersLabel?: string }>()
const emit = defineEmits<{ changeFilters: [], resetSkipped: [] }>()
</script>

<template>
  <div class="mt-1 flex flex-col gap-2.5 rounded-[18px] border border-default bg-elevated px-[22px] py-7">
    <h2 class="m-0 text-2xl leading-tight font-extrabold tracking-tight">{{ t.jobs.caughtUp.title }}</h2>
    <p class="m-0 text-[15px] leading-normal font-medium text-muted">{{ t.jobs.caughtUp.text }}</p>
    <UButton :label="filtersLabel ?? t.jobs.caughtUp.changeFilters" size="lg" block class="mt-1.5 justify-center" @click="emit('changeFilters')" />
    <UButton :label="t.jobs.caughtUp.bringBack" color="neutral" variant="outline" size="lg" block class="justify-center" @click="emit('resetSkipped')" />
    <UButton :to="{ name: 'crashr-saved' }" :label="t.jobs.caughtUp.seeSaved" variant="link" color="neutral" class="justify-center text-muted underline" />
  </div>
</template>
