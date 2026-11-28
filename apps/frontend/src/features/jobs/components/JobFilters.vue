<script setup lang="ts">
import { computed } from 'vue'
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import { t } from '@/features/common/i18n'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { marketNote } from '../match'
import { useJobSearchStore } from '../stores/search'
import DomainFilter from './DomainFilter.vue'

const props = defineProps<{ count: number }>()
const emit = defineEmits<{ show: [] }>()
const search = useJobSearchStore()
const profile = useProfileQuery()
const note = computed(() => (profile.data.value ? marketNote(profile.data.value) : ''))

const showLabel = computed(() => (props.count ? t.value.jobs.search.show(props.count) : t.value.jobs.search.noneYet))
</script>

<template>
  <div class="mx-auto flex w-full max-w-[620px] flex-col gap-4">
    <div class="flex flex-col gap-1">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">{{ t.jobs.search.title }}</h1>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">{{ t.jobs.search.intro }}</p>
      <p v-if="note" class="m-0 mt-1.5 flex items-start gap-1.5 text-[13px] leading-snug font-semibold text-ink-300" data-test="market-note">
        <UIcon name="i-lucide-map-pin" class="mt-px size-4 shrink-0 text-hoply-400" />
        <span>{{ note }} <RouterLink :to="{ name: 'hoply-profile' }" class="underline">{{ t.jobs.search.changeInProfile }}</RouterLink></span>
      </p>
    </div>

    <UFormField :label="t.jobs.search.what" :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase' }">
      <UInput v-model="search.query" :placeholder="t.jobs.search.whatPlaceholder" size="lg" class="w-full" />
    </UFormField>

    <div class="flex flex-col gap-2">
      <UFormField :label="t.jobs.search.where" :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase' }">
        <UInput v-model="search.location" :placeholder="t.jobs.search.wherePlaceholder" size="lg" class="w-full" />
      </UFormField>
      <div class="flex flex-wrap gap-2" role="group" :aria-label="t.jobs.search.where">
        <ToggleChip v-for="city in search.cityOptions" :key="city.label" :label="city.label" :pressed="city.pressed" @toggle="search.location = city.value" />
        <ToggleChip v-if="search.location.trim()" :label="t.jobs.search.includeRemote" :pressed="search.includeRemote" @toggle="search.includeRemote = !search.includeRemote" />
      </div>
    </div>

    <div v-if="search.domains.length" class="flex flex-col gap-2">
      <SectionLabel as="span">{{ t.jobs.search.domain }}</SectionLabel>
      <DomainFilter />
    </div>

    <div class="flex flex-col gap-2" role="group" aria-labelledby="level-label">
      <SectionLabel id="level-label" as="span">{{ t.jobs.search.level }}</SectionLabel>
      <div class="flex flex-wrap gap-2">
        <ToggleChip v-for="level in search.levelOptions" :key="level.value" :label="level.label" :pressed="level.pressed" @toggle="search.level = level.value" />
      </div>
    </div>

    <div class="mt-1 flex flex-col gap-2">
      <UButton :label="showLabel" :disabled="!count" size="lg" block class="justify-center" @click="emit('show')" />
      <UButton :label="t.jobs.search.reset" variant="link" color="neutral" class="justify-center text-muted underline" @click="search.reset()" />
    </div>
    <div class="flex items-center gap-3 text-xs font-extrabold tracking-wider text-ink-500 uppercase" aria-hidden="true">
      <span class="h-px flex-1 bg-(--ui-border)" />{{ t.jobs.search.alreadyFound }}<span class="h-px flex-1 bg-(--ui-border)" />
    </div>
    <RouterLink
      :to="{ name: 'hoply-job-add' }"
      class="flex w-full items-center gap-3.5 rounded-[14px] border-[1.5px] border-dashed border-ink-600 bg-elevated px-4 py-3.5 text-left hover:bg-accented"
    >
      <span class="flex size-10 shrink-0 items-center justify-center rounded-[10px] bg-accented text-hoply-400">
        <UIcon name="i-lucide-link" class="size-5" />
      </span>
      <span class="flex min-w-0 flex-1 flex-col gap-0.5">
        <span class="text-[15px] font-extrabold">{{ t.jobs.search.addTitle }}</span>
        <span class="text-[13px] leading-snug font-medium text-muted">{{ t.jobs.search.addText }}</span>
      </span>
      <UIcon name="i-lucide-chevron-right" class="size-[18px] shrink-0 text-muted" />
    </RouterLink>
  </div>
</template>
