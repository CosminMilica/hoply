<script setup lang="ts">
import { ref, watch } from 'vue'

defineProps<{ ongoingLabel: string, errors?: { start?: string, end?: string } }>()
const start = defineModel<string>('start', { required: true })
const end = defineModel<string>('end', { required: true })
const ongoing = ref(Boolean(start.value) && !end.value)
const label = { label: 'text-xs font-bold text-muted' }

watch(ongoing, (value) => {
  if (value) end.value = ''
})
</script>

<template>
  <div class="grid grid-cols-2 gap-2">
    <UFormField label="Start month" :error="errors?.start" :ui="label">
      <UInput v-model="start" type="month" placeholder="YYYY-MM" class="w-full" />
    </UFormField>
    <UFormField v-if="!ongoing" label="End month" :error="errors?.end" :ui="label">
      <UInput v-model="end" type="month" placeholder="YYYY-MM" class="w-full" />
    </UFormField>
  </div>
  <USwitch v-model="ongoing" :label="ongoingLabel" size="sm" />
</template>
