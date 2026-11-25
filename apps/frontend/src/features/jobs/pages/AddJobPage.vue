<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useApplicationMutations } from '@/features/applications/composables/useApplications'
import { serviceErrorMessage } from '@/features/common/errors'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { unreadableReason } from '../api/jobsApi'
import { jobInputSchema, type JobDraft, type ReadRequest, type UnreadableReason } from '../api/jobs.schema'
import JobDraftPreview from '../components/JobDraftPreview.vue'
import { useAddJob } from '../composables/useJobs'
import { isLink, MIN_AD_LENGTH, unreadableMessage } from '../reading'

type Phase = 'input' | 'loading' | 'failed' | 'preview'

const route = useRoute()
const router = useRouter()
const profile = useProfileQuery()
const { read, create } = useAddJob()
const { setStatus } = useApplicationMutations()

const fromSaved = computed(() => route.query.from === 'saved')
const phase = ref<Phase>('input')
const tab = ref<'link' | 'text'>('link')
const form = reactive({ url: '', text: '', title: '', company: '', link: '' })
const error = ref('')
const reason = ref<UnreadableReason | 'unknown'>('unknown')
const draft = ref<JobDraft | null>(null)
const saving = ref(false)
const person = computed(() => profile.data.value)
const failure = computed(() => unreadableMessage(reason.value, form.url))

function pick(next: 'link' | 'text') {
  tab.value = next
  error.value = ''
}

function back() {
  if (phase.value !== 'input') {
    phase.value = 'input'
    draft.value = null
    error.value = ''
    return
  }
  router.push({ name: fromSaved.value ? 'hoply-saved' : 'hoply-jobs' })
}

function readJob(body: ReadRequest) {
  error.value = ''
  phase.value = 'loading'
  read.mutate(body, {
    onSuccess: (result) => {
      draft.value = result
      phase.value = 'preview'
    },
    onError: async (problem) => {
      const why = unreadableReason(problem)
      if (why && 'url' in body && !('text' in body)) {
        reason.value = why
        phase.value = 'failed'
        return
      }
      error.value = await serviceErrorMessage(problem)
      phase.value = 'input'
    },
  })
}

function readLink() {
  if (!isLink(form.url)) {
    error.value = 'Enter the full link, starting with https://'
    return
  }
  readJob({ url: form.url.trim() })
}

function readText() {
  if (form.text.trim().length < MIN_AD_LENGTH) {
    error.value = 'Paste the whole ad: a few lines with what they ask for and what you would do.'
    return
  }
  const optional = Object.fromEntries(
    Object.entries({ title: form.title, company: form.company, url: form.link }).map(([key, value]) => [key, value.trim()]).filter(([, value]) => value),
  )
  readJob({ text: form.text.trim(), ...optional })
}

function pasteInstead() {
  form.link = form.url.trim()
  tab.value = 'text'
  phase.value = 'input'
}

function tryAnother() {
  tab.value = 'link'
  phase.value = 'input'
}

