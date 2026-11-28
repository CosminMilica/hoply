<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import type { Profile } from '../api/profile.schema'
import { hasWork, profileReadiness, READY_PERCENT } from '../readiness'

/** Where Hoply would ask Claude for a profile that says too little: how complete it is, what to add next, and the way there. */
const props = defineProps<{ profile: Profile }>()
const readiness = computed(() => profileReadiness(props.profile))
/** The next thing to add: some work when there is none, else the readiness's own next step. */
const next = computed(() => (hasWork(props.profile) ? readiness.value.hint : t.value.profile.gate.noWork))
</script>

<template>
  <section class="flex flex-col gap-3 rounded-2xl border border-hoply-700 bg-hoply-950 p-5" data-test="profile-gate" role="status">
    <div class="flex items-start gap-3">
      <span class="flex size-10 shrink-0 items-center justify-center rounded-[12px] bg-hoply-600 text-white">
        <UIcon name="i-lucide-user-round-pen" class="size-5" />
      </span>
      <div class="flex flex-col gap-1">
        <h2 class="m-0 text-[17px] font-extrabold text-highlighted">{{ t.profile.gate.title(readiness.percent) }}</h2>
        <p class="m-0 text-sm leading-snug font-medium text-toned">{{ t.profile.gate.text(READY_PERCENT) }}</p>
        <p class="m-0 text-[13px] leading-snug font-bold text-hoply-300" data-test="profile-gate-next">{{ next }}</p>
      </div>
    </div>
    <UButton :to="{ name: 'hoply-profile' }" :label="t.profile.gate.action" trailing-icon="i-lucide-arrow-right" class="self-start" />
  </section>
</template>
