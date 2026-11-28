<script setup lang="ts">
import { computed, ref } from 'vue'
import { t } from '@/features/common/i18n'
import { languageInputSchema, type LanguageLevel, type Profile } from '../api/profile.schema'
import { useProfileMutations } from '../composables/useProfile'

defineProps<{ profile: Profile }>()
const { languages } = useProfileMutations()
const LEVELS: LanguageLevel[] = ['A2', 'B1', 'B2', 'C1', 'C2', 'Native']
const levels = computed(() => LEVELS.map(value => ({ label: t.value.profile.languageLevels[value], value })))
const name = ref('')
const level = ref<LanguageLevel>('B2')
const error = ref<string>()

function add() {
  const parsed = languageInputSchema.safeParse({ name: name.value, level: level.value })
  if (!parsed.success) {
    error.value = parsed.error.issues[0]?.message
    return
  }
  error.value = undefined
  languages.add.mutate(parsed.data, { onSuccess: () => { name.value = '' } })
}
</script>

<template>
  <ul class="-mx-0.5 flex flex-wrap">
    <li v-for="language in profile.languages" :key="language.id">
      <button type="button" class="flex min-h-11 items-center px-0.5" :aria-label="t.profile.entry.remove(language.name)" @click="languages.remove.mutate(language.id)">
        <span class="flex h-8 items-center gap-1.5 rounded-lg bg-accented px-3 text-sm font-extrabold text-highlighted">
          {{ language.name }} · {{ (t.profile.languageLevels as Record<string, string>)[language.level] ?? language.level }}<span aria-hidden="true" class="text-[15px] opacity-60">×</span>
        </span>
      </button>
    </li>
  </ul>
  <form class="flex gap-2" @submit.prevent="add">
    <UFormField :label="t.profile.languages.language" :error="error" :ui="{ label: 'sr-only' }" class="min-w-0 flex-1">
      <UInput v-model="name" :placeholder="t.profile.languages.placeholder" maxlength="120" class="w-full" />
    </UFormField>
    <UFormField :label="t.profile.languages.level" :ui="{ label: 'sr-only' }">
      <USelect v-model="level" :items="levels" class="w-24" />
    </UFormField>
    <UButton type="submit" color="neutral" :label="t.profile.entry.add" class="px-[18px]" />
  </form>
</template>
