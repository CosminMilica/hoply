<script setup lang="ts">
import { ref } from 'vue'
import { t } from '@/features/common/i18n'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import { experienceInput, type Experience, type Profile } from '../api/profile.schema'
import { useEntryEditor } from '../composables/useEntryEditor'
import { useProfileMutations } from '../composables/useProfile'
import { monthRange } from '../format'
import BulletList from './BulletList.vue'
import EntryActions from './EntryActions.vue'
import ExperienceForm from './ExperienceForm.vue'
import NotesList from './NotesList.vue'

const props = defineProps<{ profile: Profile }>()
const { experience } = useProfileMutations()

function removeNote(role: Experience, index: number) {
  experience.update.mutate({ id: role.id, body: experienceInput(role, { notes: role.notes.filter((_, at) => at !== index) }) })
}
const editor = useEntryEditor(experience)
const expanded = ref<Record<string, boolean>>(props.profile.experience[0] ? { [props.profile.experience[0].id]: true } : {})

const details = (role: Profile['experience'][number]) => [
  (t.value.profile.employmentTypes as Record<string, string>)[role.employment_type],
  role.industry,
  role.location,
  role.team_size !== null ? t.value.profile.experience.teamOf(role.team_size) : '',
  role.people_managed ? t.value.profile.experience.managed(role.people_managed) : '',
].filter(Boolean).join(' · ')
</script>

<template>
  <p class="m-0 rounded-[10px] bg-accented px-3 py-2.5 text-[13px] leading-snug font-medium text-toned">
    {{ t.profile.experience.intro }}
  </p>

  <UCollapsible
    v-for="role in profile.experience"
    :key="role.id"
    :open="Boolean(expanded[role.id])"
    class="rounded-[10px] border border-default bg-elevated px-3 py-1"
    @update:open="(value: boolean) => (expanded = { ...expanded, [role.id]: value })"
  >
    <button type="button" class="flex min-h-13 w-full items-center gap-2.5 text-left text-highlighted">
      <span class="flex min-w-0 flex-1 flex-col">
        <span class="text-[15px] font-extrabold">{{ role.role }}</span>
        <span class="text-[13px] font-medium text-muted">{{ [role.company, monthRange(role.start, role.end) || t.profile.experience.datesNotSet].filter(Boolean).join(' · ') }}</span>
      </span>
      <span class="rounded-lg bg-accented px-2.5 py-1 text-xs font-extrabold">{{ t.profile.experience.highlights(role.bullets.length) }}</span>
    </button>
    <template #content>
      <div class="flex flex-col gap-2 pb-3">
        <ExperienceForm
          v-if="editor.editing.value === role.id"
          :initial="role"
          :pending="experience.update.isPending.value"
          :server-error="editor.serverError.value"
          @save="input => editor.update(role.id, input)"
          @cancel="editor.close"
        />
        <template v-else>
          <p v-if="details(role)" class="m-0 text-[13px] font-medium text-toned">{{ details(role) }}</p>
          <template v-if="role.bullets.length">
            <SectionLabel as="h3">{{ t.profile.experience.whatYouDid }}</SectionLabel>
            <BulletList :bullets="role.bullets" />
          </template>
          <template v-if="role.tools.length">
            <SectionLabel as="h3">{{ t.profile.experience.toolsYouUsed }}</SectionLabel>
            <ul class="flex flex-wrap gap-1.5">
              <li v-for="tool in role.tools" :key="tool" class="rounded-lg border border-ink-700 bg-elevated px-2.5 py-1 text-[13px] font-bold">{{ tool }}</li>
            </ul>
          </template>
          <NotesList :notes="role.notes" :of="role.role" :pending="experience.update.isPending.value" @remove="index => removeNote(role, index)" />
          <p v-if="!role.bullets.length && !role.tools.length" class="m-0 text-[13px] font-medium text-muted">
            {{ t.profile.experience.noHighlights }}
          </p>
          <EntryActions :name="role.role" what="role" :pending="experience.remove.isPending.value" @edit="editor.open(role.id)" @remove="experience.remove.mutate(role.id)" />
        </template>
      </div>
    </template>
  </UCollapsible>

  <ExperienceForm
    v-if="editor.adding.value"
    :pending="experience.add.isPending.value"
    :server-error="editor.serverError.value"
    @save="input => editor.add(input, role => (expanded = { ...expanded, [role.id]: true }))"
    @cancel="editor.close"
  />
  <UButton
    v-else
    color="neutral"
    variant="outline"
    :label="t.profile.experience.add"
    block
    class="border-[1.5px] border-dashed border-ink-400 text-toned ring-0"
    @click="editor.open(null)"
  />
</template>
