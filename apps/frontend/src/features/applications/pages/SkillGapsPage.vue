<script setup lang="ts">
import { computed } from 'vue'
import { askedLine, skillGaps } from '../gaps'
import { useTrackedJobs } from '../composables/useTrackedJobs'

const { tracked, skills, loading, error } = useTrackedJobs()
const total = computed(() => tracked.value.length)
const gaps = computed(() => skillGaps(tracked.value.map(entry => entry.job), skills.value))
const sections = computed(() => [
  { id: 'missing', title: 'Not in your profile', items: gaps.value.missing },
  { id: 'close', title: 'Close to what you know', items: gaps.value.close },
].filter(section => section.items.length))
</script>

<template>
  <div class="mx-auto flex w-full max-w-[1080px] flex-col gap-3.5">
    <div class="flex flex-col gap-1">
      <UButton :to="{ name: 'hoply-saved' }" icon="i-lucide-arrow-left" label="Saved jobs" variant="link" color="neutral" class="self-start px-0" />
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">Skill gaps</h1>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">What your saved jobs ask for that your profile does not show yet, the most asked first.</p>
    </div>

    <USkeleton v-if="loading" class="h-40 w-full rounded-[14px]" />
    <UAlert v-else-if="error" color="error" variant="subtle" title="We could not load your saved jobs" description="Try again in a moment." />
    <div v-else-if="!total" class="flex flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-[22px]">
      <p class="m-0 text-[15px] leading-normal font-medium text-muted">Save a few jobs first. Their requirements show here, next to your profile.</p>
      <UButton :to="{ name: 'hoply-job-add', query: { from: 'saved' } }" icon="i-lucide-plus" label="Add a job" size="lg" block class="justify-center" />
    </div>
    <p v-else-if="!sections.length" class="m-0 rounded-[14px] border border-default bg-elevated p-[22px] text-[15px] font-medium text-muted">
      Your profile covers every requirement of your saved jobs.
    </p>
    <template v-else>
      <section v-for="section in sections" :key="section.id" class="flex flex-col gap-2.5" :data-test="`gaps-${section.id}`">
        <h2 class="m-0 text-lg font-extrabold">{{ section.title }} <span class="text-muted">{{ section.items.length }}</span></h2>
        <ul class="grid grid-cols-1 gap-3 sm:grid-cols-2 min-[900px]:grid-cols-3">
          <li v-for="gap in section.items" :key="gap.skill" class="flex flex-col gap-1.5 rounded-[14px] border border-default bg-elevated p-3.5">
            <h3 class="m-0 text-[17px] leading-tight font-extrabold">{{ gap.skill }}</h3>
            <p v-if="gap.via.length" class="m-0 text-sm font-medium text-toned">You know {{ gap.via.join(', ') }} ({{ gap.category }}).</p>
            <p class="m-0 text-[13px] font-medium text-muted">{{ askedLine(gap, total) }}</p>
            <ul class="m-0 flex flex-col gap-0.5 p-0">
              <li v-for="job in gap.jobs" :key="job.id" class="list-none text-[13px]">
                <RouterLink :to="{ name: 'hoply-job', params: { id: job.id } }" class="font-semibold text-hoply-400 underline-offset-2 hover:underline">
                  {{ [job.title, job.company].filter(Boolean).join(' · ') }}
                </RouterLink>
              </li>
            </ul>
          </li>
        </ul>
      </section>
      <p class="m-0 text-sm font-medium text-muted">
        Learned one? Add it to your <RouterLink :to="{ name: 'hoply-profile' }" class="font-semibold text-hoply-400 underline">profile</RouterLink> and it leaves this list.
      </p>
    </template>
  </div>
</template>
