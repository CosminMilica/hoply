<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ name: string, what: string, pending?: boolean }>()
const emit = defineEmits<{ edit: [], remove: [] }>()
const confirming = ref(false)

function remove() {
  emit('remove')
  confirming.value = false
}
</script>

<template>
  <div v-if="!confirming" class="flex flex-wrap items-center gap-3">
    <UButton
      color="neutral"
      variant="link"
      icon="i-lucide-pencil"
      label="Edit"
      :aria-label="`Edit ${name}`"
      class="gap-1.5 px-1 text-[13px] font-bold text-default"
      :ui="{ leadingIcon: 'size-3.5' }"
      @click="emit('edit')"
    />
    <UButton variant="link" color="neutral" :label="`Remove this ${what}`" :aria-label="`Remove ${name}`" class="px-1 text-[13px] text-muted underline" @click="confirming = true" />
  </div>
  <div v-else class="flex flex-wrap items-center gap-2">
    <span class="text-[13px] font-bold">Remove {{ name }}?</span>
    <UButton color="error" variant="soft" size="sm" label="Remove" class="min-h-9 px-3 text-[13px]" :loading="pending" @click="remove" />
    <UButton color="neutral" variant="link" size="sm" label="Keep it" class="min-h-9 px-2 text-[13px] text-muted" @click="confirming = false" />
  </div>
</template>
