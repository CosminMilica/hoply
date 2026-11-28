<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { t } from '@/features/common/i18n'
import { profilePatchSchema, type Profile } from '../api/profile.schema'
import { useProfileMutations } from '../composables/useProfile'

type Field = 'email' | 'phone' | 'github' | 'linkedin' | 'city' | 'country'

const props = defineProps<{ profile: Profile }>()
const { patch } = useProfileMutations()
const FIELDS: { key: Field, type: string, icon: string }[] = [
  { key: 'email', type: 'email', icon: 'i-lucide-mail' },
  { key: 'phone', type: 'tel', icon: 'i-lucide-phone' },
  { key: 'github', type: 'text', icon: 'i-lucide-code-xml' },
  { key: 'linkedin', type: 'text', icon: 'i-lucide-link' },
  { key: 'city', type: 'text', icon: 'i-lucide-map-pin' },
  { key: 'country', type: 'text', icon: 'i-lucide-flag' },
]
const placeholders = computed<Record<Field, string>>(() => {
  const words = t.value.profile.contact
  return {
    email: words.emailPlaceholder,
    phone: '+40 700 000 000',
    github: words.githubPlaceholder,
    linkedin: words.linkedinPlaceholder,
    city: words.cityPlaceholder,
    country: words.countryPlaceholder,
  }
})
const values = reactive<Record<Field, string>>({
  email: props.profile.email,
  phone: props.profile.phone,
  github: props.profile.github,
  linkedin: props.profile.linkedin,
  city: props.profile.city,
  country: props.profile.country,
})
const errors = ref<Partial<Record<Field, string>>>({})
for (const { key } of FIELDS) {
  watch(() => props.profile[key], (value) => { values[key] = value })
}

function save(field: Field) {
  const value = values[field].trim()
  const parsed = profilePatchSchema.safeParse({ [field]: value })
  errors.value = { ...errors.value, [field]: parsed.success ? undefined : parsed.error.issues[0]?.message }
  if (parsed.success && value !== props.profile[field]) patch.mutate({ [field]: value })
}
</script>

<template>
  <UFormField v-for="field in FIELDS" :key="field.key" :label="t.profile.contact[field.key]" :error="errors[field.key]" :ui="{ label: 'sr-only' }">
    <div class="flex items-center gap-2.5">
      <span class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-accented text-hoply-400">
        <UIcon :name="field.icon" class="size-[18px]" />
      </span>
      <UInput v-model="values[field.key]" :type="field.type" :placeholder="placeholders[field.key]" maxlength="200" class="min-w-0 flex-1" @change="save(field.key)" />
    </div>
  </UFormField>
</template>
