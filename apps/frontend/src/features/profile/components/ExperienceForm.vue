<script setup lang="ts">
import { reactive, ref } from 'vue'
import { cleanList, EMPLOYMENT_TYPES, experienceInputSchema, withoutId, type Bullet, type Experience, type ExperienceInput } from '../api/profile.schema'
import HighlightsField from './HighlightsField.vue'
import MonthRangeFields from './MonthRangeFields.vue'

const NONE = 'none'
const label = { label: 'text-xs font-bold text-muted' }
const employmentItems = [{ label: 'Not set', value: NONE }, ...Object.entries(EMPLOYMENT_TYPES).map(([value, text]) => ({ label: text, value }))]

const props = defineProps<{ initial?: Experience, pending?: boolean, serverError?: string }>()
const emit = defineEmits<{ save: [input: ExperienceInput], cancel: [] }>()

const start = props.initial ? withoutId(props.initial) : null
const draft = reactive({
  role: start?.role ?? '',
  company: start?.company ?? '',
  location: start?.location ?? '',
  employmentType: start?.employment_type || NONE,
  industry: start?.industry ?? '',
  teamSize: start?.team_size?.toString() ?? '',
  peopleManaged: start?.people_managed?.toString() ?? '',
  start: start?.start ?? '',
  end: start?.end ?? '',
  tools: start?.tools.join(', ') ?? '',
})
const bullets = ref<Bullet[]>(start?.bullets.length ? start.bullets.map(bullet => ({ ...bullet })) : [{ text: '', result: '', skills: [] }])
const errors = ref<Record<string, string>>({})

const number = (value: string | number) => (String(value ?? '').trim() === '' ? null : Number(value))

function submit() {
  const parsed = experienceInputSchema.safeParse({
    role: draft.role,
    company: draft.company,
    location: draft.location,
    employment_type: draft.employmentType === NONE ? '' : draft.employmentType,
    industry: draft.industry,
    team_size: number(draft.teamSize),
    people_managed: number(draft.peopleManaged),
    start: draft.start,
    end: draft.end,
    tools: cleanList(draft.tools.split(',')),
    bullets: bullets.value.filter(bullet => bullet.text.trim()),
  })
  if (!parsed.success) {
    errors.value = Object.fromEntries(parsed.error.issues.map(issue => [String(issue.path[0]), issue.message]))
    return
  }
  errors.value = {}
  emit('save', parsed.data)
}
</script>

<template>
  <form class="flex flex-col gap-2 rounded-[10px] border-[1.5px] border-dashed border-ink-400 p-3" :aria-label="initial ? `Edit ${initial.role}` : 'Add a role'" @submit.prevent="submit">
    <UFormField label="Role" :error="errors.role" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.role" placeholder="Role, e.g. Frontend developer" maxlength="120" class="w-full" />
    </UFormField>
    <UFormField label="Company" :error="errors.company" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.company" placeholder="Company" maxlength="120" class="w-full" />
    </UFormField>
    <div class="grid grid-cols-2 gap-2">
      <UFormField label="Location" :ui="{ label: 'sr-only' }">
        <UInput v-model="draft.location" placeholder="Location, e.g. Iași" maxlength="120" class="w-full" />
      </UFormField>
      <UFormField label="Industry" :error="errors.industry" :ui="{ label: 'sr-only' }">
        <UInput v-model="draft.industry" placeholder="Industry, e.g. Fintech" maxlength="60" class="w-full" />
      </UFormField>
    </div>
    <MonthRangeFields v-model:start="draft.start" v-model:end="draft.end" ongoing-label="I still work here" :errors="{ start: errors.start, end: errors.end }" />
    <div class="grid grid-cols-3 gap-2">
      <UFormField label="Employment type" :ui="label">
        <USelect v-model="draft.employmentType" :items="employmentItems" class="w-full" />
      </UFormField>
      <UFormField label="Team size" :error="errors.team_size" :ui="label">
        <UInput v-model="draft.teamSize" type="number" min="0" inputmode="numeric" class="w-full" />
      </UFormField>
      <UFormField label="People managed" :error="errors.people_managed" :ui="label">
        <UInput v-model="draft.peopleManaged" type="number" min="0" inputmode="numeric" class="w-full" />
      </UFormField>
    </div>
    <UFormField label="Tools you used, separated by commas" :error="errors.tools" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.tools" placeholder="Tools, e.g. React, SQL, Figma" class="w-full" />
    </UFormField>
    <HighlightsField v-model="bullets" :error="errors.bullets" />
    <p v-if="serverError" role="alert" class="m-0 text-sm font-bold text-error">{{ serverError }}</p>
    <div class="flex gap-2">
      <UButton type="submit" color="neutral" label="Save" block class="flex-1" :loading="pending" />
      <UButton color="neutral" variant="outline" label="Cancel" @click="emit('cancel')" />
    </div>
  </form>
</template>
