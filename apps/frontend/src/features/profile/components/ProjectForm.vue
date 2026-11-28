<script setup lang="ts">
import { reactive, ref } from 'vue'
import { t } from '@/features/common/i18n'
import { cleanList, projectInputSchema, withoutId, type Bullet, type Project, type ProjectInput } from '../api/profile.schema'
import HighlightsField from './HighlightsField.vue'
import MonthRangeFields from './MonthRangeFields.vue'

const props = defineProps<{ initial?: Project, pending?: boolean, serverError?: string }>()
const emit = defineEmits<{ save: [input: ProjectInput], cancel: [] }>()

const start = props.initial ? withoutId(props.initial) : null
const draft = reactive({
  name: start?.name ?? '',
  role: start?.role ?? '',
  url: start?.url ?? '',
  start: start?.start ?? '',
  end: start?.end ?? '',
  tools: start?.tools.join(', ') ?? '',
})
const bullets = ref<Bullet[]>(start?.bullets.length ? start.bullets.map(bullet => ({ ...bullet })) : [{ text: '', result: '', skills: [] }])
const errors = ref<Record<string, string>>({})

function submit() {
  const parsed = projectInputSchema.safeParse({
    ...draft,
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
  <form class="flex flex-col gap-2 rounded-[10px] border-[1.5px] border-dashed border-ink-400 p-3" :aria-label="initial ? t.profile.entry.edit(initial.name) : t.profile.projectForm.add" @submit.prevent="submit">
    <UFormField :label="t.profile.projectForm.name" :error="errors.name" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.name" :placeholder="t.profile.projectForm.namePlaceholder" maxlength="120" class="w-full" />
    </UFormField>
    <div class="grid grid-cols-2 gap-2">
      <UFormField :label="t.profile.projectForm.role" :error="errors.role" :ui="{ label: 'sr-only' }">
        <UInput v-model="draft.role" :placeholder="t.profile.projectForm.rolePlaceholder" maxlength="120" class="w-full" />
      </UFormField>
      <UFormField :label="t.profile.projectForm.link" :error="errors.url" :ui="{ label: 'sr-only' }">
        <UInput v-model="draft.url" :placeholder="t.profile.projectForm.linkPlaceholder" maxlength="200" class="w-full" />
      </UFormField>
    </div>
    <MonthRangeFields v-model:start="draft.start" v-model:end="draft.end" :ongoing-label="t.profile.projectForm.ongoing" :errors="{ start: errors.start, end: errors.end }" />
    <UFormField :label="t.profile.experienceForm.tools" :error="errors.tools" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.tools" :placeholder="t.profile.projectForm.toolsPlaceholder" class="w-full" />
    </UFormField>
    <HighlightsField v-model="bullets" :error="errors.bullets" />
    <p v-if="serverError" role="alert" class="m-0 text-sm font-bold text-error">{{ serverError }}</p>
    <div class="flex gap-2">
      <UButton type="submit" color="neutral" :label="t.common.actions.save" block class="flex-1" :loading="pending" />
      <UButton color="neutral" variant="outline" :label="t.common.actions.cancel" @click="emit('cancel')" />
    </div>
  </form>
</template>
