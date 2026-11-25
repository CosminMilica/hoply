<script setup lang="ts">
import { computed, ref } from 'vue'
import ScoreTile from '@/features/common/components/ScoreTile.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import type { Profile } from '@/features/profile/api/profile.schema'
import { knownSkills } from '@/features/profile/skills'
import { JOB_LEVELS, JOB_WORK_MODES, type JobDraft } from '../api/jobs.schema'
import { knockoutFor, matchJob, yearsLine } from '../match'
import { readFromLine } from '../reading'

const MAX_REQUIREMENTS = 40

const props = defineProps<{ profile: Profile | undefined, error?: string, saving?: boolean }>()
const draft = defineModel<JobDraft>({ required: true })
const emit = defineEmits<{ save: [tailor: boolean] }>()
const requirement = ref('')
const label = { label: 'text-xs font-bold text-muted' }

const skills = computed(() => (props.profile ? knownSkills(props.profile) : []))
const have = computed(() => new Set(skills.value.map(skill => skill.toLowerCase())))
const match = computed(() => matchJob(draft.value, skills.value))
const knockout = computed(() => (props.profile ? knockoutFor(draft.value, props.profile) : ''))
const years = computed(() => (props.profile ? yearsLine(draft.value, props.profile) : ''))

function remove(list: 'must_have' | 'nice_to_have', item: string) {
  draft.value = { ...draft.value, [list]: draft.value[list].filter(other => other !== item) }
}
function add(list: 'must_have' | 'nice_to_have') {
  const value = requirement.value.trim()
  requirement.value = ''
  const all = [...draft.value.must_have, ...draft.value.nice_to_have].map(item => item.toLowerCase())
  if (!value || all.includes(value.toLowerCase()) || draft.value[list].length >= MAX_REQUIREMENTS) return
  draft.value = { ...draft.value, [list]: [...draft.value[list], value] }
}
const set = (patch: Partial<JobDraft>) => {
  draft.value = { ...draft.value, ...patch }
}
</script>

