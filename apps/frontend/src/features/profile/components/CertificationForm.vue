<script setup lang="ts">
import { reactive, ref } from 'vue'
import { t } from '@/features/common/i18n'
import { certificationInputSchema, withoutId, type Certification, type CertificationInput } from '../api/profile.schema'

const label = { label: 'text-xs font-bold text-muted' }

const props = defineProps<{ initial?: Certification, pending?: boolean, serverError?: string }>()
const emit = defineEmits<{ save: [input: CertificationInput], cancel: [] }>()

const draft = reactive(props.initial ? withoutId(props.initial) : { name: '', issuer: '', issued: '', expires: '', credential_id: '', url: '' })
const errors = ref<Record<string, string>>({})

function submit() {
  const parsed = certificationInputSchema.safeParse(draft)
  if (!parsed.success) {
    errors.value = Object.fromEntries(parsed.error.issues.map(issue => [String(issue.path[0]), issue.message]))
    return
  }
  errors.value = {}
  emit('save', parsed.data)
}
</script>

<template>
  <form class="flex flex-col gap-2 rounded-[10px] border-[1.5px] border-dashed border-ink-400 p-3" :aria-label="initial ? t.profile.entry.edit(initial.name) : t.profile.certificationForm.add" @submit.prevent="submit">
    <UFormField :label="t.profile.certificationForm.name" :error="errors.name" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.name" :placeholder="t.profile.certificationForm.namePlaceholder" maxlength="120" class="w-full" />
    </UFormField>
    <UFormField :label="t.profile.certificationForm.issuer" :ui="{ label: 'sr-only' }">
      <UInput v-model="draft.issuer" :placeholder="t.profile.certificationForm.issuerPlaceholder" maxlength="120" class="w-full" />
    </UFormField>
    <div class="grid grid-cols-2 gap-2">
      <UFormField :label="t.profile.certificationForm.issued" :error="errors.issued" :ui="label">
        <UInput v-model="draft.issued" type="month" :placeholder="t.profile.monthRange.placeholder" class="w-full" />
      </UFormField>
      <UFormField :label="t.profile.certificationForm.expires" :error="errors.expires" :ui="label">
        <UInput v-model="draft.expires" type="month" :placeholder="t.profile.monthRange.placeholder" class="w-full" />
      </UFormField>
    </div>
    <div class="grid grid-cols-2 gap-2">
      <UFormField :label="t.profile.certificationForm.credentialId" :ui="{ label: 'sr-only' }">
        <UInput v-model="draft.credential_id" :placeholder="t.profile.certificationForm.credentialId" maxlength="120" class="w-full" />
      </UFormField>
      <UFormField :label="t.profile.certificationForm.credentialLink" :error="errors.url" :ui="{ label: 'sr-only' }">
        <UInput v-model="draft.url" :placeholder="t.profile.certificationForm.credentialLinkPlaceholder" maxlength="200" class="w-full" />
      </UFormField>
    </div>
    <p v-if="serverError" role="alert" class="m-0 text-sm font-bold text-error">{{ serverError }}</p>
    <div class="flex gap-2">
      <UButton type="submit" color="neutral" :label="t.common.actions.save" block class="flex-1" :loading="pending" />
      <UButton color="neutral" variant="outline" :label="t.common.actions.cancel" @click="emit('cancel')" />
    </div>
  </form>
</template>
