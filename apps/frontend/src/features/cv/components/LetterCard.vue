<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import { cvApi } from '../api/cvApi'
import { CONNECTION_NOTE_LIMIT, type CvTemplate, type Letter } from '../api/cv.schema'
import { letterFailure, modelLabel } from '../authorship'
import { useLetterQuery, useWriteLetter } from '../composables/useCv'

const props = defineProps<{ cvId: string, template: CvTemplate }>()
const { data: letter, isPending } = useLetterQuery(() => props.cvId)
const write = useWriteLetter(() => props.cvId)
const recruiter = ref('')

const TABS = [
  { id: 'letter', label: 'Cover letter', hint: 'Paste it into the application, or download it with your CV\'s look.' },
  { id: 'note', label: 'LinkedIn note', hint: 'Send it with a connection request to the recruiter or the hiring manager.' },
  { id: 'message', label: 'Message', hint: 'Send it right after you apply, on LinkedIn or by email.' },
  { id: 'follow_up', label: 'Follow-up', hint: 'Send it once, about a week after you apply, if you have heard nothing.' },
] as const
type Tab = typeof TABS[number]['id']
const tab = ref<Tab>('letter')
const current = computed(() => TABS.find(item => item.id === tab.value)!)

type Drafts = Record<Tab, { subject: string, body: string }>
const written = (value: Letter | undefined): Drafts => {
  const texts = value?.texts
  return {
    letter: { subject: '', body: texts?.cover_letter ?? '' },
    note: { subject: '', body: texts?.connection_note ?? '' },
    message: { subject: texts?.message.subject ?? '', body: texts?.message.body ?? '' },
    follow_up: { subject: texts?.follow_up.subject ?? '', body: texts?.follow_up.body ?? '' },
  }
}
const drafts = ref<Drafts>(written(undefined))
watch(() => letter.value?.texts, () => {
  drafts.value = written(letter.value)
  if (letter.value?.recruiter) recruiter.value = letter.value.recruiter
}, { immediate: true })

const draft = computed(() => drafts.value[tab.value])
const edited = computed(() => {
  const original = written(letter.value)[tab.value]
  return draft.value.body !== original.body || draft.value.subject !== original.subject
})
const hasSubject = computed(() => tab.value === 'message' || tab.value === 'follow_up')
const words = (text: string) => text.trim().split(/\s+/).filter(Boolean).length
const count = computed(() => tab.value === 'note'
  ? `${draft.value.body.length} / ${CONNECTION_NOTE_LIMIT} characters`
  : `${words(draft.value.body)} words`)
const overLimit = computed(() => tab.value === 'note' && draft.value.body.length > CONNECTION_NOTE_LIMIT)

const copied = ref('')
async function copy(what: 'subject' | 'body') {
  try {
    await navigator.clipboard.writeText(what === 'subject' ? draft.value.subject : draft.value.body)
    copied.value = what
    setTimeout(() => { copied.value = '' }, 2000)
  }
  catch {
    copied.value = ''
  }
}

const downloading = ref(false)
const downloadFailed = ref(false)
async function download(format: 'pdf' | 'docx') {
  downloading.value = true
  downloadFailed.value = false
  try {
    const file = await cvApi.letterFile(props.cvId, format, props.template, drafts.value.letter.body)
    const url = URL.createObjectURL(file.blob)
    const link = document.createElement('a')
    link.href = url
    link.download = file.name
    link.click()
    URL.revokeObjectURL(url)
  }
  catch {
    downloadFailed.value = true
  }
  finally {
    downloading.value = false
  }
}

const startWriting = () => write.mutate(recruiter.value)
</script>

