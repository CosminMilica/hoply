<script setup lang="ts">
import { areaLabel, t } from '@/features/common/i18n'
import { DOMAIN_KEYS, type Domain } from '@/features/profile/api/profile.schema'
import { DOMAIN_AREAS, DOMAIN_ICONS } from '@/features/jobs/api/jobs.schema'

/** The fields of work as cards to pick one or more of, each with what it covers and the filters it brings. */
const model = defineModel<Domain[]>({ required: true })

function toggle(domain: Domain) {
  model.value = model.value.includes(domain) ? model.value.filter(item => item !== domain) : DOMAIN_KEYS.filter(item => item === domain || model.value.includes(item))
}
</script>

<template>
  <div role="group" :aria-label="t.onboarding.domainsGroup" class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3" data-test="domain-picker">
    <button
      v-for="domain in DOMAIN_KEYS"
      :key="domain"
      type="button"
      :aria-pressed="model.includes(domain)"
      class="flex min-h-[176px] flex-col gap-2.5 rounded-[16px] border-2 p-[18px] text-left transition-colors"
      :class="model.includes(domain) ? 'border-crashr-500 bg-crashr-950' : 'border-default bg-elevated hover:border-accented'"
      :data-test="`domain-${domain}`"
      @click="toggle(domain)"
    >
      <span class="flex items-center justify-between">
        <span class="flex size-10 items-center justify-center rounded-[12px]" :class="model.includes(domain) ? 'bg-crashr-600 text-white' : 'bg-accented text-highlighted'">
          <UIcon :name="DOMAIN_ICONS[domain]" class="size-5" />
        </span>
        <UIcon v-if="model.includes(domain)" name="i-lucide-circle-check" class="size-6 text-crashr-400" />
      </span>
      <span class="text-[19px] font-extrabold text-highlighted">{{ t.common.domains[domain] }}</span>
      <span class="text-[14px] leading-snug font-medium text-muted">{{ t.common.domainAbout[domain] }}</span>
      <span class="mt-auto text-[12px] leading-snug font-bold text-crashr-300">
        {{ t.onboarding.filters }}: {{ DOMAIN_AREAS[domain].map(areaLabel).join(' · ') }}
      </span>
    </button>
  </div>
</template>
