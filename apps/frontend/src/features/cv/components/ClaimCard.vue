<script setup lang="ts">
import { computed } from 'vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { PrepQuestions } from '../api/cv.schema'
import { claimKey, type Prep } from '../composables/usePrep'
import RolePicker from './RolePicker.vue'
import StatusTag from './StatusTag.vue'

const props = defineProps<{ claim: PrepQuestions['responsibilities'][number], prep: Prep }>()
const STATUS_LABEL = { added: 'Added', picked: 'From your profile', skipped: 'Left out' }
const key = computed(() => claimKey(props.claim.index))
const answers = computed(() => props.prep.answers)
const answer = computed(() => answers.value.answer(key.value))
const suggestions = computed(() => props.prep.suggestionsFor(props.claim.text))
</script>

<template>
  <div class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3" data-test="claim">
    <h3 class="m-0 text-sm leading-snug font-bold">{{ claim.question }}</h3>
    <template v-if="!answer">
      <div v-if="suggestions.length" class="flex flex-col gap-1.5" role="group" :aria-label="`From your profile: ${claim.question}`">
        <SectionLabel as="span">From your profile</SectionLabel>
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
        aria-label="What exactly did you do?"
        placeholder="In your own words: what did you do, where, and what changed? Numbers help."
        :rows="3"
        class="w-full"
        @update:model-value="(value: string) => answers.setDraft(key, String(value))"
      />
      <UButton
        v-if="!answers.draft(key).trim()"
        label="Start from the job’s wording, then make it yours"
        variant="link"
        size="sm"
        class="self-start px-1 text-[13px] text-hoply-400 underline"
        @click="answers.setDraft(key, claim.prefill)"
      />
      <RolePicker :name="claim.question" :choices="prep.roleChoices(key)" @pick="id => prep.pickRole(key, id)" />
      <div class="flex flex-wrap items-center gap-2.5">
        <UButton label="Add to my CV" :disabled="!answers.canAdd(key)" :loading="answers.state.saving[key]" @click="prep.claimYes(claim.index)" />
        <UButton label="Not really" color="neutral" variant="outline" @click="answers.skip(key)" />
      </div>
    </template>
    <template v-else>
      <div class="flex items-center justify-between gap-2">
        <StatusTag :label="STATUS_LABEL[answer.status]" :tone="answer.status === 'skipped' ? 'muted' : 'added'" />
        <UButton label="Change" variant="link" color="neutral" size="sm" class="px-1 text-[13px] text-muted underline" :loading="answers.state.saving[key]" @click="answers.undo(key)" />
      </div>
      <template v-if="answer.status !== 'skipped'">
        <p class="m-0 text-sm leading-snug font-medium">{{ answers.textOf(key) }}</p>
        <p v-if="answer.status === 'added'" class="m-0 text-[13px] font-medium text-muted">Saved in your profile under {{ answers.savedUnder(key) }}</p>
      </template>
    </template>
    <p v-if="answers.state.errors[key]" role="alert" class="m-0 text-sm font-bold text-error">{{ answers.state.errors[key] }}</p>
  </div>
</template>
