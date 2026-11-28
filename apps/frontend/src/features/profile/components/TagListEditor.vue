<script setup lang="ts">
import { ref } from 'vue'
import { t } from '@/features/common/i18n'
import { cleanList } from '../api/profile.schema'

const props = withDefaults(defineProps<{ items: string[], label: string, placeholder: string, maxlength?: number, max?: number }>(), { maxlength: 60, max: 50 })
const emit = defineEmits<{ change: [items: string[]] }>()
const draft = ref('')

function add() {
  const value = draft.value.trim()
  if (!value) return
  draft.value = ''
  if (props.items.some(item => item.toLowerCase() === value.toLowerCase()) || props.items.length >= props.max) return
  emit('change', cleanList([...props.items, value]))
}
</script>

<template>
  <ul v-if="items.length" class="-mx-0.5 flex flex-wrap">
    <li v-for="item in items" :key="item">
      <button type="button" class="flex min-h-11 items-center px-0.5" :aria-label="t.profile.entry.remove(item)" @click="emit('change', items.filter(other => other !== item))">
        <span class="flex h-8 items-center gap-1.5 rounded-lg bg-accented px-3 text-sm font-extrabold text-highlighted">
          {{ item }}<span aria-hidden="true" class="text-[15px] opacity-60">×</span>
        </span>
      </button>
    </li>
  </ul>
  <form class="flex gap-2" @submit.prevent="add">
    <UFormField :label="label" :ui="{ label: 'sr-only' }" class="min-w-0 flex-1">
      <UInput v-model="draft" :placeholder="placeholder" :maxlength="maxlength" class="w-full" />
    </UFormField>
    <UButton type="submit" color="neutral" :label="t.profile.entry.add" class="px-[18px]" />
  </form>
</template>
