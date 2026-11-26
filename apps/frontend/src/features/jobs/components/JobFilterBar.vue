<script setup lang="ts">
import { computed } from 'vue'
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { AREAS, LEVELS } from '../api/jobs.schema'
import { marketNote } from '../match'
import { useJobSearchStore } from '../stores/search'

/** The filters above the grid: every change shows its jobs at once. */
const search = useJobSearchStore()
const profile = useProfileQuery()
const note = computed(() => (profile.data.value ? marketNote(profile.data.value) : ''))
</script>

<template>
  <section aria-label="Filters" class="flex flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-3.5" data-test="filter-bar">
    <div class="flex flex-col gap-2 sm:flex-row">
      <UInput v-model="search.query" icon="i-lucide-search" placeholder="Job title, skill or company" aria-label="What" class="flex-1" />
      <UInput v-model="search.location" icon="i-lucide-map-pin" placeholder="City or country" aria-label="Where" class="sm:w-56" />
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <div role="group" aria-label="City" class="flex flex-wrap gap-2">
        <ToggleChip v-for="city in search.cityOptions" :key="city.label" :label="city.label" :pressed="city.pressed" @toggle="search.location = city.value" />
        <ToggleChip v-if="search.location.trim()" label="Include remote" :pressed="search.includeRemote" @toggle="search.includeRemote = !search.includeRemote" />
      </div>
      <span aria-hidden="true" class="mx-1 hidden h-5 w-px bg-(--ui-border) sm:block" />
      <div role="group" aria-label="Level" class="flex flex-wrap gap-2">
        <ToggleChip v-for="level in LEVELS" :key="level" :label="level" :pressed="search.level === level" @toggle="search.level = level" />
      </div>
    </div>
    <div role="group" aria-label="Area" class="flex flex-wrap gap-2">
      <ToggleChip v-for="area in AREAS" :key="area" :label="area" :pressed="search.areas.includes(area)" @toggle="search.toggleArea(area)" />
    </div>
    <div class="flex flex-wrap items-center justify-between gap-2">
      <span v-if="note" class="text-xs font-semibold text-muted" data-test="market-note">
        {{ note }} <RouterLink :to="{ name: 'hoply-profile' }" class="underline">Change it in Profile.</RouterLink>
      </span>
      <UButton label="Reset filters" variant="link" color="neutral" size="sm" class="ms-auto text-muted underline" @click="search.reset()" />
    </div>
  </section>
</template>