async function save(tailor: boolean) {
  if (!draft.value) return
  const { host: _host, ...job } = draft.value
  const parsed = jobInputSchema.safeParse({ ...job, source: job.source === 'link' ? 'link' : 'text' })
  if (!parsed.success) {
    error.value = parsed.error.issues[0]?.message ?? 'Check the job.'
    return
  }
  error.value = ''
  saving.value = true
  try {
    const saved = await create.mutateAsync(parsed.data)
    await setStatus.mutateAsync({ jobId: saved.id, status: 'saved' })
    await router.push(tailor ? { name: 'hoply-prep', params: { id: saved.id } } : { name: 'hoply-saved' })
  }
  catch (problem) {
    error.value = await serviceErrorMessage(problem)
  }
  finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="mx-auto flex w-full max-w-[620px] flex-col gap-4">
    <UButton variant="ghost" color="neutral" icon="i-lucide-chevron-left" label="Back" class="self-start px-1 text-toned" @click="back" />
    <div class="-mt-2 flex flex-col gap-1">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">Add a job</h1>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">
        Found a job somewhere else? Give us its link or paste its text. We read it, match it to your profile, and you can tailor a CV for it.
      </p>
    </div>

    <template v-if="phase === 'input'">
      <div role="tablist" aria-label="Add from" class="flex gap-1 rounded-[10px] bg-accented p-1">
        <button
          v-for="option in [{ id: 'link', label: 'Link', icon: 'i-lucide-link' }, { id: 'text', label: 'Text', icon: 'i-lucide-text' }] as const"
          :key="option.id"
          type="button"
          role="tab"
          :aria-selected="tab === option.id"
          class="flex min-h-9 flex-1 items-center justify-center gap-1.5 rounded-lg text-sm font-extrabold"
          :class="tab === option.id ? 'bg-ink-700 text-highlighted' : 'text-toned'"
          @click="pick(option.id)"
        >
          <UIcon :name="option.icon" class="size-4" />{{ option.label }}
        </button>
      </div>

      <form v-if="tab === 'link'" role="tabpanel" aria-label="Link" class="flex flex-col gap-2" @submit.prevent="readLink">
        <UFormField label="Link to the job ad" :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase' }">
          <UInput v-model="form.url" type="url" placeholder="https://" class="w-full" />
        </UFormField>
        <span class="text-[13px] leading-snug font-medium text-muted">
          Works with most job pages: company career sites, Greenhouse, Lever, Ashby, SmartRecruiters, eJobs. Pages behind a login, like LinkedIn, need the text instead.
        </span>
        <p v-if="error" role="alert" class="m-0 text-[13px] font-bold text-ember-400">{{ error }}</p>
        <UButton type="submit" label="Read the job" size="lg" block class="mt-1 justify-center" />
      </form>

      <form v-else role="tabpanel" aria-label="Text" class="flex flex-col gap-2" @submit.prevent="readText">
        <UFormField label="The job ad" :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase' }">
          <UTextarea
            v-model="form.text"
            :rows="9"
            placeholder="Paste the whole ad: title, company, location, requirements and responsibilities. English or Romanian."
            maxlength="50000"
            class="w-full"
          />
        </UFormField>
        <div class="grid grid-cols-2 gap-2">
          <UFormField label="Job title (optional)" :ui="{ label: 'text-xs font-bold text-muted' }">
            <UInput v-model="form.title" placeholder="We find it if empty" maxlength="160" class="w-full" />
          </UFormField>
          <UFormField label="Company (optional)" :ui="{ label: 'text-xs font-bold text-muted' }">
            <UInput v-model="form.company" placeholder="We find it if empty" maxlength="120" class="w-full" />
          </UFormField>
        </div>
        <UFormField label="Link to the ad (optional, to find it again)" :ui="{ label: 'text-xs font-bold text-muted' }">
          <UInput v-model="form.link" type="url" placeholder="https://" maxlength="2000" class="w-full" />
        </UFormField>
        <p v-if="error" role="alert" class="m-0 text-[13px] font-bold text-ember-400">{{ error }}</p>
        <UButton type="submit" label="Read the job" size="lg" block class="mt-1 justify-center" />
      </form>
    </template>

    <div v-else-if="phase === 'loading'" role="status" class="flex items-center gap-3.5 rounded-[14px] border border-default bg-elevated p-5">
      <span class="size-[22px] shrink-0 animate-pulse rounded-full bg-hoply-500" aria-hidden="true" />
      <span class="flex flex-col gap-0.5">
        <span class="text-[15px] font-extrabold">{{ tab === 'link' ? 'Reading the page' : 'Reading the ad' }}</span>
        <span class="text-[13px] font-medium text-muted">Looking for the job's title, company, location, requirements and responsibilities.</span>
      </span>
    </div>

    <div v-else-if="phase === 'failed'" class="flex flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-[18px]" data-test="read-failed">
      <h2 class="m-0 text-[19px] font-extrabold tracking-tight">We could not read this page</h2>
      <p class="m-0 text-sm leading-normal font-medium text-ink-300">{{ failure }}</p>
      <div class="flex flex-wrap items-center gap-2.5">
        <UButton label="Paste the text instead" @click="pasteInstead" />
        <UButton label="Try another link" variant="link" color="neutral" size="sm" class="px-1 text-[13px] text-muted underline" @click="tryAnother" />
      </div>
    </div>

    <JobDraftPreview v-else-if="draft" v-model="draft" :profile="person" :error="error" :saving="saving" @save="save" />
  </div>
</template>
