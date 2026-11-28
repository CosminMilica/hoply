<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { t } from '@/features/common/i18n'
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import {
  EMPLOYMENT_TYPES,
  profilePatchSchema,
  SALARY_PERIODS,
  SENIORITIES,
  WORK_AUTHORIZATIONS,
  WORK_MODES,
  type Profile,
  type ProfilePatch,
} from '../api/profile.schema'
import { useProfileMutations } from '../composables/useProfile'
import TagListEditor from './TagListEditor.vue'

const props = defineProps<{ profile: Profile }>()
const { patch } = useProfileMutations()
const label = { label: 'text-xs font-bold text-muted' }
const options = (values: readonly string[], labels: Record<string, string>) => values.map(value => ({ label: labels[value]!, value }))
// Nuxt UI's select takes no empty value, so "Not set" is a sentinel that saves "".
const NONE = 'none'
const clearable = (values: readonly string[], labels: Record<string, string>) => [{ label: t.value.profile.preferences.notSet, value: NONE }, ...options(values, labels)]
const orEmpty = (value: unknown) => (value === NONE ? '' : value)

const text = (profile: Profile) => ({
  region: profile.work_authorization_region,
  notice: profile.notice_period_weeks?.toString() ?? '',
  available: profile.available_from,
  salary: profile.salary_min?.toString() ?? '',
  currency: profile.salary_currency,
})
const values = reactive(text(props.profile))
const errors = ref<Partial<Record<keyof ProfilePatch, string>>>({})
for (const key of Object.keys(values) as (keyof typeof values)[]) {
  watch(() => text(props.profile)[key], (value) => { values[key] = value })
}

/** Checks one change against the contract's rules, then sends it. */
function save(change: ProfilePatch) {
  const parsed = profilePatchSchema.safeParse(change)
  const field = Object.keys(change)[0] as keyof ProfilePatch
  errors.value = { ...errors.value, [field]: parsed.success ? undefined : parsed.error.issues[0]?.message }
  if (parsed.success) patch.mutate(parsed.data)
}
const number = (value: string | number) => (String(value ?? '').trim() === '' ? null : Number(value))
function toggle<T extends string>(list: string[], value: T): T[] {
  return (list.includes(value) ? list.filter(item => item !== value) : [...list, value]) as T[]
}
</script>

<template>
  <div class="flex flex-col gap-1.5" role="group" aria-labelledby="work-modes">
    <SectionLabel id="work-modes" as="span">{{ t.profile.preferences.workModes }}</SectionLabel>
    <div class="flex flex-wrap gap-1.5">
      <ToggleChip
        v-for="mode in WORK_MODES"
        :key="mode"
        :label="t.profile.workModes[mode]"
        :pressed="profile.work_modes.includes(mode)"
        @toggle="save({ work_modes: toggle(profile.work_modes, mode) })"
      />
    </div>
  </div>
  <USwitch :model-value="profile.open_to_relocation" :label="t.profile.preferences.relocation" size="sm" @update:model-value="(value: boolean) => save({ open_to_relocation: value })" />

  <div class="grid grid-cols-2 gap-2">
    <UFormField :label="t.profile.preferences.workAuthorization" :ui="label">
      <USelect
        :model-value="profile.work_authorization || undefined"
        :items="clearable(WORK_AUTHORIZATIONS, t.profile.workAuthorizations)"
        :placeholder="t.profile.preferences.choose"
        class="w-full"
        @update:model-value="value => save({ work_authorization: orEmpty(value) as ProfilePatch['work_authorization'] })"
      />
    </UFormField>
    <UFormField :label="t.profile.preferences.region" :error="errors.work_authorization_region" :ui="label">
      <UInput v-model="values.region" :placeholder="t.profile.preferences.regionPlaceholder" maxlength="60" class="w-full" @change="save({ work_authorization_region: values.region })" />
    </UFormField>
    <UFormField :label="t.profile.preferences.notice" :error="errors.notice_period_weeks" :ui="label">
      <UInput v-model="values.notice" type="number" min="0" max="52" inputmode="numeric" class="w-full" @change="save({ notice_period_weeks: number(values.notice) })" />
    </UFormField>
    <UFormField :label="t.profile.preferences.availableFrom" :error="errors.available_from" :ui="label">
      <UInput v-model="values.available" type="month" :placeholder="t.profile.monthRange.placeholder" class="w-full" @change="save({ available_from: values.available })" />
    </UFormField>
  </div>

  <div class="flex flex-col gap-1.5">
    <SectionLabel as="span">{{ t.profile.preferences.targetRoles }}</SectionLabel>
    <TagListEditor
      :items="profile.target_roles"
      :label="t.profile.preferences.addTargetRole"
      :placeholder="t.profile.preferences.targetRolePlaceholder"
      :maxlength="120"
      :max="10"
      @change="roles => save({ target_roles: roles })"
    />
  </div>
  <div class="flex flex-col gap-1.5" role="group" aria-labelledby="seniority">
    <SectionLabel id="seniority" as="span">{{ t.profile.preferences.seniority }}</SectionLabel>
    <div class="flex flex-wrap gap-1.5">
      <ToggleChip
        v-for="level in SENIORITIES"
        :key="level"
        :label="t.profile.seniorities[level]"
        :pressed="profile.target_seniority.includes(level)"
        @toggle="save({ target_seniority: toggle(profile.target_seniority, level) })"
      />
    </div>
  </div>
  <div class="flex flex-col gap-1.5" role="group" aria-labelledby="employment">
    <SectionLabel id="employment" as="span">{{ t.profile.preferences.employmentTypes }}</SectionLabel>
    <div class="flex flex-wrap gap-1.5">
      <ToggleChip
        v-for="type in EMPLOYMENT_TYPES"
        :key="type"
        :label="t.profile.employmentTypes[type]"
        :pressed="profile.employment_types.includes(type)"
        @toggle="save({ employment_types: toggle(profile.employment_types, type) })"
      />
    </div>
  </div>

  <div class="grid grid-cols-[minmax(0,1fr)_5.5rem_minmax(0,1fr)] gap-2">
    <UFormField :label="t.profile.preferences.salary" :error="errors.salary_min" :ui="label">
      <UInput v-model="values.salary" type="number" min="0" inputmode="numeric" class="w-full" @change="save({ salary_min: number(values.salary) })" />
    </UFormField>
    <UFormField :label="t.profile.preferences.currency" :error="errors.salary_currency" :ui="label">
      <UInput v-model="values.currency" placeholder="EUR" maxlength="3" class="w-full uppercase" @change="save({ salary_currency: values.currency })" />
    </UFormField>
    <UFormField :label="t.profile.preferences.period" :ui="label">
      <USelect
        :model-value="profile.salary_period || undefined"
        :items="clearable(SALARY_PERIODS, t.profile.salaryPeriods)"
        :placeholder="t.profile.preferences.choose"
        class="w-full"
        @update:model-value="value => save({ salary_period: orEmpty(value) as ProfilePatch['salary_period'] })"
      />
    </UFormField>
  </div>
</template>
