<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { Profile } from '@/features/profile/api/profile.schema'
import { READY_PERCENT, readyForClaude } from '@/features/profile/readiness'
import { useProfileProbeQuery } from '../composables/useCv'
import { useAnswers } from '../composables/useAnswers'
import { useNotes } from '../composables/useNotes'
import ProbeQuestionCard from './ProbeQuestionCard.vue'

const props = defineProps<{ profile: Profile }>()
const ready = computed(() => readyForClaude(props.profile))
const probe = useProfileProbeQuery(ready)
const asked = computed(() => (probe.data.value?.status === 'ready' ? probe.data.value.questions : []))
const answers = useAnswers(useNotes(() => props.profile))
</script>

<template>
  <section v-if="!ready || probe.data.value?.status === 'incomplete'" class="flex flex-col gap-2 rounded-2xl border border-dashed border-default p-4" data-test="profile-probe-later">
    <SectionLabel as="h2">{{ t.cv.profileProbe.title }}</SectionLabel>
    <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ t.profile.gate.questionsLater(READY_PERCENT) }}</p>
  </section>
  <section v-else-if="probe.data.value?.status === 'pending'" class="flex flex-col gap-2 rounded-2xl border border-default p-4" aria-live="polite" data-test="profile-probe-pending">
    <SectionLabel as="h2">{{ t.cv.profileProbe.title }}</SectionLabel>
    <p class="m-0 text-[13px] font-medium text-muted">{{ t.cv.profileProbe.reading }}</p>
    <USkeleton class="h-16 w-full rounded-[10px]" />
  </section>
  <section v-else-if="asked.length" class="flex flex-col gap-2.5 rounded-2xl border border-default p-4" :aria-label="t.cv.profileProbe.title" data-test="profile-probe">
    <SectionLabel as="h2">{{ t.cv.profileProbe.title }}</SectionLabel>
    <p class="m-0 text-[13px] leading-snug font-medium text-muted">
      {{ t.cv.profileProbe.intro }}
    </p>
    <ProbeQuestionCard
      v-for="(item, index) in asked"
      :key="index"
      :item="item"
      :answer="answers.shown(`${index}`)"
      :draft="answers.draft(`${index}`)"
      :can-add="answers.canAdd(`${index}`)"
      :saving="answers.state.saving[index]"
      :error="answers.state.errors[index]"
      :add-label="t.cv.cards.addToProfile"
      @update:draft="(value: string) => answers.setDraft(`${index}`, value)"
      @add="answers.add(`${index}`, item.place)"
      @skip="answers.skip(`${index}`)"
      @undo="answers.undo(`${index}`)"
    />
  </section>
</template>
