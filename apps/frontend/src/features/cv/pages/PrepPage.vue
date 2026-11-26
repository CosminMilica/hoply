<script setup lang="ts">
import ProbeQuestionCard, { type ProbeAnswer } from '../components/ProbeQuestionCard.vue'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
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
import GeneratingSteps from '../components/GeneratingSteps.vue'
import RolePicker from '../components/RolePicker.vue'
import StatusTag from '../components/StatusTag.vue'
import { useCvProbeQuery, useCvProgress, useCvQuestionsQuery, useGenerateCv } from '../composables/useCv'
import { usePrep } from '../composables/usePrep'

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
  if (done && status === 'ready') router.replace({ name: 'hoply-cv', params: { id: props.id } })
})
onBeforeUnmount(clearTimers)

type Answer = { status: string, remembered?: { text: string, place: string }, text?: string }
const addedText = (answer: Answer) => answer.remembered?.text ?? answer.text ?? ''
const addedPlace = (answer: Answer) => answer.remembered?.place ?? ''
function probeAnswer(index: number): ProbeAnswer {
  const answer = prep.state.probes[index]
  if (!answer) return undefined
  return answer.status === 'added' ? { status: 'added', text: addedText(answer), where: prep.whereLabel(addedPlace(answer)) } : { status: 'skipped' }
}

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
    <UAlert v-if="failed" color="error" variant="subtle" title="We could not prepare this CV" description="Check that your profile loads, then try again." />
    <USkeleton v-else-if="!ready" class="h-64 w-full" />
    <div v-else-if="job.data.value && questions.data.value" class="flex flex-col gap-4 min-[900px]:grid min-[900px]:grid-cols-[minmax(0,1fr)_340px] min-[900px]:items-start min-[900px]:gap-x-12">
      <BackButton :fallback="{ name: 'hoply-job', params: { id } }" class="min-[900px]:col-span-2" />
      <div class="-mt-2.5 flex flex-col gap-1 min-[900px]:col-start-1">
        <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">Prepare your CV</h1>
        <p class="m-0 text-[15px] font-medium text-muted">{{ job.data.value.title }} · {{ job.data.value.company }}</p>
      </div>

      <aside class="contents min-[900px]:sticky min-[900px]:top-0 min-[900px]:col-start-2 min-[900px]:row-span-2 min-[900px]:row-start-2 min-[900px]:flex min-[900px]:flex-col min-[900px]:gap-4" aria-label="Your progress">
        <div class="flex flex-col gap-2.5 rounded-2xl border border-default bg-elevated p-4">
          <div class="flex items-baseline justify-between">
            <span class="text-xs font-extrabold tracking-wider text-ink-300 uppercase">Keyword match</span>
            <span class="text-2xl font-extrabold text-hoply-400" data-test="match">{{ prep.matchLabel.value }}</span>
          </div>
          <ProgressBar :value="prep.score.value" label="Keyword match" />
          <span class="text-[13px] leading-snug font-medium text-ink-300">The more real detail you give, the stronger the CV. We rewrite everything in this job’s words, but only from what you tell us.</span>
          <span class="text-[13px] font-extrabold">{{ prep.progress.value }}</span>
        </div>
        <div class="order-last flex flex-col gap-2 min-[900px]:order-none">
          <UButton :label="`Generate CV · ${prep.score.value}% match`" size="lg" block class="justify-center" :loading="generateCv.isPending.value" @click="generate" />
          <p v-if="writeFailed" role="alert" class="m-0 text-sm font-bold text-error">We could not write this CV. Try again.</p>
        </div>
      </aside>

      <div class="flex flex-col gap-4 min-[900px]:col-start-1">
        <section v-if="questions.data.value.covered.length" class="flex flex-col gap-2">
          <SectionLabel>Already in your profile</SectionLabel>
          <ul class="flex flex-wrap gap-1.5">
            <li v-for="skill in questions.data.value.covered" :key="skill" class="rounded-lg bg-hoply-900 px-2.5 py-1 text-[13px] font-bold text-hoply-300">{{ skill }}</li>
          </ul>
        </section>

        <section v-if="probe.data.value?.status === 'pending'" class="flex flex-col gap-2" aria-live="polite" data-test="probe-pending">
          <SectionLabel>Make your CV stronger for this job</SectionLabel>
          <p class="m-0 text-[13px] font-medium text-muted">Reading the job and your profile to find what is worth asking…</p>
          <USkeleton class="h-20 w-full rounded-[10px]" />
        </section>
        <section v-else-if="asked.length" class="flex flex-col gap-2.5">
          <SectionLabel>Make your CV stronger for this job</SectionLabel>
          <p class="m-0 text-[13px] leading-snug font-medium text-muted">What this job cares about that your profile does not say yet. Answer in your own words; we write the CV from it.</p>
          <ProbeQuestionCard
            v-for="(item, index) in asked"
            :key="index"
            :item="item"
            :answer="probeAnswer(index)"
            :draft="prep.draft(`p:${index}`)"
            :can-add="prep.canAdd(`p:${index}`)"
            :saving="prep.state.saving[`p:${index}`]"
            :error="prep.state.errors[`p:${index}`]"
            add-label="Add to my CV"
            @update:draft="(value: string) => prep.setDraft(`p:${index}`, value)"
            @add="prep.answerProbe(index, item)"
            @skip="prep.skipProbe(index)"
            @undo="prep.undoProbe(index)"
          />
        </section>

        <section v-if="questions.data.value.related.length" class="flex flex-col gap-2.5">
          <SectionLabel>Close to what you know</SectionLabel>
          <div v-for="close in questions.data.value.related" :key="close.skill" class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3" data-test="close">
            <div class="flex items-center justify-between gap-2">
              <h3 class="m-0 text-[17px] font-extrabold tracking-tight">{{ close.skill }}</h3>
              <StatusTag v-if="prep.state.gaps[close.skill]?.status === 'added'" label="Added" tone="added" />
              <StatusTag v-else label="Close" tone="close" />
            </div>
            <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ close.note }}</p>
            <template v-if="prep.state.gaps[close.skill]?.status === 'added'">
              <p class="m-0 text-sm leading-snug font-medium">{{ addedText(prep.state.gaps[close.skill]!) }}</p>
              <UButton label="Change" variant="link" color="neutral" size="sm" class="self-start px-1 text-[13px] text-muted underline" :loading="prep.state.saving[`g:${close.skill}`]" @click="prep.undoGap(close.skill)" />
            </template>
            <details v-else class="flex flex-col gap-2">
              <summary class="cursor-pointer text-[13px] font-bold text-hoply-300">I have used {{ close.skill }} myself</summary>
              <div class="mt-2 flex flex-col gap-2">
                <UTextarea
                  :model-value="prep.draft(`g:${close.skill}`)"
                  :aria-label="`Your experience with ${close.skill}`"
                  :placeholder="`Where you used ${close.skill} and what you did with it`"
                  :rows="3"
                  class="w-full"
                  @update:model-value="(value: string) => prep.setDraft(`g:${close.skill}`, String(value))"
                />
                <RolePicker :name="close.skill" :choices="prep.roleChoices(`g:${close.skill}`)" @pick="id => prep.pickRole(`g:${close.skill}`, id)" />
                <UButton label="Add to my CV" class="self-start" :disabled="!prep.canAdd(`g:${close.skill}`)" :loading="prep.state.saving[`g:${close.skill}`]" @click="prep.addGap(close.skill)" />
              </div>
            </details>
            <p v-if="prep.state.errors[`g:${close.skill}`]" role="alert" class="m-0 text-sm font-bold text-error">{{ prep.state.errors[`g:${close.skill}`] }}</p>
          </div>
        </section>

        <section v-if="questions.data.value.gaps.length" class="flex flex-col gap-2.5">
          <SectionLabel>Fill the gaps</SectionLabel>
          <div v-for="gap in questions.data.value.gaps" :key="gap.skill" class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3" data-test="gap">
            <div class="flex items-center justify-between gap-2">
              <h3 class="m-0 text-[17px] font-extrabold tracking-tight">{{ gap.skill }}</h3>
              <StatusTag v-if="!prep.state.gaps[gap.skill]" label="Missing" tone="missing" />
              <StatusTag v-else-if="prep.state.gaps[gap.skill]!.status === 'added'" label="Added" tone="added" />
              <StatusTag v-else label="Skipped" tone="muted" />
            </div>
            <template v-if="!prep.state.gaps[gap.skill]">
              <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ gap.ask }}</p>
              <UTextarea
                :model-value="prep.draft(`g:${gap.skill}`)"
                :aria-label="`Your experience with ${gap.skill}`"
                :placeholder="gap.example"
                :rows="3"
                class="w-full"
                @update:model-value="(value: string) => prep.setDraft(`g:${gap.skill}`, String(value))"
              />
              <RolePicker :name="gap.skill" :choices="prep.roleChoices(`g:${gap.skill}`)" @pick="id => prep.pickRole(`g:${gap.skill}`, id)" />
              <div class="flex items-center gap-2.5">
                <UButton label="Add to my CV" :disabled="!prep.canAdd(`g:${gap.skill}`)" :loading="prep.state.saving[`g:${gap.skill}`]" @click="prep.addGap(gap.skill)" />
                <UButton label="Skip" color="neutral" variant="outline" @click="prep.skipGap(gap.skill)" />
              </div>
            </template>
            <template v-else-if="prep.state.gaps[gap.skill]!.status === 'added'">
              <p class="m-0 text-sm leading-snug font-medium">{{ addedText(prep.state.gaps[gap.skill]!) }}</p>
              <p class="m-0 text-[13px] font-medium text-muted">Saved in your profile under {{ prep.whereLabel(addedPlace(prep.state.gaps[gap.skill]!)) }}</p>
              <div class="flex flex-wrap items-center gap-3">
                <span v-if="skills.includes(gap.skill.toLowerCase())" class="text-[13px] font-bold text-hoply-300">In your profile skills</span>
                <UButton v-else :label="`Also add ${gap.skill} to my skills`" color="neutral" variant="outline" size="sm" @click="addToProfile(gap.skill)" />
                <UButton label="Change" variant="link" color="neutral" size="sm" class="px-1 text-[13px] text-muted underline" :loading="prep.state.saving[`g:${gap.skill}`]" @click="prep.undoGap(gap.skill)" />
              </div>
            </template>
            <UButton v-else label="Change my mind" variant="link" color="neutral" size="sm" class="self-start px-1 text-[13px] text-muted underline" @click="prep.undoGap(gap.skill)" />
            <p v-if="prep.state.errors[`g:${gap.skill}`]" role="alert" class="m-0 text-sm font-bold text-error">{{ prep.state.errors[`g:${gap.skill}`] }}</p>
          </div>
        </section>

        <section v-if="!asked.length" class="flex flex-col gap-2.5">
          <SectionLabel>What this job will ask you to do</SectionLabel>
          <div v-for="claim in questions.data.value.responsibilities" :key="claim.index" class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3" data-test="claim">
            <h3 class="m-0 text-sm leading-snug font-bold">{{ claim.question }}</h3>
            <template v-if="!prep.state.claims[claim.index]">
              <div v-if="prep.suggestionsFor(claim.text).length" class="flex flex-col gap-1.5" role="group" :aria-label="`From your profile: ${claim.question}`">
                <SectionLabel as="span">From your profile</SectionLabel>
                <button
                  v-for="suggestion in prep.suggestionsFor(claim.text)"
                  :key="suggestion.text"
                  type="button"
                  :aria-pressed="false"
                  class="flex min-h-11 items-start gap-2 rounded-lg border border-default px-3 py-2 text-left text-[13px] font-medium hover:bg-accented"
                  @click="prep.claimPick(claim.index, suggestion.text)"
                >
                  <UIcon name="i-lucide-check" class="mt-0.5 size-3.5 shrink-0 text-hoply-400" />
                  <span><span class="font-bold">{{ suggestion.label }}:</span> {{ suggestion.text }}</span>
                </button>
              </div>
              <UTextarea
                :model-value="prep.draft(`c:${claim.index}`)"
                aria-label="What exactly did you do?"
                placeholder="In your own words: what did you do, where, and what changed? Numbers help."
                :rows="3"
                class="w-full"
                @update:model-value="(value: string) => prep.setDraft(`c:${claim.index}`, String(value))"
              />
              <UButton
                v-if="!prep.draft(`c:${claim.index}`).trim()"
                label="Start from the job’s wording, then make it yours"
                variant="link"
                size="sm"
                class="self-start px-1 text-[13px] text-hoply-400 underline"
                @click="prep.setDraft(`c:${claim.index}`, claim.prefill)"
              />
              <RolePicker :name="claim.question" :choices="prep.roleChoices(`c:${claim.index}`)" @pick="id => prep.pickRole(`c:${claim.index}`, id)" />
              <div class="flex flex-wrap items-center gap-2.5">
                <UButton label="Add to my CV" :disabled="!prep.canAdd(`c:${claim.index}`)" :loading="prep.state.saving[`c:${claim.index}`]" @click="prep.claimYes(claim.index)" />
                <UButton label="Not really" color="neutral" variant="outline" @click="prep.claimNo(claim.index)" />
              </div>
            </template>
            <template v-else>
              <div class="flex items-center justify-between gap-2">
                <StatusTag
                  :label="{ added: 'Added', picked: 'From your profile', no: 'Left out' }[prep.state.claims[claim.index]!.status]"
                  :tone="prep.state.claims[claim.index]!.status === 'no' ? 'muted' : 'added'"
                />
                <UButton label="Change" variant="link" color="neutral" size="sm" class="px-1 text-[13px] text-muted underline" :loading="prep.state.saving[`c:${claim.index}`]" @click="prep.claimUndo(claim.index)" />
              </div>
              <template v-if="prep.state.claims[claim.index]!.status !== 'no'">
                <p class="m-0 text-sm leading-snug font-medium">{{ addedText(prep.state.claims[claim.index]!) }}</p>
                <p v-if="prep.state.claims[claim.index]!.status === 'added'" class="m-0 text-[13px] font-medium text-muted">
                  Saved in your profile under {{ prep.whereLabel(addedPlace(prep.state.claims[claim.index]!)) }}
                </p>
              </template>
            </template>
            <p v-if="prep.state.errors[`c:${claim.index}`]" role="alert" class="m-0 text-sm font-bold text-error">{{ prep.state.errors[`c:${claim.index}`] }}</p>
          </div>
        </section>

        <section v-if="prep.impactRows.value.length" class="flex flex-col gap-2.5">
          <SectionLabel>Make your bullets hit harder</SectionLabel>
          <p class="m-0 text-[13px] leading-snug font-medium text-muted">Screening favors results you can measure. What changed because of each of these? A number, a time saved, a cost, a count of people. We save it with the highlight in your profile.</p>
          <div v-for="row in prep.impactRows.value" :key="row.key" class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated p-3">
            <p class="m-0 text-sm leading-snug font-bold">{{ row.bullet }}</p>
            <UInput
              v-model="prep.impacts[row.key]"
              aria-label="What changed because of it"
              placeholder="e.g. cut release time from 2 hours to 15 minutes"
              maxlength="200"
              class="w-full"
              @change="prep.saveImpact(row.key, prep.impacts[row.key] ?? '')"
            />
            <p v-if="prep.state.errors[`i:${row.key}`]" role="alert" class="m-0 text-sm font-bold text-error">{{ prep.state.errors[`i:${row.key}`] }}</p>
          </div>
        </section>

        <section class="flex flex-col gap-2">
          <SectionLabel>Roles to feature</SectionLabel>
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
              :class="role.included ? 'border-hoply-600 bg-hoply-600 text-white' : 'border-ink-700 text-toned'"
            >{{ role.included ? 'Included' : 'Left out' }}</span>
          </button>
        </section>

        <section v-if="prep.focusChips.value.length" class="flex flex-col gap-2">
          <SectionLabel>Put these up front (up to 3)</SectionLabel>
          <div class="flex flex-wrap gap-2" role="group" aria-label="Put these up front">
            <ToggleChip v-for="chip in prep.focusChips.value" :key="chip.label" :label="chip.label" :pressed="chip.pressed" @toggle="prep.toggleFocus(chip.label)" />
          </div>
        </section>

        <UFormField
          label="Anything else that makes you a better fit?"
          description="Certifications, side projects, awards, relocation. One per line. We keep them in your profile’s additional information."
          :error="prep.state.errors.note"
          :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase', description: 'text-[13px] leading-snug font-medium' }"
        >
          <UTextarea
            v-model="prep.note.value"
            :rows="3"
            :placeholder="'e.g. HashiCorp Terraform Associate, 2025\nBuilt a home lab running 12 containers'"
            class="w-full"
            @change="prep.saveNote()"
          />
        </UFormField>
      </div>
    </div>
  </div>
</template>
