<script setup lang="ts">
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { Profile } from '../api/profile.schema'
import { useEntryEditor } from '../composables/useEntryEditor'
import { useProfileMutations } from '../composables/useProfile'
import { monthRange } from '../format'
import BulletList from './BulletList.vue'
import EntryActions from './EntryActions.vue'
import ProjectForm from './ProjectForm.vue'

defineProps<{ profile: Profile }>()
const { projects } = useProfileMutations()
const editor = useEntryEditor(projects)
</script>

<template>
  <p v-if="!profile.projects.length && !editor.adding.value" class="m-0 text-[13px] font-medium text-muted">
    Side projects, open source or a home lab count too.
  </p>
  <div v-for="project in profile.projects" :key="project.id" class="flex flex-col gap-2 rounded-[10px] border border-default bg-elevated px-3 py-2.5">
    <ProjectForm
      v-if="editor.editing.value === project.id"
      :initial="project"
      :pending="projects.update.isPending.value"
      :server-error="editor.serverError.value"
      @save="input => editor.update(project.id, input)"
      @cancel="editor.close"
    />
    <template v-else>
      <div class="flex flex-col">
        <h3 class="m-0 text-[15px] font-extrabold">{{ project.name }}</h3>
        <span class="text-[13px] font-medium text-muted">{{ [project.role, monthRange(project.start, project.end)].filter(Boolean).join(' · ') }}</span>
        <a v-if="project.url" :href="project.url.startsWith('http') ? project.url : `https://${project.url}`" target="_blank" rel="noopener" class="text-[13px] font-bold text-hoply-300 underline">{{ project.url }}</a>
      </div>
      <BulletList v-if="project.bullets.length" :bullets="project.bullets" />
      <template v-if="project.tools.length">
        <SectionLabel as="h4">Tools</SectionLabel>
        <ul class="flex flex-wrap gap-1.5">
          <li v-for="tool in project.tools" :key="tool" class="rounded-lg border border-ink-700 bg-elevated px-2.5 py-1 text-[13px] font-bold">{{ tool }}</li>
        </ul>
      </template>
      <EntryActions :name="project.name" what="project" :pending="projects.remove.isPending.value" @edit="editor.open(project.id)" @remove="projects.remove.mutate(project.id)" />
    </template>
  </div>
  <ProjectForm v-if="editor.adding.value" :pending="projects.add.isPending.value" :server-error="editor.serverError.value" @save="input => editor.add(input)" @cancel="editor.close" />
  <UButton
    v-else
    color="neutral"
    variant="outline"
    label="+ Add a project"
    block
    class="border-[1.5px] border-dashed border-ink-400 text-toned ring-0"
    @click="editor.open(null)"
  />
</template>
