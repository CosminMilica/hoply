<script setup lang="ts">
import { computed } from 'vue'
import { useApplicationMutations, useApplicationStatuses } from '@/features/applications/composables/useApplications'
import BackButton from '@/features/common/components/BackButton.vue'
import KeywordChips from '@/features/common/components/KeywordChips.vue'
import ScoreTile from '@/features/common/components/ScoreTile.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { knownSkills } from '@/features/profile/skills'
import FollowCompanyCard from '../components/FollowCompanyCard.vue'
import { analyzeImported, useJobQuery } from '../composables/useJobs'
import { lookFor, matchJob, placeOf, sourceLine, tagline } from '../match'

const props = defineProps<{ id: string }>()
const { data: job, isPending, isError } = useJobQuery(() => props.id)
const profile = useProfileQuery()
const { statuses } = useApplicationStatuses()
const { setStatus, remove } = useApplicationMutations()

const match = computed(() => (job.value ? matchJob(job.value, profile.data.value ? knownSkills(profile.data.value) : []) : null))
const status = computed(() => statuses.value.get(props.id))
const lists = computed(() => (job.value
  ? [
      { title: 'What you will do', items: job.value.responsibilities },
      { title: 'What we look for', items: lookFor(job.value) },
      { title: 'Why people like it here', items: job.value.perks },
    ].filter(list => list.items.length)
  : []))
const prepareLabel = computed(() => (status.value === 'cv_ready' || status.value === 'applied' ? 'Prepare a new CV' : 'Prepare my CV'))

function toggleSave() {
  if (status.value === 'saved') remove.mutate(props.id)
  else if (!status.value) {
    setStatus.mutate({ jobId: props.id, status: 'saved' })
    if (job.value) analyzeImported(job.value)
  }
}
</script>

<template>
  <div class="mx-auto w-full max-w-[1080px]">
    <USkeleton v-if="isPending" class="h-64 w-full" />
    <UAlert v-else-if="isError || !job" color="error" variant="subtle" title="We could not find this job" />
    <div v-else class="flex flex-col gap-4 min-[900px]:grid min-[900px]:grid-cols-[minmax(0,1fr)_340px] min-[900px]:items-start min-[900px]:gap-x-12">
      <BackButton :fallback="{ name: 'hoply-saved' }" class="min-[900px]:col-span-2" />
      <div class="-mt-2.5 flex flex-col gap-1.5 min-[900px]:col-start-1">
        <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">{{ job.title }}</h1>
        <p class="m-0 text-base font-medium text-muted">{{ [job.company, placeOf(job)].filter(Boolean).join(' · ') }}</p>
        <p class="m-0 text-[13px] font-bold">{{ [sourceLine(job), tagline(job)].filter(Boolean).join(' · ') }}</p>
        <UAlert v-if="job.closed_at" color="neutral" variant="subtle" icon="i-lucide-archive" title="The company no longer lists this job" data-test="closed" />
        <a v-if="job.url" :href="job.url" target="_blank" rel="noopener noreferrer" class="self-start text-[13px] font-bold text-hoply-400 underline-offset-2 hover:underline">View the original ad ↗</a>
        <FollowCompanyCard v-if="job.added_by_me && job.url" :job="job" class="mt-1.5" />
      </div>

      <aside class="contents min-[900px]:sticky min-[900px]:top-0 min-[900px]:col-start-2 min-[900px]:row-span-2 min-[900px]:row-start-2 min-[900px]:flex min-[900px]:flex-col min-[900px]:gap-4" aria-label="Your match">
        <div v-if="match" class="flex flex-col gap-3 rounded-[14px] bg-accented p-4">
          <div class="flex items-center gap-3.5">
            <ScoreTile :score="match.score" />
            <span class="text-[15px] leading-snug font-bold">{{ match.verdict }}</span>
          </div>
          <KeywordChips :matched="match.matched" :related="match.related" :missing="match.missing" size="md" />
          <p v-if="job.requirements_from === 'analysis'" data-test="requirements-from" class="m-0 text-xs text-muted">Requirements as Claude read them from the ad.</p>
        </div>
        <div class="order-last flex flex-col gap-2.5 min-[900px]:order-none">
          <UButton :to="{ name: 'hoply-prep', params: { id: job.id } }" :label="prepareLabel" size="lg" block class="justify-center" />
          <UButton
            v-if="!status || status === 'saved'"
            :label="status === 'saved' ? 'Saved · tap to remove' : 'Save for later'"
            color="neutral"
            variant="outline"
            size="lg"
            block
            class="justify-center"
            @click="toggleSave"
          />
        </div>
      </aside>

      <div class="flex flex-col gap-4 min-[900px]:col-start-1">
        <section v-if="job.about" class="flex flex-col gap-2">
          <SectionLabel>About the role</SectionLabel>
          <p class="m-0 text-base leading-relaxed font-medium">{{ job.about }}</p>
        </section>
        <section v-for="list in lists" :key="list.title" class="flex flex-col gap-2">
          <SectionLabel>{{ list.title }}</SectionLabel>
          <ul class="flex flex-col gap-2">
            <li v-for="item in list.items" :key="item" class="flex gap-2.5 text-base leading-normal font-medium">
              <span aria-hidden="true" class="font-extrabold">·</span><span>{{ item }}</span>
            </li>
          </ul>
        </section>
        <section v-if="job.about_company" class="flex flex-col gap-2">
          <SectionLabel>About {{ job.company }}</SectionLabel>
          <p class="m-0 text-[15px] leading-relaxed font-medium text-toned">{{ job.about_company }}</p>
        </section>
      </div>
    </div>
  </div>
</template>
