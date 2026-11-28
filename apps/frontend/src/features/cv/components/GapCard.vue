<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import type { PrepQuestions } from '../api/cv.schema'
import { gapKey, type Prep } from '../composables/usePrep'
import RolePicker from './RolePicker.vue'
import StatusTag from './StatusTag.vue'

const props = defineProps<{ gap: PrepQuestions['gaps'][number], prep: Prep, inSkills: boolean }>()
defineEmits<{ addSkill: [skill: string] }>()
const key = computed(() => gapKey(props.gap.skill))
const answers = computed(() => props.prep.answers)
const answer = computed(() => answers.value.answer(key.value))
</script>

<template>
  <div class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3" data-test="gap">
    <div class="flex items-center justify-between gap-2">
      <h3 class="m-0 text-[17px] font-extrabold tracking-tight">{{ gap.skill }}</h3>
      <StatusTag v-if="!answer" :label="t.cv.cards.missing" tone="missing" />
      <StatusTag v-else-if="answer.status === 'added'" :label="t.cv.cards.added" tone="added" />
      <StatusTag v-else :label="t.cv.cards.skipped" tone="muted" />
    </div>
    <template v-if="!answer">
      <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ gap.ask }}</p>
      <UTextarea
        :model-value="answers.draft(key)"
        :aria-label="t.cv.cards.experienceWith(gap.skill)"
        :placeholder="gap.example"
        :rows="3"
        class="w-full"
        @update:model-value="(value: string) => answers.setDraft(key, String(value))"
      />
      <RolePicker :name="gap.skill" :choices="prep.roleChoices(key)" @pick="id => prep.pickRole(key, id)" />
      <div class="flex items-center gap-2.5">
        <UButton :label="t.cv.cards.addToCv" :disabled="!answers.canAdd(key)" :loading="answers.state.saving[key]" @click="prep.addGap(gap.skill)" />
        <UButton :label="t.cv.cards.skip" color="neutral" variant="outline" @click="answers.skip(key)" />
      </div>
    </template>
    <template v-else-if="answer.status === 'added'">
      <p class="m-0 text-sm leading-snug font-medium">{{ answers.textOf(key) }}</p>
      <p class="m-0 text-[13px] font-medium text-muted">{{ t.cv.cards.savedUnder(answers.savedUnder(key)) }}</p>
      <div class="flex flex-wrap items-center gap-3">
        <span v-if="inSkills" class="text-[13px] font-bold text-hoply-300">{{ t.cv.gap.inSkills }}</span>
        <UButton v-else :label="t.cv.gap.alsoAdd(gap.skill)" color="neutral" variant="outline" size="sm" @click="$emit('addSkill', gap.skill)" />
        <UButton :label="t.cv.cards.change" variant="link" color="neutral" size="sm" class="px-1 text-[13px] text-muted underline" :loading="answers.state.saving[key]" @click="answers.undo(key)" />
      </div>
    </template>
    <UButton v-else :label="t.cv.cards.changeMind" variant="link" color="neutral" size="sm" class="self-start px-1 text-[13px] text-muted underline" @click="answers.undo(key)" />
    <p v-if="answers.state.errors[key]" role="alert" class="m-0 text-sm font-bold text-error">{{ answers.state.errors[key] }}</p>
  </div>
</template>
