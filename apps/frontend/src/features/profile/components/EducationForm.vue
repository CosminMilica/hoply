<script setup lang="ts">
import { reactive, ref } from 'vue'
import { EDUCATION_LEVELS, educationInputSchema, type Education, type EducationInput } from '../api/profile.schema'

const label = { label: 'text-xs font-bold text-muted' }
const levels = Object.entries(EDUCATION_LEVELS).map(([value, text]) => ({ label: text, value }))

const props = defineProps<{ initial?: Education, pending?: boolean, serverError?: string }>()
const emit = defineEmits<{ save: [input: EducationInput], cancel: [] }>()

const draft = reactive({
  level: props.initial?.level || undefined,
  field: props.initial?.field ?? '',
  degree: props.initial?.degree ?? '',
  school: props.initial?.school ?? '',
  start_year: props.initial?.start_year ?? '',
  end_year: props.initial?.end_year ?? '',
})
const errors = ref<Record<string, string>>({})

function submit() {
  const parsed = educationInputSchema.safeParse(draft)
  if (!parsed.success) {
    errors.value = Object.fromEntries(parsed.error.issues.map(issue => [String(issue.path[0]), issue.message]))
    return
  }
  errors.value = {}
  emit('save', parsed.data)
}
</script>

<template>
  <form class="flex flex-col gap-2 rounded-[10px] border-[1.5px] border-dashed border-ink-400 p-3" :aria-label="initial ? `Edit ${initial.degree || initial.field || 'education'}` : 'Add education'" @submit.prevent="submit">
    <UFormField label="Level" :error="errors.level" :ui="label" required>
      <USelect v-model="draft.level" :items="levels" placeholder="Choose the level" class="w-full" />
    </UFormField>
    <UFormField label="Field of study" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.field" placeholder="Field, e.g. Computer Science" maxlength="120" class="w-full" />
    </UFormField>
    <UFormField label="Degree title" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.degree" placeholder="Degree title (optional), e.g. BSc Computer Science" maxlength="120" class="w-full" />
    </UFormField>
    <UFormField label="School" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.school" placeholder="School" maxlength="120" class="w-full" />
    </UFormField>
    <div class="grid grid-cols-2 gap-2">
      <UFormField label="Start year" :error="errors.start_year" :ui="label">
        <UInput v-model="draft.start_year" placeholder="e.g. 2016" inputmode="numeric" maxlength="4" class="w-full" />
      </UFormField>
      <UFormField label="End year" :error="errors.end_year" :ui="label">
        <UInput v-model="draft.end_year" placeholder="Empty = ongoing" inputmode="numeric" maxlength="4" class="w-full" />
      </UFormField>
    </div>
    <p v-if="serverError" role="alert" class="m-0 text-sm font-bold text-error">{{ serverError }}</p>
    <div class="flex gap-2">
      <UButton type="submit" color="neutral" label="Save" block class="flex-1" :loading="pending" />
      <UButton color="neutral" variant="outline" label="Cancel" @click="emit('cancel')" />
    </div>
  </form>
</template>
