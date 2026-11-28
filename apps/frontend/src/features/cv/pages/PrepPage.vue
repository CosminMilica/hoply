<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { t } from '@/features/common/i18n'
import { useApplicationMutations, useApplicationStatuses } from '@/features/applications/composables/useApplications'
import BackButton from '@/features/common/components/BackButton.vue'
import ProgressBar from '@/features/common/components/ProgressBar.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import { useJobQuery } from '@/features/jobs/composables/useJobs'
import { requirementsOf } from '@/features/jobs/match'
import { cleanList } from '@/features/profile/api/profile.schema'
import { useProfileMutations, useProfileQuery } from '@/features/profile/composables/useProfile'
import type { CvStatus } from '../api/cv.schema'
import ClaimCard from '../components/ClaimCard.vue'
import CloseCard from '../components/CloseCard.vue'
import GapCard from '../components/GapCard.vue'
import GeneratingSteps from '../components/GeneratingSteps.vue'
import ImpactRows from '../components/ImpactRows.vue'
import ProbeQuestionCard from '../components/ProbeQuestionCard.vue'
import { useCvProbeQuery, useCvProgress, useCvQuestionsQuery, useGenerateCv } from '../composables/useCv'
import { probeKey, usePrep } from '../composables/usePrep'
import { CV_LANGUAGES } from '../language'

const props = defineProps<{ id: string }>()
const STEP_AT = [900, 1800, 2700, 3600]
const DONE_AT = 4500

const router = useRouter()
const job = useJobQuery(() => props.id)
const profile = useProfileQuery()
const questions = useCvQuestionsQuery(() => props.id)
const probe = useCvProbeQuery(() => props.id)
const asked = computed(() => (probe.data.value?.status === 'ready' ? probe.data.value.questions : []))
const { setSkills } = useProfileMutations()
const { statuses } = useApplicationStatuses()
const { setStatus } = useApplicationMutations()
const generateCv = useGenerateCv()
const prep = usePrep({ job: () => job.data.value, profile: () => profile.data.value, questions: () => questions.data.value, probe: () => asked.value })

const ready = computed(() => job.data.value && profile.data.value && questions.data.value)
const failed = computed(() => job.isError.value || profile.isError.value || questions.isError.value)
const skills = computed(() => (profile.data.value?.skills ?? []).map(skill => skill.toLowerCase()))

const generating = ref(false)
const step = ref(0)
const elapsed = ref(false)
const cvId = ref<string | null>(null)
const writeFailed = ref(false)
const progress = useCvProgress(cvId)
let timers: number[] = []

/** How far the service has got: queued, then written, then checked. */
const SERVER_STEP: Record<CvStatus, number> = { queued: 0, writing: 2, checking: 3, ready: 4, failed: 0 }
const cv = computed(() => (cvId.value ? progress.data.value : undefined))
const shownStep = computed(() => Math.min(step.value, cv.value ? SERVER_STEP[cv.value.status] : 0))

function clearTimers() {
  timers.forEach(timer => window.clearTimeout(timer))
  timers = []
}

function stop() {
  clearTimers()
  generating.value = false
  cvId.value = null
}

function generate() {
  clearTimers()
  generating.value = true
  writeFailed.value = false
  step.value = 0
  elapsed.value = false
  cvId.value = null
  timers = [
    ...STEP_AT.map((ms, index) => window.setTimeout(() => { step.value = index + 1 }, ms)),
    window.setTimeout(() => { elapsed.value = true }, DONE_AT),
  ]
  generateCv.mutate(prep.toRequest(props.id), {
    onSuccess: (queued) => { cvId.value = queued.id },
    onError: stop,
  })
}

watch(() => cv.value?.status, (status) => {
  if (status === 'ready') {
    setStatus.mutate({ jobId: props.id, status: statuses.value.get(props.id) === 'applied' ? 'applied' : 'cv_ready' })
  }
})
watch(() => cv.value?.status === 'failed' || (Boolean(cvId.value) && progress.isError.value), (failed) => {
  if (!failed) return
  stop()
  writeFailed.value = true
})
watch([elapsed, () => cv.value?.status], ([done, status]) => {
  if (done && status === 'ready') router.replace({ name: 'crashr-cv', params: { id: props.id } })
})
onBeforeUnmount(clearTimers)

