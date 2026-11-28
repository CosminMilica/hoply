<script setup lang="ts">
import { ref } from 'vue'
import { t } from '@/features/common/i18n'

defineProps<{ name: string, what: 'role' | 'project', pending?: boolean }>()
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
      :label="t.common.actions.edit"
      :aria-label="t.profile.entry.edit(name)"
      class="gap-1.5 px-1 text-[13px] font-bold text-default"
      :ui="{ leadingIcon: 'size-3.5' }"
      @click="emit('edit')"
    />
    <UButton variant="link" color="neutral" :label="t.profile.entry.removeThis[what]" :aria-label="t.profile.entry.remove(name)" class="px-1 text-[13px] text-muted underline" @click="confirming = true" />
  </div>
  <div v-else class="flex flex-wrap items-center gap-2">
    <span class="text-[13px] font-bold">{{ t.profile.entry.confirmRemove(name) }}</span>
    <UButton color="error" variant="soft" size="sm" :label="t.common.actions.remove" class="min-h-9 px-3 text-[13px]" :loading="pending" @click="remove" />
    <UButton color="neutral" variant="link" size="sm" :label="t.profile.entry.keep" class="min-h-9 px-2 text-[13px] text-muted" @click="confirming = false" />
  </div>
</template>
