<script setup lang="ts">
import { computed, ref } from 'vue'
import { t } from '@/features/common/i18n'
import ScoreTile from '@/features/common/components/ScoreTile.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import type { Profile } from '@/features/profile/api/profile.schema'
import { knownSkills } from '@/features/profile/skills'
import type { JobDraft } from '../api/jobs.schema'
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
      <h2 class="m-0 text-xl font-extrabold tracking-tight">{{ t.jobs.draft.title }}</h2>
      <span class="text-[13px] font-semibold text-muted">{{ readFromLine(draft) }}. {{ t.jobs.draft.fix }}</span>
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

    <UFormField :label="t.jobs.draft.jobTitle" :ui="label">
      <UInput :model-value="draft.title" maxlength="160" class="w-full" @update:model-value="(value: string) => set({ title: value })" />
    </UFormField>
    <div class="grid grid-cols-2 gap-2">
      <UFormField :label="t.jobs.draft.company" :ui="label">
        <UInput :model-value="draft.company" :placeholder="t.jobs.draft.notFound" maxlength="120" class="w-full" @update:model-value="(value: string) => set({ company: value })" />
      </UFormField>
      <UFormField :label="t.jobs.draft.city" :ui="label">
        <UInput :model-value="draft.city" :placeholder="t.jobs.draft.notFound" maxlength="120" class="w-full" @update:model-value="(value: string) => set({ city: value })" />
      </UFormField>
    </div>
    <div class="flex flex-wrap gap-3.5">
      <div class="flex flex-col gap-1.5" role="group" :aria-label="t.jobs.draft.workMode">
        <span class="text-xs font-bold text-muted">{{ t.jobs.draft.workMode }}</span>
        <div class="flex flex-wrap gap-1.5">
          <ToggleChip
            v-for="(title, mode) in t.jobs.workModes"
            :key="mode"
            :label="title"
            :pressed="draft.work_mode === mode"
            @toggle="set({ work_mode: draft.work_mode === mode ? '' : mode })"
          />
        </div>
      </div>
      <div class="flex flex-col gap-1.5" role="group" :aria-label="t.jobs.draft.level">
        <span class="text-xs font-bold text-muted">{{ t.jobs.draft.level }}</span>
        <div class="flex flex-wrap gap-1.5">
          <ToggleChip v-for="(title, level) in t.jobs.jobLevels" :key="level" :label="title" :pressed="draft.level === level" @toggle="set({ level })" />
        </div>
      </div>
    </div>

    <section class="flex flex-col gap-2" aria-labelledby="requirements">
      <div class="flex items-baseline justify-between gap-2">
        <SectionLabel id="requirements" as="h3">{{ t.jobs.draft.requirements }}</SectionLabel>
        <span class="flex items-center gap-1.5 text-xs font-semibold text-muted"><span class="size-2 rounded-xs bg-crashr-500" aria-hidden="true" />{{ t.jobs.draft.inProfile }}</span>
      </div>
      <p v-if="!draft.must_have.length && !draft.nice_to_have.length" class="m-0 text-[13px] leading-snug font-medium text-ink-300">
        {{ t.jobs.draft.noSkills }}
      </p>
      <template v-for="list in (['must_have', 'nice_to_have'] as const)" :key="list">
        <template v-if="draft[list].length">
          <span class="text-xs font-bold text-toned">{{ list === 'must_have' ? t.jobs.draft.mustHave : t.jobs.draft.niceToHave }}</span>
          <ul class="flex flex-wrap gap-1.5">
            <li v-for="item in draft[list]" :key="item">
              <button
                type="button"
                :aria-label="t.jobs.draft.remove(item)"
                class="flex min-h-8 items-center gap-1.5 rounded-lg border-[1.5px] px-2.5 text-[13px] font-bold"
                :class="[
                  list === 'nice_to_have' ? 'border-dashed' : '',
                  have.has(item.toLowerCase()) ? 'border-crashr-500 bg-crashr-900 text-crashr-300' : 'border-ink-400 text-toned',
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
        <UFormField :label="t.jobs.draft.addRequirement" :ui="{ label: 'sr-only' }" class="min-w-40 flex-1">
          <UInput v-model="requirement" :placeholder="t.jobs.draft.addPlaceholder" maxlength="60" size="sm" class="w-full" @keydown.enter.prevent="add('must_have')" />
        </UFormField>
        <UButton :label="t.jobs.draft.addMust" color="neutral" variant="outline" size="sm" @click="add('must_have')" />
        <UButton :label="t.jobs.draft.addNice" color="neutral" variant="outline" size="sm" class="border-dashed" @click="add('nice_to_have')" />
      </div>
    </section>

    <section v-if="draft.responsibilities.length" class="flex flex-col gap-1.5">
      <SectionLabel as="h3">{{ t.jobs.draft.responsibilities(draft.responsibilities.length) }}</SectionLabel>
      <ul class="flex flex-col gap-1.5">
        <li v-for="(item, index) in draft.responsibilities.slice(0, 6)" :key="index" class="flex gap-2 text-sm leading-snug font-medium">
          <span aria-hidden="true" class="font-extrabold">·</span><span>{{ item }}</span>
        </li>
      </ul>
      <span class="text-[13px] font-medium text-muted">{{ t.jobs.draft.askHow }}</span>
    </section>

    <p v-if="error" role="alert" class="m-0 text-[13px] font-bold text-ember-400">{{ error }}</p>
    <div class="flex flex-col gap-2">
      <UButton :label="t.jobs.draft.saveAndPrepare" size="lg" block class="justify-center" :loading="saving" @click="emit('save', true)" />
      <UButton :label="t.jobs.draft.saveToJobs" color="neutral" variant="outline" block class="justify-center" :disabled="saving" @click="emit('save', false)" />
    </div>
    <p class="m-0 text-xs leading-snug font-medium text-ink-500">{{ t.jobs.draft.howWeRead }}</p>
  </div>
</template>
