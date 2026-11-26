<script setup lang="ts">
import { computed } from 'vue'
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { AREAS, CITIES, LEVELS } from '../api/jobs.schema'
import { marketNote } from '../match'
import { useJobSearchStore } from '../stores/search'

const props = defineProps<{ count: number }>()
const emit = defineEmits<{ show: [] }>()
const search = useJobSearchStore()
const profile = useProfileQuery()
const note = computed(() => (profile.data.value ? marketNote(profile.data.value) : ''))

const cities = computed(() => [{ label: search.anywhere, value: '' }, ...CITIES.map(city => ({ label: city, value: city }))]
  .map(city => ({ ...city, pressed: search.location.trim().toLowerCase() === city.value.toLowerCase() })))
const showLabel = computed(() => (props.count ? `Show ${props.count} ${props.count === 1 ? 'job' : 'jobs'}` : 'No jobs match yet'))
</script>

<template>
  <div class="mx-auto flex w-full max-w-[620px] flex-col gap-4">
    <div class="flex flex-col gap-1">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">Find jobs</h1>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">Tell us what you are looking for, then swipe through the matches.</p>
      <p v-if="note" class="m-0 mt-1.5 flex items-start gap-1.5 text-[13px] leading-snug font-semibold text-ink-300" data-test="market-note">
        <UIcon name="i-lucide-map-pin" class="mt-px size-4 shrink-0 text-hoply-400" />
        <span>{{ note }} <RouterLink :to="{ name: 'hoply-profile' }" class="underline">Change it in Profile.</RouterLink></span>
      </p>
    </div>

    <UFormField label="What" :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase' }">
      <UInput v-model="search.query" placeholder="Job title, skill or company" size="lg" class="w-full" />
    </UFormField>

    <div class="flex flex-col gap-2">
      <UFormField label="Where" :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase' }">
        <UInput v-model="search.location" placeholder="City or country" size="lg" class="w-full" />
      </UFormField>
      <div class="flex flex-wrap gap-2" role="group" aria-label="Where">
        <ToggleChip v-for="city in cities" :key="city.label" :label="city.label" :pressed="city.pressed" @toggle="search.location = city.value" />
        <ToggleChip v-if="search.location.trim()" label="Include remote" :pressed="search.includeRemote" @toggle="search.includeRemote = !search.includeRemote" />
      </div>
    </div>

    <div class="flex flex-col gap-2" role="group" aria-labelledby="area-label">
      <SectionLabel id="area-label" as="span">Position area</SectionLabel>
      <div class="flex flex-wrap gap-2">
        <ToggleChip v-for="area in AREAS" :key="area" :label="area" :pressed="search.areas.includes(area)" @toggle="search.toggleArea(area)" />
      </div>
    </div>

    <div class="flex flex-col gap-2" role="group" aria-labelledby="level-label">
      <SectionLabel id="level-label" as="span">Level</SectionLabel>
      <div class="flex flex-wrap gap-2">
        <ToggleChip v-for="level in LEVELS" :key="level" :label="level" :pressed="search.level === level" @toggle="search.level = level" />
      </div>
    </div>

    <div class="mt-1 flex flex-col gap-2">
      <UButton :label="showLabel" :disabled="!count" size="lg" block class="justify-center" @click="emit('show')" />
      <UButton label="Reset filters" variant="link" color="neutral" class="justify-center text-muted underline" @click="search.reset()" />
    </div>
    <div class="flex items-center gap-3 text-xs font-extrabold tracking-wider text-ink-500 uppercase" aria-hidden="true">
      <span class="h-px flex-1 bg-(--ui-border)" />Already found a job?<span class="h-px flex-1 bg-(--ui-border)" />
    </div>
    <RouterLink
      :to="{ name: 'hoply-job-add' }"
      class="flex w-full items-center gap-3.5 rounded-[14px] border-[1.5px] border-dashed border-ink-600 bg-elevated px-4 py-3.5 text-left hover:bg-accented"
    >
      <span class="flex size-10 shrink-0 items-center justify-center rounded-[10px] bg-accented text-hoply-400">
        <UIcon name="i-lucide-link" class="size-5" />
      </span>
      <span class="flex min-w-0 flex-1 flex-col gap-0.5">
        <span class="text-[15px] font-extrabold">Add a job from a link or its text</span>
        <span class="text-[13px] leading-snug font-medium text-muted">Seen it on LinkedIn, eJobs or a company site? Paste it here. We match it to your profile and you can tailor a CV for it.</span>
      </span>
      <UIcon name="i-lucide-chevron-right" class="size-[18px] shrink-0 text-muted" />
    </RouterLink>
  </div>
</template>
