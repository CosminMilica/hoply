<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import type { Job } from '../api/jobs.schema'
import { useBoardOfQuery, useFollowBoard } from '../composables/useJobs'
import { platformLabel } from '../match'

/** For a job added from a link on a board Hoply reads: follow the company, and its jobs in Romania come to Find jobs. */
const props = defineProps<{ job: Pick<Job, 'url' | 'company'> }>()
const board = useBoardOfQuery(() => props.job.url)
const follow = useFollowBoard()
const known = computed(() => board.data.value?.company || props.job.company.trim())
const company = computed(() => known.value || t.value.jobs.followCompany.thisCompany)
const sentCompany = computed(() => known.value || 'This company')
</script>

<template>
  <div v-if="board.data.value" class="flex flex-col gap-2 rounded-[14px] border border-default bg-elevated p-3.5" data-test="follow-company">
    <p v-if="follow.isSuccess.value" class="m-0 text-sm font-medium">
      {{ t.jobs.followCompany.following(company) }}
    </p>
    <p v-else-if="board.data.value.imported" class="m-0 text-sm font-medium text-muted">
      {{ t.jobs.followCompany.imported(company) }}
    </p>
    <template v-else>
      <p class="m-0 text-sm font-bold">{{ t.jobs.followCompany.postsOn(company, platformLabel(board.data.value.platform)) }}</p>
      <p class="m-0 text-sm font-medium text-muted">{{ t.jobs.followCompany.pitch }}</p>
      <UButton
        :label="t.jobs.followCompany.follow(company)"
        icon="i-lucide-bell-plus"
        color="neutral"
        variant="outline"
        :loading="follow.isPending.value"
        class="self-start"
        @click="follow.mutate({ url: job.url, company: sentCompany })"
      />
      <p v-if="follow.isError.value" role="alert" class="m-0 text-sm font-medium text-error">{{ t.jobs.followCompany.failed(company) }}</p>
    </template>
  </div>
</template>