function addToProfile(skill: string) {
  setSkills.mutate(cleanList([...(profile.data.value?.skills ?? []), skill]))
}
</script>

<template>
  <GeneratingSteps
    v-if="generating && job.data.value"
    :step="shownStep"
    :title="job.data.value.title"
    :company="job.data.value.company"
    :keywords="requirementsOf(job.data.value)"
    :matched="prep.matched.value"
    :preview="cv?.preview"
    :highlights="prep.matched.value"
  />
  <div v-else class="mx-auto w-full max-w-[1080px]">
    <UAlert v-if="failed" color="error" variant="subtle" :title="t.cv.prep.failedTitle" :description="t.cv.prep.failedDescription" />
    <USkeleton v-else-if="!ready" class="h-64 w-full" />
    <div v-else-if="job.data.value && questions.data.value" class="flex flex-col gap-4 min-[900px]:grid min-[900px]:grid-cols-[minmax(0,1fr)_340px] min-[900px]:items-start min-[900px]:gap-x-12">
      <BackButton :fallback="{ name: 'crashr-job', params: { id } }" class="min-[900px]:col-span-2" />
      <div class="-mt-2.5 flex flex-col gap-1 min-[900px]:col-start-1">
        <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">{{ t.cv.prep.title }}</h1>
        <p class="m-0 text-[15px] font-medium text-muted">{{ job.data.value.title }} · {{ job.data.value.company }}</p>
      </div>

      <aside class="contents min-[900px]:sticky min-[900px]:top-0 min-[900px]:col-start-2 min-[900px]:row-span-2 min-[900px]:row-start-2 min-[900px]:flex min-[900px]:flex-col min-[900px]:gap-4" :aria-label="t.cv.prep.progressLabel">
        <div class="flex flex-col gap-2.5 rounded-2xl border border-default bg-elevated p-4">
          <div class="flex items-baseline justify-between">
            <span class="text-xs font-extrabold tracking-wider text-ink-300 uppercase">{{ t.cv.prep.keywordMatch }}</span>
            <span class="text-2xl font-extrabold text-crashr-400" data-test="match">{{ prep.matchLabel.value }}</span>
          </div>
          <ProgressBar :value="prep.score.value" :label="t.cv.prep.keywordMatch" />
          <span class="text-[13px] leading-snug font-medium text-ink-300">{{ t.cv.prep.matchHint }}</span>
          <span class="text-[13px] font-extrabold">{{ prep.progress.value }}</span>
        </div>
        <div class="order-last flex flex-col gap-2 min-[900px]:order-none">
          <div class="flex items-center justify-between gap-3">
            <span id="cv-language" class="text-xs font-extrabold tracking-wider text-ink-300 uppercase">{{ t.cv.prep.cvLanguage }}</span>
            <div role="group" aria-labelledby="cv-language" class="flex gap-0.5 rounded-[10px] bg-accented p-0.5">
              <UButton
                v-for="option in CV_LANGUAGES"
                :key="option.id"
                :label="option.label"
                :aria-pressed="prep.language.value === option.id"
                size="sm"
                color="neutral"
                :variant="prep.language.value === option.id ? 'solid' : 'ghost'"
                @click="prep.state.language = option.id"
              />
            </div>
          </div>
          <UButton :label="t.cv.prep.generate(prep.score.value)" size="lg" block class="justify-center" :loading="generateCv.isPending.value" @click="generate" />
          <p v-if="writeFailed" role="alert" class="m-0 text-sm font-bold text-error">{{ t.cv.prep.writeFailed }}</p>
        </div>
      </aside>

      <div class="flex flex-col gap-4 min-[900px]:col-start-1">
        <section v-if="questions.data.value.covered.length" class="flex flex-col gap-2">
          <SectionLabel>{{ t.cv.prep.covered }}</SectionLabel>
          <ul class="flex flex-wrap gap-1.5">
            <li v-for="skill in questions.data.value.covered" :key="skill" class="rounded-lg bg-crashr-900 px-2.5 py-1 text-[13px] font-bold text-crashr-300">{{ skill }}</li>
          </ul>
        </section>

        <section v-if="probe.data.value?.status === 'pending'" class="flex flex-col gap-2" aria-live="polite" data-test="probe-pending">
          <SectionLabel>{{ t.cv.prep.probeTitle }}</SectionLabel>
          <p class="m-0 text-[13px] font-medium text-muted">{{ t.cv.prep.probeReading }}</p>
          <USkeleton class="h-20 w-full rounded-[10px]" />
        </section>
        <section v-else-if="asked.length" class="flex flex-col gap-2.5">
          <SectionLabel>{{ t.cv.prep.probeTitle }}</SectionLabel>
          <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ t.cv.prep.probeIntro }}</p>
          <ProbeQuestionCard
            v-for="(item, index) in asked"
            :key="index"
            :item="item"
            :answer="prep.answers.shown(probeKey(index))"
            :draft="prep.answers.draft(probeKey(index))"
            :can-add="prep.answers.canAdd(probeKey(index))"
            :saving="prep.answers.state.saving[probeKey(index)]"
            :error="prep.answers.state.errors[probeKey(index)]"
            :add-label="t.cv.cards.addToCv"
            @update:draft="(value: string) => prep.answers.setDraft(probeKey(index), value)"
            @add="prep.answerProbe(index, item)"
            @skip="prep.answers.skip(probeKey(index))"
            @undo="prep.answers.undo(probeKey(index))"
          />
        </section>

        <section v-if="questions.data.value.related.length" class="flex flex-col gap-2.5">
          <SectionLabel>{{ t.cv.prep.close }}</SectionLabel>
          <CloseCard v-for="close in questions.data.value.related" :key="close.skill" :close="close" :prep="prep" />
        </section>

        <section v-if="questions.data.value.gaps.length" class="flex flex-col gap-2.5">
          <SectionLabel>{{ t.cv.prep.gaps }}</SectionLabel>
          <GapCard
            v-for="gap in questions.data.value.gaps"
            :key="gap.skill"
            :gap="gap"
            :prep="prep"
            :in-skills="skills.includes(gap.skill.toLowerCase())"
            @add-skill="addToProfile"
          />
        </section>

        <section v-if="!asked.length" class="flex flex-col gap-2.5">
          <SectionLabel>{{ t.cv.prep.claims }}</SectionLabel>
          <ClaimCard v-for="claim in questions.data.value.responsibilities" :key="claim.index" :claim="claim" :prep="prep" />
        </section>

        <ImpactRows :prep="prep" />

        <section class="flex flex-col gap-2">
          <SectionLabel>{{ t.cv.prep.roles }}</SectionLabel>
          <button
            v-for="role in prep.roles.value"
            :key="role.id"
            type="button"
            :aria-pressed="role.included"
            class="flex min-h-14 items-center gap-2.5 rounded-[10px] border border-default bg-elevated px-3 py-2 text-left"
            @click="prep.toggleRole(role.id)"
          >
            <span class="flex min-w-0 flex-1 flex-col">
              <span class="text-[15px] font-extrabold">{{ role.role }}</span>
              <span class="text-[13px] font-medium text-muted">{{ role.meta }}</span>
              <span class="text-xs font-bold text-toned">{{ role.overlap }}</span>
            </span>
            <span
              class="rounded-lg border-[1.5px] px-3 py-1 text-xs font-extrabold"
              :class="role.included ? 'border-crashr-600 bg-crashr-600 text-white' : 'border-ink-700 text-toned'"
            >{{ role.included ? t.cv.prep.included : t.cv.prep.leftOut }}</span>
          </button>
        </section>

        <section v-if="prep.focusChips.value.length" class="flex flex-col gap-2">
          <SectionLabel>{{ t.cv.prep.focus }}</SectionLabel>
          <div class="flex flex-wrap gap-2" role="group" :aria-label="t.cv.prep.focusLabel">
            <ToggleChip v-for="chip in prep.focusChips.value" :key="chip.label" :label="chip.label" :pressed="chip.pressed" @toggle="prep.toggleFocus(chip.label)" />
          </div>
        </section>

        <UFormField
          :label="t.cv.prep.extraLabel"
          :description="t.cv.prep.extraDescription"
          :error="prep.answers.state.errors.note"
          :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase', description: 'text-[13px] leading-snug font-medium' }"
        >
          <UTextarea
            v-model="prep.note.value"
            :rows="3"
            :placeholder="t.cv.prep.extraPlaceholder"
            class="w-full"
            @change="prep.saveNote()"
          />
        </UFormField>
      </div>
    </div>
  </div>
</template>
