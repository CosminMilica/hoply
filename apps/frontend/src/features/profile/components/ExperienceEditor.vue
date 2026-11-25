<script setup lang="ts">
import { ref } from 'vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import { EMPLOYMENT_TYPES, type Profile } from '../api/profile.schema'
import { useEntryEditor } from '../composables/useEntryEditor'
import { useProfileMutations } from '../composables/useProfile'
import { monthRange, plural } from '../format'
import BulletList from './BulletList.vue'
import EntryActions from './EntryActions.vue'
import ExperienceForm from './ExperienceForm.vue'

const props = defineProps<{ profile: Profile }>()
const { experience } = useProfileMutations()
const editor = useEntryEditor(experience)
const expanded = ref<Record<string, boolean>>(props.profile.experience[0] ? { [props.profile.experience[0].id]: true } : {})

const details = (role: Profile['experience'][number]) => [
  EMPLOYMENT_TYPES[role.employment_type as keyof typeof EMPLOYMENT_TYPES],
  role.industry,
  role.location,
  role.team_size !== null ? `team of ${role.team_size}` : '',
  role.people_managed ? `managed ${plural(role.people_managed, 'person', 'people')}` : '',
].filter(Boolean).join(' · ')
</script>

<template>
  <p class="m-0 rounded-[10px] bg-accented px-3 py-2.5 text-[13px] leading-snug font-medium text-toned">
    Keep it simple: what you did, what came of it, and which tools you used. For every job you apply to, we rewrite it with that job's keywords.
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
        <span class="text-[13px] font-medium text-muted">{{ [role.company, monthRange(role.start, role.end) || 'dates not set'].filter(Boolean).join(' · ') }}</span>
      </span>
      <span class="rounded-lg bg-accented px-2.5 py-1 text-xs font-extrabold">{{ plural(role.bullets.length, 'highlight') }}</span>
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
            <SectionLabel as="h3">What you did</SectionLabel>
            <BulletList :bullets="role.bullets" />
          </template>
          <template v-if="role.tools.length">
            <SectionLabel as="h3">Tools you used</SectionLabel>
            <ul class="flex flex-wrap gap-1.5">
              <li v-for="tool in role.tools" :key="tool" class="rounded-lg border border-ink-700 bg-elevated px-2.5 py-1 text-[13px] font-bold">{{ tool }}</li>
            </ul>
          </template>
          <p v-if="!role.bullets.length && !role.tools.length" class="m-0 text-[13px] font-medium text-muted">
            No highlights yet. Edit the role to add what you did.
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
    label="+ Add a role"
    block
    class="border-[1.5px] border-dashed border-ink-400 text-toned ring-0"
    @click="editor.open(null)"
  />
</template>