<template>
  <section class="flex flex-col gap-3 rounded-2xl border border-default bg-elevated p-4" aria-label="Cover letter and messages" data-test="letter">
    <SectionLabel as="h2">Cover letter and messages</SectionLabel>
    <USkeleton v-if="isPending" class="h-20 w-full" />
    <div v-else-if="letter?.status === 'writing'" class="flex flex-col gap-2" data-test="letter-writing">
      <p class="m-0 text-sm font-semibold">Claude is writing them from this CV. It takes about half a minute.</p>
      <USkeleton class="h-3 w-full" />
      <USkeleton class="h-3 w-11/12" />
      <USkeleton class="h-3 w-4/5" />
    </div>
    <template v-else-if="letter?.status === 'ready' && letter.texts">
      <div class="flex gap-1 overflow-x-auto rounded-[10px] bg-accented p-1" role="group" aria-label="Text">
        <UButton
          v-for="item in TABS"
          :key="item.id"
          :label="item.label"
          :aria-pressed="tab === item.id"
          color="neutral"
          :variant="tab === item.id ? 'soft' : 'ghost'"
          size="sm"
          class="shrink-0 flex-1 justify-center"
          :class="tab === item.id ? 'bg-ink-700' : ''"
          @click="tab = item.id"
        />
      </div>
      <p class="m-0 text-[13px] leading-snug font-medium text-muted">{{ current.hint }}</p>
      <p v-if="!draft.body" class="m-0 text-sm font-medium text-muted" data-test="letter-empty">Nothing passed the checks for this one. Write them again to get a new one.</p>
      <template v-else>
        <div v-if="hasSubject" class="flex items-end gap-2">
          <UFormField label="Subject" class="flex-1">
            <UInput v-model="draft.subject" class="w-full" />
          </UFormField>
          <UButton :label="copied === 'subject' ? 'Copied' : 'Copy'" aria-label="Copy the subject" color="neutral" variant="outline" @click="copy('subject')" />
        </div>
        <UTextarea v-model="draft.body" autoresize :rows="tab === 'letter' ? 12 : 5" :aria-label="current.label" class="w-full" data-test="letter-text" />
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span class="text-xs font-semibold" :class="overLimit ? 'text-error' : 'text-muted'" data-test="letter-count">{{ count }}</span>
          <div class="flex gap-2">
            <UButton v-if="edited" label="Undo my edits" color="neutral" variant="ghost" size="sm" @click="drafts[tab] = written(letter)[tab]" />
            <UButton :label="copied === 'body' ? 'Copied' : 'Copy'" :aria-label="`Copy the ${current.label.toLowerCase()}`" size="sm" @click="copy('body')" />
          </div>
        </div>
        <div v-if="tab === 'letter'" class="flex gap-2">
          <UButton label="Download PDF" aria-label="Download the cover letter as PDF" :loading="downloading" class="flex-1 justify-center" @click="download('pdf')" />
          <UButton label="DOCX" aria-label="Download the cover letter as DOCX" color="neutral" variant="outline" :disabled="downloading" class="justify-center" @click="download('docx')" />
        </div>
        <p v-if="downloadFailed" class="m-0 text-[13px] font-medium text-error">The file could not be made. Try again in a moment.</p>
      </template>
      <p class="m-0 text-xs font-medium text-dimmed">
        Written by Claude{{ letter.model ? ` (${modelLabel(letter.model)})` : '' }} from this CV and your profile. Your edits stay on this page; downloads include them.
      </p>
    </template>
    <p v-else-if="letter?.status === 'failed'" class="m-0 text-sm font-medium text-muted" data-test="letter-failed">{{ letterFailure(letter) }}</p>
    <p v-else class="m-0 text-sm leading-snug font-medium text-muted">
      A cover letter, a LinkedIn note, a message for after you apply and a follow-up, written from this CV and checked against your profile.
    </p>
    <form v-if="!isPending && letter?.status !== 'writing'" class="flex flex-col gap-2" @submit.prevent="startWriting">
      <UFormField label="Recruiter's name (optional)">
        <UInput v-model="recruiter" maxlength="80" placeholder="Ana Popescu" class="w-full" />
      </UFormField>
      <UButton
        type="submit"
        :label="letter?.status === 'ready' ? 'Write them again' : letter?.status === 'failed' ? 'Try again' : 'Write them'"
        :color="letter?.status === 'ready' ? 'neutral' : 'primary'"
        :variant="letter?.status === 'ready' ? 'outline' : 'solid'"
        :loading="write.isPending.value"
        block
        class="justify-center"
      />
    </form>
  </section>
</template>
