<script setup lang="ts">
import { ref, watch } from 'vue'
import { t } from '@/features/common/i18n'
import type { Profile } from '../api/profile.schema'
import { useProfileMutations } from '../composables/useProfile'

const props = defineProps<{ profile: Profile }>()
const { patch } = useProfileMutations()
const name = ref(props.profile.name)
const headline = ref(props.profile.headline)
const about = ref(props.profile.about)
watch(() => props.profile.name, (value) => { name.value = value })
watch(() => props.profile.headline, (value) => { headline.value = value })
watch(() => props.profile.about, (value) => { about.value = value })

function save(field: 'name' | 'headline' | 'about', value: string) {
  if (value.trim() !== props.profile[field]) patch.mutate({ [field]: value.trim() })
}
</script>

<template>
  <UFormField :label="t.profile.about.name" :ui="{ label: 'sr-only' }">
    <UInput v-model="name" :placeholder="t.profile.about.namePlaceholder" maxlength="120" autocomplete="name" class="w-full" @change="save('name', name)" />
  </UFormField>
  <UFormField :label="t.profile.about.headline" :ui="{ label: 'sr-only' }">
    <UInput v-model="headline" :placeholder="t.profile.about.headlinePlaceholder" maxlength="120" class="w-full" @change="save('headline', headline)" />
  </UFormField>
  <UFormField :label="t.profile.about.about" :ui="{ label: 'sr-only' }">
    <UTextarea v-model="about" :rows="3" maxlength="280" class="w-full" @change="save('about', about)" />
  </UFormField>
  <div class="flex justify-between gap-2 text-[13px] font-medium text-muted">
    <span>{{ t.profile.about.hint }}</span>
    <span>{{ about.length }}/280</span>
  </div>
</template>
