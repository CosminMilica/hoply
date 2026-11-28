<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useApplicationMutations } from '@/features/applications/composables/useApplications'
import { serviceErrorMessage } from '@/features/common/errors'
import { t } from '@/features/common/i18n'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { unreadableReason } from '../api/jobsApi'
import { jobInputSchema, type Job, type JobDraft, type ReadRequest, type UnreadableReason } from '../api/jobs.schema'
import JobDraftPreview from '../components/JobDraftPreview.vue'
import SendToHoplyCard from '../components/SendToHoplyCard.vue'
import { useAddJob } from '../composables/useJobs'
import { isLink, MIN_AD_LENGTH, unreadableMessage } from '../reading'
import { sentJob } from '../sendToHoply'

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
const created = ref<Job | null>(null)
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
  created.value = null
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
    error.value = t.value.jobs.addJob.fullLink
    return
  }
  readJob({ url: form.url.trim() })
}

function readText() {
  if (form.text.trim().length < MIN_AD_LENGTH) {
    error.value = t.value.jobs.addJob.wholeAd
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

// A job the Send to Hoply bookmark sent is read at once; the hash leaves the address.
onMounted(() => {
  const sent = sentJob(route.hash)
  if (!sent) return
  router.replace({ query: route.query, hash: '' })
  tab.value = 'text'
  Object.assign(form, { text: sent.text, title: sent.title, company: sent.company, link: sent.url })
  readText()
})

async function save(tailor: boolean) {
  if (!draft.value) return
  const { host: _host, ...job } = draft.value
  const parsed = jobInputSchema.safeParse({ ...job, source: job.source === 'link' ? 'link' : 'text' })
  if (!parsed.success) {
    error.value = parsed.error.issues[0]?.message ?? t.value.jobs.addJob.checkJob
    return
  }
  error.value = ''
  saving.value = true
  try {
    // A retry after saving the job but not its status saves only the status: the job is not added twice.
    const saved = created.value ?? await create.mutateAsync(parsed.data)
    created.value = saved
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
    <UButton variant="ghost" color="neutral" icon="i-lucide-chevron-left" :label="t.common.actions.back" class="self-start px-1 text-toned" @click="back" />
    <div class="-mt-2 flex flex-col gap-1">
      <h1 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">{{ t.jobs.addJob.title }}</h1>
      <p class="m-0 text-[15px] leading-snug font-medium text-muted">
        {{ t.jobs.addJob.intro }}
      </p>
    </div>

    <template v-if="phase === 'input'">
      <div role="tablist" :aria-label="t.jobs.addJob.addFrom" class="flex gap-1 rounded-[10px] bg-accented p-1">
        <button
          v-for="option in [{ id: 'link', label: t.jobs.addJob.link, icon: 'i-lucide-link' }, { id: 'text', label: t.jobs.addJob.text, icon: 'i-lucide-text' }] as const"
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

      <form v-if="tab === 'link'" role="tabpanel" :aria-label="t.jobs.addJob.link" class="flex flex-col gap-2" @submit.prevent="readLink">
        <UFormField :label="t.jobs.addJob.linkLabel" :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase' }">
          <UInput v-model="form.url" type="url" placeholder="https://" class="w-full" />
        </UFormField>
        <span class="text-[13px] leading-snug font-medium text-muted">
          {{ t.jobs.addJob.worksWith }}
        </span>
        <p v-if="error" role="alert" class="m-0 text-[13px] font-bold text-ember-400">{{ error }}</p>
        <UButton type="submit" :label="t.jobs.addJob.read" size="lg" block class="mt-1 justify-center" />
      </form>

      <form v-else role="tabpanel" :aria-label="t.jobs.addJob.text" class="flex flex-col gap-2" @submit.prevent="readText">
        <UFormField :label="t.jobs.addJob.adLabel" :ui="{ label: 'text-xs font-extrabold tracking-wider text-muted uppercase' }">
          <UTextarea
            v-model="form.text"
            :rows="9"
            :placeholder="t.jobs.addJob.adPlaceholder"
            maxlength="50000"
            class="w-full"
          />
        </UFormField>
        <div class="grid grid-cols-2 gap-2">
          <UFormField :label="t.jobs.addJob.titleOptional" :ui="{ label: 'text-xs font-bold text-muted' }">
            <UInput v-model="form.title" :placeholder="t.jobs.addJob.findIt" maxlength="160" class="w-full" />
          </UFormField>
          <UFormField :label="t.jobs.addJob.companyOptional" :ui="{ label: 'text-xs font-bold text-muted' }">
            <UInput v-model="form.company" :placeholder="t.jobs.addJob.findIt" maxlength="120" class="w-full" />
          </UFormField>
        </div>
        <UFormField :label="t.jobs.addJob.linkOptional" :ui="{ label: 'text-xs font-bold text-muted' }">
          <UInput v-model="form.link" type="url" placeholder="https://" maxlength="2000" class="w-full" />
        </UFormField>
        <p v-if="error" role="alert" class="m-0 text-[13px] font-bold text-ember-400">{{ error }}</p>
        <UButton type="submit" :label="t.jobs.addJob.read" size="lg" block class="mt-1 justify-center" />
      </form>
      <SendToHoplyCard />
    </template>

    <div v-else-if="phase === 'loading'" role="status" class="flex items-center gap-3.5 rounded-[14px] border border-default bg-elevated p-5">
      <span class="size-[22px] shrink-0 animate-pulse rounded-full bg-hoply-500" aria-hidden="true" />
      <span class="flex flex-col gap-0.5">
        <span class="text-[15px] font-extrabold">{{ tab === 'link' ? t.jobs.addJob.readingPage : t.jobs.addJob.readingAd }}</span>
        <span class="text-[13px] font-medium text-muted">{{ t.jobs.addJob.lookingFor }}</span>
      </span>
    </div>

    <div v-else-if="phase === 'failed'" class="flex flex-col gap-2.5 rounded-[14px] border border-default bg-elevated p-[18px]" data-test="read-failed">
      <h2 class="m-0 text-[19px] font-extrabold tracking-tight">{{ t.jobs.addJob.failed }}</h2>
      <p class="m-0 text-sm leading-normal font-medium text-ink-300">{{ failure }}</p>
      <div class="flex flex-wrap items-center gap-2.5">
        <UButton :label="t.jobs.addJob.pasteInstead" @click="pasteInstead" />
        <UButton :label="t.jobs.addJob.tryAnother" variant="link" color="neutral" size="sm" class="px-1 text-[13px] text-muted underline" @click="tryAnother" />
      </div>
    </div>

    <JobDraftPreview v-else-if="draft" v-model="draft" :profile="person" :error="error" :saving="saving" @save="save" />
  </div>
</template>
