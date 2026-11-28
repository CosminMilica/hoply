<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useLocalStorage } from '@vueuse/core'
import { useApplicationMutations, useApplicationStatuses } from '@/features/applications/composables/useApplications'
import { saveFile } from '@/features/common/download'
import { t } from '@/features/common/i18n'
import AtsCard from '../components/AtsCard.vue'
import CvPaper from '../components/CvPaper.vue'
import LetterCard from '../components/LetterCard.vue'
import TailoringCard from '../components/TailoringCard.vue'
import { authorshipLine } from '../authorship'
import { useLatestCvQuery } from '../composables/useCv'
import { CV_TEMPLATES, type CvTemplate } from '../api/cv.schema'
import { cvApi } from '../api/cvApi'

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
const templates = Object.keys(CV_TEMPLATES) as CvTemplate[]

const applied = computed(() => statuses.value.get(props.id) === 'applied')
const document = computed(() => (cv.value ? cv.value[mode.value] : null))

const downloading = ref<'pdf' | 'docx' | ''>('')
const downloadFailed = ref(false)
/** Downloads the version on screen, as the cv service renders it. */
async function download(format: 'pdf' | 'docx') {
  if (!cv.value) return
  downloading.value = format
  downloadFailed.value = false
  try {
    saveFile(await cvApi.file(cv.value.id, format, mode.value, template.value))
  }
  catch {
    downloadFailed.value = true
  }
  finally {
    downloading.value = ''
  }
}
</script>

<template>
  <div class="mx-auto w-full max-w-[1080px]">
    <USkeleton v-if="isPending" class="h-64 w-full" />
    <div v-else-if="!cv || !document" class="flex flex-col gap-2.5 pt-6">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">{{ t.cv.cvPage.emptyTitle }}</h1>
      <p class="m-0 text-base leading-normal font-medium text-muted">{{ t.cv.cvPage.emptyText }}</p>
      <UButton :to="{ name: 'crashr-job-add' }" :label="t.cv.cvPage.addJob" size="lg" block class="mt-2 justify-center" />
    </div>
    <div v-else class="flex flex-col gap-4 min-[900px]:grid min-[900px]:grid-cols-[380px_minmax(0,1fr)] min-[900px]:items-start min-[900px]:gap-x-12">
      <div class="contents min-[900px]:col-start-1 min-[900px]:row-start-1 min-[900px]:flex min-[900px]:flex-col min-[900px]:gap-4">
        <UButton :to="{ name: 'crashr-applications' }" :label="t.cv.cvPage.applications" icon="i-lucide-chevron-left" variant="ghost" color="neutral" class="order-0 self-start px-1 text-toned" />
        <div class="order-1 -mt-2.5 flex flex-col gap-1">
          <h1 class="m-0 text-[26px] leading-tight font-extrabold tracking-tight">{{ t.cv.cvPage.ready }}</h1>
          <p class="m-0 text-[15px] font-medium text-muted">{{ t.cv.cvPage.forJob(cv.job_title, cv.company) }}</p>
          <p class="m-0 text-[13px] font-medium text-dimmed" data-test="written-by">{{ authorshipLine(cv) }}</p>
        </div>
        <div class="order-2 flex gap-1.5 rounded-[10px] bg-accented p-1" role="group" :aria-label="t.cv.cvPage.version">
          <UButton
            v-for="option in [{ id: 'profile', label: t.cv.cvPage.profileVersion }, { id: 'tailored', label: t.cv.cvPage.tailored }] as const"
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
          <span class="text-[13px] font-bold text-muted">{{ t.cv.cvPage.template }}</span>
          <div class="flex gap-1.5 rounded-[10px] bg-accented p-1" role="group" :aria-label="t.cv.cvPage.template">
            <UButton
              v-for="name in templates"
              :key="name"
              :label="t.cv.cvPage.templates[name]"
              :aria-pressed="template === name"
              color="neutral"
              :variant="template === name ? 'soft' : 'ghost'"
              class="flex-1 justify-center text-[15px]"
              :class="template === name ? 'bg-ink-700' : ''"
              @click="template = name"
            />
          </div>
        </div>
        <AtsCard v-if="mode === 'tailored'" :ats="cv.ats" class="order-3" @improve="router.push({ name: 'crashr-prep', params: { id } })" />
        <TailoringCard v-if="mode === 'tailored' && cv.tailoring" :tailoring="cv.tailoring" class="order-3" />
        <div class="order-6 flex flex-col gap-2.5">
          <div class="flex gap-2">
            <UButton :label="t.cv.cvPage.downloadPdf" :loading="downloading === 'pdf'" size="lg" class="flex-1 justify-center" @click="download('pdf')" />
            <UButton label="DOCX" :loading="downloading === 'docx'" color="neutral" variant="outline" size="lg" class="justify-center" @click="download('docx')" />
          </div>
          <p v-if="downloadFailed" role="alert" class="m-0 text-center text-[13px] font-bold text-ember-400">{{ t.cv.cvPage.downloadFailed }}</p>
          <span class="text-center text-xs font-medium text-muted">{{ cv.file_name }}</span>
          <UButton
            :label="applied ? t.cv.cvPage.applied : t.cv.cvPage.markApplied"
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
          <mark class="rounded-md bg-crashr-200 px-1.5 py-px font-extrabold text-crashr-950">{{ t.cv.cvPage.highlighted }}</mark>
          {{ t.cv.cvPage.highlightNote }}
        </p>
        <LetterCard class="order-7" :cv-id="cv.id" :template="template" />
      </div>
    </div>
  </div>
</template>
