<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { PrepQuestions } from '../api/cv.schema'
import { claimKey, type Prep } from '../composables/usePrep'
import RolePicker from './RolePicker.vue'
import StatusTag from './StatusTag.vue'

const props = defineProps<{ claim: PrepQuestions['responsibilities'][number], prep: Prep }>()
const key = computed(() => claimKey(props.claim.index))
const answers = computed(() => props.prep.answers)
const answer = computed(() => answers.value.answer(key.value))
const suggestions = computed(() => props.prep.suggestionsFor(props.claim.text))
const statusLabel = computed(() => {
  const words = t.value.cv.cards
  return { added: words.added, picked: words.fromProfile, skipped: words.leftOut }[answer.value?.status ?? 'skipped']
})
</script>

<template>
  <div class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3" data-test="claim">
    <h3 class="m-0 text-sm leading-snug font-bold">{{ claim.question }}</h3>
    <template v-if="!answer">
      <div v-if="suggestions.length" class="flex flex-col gap-1.5" role="group" :aria-label="t.cv.claim.fromProfileFor(claim.question)">
        <SectionLabel as="span">{{ t.cv.cards.fromProfile }}</SectionLabel>
        <button
          v-for="suggestion in suggestions"
          :key="suggestion.text"
          type="button"
          :aria-pressed="false"
          class="flex min-h-11 items-start gap-2 rounded-lg border border-default px-3 py-2 text-left text-[13px] font-medium hover:bg-accented"
          @click="answers.pick(key, suggestion.text)"
        >
          <UIcon name="i-lucide-check" class="mt-0.5 size-3.5 shrink-0 text-hoply-400" />
          <span><span class="font-bold">{{ suggestion.label }}:</span> {{ suggestion.text }}</span>
        </button>
      </div>
      <UTextarea
        :model-value="answers.draft(key)"
        :aria-label="t.cv.claim.whatDid"
        :placeholder="t.cv.claim.placeholder"
        :rows="3"
        class="w-full"
        @update:model-value="(value: string) => answers.setDraft(key, String(value))"
      />
      <UButton
        v-if="!answers.draft(key).trim()"
        :label="t.cv.claim.prefill"
        variant="link"
        size="sm"
        class="self-start px-1 text-[13px] text-hoply-400 underline"
        @click="answers.setDraft(key, claim.prefill)"
      />
      <RolePicker :name="claim.question" :choices="prep.roleChoices(key)" @pick="id => prep.pickRole(key, id)" />
      <div class="flex flex-wrap items-center gap-2.5">
        <UButton :label="t.cv.cards.addToCv" :disabled="!answers.canAdd(key)" :loading="answers.state.saving[key]" @click="prep.claimYes(claim.index)" />
        <UButton :label="t.cv.claim.notReally" color="neutral" variant="outline" @click="answers.skip(key)" />
      </div>
    </template>
    <template v-else>
      <div class="flex items-center justify-between gap-2">
        <StatusTag :label="statusLabel" :tone="answer.status === 'skipped' ? 'muted' : 'added'" />
        <UButton :label="t.cv.cards.change" variant="link" color="neutral" size="sm" class="px-1 text-[13px] text-muted underline" :loading="answers.state.saving[key]" @click="answers.undo(key)" />
      </div>
      <template v-if="answer.status !== 'skipped'">
        <p class="m-0 text-sm leading-snug font-medium">{{ answers.textOf(key) }}</p>
        <p v-if="answer.status === 'added'" class="m-0 text-[13px] font-medium text-muted">{{ t.cv.cards.savedUnder(answers.savedUnder(key)) }}</p>
      </template>
    </template>
    <p v-if="answers.state.errors[key]" role="alert" class="m-0 text-sm font-bold text-error">{{ answers.state.errors[key] }}</p>
  </div>
</template>
