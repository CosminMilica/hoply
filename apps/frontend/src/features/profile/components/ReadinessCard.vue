<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import ProgressBar from '@/features/common/components/ProgressBar.vue'
import type { Profile } from '../api/profile.schema'
import { initialsOf, profileReadiness } from '../readiness'

const props = defineProps<{ profile: Profile }>()
const readiness = computed(() => profileReadiness(props.profile))
const initials = computed(() => initialsOf(props.profile.name))
</script>

<template>
  <div class="flex flex-col gap-4 rounded-2xl border border-default bg-elevated p-[18px]">
    <div class="flex items-center gap-3.5">
      <span class="flex size-14 shrink-0 items-center justify-center rounded-[14px] bg-hoply-600 text-xl font-extrabold text-white" data-test="avatar">
        <span v-if="initials" aria-hidden="true">{{ initials }}</span>
        <UIcon v-else name="i-lucide-user" class="size-7" />
      </span>
      <div class="flex min-w-0 flex-col gap-px">
        <h1 class="m-0 text-[26px] leading-tight font-extrabold tracking-tight">{{ profile.name.trim() || t.profile.readiness.yourProfile }}</h1>
        <span class="text-sm font-medium text-ink-300">{{ profile.headline || t.profile.readiness.addHeadline }}</span>
      </div>
    </div>
    <div class="flex flex-col gap-2">
      <div class="flex items-baseline justify-between">
        <span class="text-xs font-extrabold tracking-wider text-ink-300 uppercase">{{ t.profile.readiness.readyToTailor }}</span>
        <span class="text-[22px] font-extrabold text-hoply-400" data-test="readiness-percent">{{ readiness.percent }}%</span>
      </div>
      <ProgressBar :value="readiness.percent" :label="t.profile.readiness.progress" />
      <span class="text-[13px] leading-snug font-medium text-ink-300">{{ readiness.hint }}</span>
    </div>
  </div>
</template>
