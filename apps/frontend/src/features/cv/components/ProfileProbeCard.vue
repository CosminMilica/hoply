<script setup lang="ts">
import { computed } from 'vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { Profile } from '@/features/profile/api/profile.schema'
import { useProfileProbeQuery } from '../composables/useCv'
import { useAnswers } from '../composables/useAnswers'
import { useNotes } from '../composables/useNotes'
import ProbeQuestionCard from './ProbeQuestionCard.vue'

const props = defineProps<{ profile: Profile }>()
const probe = useProfileProbeQuery()
const asked = computed(() => (probe.data.value?.status === 'ready' ? probe.data.value.questions : []))
const answers = useAnswers(useNotes(() => props.profile))
</script>

<template>
  <section v-if="probe.data.value?.status === 'pending'" class="flex flex-col gap-2 rounded-2xl border border-default p-4" aria-live="polite" data-test="profile-probe-pending">
    <SectionLabel as="h2">Make every CV stronger</SectionLabel>
    <p class="m-0 text-[13px] font-medium text-muted">Reading your profile to find what every CV would gain from…</p>
    <USkeleton class="h-16 w-full rounded-[10px]" />
  </section>
  <section v-else-if="asked.length" class="flex flex-col gap-2.5 rounded-2xl border border-default p-4" aria-label="Make every CV stronger" data-test="profile-probe">
    <SectionLabel as="h2">Make every CV stronger</SectionLabel>
    <p class="m-0 text-[13px] leading-snug font-medium text-muted">
      Every CV leans on these. A number makes each one stronger; an honest range is fine when you do not know the exact figure.
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
      add-label="Add to my profile"
      @update:draft="(value: string) => answers.setDraft(`${index}`, value)"
      @add="answers.add(`${index}`, item.place)"
      @skip="answers.skip(`${index}`)"
      @undo="answers.undo(`${index}`)"
    />
  </section>
</template>