<template>
  <div class="flex flex-col gap-3.5">
    <div class="flex flex-col gap-0.5">
      <h2 class="m-0 text-xl font-extrabold tracking-tight">Check what we found</h2>
      <span class="text-[13px] font-semibold text-muted">{{ readFromLine(draft) }}. Fix anything we got wrong.</span>
    </div>

    <div class="flex flex-col gap-2.5 rounded-[14px] bg-accented p-3.5" data-test="draft-match">
      <div class="flex items-center gap-3">
        <ScoreTile :score="match.score" label />
        <span class="text-sm leading-snug font-bold">{{ match.verdict }}</span>
      </div>
      <p v-if="knockout" role="note" class="m-0 flex items-start gap-2 rounded-[10px] bg-ember-950 px-3 py-2.5 text-[13px] leading-snug font-semibold text-ember-100" data-test="knockout">
        <UIcon name="i-lucide-triangle-alert" class="mt-px size-4 shrink-0 text-ember-400" />
        <span>{{ knockout }}</span>
      </p>
      <span v-if="years" class="text-[13px] font-semibold text-toned" data-test="years">{{ years }}</span>
    </div>

    <UFormField label="Job title" :ui="label">
      <UInput :model-value="draft.title" maxlength="160" class="w-full" @update:model-value="(value: string) => set({ title: value })" />
    </UFormField>
    <div class="grid grid-cols-2 gap-2">
      <UFormField label="Company" :ui="label">
        <UInput :model-value="draft.company" placeholder="Not found" maxlength="120" class="w-full" @update:model-value="(value: string) => set({ company: value })" />
      </UFormField>
      <UFormField label="City" :ui="label">
        <UInput :model-value="draft.city" placeholder="Not found" maxlength="120" class="w-full" @update:model-value="(value: string) => set({ city: value })" />
      </UFormField>
    </div>
    <div class="flex flex-wrap gap-3.5">
      <div class="flex flex-col gap-1.5" role="group" aria-label="Work mode">
        <span class="text-xs font-bold text-muted">Work mode</span>
        <div class="flex flex-wrap gap-1.5">
          <ToggleChip
            v-for="(title, mode) in JOB_WORK_MODES"
            :key="mode"
            :label="title"
            :pressed="draft.work_mode === mode"
            @toggle="set({ work_mode: draft.work_mode === mode ? '' : mode })"
          />
        </div>
      </div>
      <div class="flex flex-col gap-1.5" role="group" aria-label="Level">
        <span class="text-xs font-bold text-muted">Level</span>
        <div class="flex flex-wrap gap-1.5">
          <ToggleChip v-for="(title, level) in JOB_LEVELS" :key="level" :label="title" :pressed="draft.level === level" @toggle="set({ level })" />
        </div>
      </div>
    </div>

    <section class="flex flex-col gap-2" aria-labelledby="requirements">
      <div class="flex items-baseline justify-between gap-2">
        <SectionLabel id="requirements" as="h3">Requirements we found</SectionLabel>
        <span class="flex items-center gap-1.5 text-xs font-semibold text-muted"><span class="size-2 rounded-xs bg-hoply-500" aria-hidden="true" />in your profile</span>
      </div>
      <p v-if="!draft.must_have.length && !draft.nice_to_have.length" class="m-0 text-[13px] leading-snug font-medium text-ink-300">
        We did not recognise any skills in this ad. Add the ones it asks for, so the match and your CV use them.
      </p>
      <template v-for="list in (['must_have', 'nice_to_have'] as const)" :key="list">
        <template v-if="draft[list].length">
          <span class="text-xs font-bold text-toned">{{ list === 'must_have' ? 'Must have' : 'Nice to have' }}</span>
          <ul class="flex flex-wrap gap-1.5">
            <li v-for="item in draft[list]" :key="item">
              <button
                type="button"
                :aria-label="`Remove ${item}`"
                class="flex min-h-8 items-center gap-1.5 rounded-lg border-[1.5px] px-2.5 text-[13px] font-bold"
                :class="[
                  list === 'nice_to_have' ? 'border-dashed' : '',
                  have.has(item.toLowerCase()) ? 'border-hoply-500 bg-hoply-900 text-hoply-300' : 'border-ink-400 text-toned',
                ]"
                @click="remove(list, item)"
              >
                {{ item }}<span aria-hidden="true" class="opacity-60">×</span>
              </button>
            </li>
          </ul>
        </template>
      </template>
      <div class="flex flex-wrap gap-1.5">
        <UFormField label="Add a requirement" :ui="{ label: 'sr-only' }" class="min-w-40 flex-1">
          <UInput v-model="requirement" placeholder="Add a requirement, e.g. Kubernetes" maxlength="60" size="sm" class="w-full" @keydown.enter.prevent="add('must_have')" />
        </UFormField>
        <UButton label="+ Must" color="neutral" variant="outline" size="sm" @click="add('must_have')" />
        <UButton label="+ Nice" color="neutral" variant="outline" size="sm" class="border-dashed" @click="add('nice_to_have')" />
      </div>
    </section>

    <section v-if="draft.responsibilities.length" class="flex flex-col gap-1.5">
      <SectionLabel as="h3">Responsibilities we found · {{ draft.responsibilities.length }}</SectionLabel>
      <ul class="flex flex-col gap-1.5">
        <li v-for="(item, index) in draft.responsibilities.slice(0, 6)" :key="index" class="flex gap-2 text-sm leading-snug font-medium">
          <span aria-hidden="true" class="font-extrabold">·</span><span>{{ item }}</span>
        </li>
      </ul>
      <span class="text-[13px] font-medium text-muted">When you prepare your CV, we ask how you did these.</span>
    </section>

    <p v-if="error" role="alert" class="m-0 text-[13px] font-bold text-ember-400">{{ error }}</p>
    <div class="flex flex-col gap-2">
      <UButton label="Save and prepare my CV" size="lg" block class="justify-center" :loading="saving" @click="emit('save', true)" />
      <UButton label="Save to my jobs" color="neutral" variant="outline" block class="justify-center" :disabled="saving" @click="emit('save', false)" />
    </div>
    <p class="m-0 text-xs leading-snug font-medium text-ink-500">We read ads with rules and a skills dictionary, not with AI. AI is used only when we write your CV.</p>
  </div>
</template>
