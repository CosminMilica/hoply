<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import ScoreTile from '@/features/common/components/ScoreTile.vue'
import { jobsApi } from '@/features/jobs/api/jobsApi'
import { jobKeys } from '@/features/jobs/composables/useJobs'
import { matchJob, placeOf } from '@/features/jobs/match'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { knownSkills } from '@/features/profile/skills'
import { STATUS_LABELS, type ApplicationStatus } from '../api/applications.schema'
import { useApplicationMutations, useApplicationsQuery } from '../composables/useApplications'

type Tab = 'all' | ApplicationStatus

const TABS: { id: Tab, label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'saved', label: STATUS_LABELS.saved },
  { id: 'cv_ready', label: STATUS_LABELS.cv_ready },
  { id: 'applied', label: STATUS_LABELS.applied },
]
const BADGE: Record<ApplicationStatus, string> = {
  saved: 'bg-accented text-toned',
  cv_ready: 'bg-hoply-600 text-white',
  applied: 'bg-hoply-900 text-hoply-300',
}

const router = useRouter()
const tab = ref<Tab>('all')
const applications = useApplicationsQuery()
const jobs = useQuery({ queryKey: jobKeys.list({}), queryFn: () => jobsApi.list() })
const profile = useProfileQuery()
const { setStatus, remove } = useApplicationMutations()

const tracked = computed(() => {
  const byId = new Map((jobs.data.value?.items ?? []).map(job => [job.id, job]))
  const skills = profile.data.value ? knownSkills(profile.data.value) : []
  return (applications.data.value ?? []).flatMap((application) => {
    const job = byId.get(application.job_id)
    return job ? [{ job, status: application.status, ...matchJob(job, skills) }] : []
  })
})
const tabs = computed(() => TABS.map(item => ({
  ...item,
  count: item.id === 'all' ? tracked.value.length : tracked.value.filter(entry => entry.status === item.id).length,
})))
const visible = computed(() => tracked.value.filter(entry => tab.value === 'all' || entry.status === tab.value))
const loading = computed(() => applications.isPending.value || jobs.isPending.value)

function primary(jobId: string, status: ApplicationStatus) {
  router.push({ name: status === 'saved' ? 'hoply-prep' : 'hoply-cv', params: { id: jobId } })
}
</script>

<template>
  <div class="mx-auto flex w-full max-w-[1080px] flex-col gap-3.5">
    <div class="flex flex-col gap-1">
      <div class="flex items-center justify-between gap-3">
        <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">Saved jobs</h1>
        <UButton :to="{ name: 'hoply-job-add', query: { from: 'saved' } }" icon="i-lucide-plus" label="Add a job" color="neutral" variant="outline" class="shrink-0" />
      </div>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">Pick the ones worth a CV, and we will tailor it for you.</p>
    </div>
    <div class="flex flex-wrap gap-2" role="group" aria-label="Show">
      <ToggleChip v-for="item in tabs" :key="item.id" :label="`${item.label} ${item.count}`" :pressed="tab === item.id" @toggle="tab = item.id" />
    </div>

    <USkeleton v-if="loading" class="h-40 w-full rounded-[14px]" />
    <ul v-else-if="visible.length" class="grid grid-cols-1 gap-3.5 sm:grid-cols-2 min-[900px]:grid-cols-[repeat(auto-fill,minmax(320px,1fr))]">
      <li v-for="entry in visible" :key="entry.job.id" class="flex flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-3.5">
        <div class="flex items-center gap-3">
          <span class="flex min-w-0 flex-1 flex-col gap-0.5">
            <h2 class="m-0 text-[17px] leading-tight font-extrabold">{{ entry.job.title }}</h2>
            <span class="text-sm font-medium text-muted">{{ [entry.job.company, placeOf(entry.job)].filter(Boolean).join(' · ') }}</span>
          </span>
          <ScoreTile :score="entry.score" size="sm" />
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <span class="rounded-lg px-2.5 py-1 text-xs font-extrabold" :class="BADGE[entry.status]">{{ STATUS_LABELS[entry.status] }}</span>
          <span class="text-[13px] font-medium text-muted">{{ entry.missing.length ? `Missing: ${entry.missing.join(', ')}` : 'Every keyword is covered' }}</span>
        </div>
        <div class="flex items-center gap-2">
          <UButton :label="entry.status === 'saved' ? 'Prepare CV' : 'View CV'" block class="flex-1 justify-center" @click="primary(entry.job.id, entry.status)" />
          <UButton :to="{ name: 'hoply-job', params: { id: entry.job.id } }" label="Details" color="neutral" variant="outline" />
        </div>
        <UButton
          v-if="entry.status === 'cv_ready'"
          label="I applied with this CV"
          color="neutral"
          variant="soft"
          block
          class="justify-center"
          @click="setStatus.mutate({ jobId: entry.job.id, status: 'applied' })"
        />
        <UButton
          v-if="entry.status === 'saved'"
          label="Remove from saved"
          variant="link"
          color="neutral"
          class="self-start px-1 text-[13px] text-muted underline"
          @click="remove.mutate(entry.job.id)"
        />
      </li>
    </ul>
    <div v-else class="flex flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-[22px]">
      <p class="m-0 text-[15px] leading-normal font-medium text-muted">
        {{ tab === 'all' ? 'Nothing saved yet. Add a job you found, from its link or its text, and it lands here.' : 'No jobs with this status yet.' }}
      </p>
      <UButton :to="{ name: 'hoply-job-add', query: { from: 'saved' } }" icon="i-lucide-plus" label="Add a job" size="lg" block class="justify-center" />
    </div>
  </div>
</template>
