<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { t } from '@/features/common/i18n'
import { placeOf } from '@/features/jobs/match'
import { statusLabel } from '../api/applications.schema'
import { useApplicationMutations } from '../composables/useApplications'
import { useTrackedJobs } from '../composables/useTrackedJobs'

type Stage = 'cv_ready' | 'applied'

const STAGES: Stage[] = ['cv_ready', 'applied']

const router = useRouter()
// A phone shows one stage at a time; a wide screen shows both side by side.
const shown = ref<Stage>('cv_ready')
const { tracked, loading, error } = useTrackedJobs()
const { setStatus } = useApplicationMutations()

const stages = computed(() => STAGES.map(id => ({
  id,
  label: statusLabel(id),
  empty: t.value.applications.applicationsPage.stageEmpty[id],
  entries: tracked.value.filter(entry => entry.status === id),
})))
const empty = computed(() => stages.value.every(stage => !stage.entries.length))
</script>

<template>
  <div class="mx-auto flex w-full max-w-[1080px] flex-col gap-4">
    <div class="flex flex-col gap-1">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">{{ t.applications.applicationsPage.title }}</h1>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">{{ t.applications.applicationsPage.subtitle }}</p>
    </div>

    <USkeleton v-if="loading" class="h-40 w-full rounded-[14px]" />
    <UAlert v-else-if="error" color="error" variant="subtle" :title="t.applications.applicationsPage.loadError" :description="t.applications.tryLater" />
    <div v-else-if="empty" class="flex flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-[22px]">
      <p class="m-0 text-[15px] leading-normal font-medium text-muted">{{ t.applications.applicationsPage.empty }}</p>
      <UButton :to="{ name: 'hoply-saved' }" :label="t.applications.applicationsPage.goToSaved" size="lg" block class="justify-center" />
    </div>
    <template v-else>
      <div role="group" :aria-label="t.applications.applicationsPage.stage" class="flex gap-1 rounded-[12px] bg-elevated p-0.5 min-[900px]:hidden">
        <UButton
          v-for="stage in stages"
          :key="stage.id"
          :label="`${stage.label} ${stage.entries.length}`"
          :aria-pressed="shown === stage.id"
          color="neutral"
          :variant="shown === stage.id ? 'solid' : 'ghost'"
          class="flex-1 justify-center"
          @click="shown = stage.id"
        />
      </div>
      <div class="grid grid-cols-1 items-start gap-5 min-[900px]:grid-cols-2">
        <section
          v-for="stage in stages"
          :key="stage.id"
          :aria-label="stage.label"
          class="flex-col gap-3 rounded-[16px] border border-default bg-muted p-4 min-[900px]:flex"
          :class="shown === stage.id ? 'flex' : 'hidden'"
          :data-test="`stage-${stage.id}`"
        >
          <div class="flex items-center gap-2.5">
            <span class="size-2.5 rounded-full" :class="stage.id === 'cv_ready' ? 'bg-hoply-500' : 'bg-ink-400'" aria-hidden="true" />
            <h2 class="m-0 text-base font-extrabold">{{ stage.label }}</h2>
            <span class="text-sm font-bold text-dimmed">{{ stage.entries.length }}</span>
          </div>
          <p v-if="!stage.entries.length" class="m-0 py-2 text-sm font-medium text-muted">{{ stage.empty }}</p>
          <article
            v-for="entry in stage.entries"
            :key="entry.job.id"
            class="flex flex-col gap-3 rounded-[12px] border border-default bg-elevated p-3.5"
          >
            <div class="flex flex-col gap-0.5">
              <h3 class="m-0 text-base leading-tight font-extrabold">{{ entry.job.title }}</h3>
              <span class="text-sm font-medium text-muted">{{ [entry.job.company, placeOf(entry.job)].filter(Boolean).join(' · ') }}</span>
              <span v-if="entry.since" class="text-xs font-bold" :class="stage.id === 'cv_ready' ? 'text-hoply-300' : 'text-dimmed'" data-test="since">{{ entry.since }}</span>
            </div>
            <div class="flex flex-wrap gap-2">
              <UButton
                v-if="stage.id === 'cv_ready'"
                icon="i-lucide-check"
                :label="t.applications.applicationsPage.applied"
                class="flex-1 justify-center"
                @click="setStatus.mutate({ jobId: entry.job.id, status: 'applied' })"
              />
              <UButton
                :label="t.applications.applicationsPage.viewCv"
                color="neutral"
                variant="outline"
                :class="stage.id === 'cv_ready' ? 'flex-1 justify-center' : ''"
                @click="router.push({ name: 'hoply-cv', params: { id: entry.job.id } })"
              />
              <UButton :to="{ name: 'hoply-job', params: { id: entry.job.id } }" :label="t.applications.details" color="neutral" variant="outline" />
            </div>
          </article>
        </section>
      </div>
    </template>
  </div>
</template>
