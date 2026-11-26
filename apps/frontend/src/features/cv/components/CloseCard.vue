<script setup lang="ts">
import { computed } from 'vue'
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
      <StatusTag v-if="added" label="Added" tone="added" />
      <StatusTag v-else label="Close" tone="close" />
    </div>
    <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ close.note }}</p>
    <template v-if="added">
      <p class="m-0 text-sm leading-snug font-medium">{{ answers.textOf(key) }}</p>
      <UButton label="Change" variant="link" color="neutral" size="sm" class="self-start px-1 text-[13px] text-muted underline" :loading="answers.state.saving[key]" @click="answers.undo(key)" />
    </template>
    <details v-else class="flex flex-col gap-2">
      <summary class="cursor-pointer text-[13px] font-bold text-hoply-300">I have used {{ close.skill }} myself</summary>
      <div class="mt-2 flex flex-col gap-2">
        <UTextarea
          :model-value="answers.draft(key)"
          :aria-label="`Your experience with ${close.skill}`"
          :placeholder="`Where you used ${close.skill} and what you did with it`"
          :rows="3"
          class="w-full"
          @update:model-value="(value: string) => answers.setDraft(key, String(value))"
        />
        <RolePicker :name="close.skill" :choices="prep.roleChoices(key)" @pick="id => prep.pickRole(key, id)" />
        <UButton label="Add to my CV" class="self-start" :disabled="!answers.canAdd(key)" :loading="answers.state.saving[key]" @click="prep.addGap(close.skill)" />
      </div>
    </details>
    <p v-if="answers.state.errors[key]" role="alert" class="m-0 text-sm font-bold text-error">{{ answers.state.errors[key] }}</p>
  </div>
</template>
