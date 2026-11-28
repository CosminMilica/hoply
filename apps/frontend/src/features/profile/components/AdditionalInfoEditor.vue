<script setup lang="ts">
import { ref } from 'vue'
import { t } from '@/features/common/i18n'
import type { Profile } from '../api/profile.schema'
import { useProfileMutations } from '../composables/useProfile'
import NotesList from './NotesList.vue'

const MAX = 10
const props = defineProps<{ profile: Profile }>()
const { patch } = useProfileMutations()
const draft = ref('')

function add() {
  const line = draft.value.trim()
  if (!line || props.profile.additional_info.length >= MAX) return
  draft.value = ''
  patch.mutate({ additional_info: [...props.profile.additional_info, line] })
}
function remove(index: number) {
  patch.mutate({ additional_info: props.profile.additional_info.filter((_, at) => at !== index) })
}
function removeNote(index: number) {
  patch.mutate({ notes: props.profile.notes.filter((_, at) => at !== index) })
}
</script>

<template>
  <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ t.profile.additional.intro }}</p>
  <ul v-if="profile.additional_info.length" class="flex flex-col gap-1.5">
    <li v-for="(line, index) in profile.additional_info" :key="index" class="flex items-center gap-1 rounded-[10px] border border-default bg-elevated py-1 ps-3 pe-1 text-sm font-medium">
      <span class="min-w-0 flex-1">{{ line }}</span>
      <UButton color="neutral" variant="ghost" icon="i-lucide-x" :aria-label="t.profile.entry.remove(line)" class="size-9 shrink-0 justify-center text-muted" @click="remove(index)" />
    </li>
  </ul>
  <form v-if="profile.additional_info.length < MAX" class="flex gap-2" @submit.prevent="add">
    <UFormField :label="t.profile.additional.addLine" :ui="{ label: 'sr-only' }" class="min-w-0 flex-1">
      <UInput v-model="draft" :placeholder="t.profile.additional.placeholder" maxlength="300" class="w-full" />
    </UFormField>
    <UButton type="submit" color="neutral" :label="t.profile.entry.add" class="px-[18px]" />
  </form>
  <NotesList :notes="profile.notes" :of="t.profile.additional.yourNotes" :pending="patch.isPending.value" @remove="removeNote" />
</template>
