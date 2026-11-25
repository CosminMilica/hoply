<script setup lang="ts">
import type { Profile } from '../api/profile.schema'
import { useEntryEditor } from '../composables/useEntryEditor'
import { useProfileMutations } from '../composables/useProfile'
import { educationTitle, yearRange } from '../format'
import EducationForm from './EducationForm.vue'

defineProps<{ profile: Profile }>()
const { education } = useProfileMutations()
const editor = useEntryEditor(education)
</script>

<template>
  <ul v-if="profile.education.length" class="flex flex-col gap-2.5">
    <li v-for="item in profile.education" :key="item.id">
      <EducationForm
        v-if="editor.editing.value === item.id"
        :initial="item"
        :pending="education.update.isPending.value"
        :server-error="editor.serverError.value"
        @save="input => editor.update(item.id, input)"
        @cancel="editor.close"
      />
      <div v-else class="flex items-center gap-1 rounded-[10px] border border-default bg-elevated py-1 ps-3 pe-1">
        <span class="flex min-w-0 flex-1 flex-col">
          <span class="text-[15px] font-extrabold">{{ educationTitle(item) }}</span>
          <span class="text-[13px] font-medium text-muted">{{ [item.degree ? item.field : '', item.school, yearRange(item.start_year, item.end_year)].filter(Boolean).join(' · ') }}</span>
        </span>
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-pencil"
          :aria-label="`Edit ${educationTitle(item)}`"
          class="size-9 justify-center text-muted"
          @click="editor.open(item.id)"
        />
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-x"
          :aria-label="`Remove ${educationTitle(item)}`"
          class="size-9 justify-center text-muted"
          @click="education.remove.mutate(item.id)"
        />
      </div>
    </li>
  </ul>
  <EducationForm v-if="editor.adding.value" :pending="education.add.isPending.value" :server-error="editor.serverError.value" @save="input => editor.add(input)" @cancel="editor.close" />
  <UButton
    v-else
    color="neutral"
    variant="outline"
    label="+ Add education"
    block
    class="border-[1.5px] border-dashed border-ink-400 text-toned ring-0"
    @click="editor.open(null)"
  />
</template>
