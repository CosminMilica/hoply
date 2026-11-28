<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import type { PrepQuestions } from '../api/cv.schema'
import { gapKey, type Prep } from '../composables/usePrep'
import RolePicker from './RolePicker.vue'
import StatusTag from './StatusTag.vue'

const props = defineProps<{ close: PrepQuestions['related'][number], prep: Prep }>()
const key = computed(() => gapKey(props.close.skill))
const answers = computed(() => props.prep.answers)
const added = computed(() => answers.value.answer(key.value)?.status === 'added')
</script>

<template>
  <div class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3" data-test="close">
    <div class="flex items-center justify-between gap-2">
      <h3 class="m-0 text-[17px] font-extrabold tracking-tight">{{ close.skill }}</h3>
      <StatusTag v-if="added" :label="t.cv.cards.added" tone="added" />
      <StatusTag v-else :label="t.cv.cards.close" tone="close" />
    </div>
    <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ t.cv.closeCard.note(close) }}</p>
    <template v-if="added">
      <p class="m-0 text-sm leading-snug font-medium">{{ answers.textOf(key) }}</p>
      <UButton :label="t.cv.cards.change" variant="link" color="neutral" size="sm" class="self-start px-1 text-[13px] text-muted underline" :loading="answers.state.saving[key]" @click="answers.undo(key)" />
    </template>
    <details v-else class="flex flex-col gap-2">
      <summary class="cursor-pointer text-[13px] font-bold text-hoply-300">{{ t.cv.closeCard.usedMyself(close.skill) }}</summary>
      <div class="mt-2 flex flex-col gap-2">
        <UTextarea
          :model-value="answers.draft(key)"
          :aria-label="t.cv.cards.experienceWith(close.skill)"
          :placeholder="t.cv.closeCard.placeholder(close.skill)"
          :rows="3"
          class="w-full"
          @update:model-value="(value: string) => answers.setDraft(key, String(value))"
        />
        <RolePicker :name="close.skill" :choices="prep.roleChoices(key)" @pick="id => prep.pickRole(key, id)" />
        <UButton :label="t.cv.cards.addToCv" class="self-start" :disabled="!answers.canAdd(key)" :loading="answers.state.saving[key]" @click="prep.addGap(close.skill)" />
      </div>
    </details>
    <p v-if="answers.state.errors[key]" role="alert" class="m-0 text-sm font-bold text-error">{{ answers.state.errors[key] }}</p>
  </div>
</template>
