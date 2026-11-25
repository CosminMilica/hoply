<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { profilePatchSchema, type Profile } from '../api/profile.schema'
import { useProfileMutations } from '../composables/useProfile'

type Field = 'email' | 'phone' | 'github' | 'linkedin' | 'city' | 'country'

const props = defineProps<{ profile: Profile }>()
const { patch } = useProfileMutations()
const FIELDS: { key: Field, label: string, type: string, placeholder: string, icon: string }[] = [
  { key: 'email', label: 'Email', type: 'email', placeholder: 'you@email.com', icon: 'i-lucide-mail' },
  { key: 'phone', label: 'Phone', type: 'tel', placeholder: '+40 700 000 000', icon: 'i-lucide-phone' },
  { key: 'github', label: 'GitHub', type: 'text', placeholder: 'github.com/username', icon: 'i-lucide-code-xml' },
  { key: 'linkedin', label: 'LinkedIn', type: 'text', placeholder: 'linkedin.com/in/username', icon: 'i-lucide-link' },
  { key: 'city', label: 'City', type: 'text', placeholder: 'City, e.g. Iași', icon: 'i-lucide-map-pin' },
  { key: 'country', label: 'Country', type: 'text', placeholder: 'Country, e.g. Romania', icon: 'i-lucide-flag' },
]
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
  <UFormField v-for="field in FIELDS" :key="field.key" :label="field.label" :error="errors[field.key]" :ui="{ label: 'sr-only' }">
    <div class="flex items-center gap-2.5">
      <span class="flex size-9 shrink-0 items-center justify-center rounded-lg bg-accented text-hoply-400">
        <UIcon :name="field.icon" class="size-[18px]" />
      </span>
      <UInput v-model="values[field.key]" :type="field.type" :placeholder="field.placeholder" maxlength="200" class="min-w-0 flex-1" @change="save(field.key)" />
    </div>
  </UFormField>
</template>
