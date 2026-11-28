<script setup lang="ts">
import { computed } from 'vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import { locale, setLocale, t, type Locale } from '@/features/common/i18n'
import { useProfileMutations, useProfileQuery } from '@/features/profile/composables/useProfile'
import type { Domain } from '@/features/profile/api/profile.schema'
import DomainPicker from '../components/DomainPicker.vue'
import LanguagePicker from '../components/LanguagePicker.vue'

/** The app's language and the fields of work, as onboarding set them; each change saves at once. */
const profile = useProfileQuery()
const { patch } = useProfileMutations()

const language = computed<Locale>({
  get: () => locale.value,
  set: (value) => {
    setLocale(value)
    patch.mutate({ app_language: value })
  },
})
const domains = computed<Domain[]>({
  get: () => (profile.data.value?.domains ?? []) as Domain[],
  set: (value) => {
    // Find jobs needs a field to show: the last one stays.
    if (value.length) patch.mutate({ domains: value })
  },
})
</script>

<template>
  <div class="mx-auto flex w-full max-w-[1040px] flex-col gap-7">
    <div class="flex flex-col gap-1">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">{{ t.onboarding.preferences.title }}</h1>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">{{ t.onboarding.preferences.intro }}</p>
    </div>
    <UAlert v-if="profile.isError.value" color="error" variant="subtle" :title="t.onboarding.loadFailed" />
    <section class="flex max-w-[720px] flex-col gap-3">
      <SectionLabel>{{ t.onboarding.languageGroup }}</SectionLabel>
      <LanguagePicker v-model="language" />
    </section>
    <section v-if="profile.data.value" class="flex flex-col gap-3">
      <div class="flex flex-wrap items-baseline justify-between gap-2">
        <SectionLabel>{{ t.onboarding.domainsGroup }}</SectionLabel>
        <span class="text-[13px] font-semibold text-muted" aria-live="polite" data-test="picked">
          {{ domains.length === 1 ? t.onboarding.preferences.atLeastOne : t.onboarding.picked(domains.length) }}
        </span>
      </div>
      <DomainPicker v-model="domains" />
    </section>
  </div>
</template>
