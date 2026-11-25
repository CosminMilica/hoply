<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useLocalStorage } from '@vueuse/core'
import { useApplicationMutations, useApplicationStatuses } from '@/features/applications/composables/useApplications'
import AtsCard from '../components/AtsCard.vue'
import CvPaper from '../components/CvPaper.vue'
import LetterCard from '../components/LetterCard.vue'
import TailoringCard from '../components/TailoringCard.vue'
import { authorshipLine } from '../authorship'
import { useLatestCvQuery } from '../composables/useCv'
import { CV_TEMPLATES, type CvTemplate } from '../api/cv.schema'

const props = defineProps<{ id: string }>()
const router = useRouter()
const { data: cv, isPending } = useLatestCvQuery(() => props.id)
const { statuses } = useApplicationStatuses()
const { setStatus } = useApplicationMutations()
const mode = ref<'profile' | 'tailored'>('tailored')
const stored = useLocalStorage<string>('hoply.cv.template', 'classic')
const template = computed<CvTemplate>({
  get: () => (Object.hasOwn(CV_TEMPLATES, stored.value) ? stored.value as CvTemplate : 'classic'),
  set: (value) => { stored.value = value },
})
const templates = Object.entries(CV_TEMPLATES) as [CvTemplate, string][]

const applied = computed(() => statuses.value.get(props.id) === 'applied')
const document = computed(() => (cv.value ? cv.value[mode.value] : null))

/** The file of the version on screen, as the cv service renders it. */
const fileUrl = (format: 'pdf' | 'docx') =>
  cv.value ? `/api/cv/v1/cvs/${encodeURIComponent(cv.value.id)}/file?format=${format}&version=${mode.value}&template=${template.value}` : ''
</script>

<template>
  <div class="mx-auto w-full max-w-[1080px]">
    <USkeleton v-if="isPending" class="h-64 w-full" />
    <div v-else-if="!cv || !document" class="flex flex-col gap-2.5 pt-6">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">No CV yet</h1>
      <p class="m-0 text-base leading-normal font-medium text-muted">Add a job, prepare your CV, and the tailored version shows up here.</p>
      <UButton :to="{ name: 'hoply-job-add' }" label="Add a job" size="lg" block class="mt-2 justify-center" />
    </div>
    <div v-else class="flex flex-col gap-4 min-[900px]:grid min-[900px]:grid-cols-[380px_minmax(0,1fr)] min-[900px]:items-start min-[900px]:gap-x-12">
      <div class="contents min-[900px]:col-start-1 min-[900px]:row-start-1 min-[900px]:flex min-[900px]:flex-col min-[900px]:gap-4">
        <UButton :to="{ name: 'hoply-saved' }" label="Saved jobs" icon="i-lucide-chevron-left" variant="ghost" color="neutral" class="order-0 self-start px-1 text-toned" />
        <div class="order-1 -mt-2.5 flex flex-col gap-1">
          <h1 class="m-0 text-[26px] leading-tight font-extrabold tracking-tight">Your CV is ready</h1>
          <p class="m-0 text-[15px] font-medium text-muted">For {{ cv.job_title }} at {{ cv.company }}</p>
          <p class="m-0 text-[13px] font-medium text-dimmed" data-test="written-by">{{ authorshipLine(cv) }}</p>
        </div>
        <div class="order-2 flex gap-1.5 rounded-[10px] bg-accented p-1" role="group" aria-label="Version">
          <UButton
            v-for="option in [{ id: 'profile', label: 'Your profile' }, { id: 'tailored', label: 'Tailored' }] as const"
            :key="option.id"
            :label="option.label"
            :aria-pressed="mode === option.id"
            color="neutral"
            :variant="mode === option.id ? 'soft' : 'ghost'"
            class="flex-1 justify-center text-[15px]"
            :class="mode === option.id ? 'bg-ink-700' : ''"
            @click="mode = option.id"
          />
        </div>
        <div class="order-2 flex flex-col gap-1.5">
          <span class="text-[13px] font-bold text-muted">Template</span>
          <div class="flex gap-1.5 rounded-[10px] bg-accented p-1" role="group" aria-label="Template">
            <UButton
              v-for="[name, label] in templates"
              :key="name"
              :label="label"
              :aria-pressed="template === name"
              color="neutral"
              :variant="template === name ? 'soft' : 'ghost'"
              class="flex-1 justify-center text-[15px]"
              :class="template === name ? 'bg-ink-700' : ''"
              @click="template = name"
            />
          </div>
        </div>
        <AtsCard v-if="mode === 'tailored'" :ats="cv.ats" class="order-3" @improve="router.push({ name: 'hoply-prep', params: { id } })" />
        <TailoringCard v-if="mode === 'tailored' && cv.tailoring" :tailoring="cv.tailoring" class="order-3" />
        <div class="order-6 flex flex-col gap-2.5">
          <div class="flex gap-2">
            <UButton :to="fileUrl('pdf')" external download label="Download PDF" size="lg" class="flex-1 justify-center" />
            <UButton :to="fileUrl('docx')" external download label="DOCX" color="neutral" variant="outline" size="lg" class="justify-center" />
          </div>
          <span class="text-center text-xs font-medium text-muted">{{ cv.file_name }}</span>
          <UButton
            :label="applied ? 'Marked as applied ✓' : 'I applied to this job'"
            color="neutral"
            :variant="applied ? 'soft' : 'outline'"
            size="lg"
            block
            class="justify-center"
            :disabled="applied"
            @click="setStatus.mutate({ jobId: id, status: 'applied' })"
          />
        </div>
      </div>
      <div class="contents min-[900px]:col-start-2 min-[900px]:row-start-1 min-[900px]:flex min-[900px]:flex-col min-[900px]:gap-4">
        <CvPaper class="order-4" :document="document" :template="template" :highlights="mode === 'tailored' ? cv.highlights : []" />
        <p v-if="mode === 'tailored'" class="order-5 m-0 text-[13px] leading-normal font-medium text-muted">
          <mark class="rounded-md bg-hoply-200 px-1.5 py-px font-extrabold text-hoply-950">Highlighted</mark>
          words are only for you. Every template exports real text in one column with standard headings, so any ATS can read it. Everything in it comes from your profile and your answers.
        </p>
        <LetterCard class="order-7" :cv-id="cv.id" :template="template" />
      </div>
    </div>
  </div>
</template>
