<script setup lang="ts">
import StatusTag from './StatusTag.vue'
import type { ProbeQuestion } from '../api/cv.schema'

/** What the person did with the question: added an answer (kept at `where`), skipped it, or nothing yet. */
export type ProbeAnswer = { status: 'added', text: string, where: string } | { status: 'skipped' } | undefined

defineProps<{
  item: ProbeQuestion
  answer: ProbeAnswer
  draft: string
  canAdd: boolean
  saving?: boolean
  error?: string
  addLabel: string
}>()
const emit = defineEmits<{ 'update:draft': [value: string], 'add': [], 'skip': [], 'undo': [] }>()
</script>

<template>
  <div class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3" data-test="probe">
    <div class="flex items-center justify-between gap-2">
      <span class="text-xs font-extrabold tracking-wider text-hoply-300 uppercase">{{ item.theme }}</span>
      <StatusTag v-if="answer?.status === 'added'" label="Added" tone="added" />
      <StatusTag v-else-if="answer" label="Skipped" tone="muted" />
    </div>
    <p v-if="item.about" class="m-0 border-s-2 border-ink-600 ps-2.5 text-[13px] leading-snug font-medium text-muted">{{ item.about }}</p>
    <h3 class="m-0 text-[15px] leading-snug font-bold">{{ item.question }}</h3>
    <template v-if="!answer">
      <p v-if="item.why" class="m-0 text-[13px] leading-snug font-medium text-muted">{{ item.why }}</p>
      <div v-if="item.choices.length" class="flex flex-wrap gap-1.5" role="group" :aria-label="`Answers to pick for: ${item.question}`">
        <UButton
          v-for="choice in item.choices"
          :key="choice"
          :label="choice"
          :aria-pressed="draft === choice"
          color="neutral"
          :variant="draft === choice ? 'soft' : 'outline'"
          size="sm"
          :class="draft === choice ? 'bg-ink-700' : ''"
          @click="emit('update:draft', choice)"
        />
      </div>
      <UTextarea
        :model-value="draft"
        :aria-label="item.question"
        :placeholder="item.choices.length ? 'Pick one above, or answer in your own words' : item.example"
        :rows="2"
        class="w-full"
        @update:model-value="(value: string) => emit('update:draft', String(value))"
      />
      <div class="flex items-center gap-2.5">
        <UButton :label="addLabel" :disabled="!canAdd" :loading="saving" @click="emit('add')" />
        <UButton label="Skip" color="neutral" variant="outline" @click="emit('skip')" />
      </div>
    </template>
    <template v-else-if="answer.status === 'added'">
      <p class="m-0 text-sm leading-snug font-medium">{{ answer.text }}</p>
      <p class="m-0 text-[13px] font-medium text-muted">Saved in your profile under {{ answer.where }}</p>
      <UButton label="Change" variant="link" color="neutral" size="sm" class="self-start px-1 text-[13px] text-muted underline" :loading="saving" @click="emit('undo')" />
    </template>
    <UButton v-else label="Change my mind" variant="link" color="neutral" size="sm" class="self-start px-1 text-[13px] text-muted underline" @click="emit('undo')" />
    <p v-if="error" role="alert" class="m-0 text-sm font-bold text-error">{{ error }}</p>
  </div>
</template>
