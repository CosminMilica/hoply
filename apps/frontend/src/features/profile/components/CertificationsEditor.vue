<script setup lang="ts">
import { t } from '@/features/common/i18n'
import type { Profile } from '../api/profile.schema'
import { useEntryEditor } from '../composables/useEntryEditor'
import { useProfileMutations } from '../composables/useProfile'
import { monthLabel } from '../format'
import CertificationForm from './CertificationForm.vue'

defineProps<{ profile: Profile }>()
const { certifications } = useProfileMutations()
const editor = useEntryEditor(certifications)

const meta = (item: Profile['certifications'][number]) => [
  item.issuer,
  item.issued ? t.value.profile.certifications.issued(monthLabel(item.issued)) : '',
  item.expires ? t.value.profile.certifications.expires(monthLabel(item.expires)) : '',
  item.credential_id ? t.value.profile.certifications.id(item.credential_id) : '',
].filter(Boolean).join(' · ')
</script>

<template>
  <ul v-if="profile.certifications.length" class="flex flex-col gap-2.5">
    <li v-for="item in profile.certifications" :key="item.id">
      <CertificationForm
        v-if="editor.editing.value === item.id"
        :initial="item"
        :pending="certifications.update.isPending.value"
        :server-error="editor.serverError.value"
        @save="input => editor.update(item.id, input)"
        @cancel="editor.close"
      />
      <div v-else class="flex items-center gap-1 rounded-[10px] border border-default bg-elevated py-1 ps-3 pe-1">
        <span class="flex min-w-0 flex-1 flex-col">
          <span class="text-[15px] font-extrabold">{{ item.name }}</span>
          <span class="text-[13px] font-medium text-muted">{{ meta(item) }}</span>
        </span>
        <UButton color="neutral" variant="ghost" icon="i-lucide-pencil" :aria-label="t.profile.entry.edit(item.name)" class="size-9 justify-center text-muted" @click="editor.open(item.id)" />
        <UButton color="neutral" variant="ghost" icon="i-lucide-x" :aria-label="t.profile.entry.remove(item.name)" class="size-9 justify-center text-muted" @click="certifications.remove.mutate(item.id)" />
      </div>
    </li>
  </ul>
  <CertificationForm v-if="editor.adding.value" :pending="certifications.add.isPending.value" :server-error="editor.serverError.value" @save="input => editor.add(input)" @cancel="editor.close" />
  <UButton
    v-else
    color="neutral"
    variant="outline"
    :label="t.profile.certifications.add"
    block
    class="border-[1.5px] border-dashed border-ink-400 text-toned ring-0"
    @click="editor.open(null)"
  />
</template>
