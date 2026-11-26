<script setup lang="ts">
import { computed } from 'vue'
import type { Job } from '../api/jobs.schema'
import { useBoardOfQuery, useFollowBoard } from '../composables/useJobs'
import { platformLabel } from '../match'

/** For a job added from a link on a board Hoply reads: follow the company, and its jobs in Romania come to Find jobs. */
const props = defineProps<{ job: Pick<Job, 'url' | 'company'> }>()
const board = useBoardOfQuery(() => props.job.url)
const follow = useFollowBoard()
const company = computed(() => board.data.value?.company || props.job.company.trim() || 'This company')
</script>

<template>
  <div v-if="board.data.value" class="flex flex-col gap-2 rounded-[14px] border border-default bg-elevated p-3.5" data-test="follow-company">
    <p v-if="follow.isSuccess.value" class="m-0 text-sm font-medium">
      Following {{ company }}. Its jobs in Romania show up in Find jobs in a few minutes.
    </p>
    <p v-else-if="board.data.value.imported" class="m-0 text-sm font-medium text-muted">
      Hoply brings {{ company }}’s jobs in Romania to Find jobs.
    </p>
    <template v-else>
      <p class="m-0 text-sm font-bold">{{ company }} posts its jobs on {{ platformLabel(board.data.value.platform) }}.</p>
      <p class="m-0 text-sm font-medium text-muted">Follow it, and its jobs in Romania come to Find jobs, checked every 6 hours.</p>
      <UButton
        :label="`Follow ${company}`"
        icon="i-lucide-bell-plus"
        color="neutral"
        variant="outline"
        :loading="follow.isPending.value"
        class="self-start"
        @click="follow.mutate({ url: job.url, company })"
      />
      <p v-if="follow.isError.value" role="alert" class="m-0 text-sm font-medium text-error">We could not follow {{ company }}. Try again.</p>
    </template>
  </div>
</template>
