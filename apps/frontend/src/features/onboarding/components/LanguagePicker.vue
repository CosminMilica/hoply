<script setup lang="ts">
import { LOCALES, t, type Locale } from '@/features/common/i18n'

/** The app's languages as two cards; picking one is a radio choice. */
const model = defineModel<Locale>({ required: true })
</script>

<template>
  <div role="radiogroup" :aria-label="t.onboarding.languageGroup" class="grid grid-cols-1 gap-3 sm:grid-cols-2" data-test="language-picker">
    <button
      v-for="language in LOCALES"
      :key="language"
      type="button"
      role="radio"
      :aria-checked="model === language"
      :lang="language"
      class="flex min-h-[148px] flex-col gap-3 rounded-[16px] border-2 p-5 text-left transition-colors"
      :class="model === language ? 'border-hoply-500 bg-hoply-950' : 'border-default bg-elevated hover:border-accented'"
      @click="model = language"
    >
      <span class="flex items-center justify-between">
        <span class="flex size-11 items-center justify-center rounded-[12px] bg-accented text-[15px] font-extrabold tracking-wider uppercase">{{ language }}</span>
        <UIcon v-if="model === language" name="i-lucide-circle-check" class="size-6 text-hoply-400" />
      </span>
      <span class="text-[22px] font-extrabold text-highlighted">{{ t.common.languages[language] }}</span>
      <span class="text-[14px] leading-snug font-medium text-muted">{{ t.onboarding.languageHints[language] }}</span>
    </button>
  </div>
</template>
