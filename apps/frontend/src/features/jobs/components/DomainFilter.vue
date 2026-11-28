<script setup lang="ts">
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import { t } from '@/features/common/i18n'
import { DOMAIN_ICONS } from '../api/jobs.schema'
import { useJobSearchStore } from '../stores/search'

/** The fields of work the person picked, as tabs, and the areas of the field shown, as chips. */
const search = useJobSearchStore()
</script>

<template>
  <div v-if="search.domains.length" class="flex flex-col gap-2.5" data-test="domain-filter">
    <div class="flex flex-wrap items-center gap-2">
      <div v-if="search.domainOptions.length" role="group" :aria-label="t.jobs.search.domain" class="flex flex-wrap gap-2">
        <UButton
          v-for="option in search.domainOptions"
          :key="option.value"
          :label="option.label"
          :icon="option.value ? DOMAIN_ICONS[option.value] : undefined"
          :aria-pressed="option.pressed"
          :color="option.pressed ? 'primary' : 'neutral'"
          :variant="option.pressed ? 'solid' : 'outline'"
          class="rounded-full px-4"
          @click="search.pickDomain(option.value)"
        />
      </div>
      <RouterLink :to="{ name: 'crashr-preferences' }" class="ms-auto flex min-h-9 items-center gap-1.5 text-[13px] font-bold text-crashr-300 hover:underline">
        <UIcon name="i-lucide-sliders-horizontal" class="size-4" />{{ t.jobs.search.changeDomains }}
      </RouterLink>
    </div>
    <div v-if="search.shownDomain" role="group" :aria-label="t.jobs.search.areasOf(t.common.domains[search.shownDomain])" class="flex flex-col gap-1.5">
      <span class="text-xs font-extrabold tracking-wide text-dimmed uppercase" aria-hidden="true">{{ t.jobs.search.areasOf(t.common.domains[search.shownDomain]) }}</span>
      <div class="flex flex-wrap gap-2">
        <ToggleChip v-for="area in search.areaOptions" :key="area.value" :label="area.label" :pressed="area.pressed" @toggle="search.toggleArea(area.value)" />
      </div>
    </div>
    <p v-else class="m-0 text-[13px] font-medium text-muted">{{ t.jobs.search.pickDomain }}</p>
  </div>
</template>
