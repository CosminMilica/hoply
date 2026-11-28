<script setup lang="ts">
import { computed, reactive } from 'vue'
import { t } from '@/features/common/i18n'
import { useProfileQuery } from '../composables/useProfile'
import AboutFields from '../components/AboutFields.vue'
import AdditionalInfoEditor from '../components/AdditionalInfoEditor.vue'
import CertificationsEditor from '../components/CertificationsEditor.vue'
import ContactFields from '../components/ContactFields.vue'
import EducationEditor from '../components/EducationEditor.vue'
import ExperienceEditor from '../components/ExperienceEditor.vue'
import LanguagesEditor from '../components/LanguagesEditor.vue'
import PreferencesFields from '../components/PreferencesFields.vue'
import ProfileSection from '../components/ProfileSection.vue'
import ProjectsEditor from '../components/ProjectsEditor.vue'
import ReadinessCard from '../components/ReadinessCard.vue'
import ProfileProbeCard from '@/features/cv/components/ProfileProbeCard.vue'
import SkillsEditor from '../components/SkillsEditor.vue'

const { data: profile, isPending, isError, refetch } = useProfileQuery()
const open = reactive({
  about: false, contact: false, preferences: false, skills: true, additional: false,
  experience: true, projects: false, education: false, certifications: false, languages: false,
})

const summaries = computed(() => {
  const p = profile.value
  if (!p) return null
  const filled = (value: string) => value.trim().length > 0
  const words = t.value.profile.page
  return {
    about: filled(p.about) && filled(p.headline) ? words.done : words.toDo,
    contact: `${[p.email, p.phone, p.github, p.linkedin, p.city, p.country].filter(filled).length}/6`,
    preferences: p.work_modes.length || filled(p.work_authorization) || p.target_roles.length ? words.set : words.toDo,
    projects: words.projects(p.projects.length),
    certifications: words.certifications(p.certifications.length),
    additional: words.lines(p.additional_info.length),
    skills: words.skills(p.skills.length),
    experience: words.roles(p.experience.length),
    education: words.degrees(p.education.length),
    languages: words.languages(p.languages.length),
  }
})
</script>

<template>
  <div class="mx-auto flex w-full max-w-[1080px] flex-col gap-3.5">
    <div v-if="isPending" class="flex flex-col gap-3" aria-busy="true">
      <USkeleton class="h-40 w-full rounded-2xl" />
      <USkeleton class="h-13 w-full" />
      <USkeleton class="h-13 w-full" />
    </div>
    <UAlert
      v-else-if="isError || !profile"
      color="error"
      variant="subtle"
      :title="t.profile.page.loadError"
      :description="t.profile.page.loadErrorHint"
      :actions="[{ label: t.common.actions.retry, color: 'neutral', onClick: () => refetch() }]"
    />
    <template v-else-if="summaries">
      <ReadinessCard :profile="profile" />
      <ProfileProbeCard :profile="profile" />
      <div class="flex flex-col gap-3.5 min-[900px]:grid min-[900px]:grid-cols-2 min-[900px]:items-start min-[900px]:gap-x-12">
        <div class="flex flex-col">
          <ProfileSection v-model:open="open.about" :title="t.profile.page.sections.about" icon="i-lucide-pencil" :summary="summaries.about">
            <AboutFields :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.contact" :title="t.profile.page.sections.contact" icon="i-lucide-contact" :summary="summaries.contact">
            <ContactFields :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.preferences" :title="t.profile.page.sections.preferences" icon="i-lucide-sliders-horizontal" :summary="summaries.preferences">
            <PreferencesFields :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.skills" :title="t.profile.page.sections.skills" icon="i-lucide-sparkle" :summary="summaries.skills">
            <SkillsEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.additional" :title="t.profile.page.sections.additional" icon="i-lucide-list-plus" :summary="summaries.additional">
            <AdditionalInfoEditor :profile="profile" />
          </ProfileSection>
        </div>
        <div class="flex flex-col">
          <ProfileSection v-model:open="open.experience" :title="t.profile.page.sections.experience" icon="i-lucide-briefcase" :summary="summaries.experience">
            <ExperienceEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.projects" :title="t.profile.page.sections.projects" icon="i-lucide-folder-git-2" :summary="summaries.projects">
            <ProjectsEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.education" :title="t.profile.page.sections.education" icon="i-lucide-graduation-cap" :summary="summaries.education">
            <EducationEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.certifications" :title="t.profile.page.sections.certifications" icon="i-lucide-badge-check" :summary="summaries.certifications">
            <CertificationsEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.languages" :title="t.profile.page.sections.languages" icon="i-lucide-globe" :summary="summaries.languages">
            <LanguagesEditor :profile="profile" />
          </ProfileSection>
        </div>
      </div>
      <UButton :to="{ name: 'crashr-job-add' }" :label="t.profile.page.addJob" size="lg" block class="justify-center min-[900px]:hidden" />
    </template>
  </div>
</template>
