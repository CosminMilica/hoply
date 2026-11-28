<script setup lang="ts">
import { saveFile } from '@/features/common/download'
import { computed, ref, watch } from 'vue'
import { t } from '@/features/common/i18n'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import { cvApi } from '../api/cvApi'
import { CONNECTION_NOTE_LIMIT, type CvTemplate, type Letter } from '../api/cv.schema'
import { letterFailure, modelLabel } from '../authorship'
import { useLetterQuery, useWriteLetter } from '../composables/useCv'
import ProfileGate from '@/features/profile/components/ProfileGate.vue'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { readyForClaude } from '@/features/profile/readiness'

const props = defineProps<{ cvId: string, template: CvTemplate }>()
const { data: letter, isPending } = useLetterQuery(() => props.cvId)
const write = useWriteLetter(() => props.cvId)
// A letter is written by Claude: a profile that says too little is sent to be completed first.
const profile = useProfileQuery()
const enough = computed(() => !profile.data.value || readyForClaude(profile.data.value))
const recruiter = ref('')

const TABS = ['letter', 'note', 'message', 'follow_up'] as const
type Tab = typeof TABS[number]
const tab = ref<Tab>('letter')
const tabs = computed(() => TABS.map(id => ({ id, ...t.value.cv.letter.tabs[id] })))
const current = computed(() => t.value.cv.letter.tabs[tab.value])

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
  ? t.value.cv.letter.characters(draft.value.body.length, CONNECTION_NOTE_LIMIT)
  : t.value.cv.letter.words(words(draft.value.body)))
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
    saveFile(await cvApi.letterFile(props.cvId, format, props.template, drafts.value.letter.body))
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
  <section class="flex flex-col gap-3 rounded-2xl border border-default bg-elevated p-4" :aria-label="t.cv.letter.title" data-test="letter">
    <SectionLabel as="h2">{{ t.cv.letter.title }}</SectionLabel>
    <USkeleton v-if="isPending" class="h-20 w-full" />
    <div v-else-if="letter?.status === 'writing'" class="flex flex-col gap-2" data-test="letter-writing">
      <p class="m-0 text-sm font-semibold">{{ t.cv.letter.writing }}</p>
      <USkeleton class="h-3 w-full" />
      <USkeleton class="h-3 w-11/12" />
      <USkeleton class="h-3 w-4/5" />
    </div>
    <template v-else-if="letter?.status === 'ready' && letter.texts">
      <div class="flex gap-1 overflow-x-auto rounded-[10px] bg-accented p-1" role="group" :aria-label="t.cv.letter.tabsLabel">
        <UButton
          v-for="item in tabs"
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
      <p v-if="!draft.body" class="m-0 text-sm font-medium text-muted" data-test="letter-empty">{{ t.cv.letter.empty }}</p>
      <template v-else>
        <div v-if="hasSubject" class="flex items-end gap-2">
          <UFormField :label="t.cv.letter.subject" class="flex-1">
            <UInput v-model="draft.subject" class="w-full" />
          </UFormField>
          <UButton :label="copied === 'subject' ? t.cv.letter.copied : t.cv.letter.copy" :aria-label="t.cv.letter.copySubject" color="neutral" variant="outline" @click="copy('subject')" />
        </div>
        <UTextarea v-model="draft.body" autoresize :rows="tab === 'letter' ? 12 : 5" :aria-label="current.label" class="w-full" data-test="letter-text" />
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span class="text-xs font-semibold" :class="overLimit ? 'text-error' : 'text-muted'" data-test="letter-count">{{ count }}</span>
          <div class="flex gap-2">
            <UButton v-if="edited" :label="t.cv.letter.undo" color="neutral" variant="ghost" size="sm" @click="drafts[tab] = written(letter)[tab]" />
            <UButton :label="copied === 'body' ? t.cv.letter.copied : t.cv.letter.copy" :aria-label="current.copy" size="sm" @click="copy('body')" />
          </div>
        </div>
        <div v-if="tab === 'letter'" class="flex gap-2">
          <UButton :label="t.cv.letter.downloadPdf" :aria-label="t.cv.letter.downloadPdfLabel" :loading="downloading" class="flex-1 justify-center" @click="download('pdf')" />
          <UButton label="DOCX" :aria-label="t.cv.letter.downloadDocxLabel" color="neutral" variant="outline" :disabled="downloading" class="justify-center" @click="download('docx')" />
        </div>
        <p v-if="downloadFailed" class="m-0 text-[13px] font-medium text-error">{{ t.cv.letter.fileFailed }}</p>
      </template>
      <p class="m-0 text-xs font-medium text-dimmed">
        {{ t.cv.letter.writtenBy(letter.model ? modelLabel(letter.model) : '') }}
      </p>
    </template>
    <p v-else-if="letter?.status === 'failed'" class="m-0 text-sm font-medium text-muted" data-test="letter-failed">{{ letterFailure(letter) }}</p>
    <p v-else class="m-0 text-sm leading-snug font-medium text-muted">
      {{ t.cv.letter.intro }}
    </p>
    <ProfileGate v-if="!enough && profile.data.value && letter?.status !== 'writing'" :profile="profile.data.value" />
    <form v-else-if="!isPending && letter?.status !== 'writing'" class="flex flex-col gap-2" @submit.prevent="startWriting">
      <UFormField :label="t.cv.letter.recruiter">
        <UInput v-model="recruiter" maxlength="80" placeholder="Ana Popescu" class="w-full" />
      </UFormField>
      <UButton
        type="submit"
        :label="letter?.status === 'ready' ? t.cv.letter.writeAgain : letter?.status === 'failed' ? t.common.actions.retry : t.cv.letter.write"
        :color="letter?.status === 'ready' ? 'neutral' : 'primary'"
        :variant="letter?.status === 'ready' ? 'outline' : 'solid'"
        :loading="write.isPending.value"
        block
        class="justify-center"
      />
    </form>
  </section>
</template>
