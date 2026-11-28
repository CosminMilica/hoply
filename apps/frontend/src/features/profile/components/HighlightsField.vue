<script setup lang="ts">
import { t } from '@/features/common/i18n'
import type { Bullet } from '../api/profile.schema'

const MAX = 10

defineProps<{ error?: string }>()
const rows = defineModel<Bullet[]>({ required: true })

function add() {
  if (rows.value.length < MAX) rows.value = [...rows.value, { text: '', result: '', skills: [] }]
}
function remove(index: number) {
  rows.value = rows.value.filter((_, at) => at !== index)
}
</script>

<template>
  <fieldset class="flex flex-col gap-2">
    <legend class="mb-2 text-xs font-extrabold tracking-wider text-muted uppercase">{{ t.profile.highlights.legend }}</legend>
    <div v-for="(row, index) in rows" :key="index" class="flex gap-2">
      <div class="flex min-w-0 flex-1 flex-col gap-1">
        <UFormField :label="t.profile.highlights.highlight(index + 1)" :ui="{ label: 'sr-only' }">
          <UInput v-model="row.text" :placeholder="t.profile.highlights.placeholders[Math.min(index, 1)]" maxlength="300" class="w-full" />
        </UFormField>
        <UFormField :label="t.profile.highlights.result(index + 1)" :ui="{ label: 'sr-only' }">
          <UInput
            v-model="row.result"
            :placeholder="t.profile.highlights.resultPlaceholder"
            maxlength="200"
            size="sm"
            icon="i-lucide-trending-up"
            class="w-full"
          />
        </UFormField>
      </div>
      <UButton
        color="neutral"
        variant="ghost"
        icon="i-lucide-x"
        :aria-label="t.profile.highlights.remove(index + 1)"
        class="size-9 shrink-0 justify-center text-muted"
        @click="remove(index)"
      />
    </div>
    <p v-if="error" class="m-0 text-sm text-error">{{ error }}</p>
    <UButton
      v-if="rows.length < MAX"
      color="neutral"
      variant="link"
      size="sm"
      :label="t.profile.highlights.add"
      class="self-start px-1 text-toned"
      @click="add"
    />
  </fieldset>
</template>
