<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { t } from '@/features/common/i18n'
import { EDUCATION_LEVELS, educationInputSchema, type Education, type EducationInput } from '../api/profile.schema'

const label = { label: 'text-xs font-bold text-muted' }
const levels = computed(() => EDUCATION_LEVELS.map(value => ({ label: t.value.profile.educationLevels[value], value: value as string })))

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
  <form class="flex flex-col gap-2 rounded-[10px] border-[1.5px] border-dashed border-ink-400 p-3" :aria-label="initial ? t.profile.entry.edit(initial.degree || initial.field || t.profile.educationForm.entry) : t.profile.educationForm.add" @submit.prevent="submit">
    <UFormField :label="t.profile.educationForm.level" :error="errors.level" :ui="label" required>
      <USelect v-model="draft.level" :items="levels" :placeholder="t.profile.educationForm.chooseLevel" class="w-full" />
    </UFormField>
    <UFormField :label="t.profile.educationForm.field" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.field" :placeholder="t.profile.educationForm.fieldPlaceholder" maxlength="120" class="w-full" />
    </UFormField>
    <UFormField :label="t.profile.educationForm.degree" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.degree" :placeholder="t.profile.educationForm.degreePlaceholder" maxlength="120" class="w-full" />
    </UFormField>
    <UFormField :label="t.profile.educationForm.school" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.school" :placeholder="t.profile.educationForm.school" maxlength="120" class="w-full" />
    </UFormField>
    <div class="grid grid-cols-2 gap-2">
      <UFormField :label="t.profile.educationForm.startYear" :error="errors.start_year" :ui="label">
        <UInput v-model="draft.start_year" :placeholder="t.profile.educationForm.startYearPlaceholder" inputmode="numeric" maxlength="4" class="w-full" />
      </UFormField>
      <UFormField :label="t.profile.educationForm.endYear" :error="errors.end_year" :ui="label">
        <UInput v-model="draft.end_year" :placeholder="t.profile.educationForm.endYearPlaceholder" inputmode="numeric" maxlength="4" class="w-full" />
      </UFormField>
    </div>
    <p v-if="serverError" role="alert" class="m-0 text-sm font-bold text-error">{{ serverError }}</p>
    <div class="flex gap-2">
      <UButton type="submit" color="neutral" :label="t.common.actions.save" block class="flex-1" :loading="pending" />
      <UButton color="neutral" variant="outline" :label="t.common.actions.cancel" @click="emit('cancel')" />
    </div>
  </form>
</template>
