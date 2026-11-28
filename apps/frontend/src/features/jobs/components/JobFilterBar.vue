<script setup lang="ts">
import { computed } from 'vue'
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import { t } from '@/features/common/i18n'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { marketNote } from '../match'
import { useJobSearchStore } from '../stores/search'
import DomainFilter from './DomainFilter.vue'

/** The filters above the grid: every change shows its jobs at once. */
const search = useJobSearchStore()
const profile = useProfileQuery()
const note = computed(() => (profile.data.value ? marketNote(profile.data.value) : ''))
</script>

<template>
  <section :aria-label="t.jobs.search.filters" class="flex flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-3.5" data-test="filter-bar">
    <div class="flex flex-col gap-2 sm:flex-row">
      <UInput v-model="search.query" icon="i-lucide-search" :placeholder="t.jobs.search.whatPlaceholder" :aria-label="t.jobs.search.what" class="flex-1" />
      <UInput v-model="search.location" icon="i-lucide-map-pin" :placeholder="t.jobs.search.wherePlaceholder" :aria-label="t.jobs.search.where" class="sm:w-56" />
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <div role="group" :aria-label="t.jobs.search.city" class="flex flex-wrap gap-2">
        <ToggleChip v-for="city in search.cityOptions" :key="city.label" :label="city.label" :pressed="city.pressed" @toggle="search.location = city.value" />
        <ToggleChip v-if="search.location.trim()" :label="t.jobs.search.includeRemote" :pressed="search.includeRemote" @toggle="search.includeRemote = !search.includeRemote" />
      </div>
      <span aria-hidden="true" class="mx-1 hidden h-5 w-px bg-(--ui-border) sm:block" />
      <div role="group" :aria-label="t.jobs.search.level" class="flex flex-wrap gap-2">
        <ToggleChip v-for="level in search.levelOptions" :key="level.value" :label="level.label" :pressed="level.pressed" @toggle="search.level = level.value" />
      </div>
    </div>
    <DomainFilter />
    <div class="flex flex-wrap items-center justify-between gap-2">
      <span v-if="note" class="text-xs font-semibold text-muted" data-test="market-note">
        {{ note }} <RouterLink :to="{ name: 'hoply-profile' }" class="underline">{{ t.jobs.search.changeInProfile }}</RouterLink>
      </span>
      <UButton :label="t.jobs.search.reset" variant="link" color="neutral" size="sm" class="ms-auto text-muted underline" @click="search.reset()" />
    </div>
  </section>
</template>
