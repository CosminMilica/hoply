<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useProfileQuery } from '../composables/useProfile'
import { plural } from '../format'
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
  return {
    about: filled(p.about) && filled(p.headline) ? 'Done' : 'To do',
    contact: `${[p.email, p.phone, p.github, p.linkedin, p.city, p.country].filter(filled).length}/6`,
    preferences: p.work_modes.length || filled(p.work_authorization) || p.target_roles.length ? 'Set' : 'To do',
    projects: plural(p.projects.length, 'project'),
    certifications: plural(p.certifications.length, 'certification'),
    additional: plural(p.additional_info.length, 'line'),
    skills: plural(p.skills.length, 'skill'),
    experience: plural(p.experience.length, 'role'),
    education: plural(p.education.length, 'degree'),
    languages: plural(p.languages.length, 'language'),
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
      title="We could not load your profile"
      description="Check that the profile service is running, then try again."
      :actions="[{ label: 'Try again', color: 'neutral', onClick: () => refetch() }]"
    />
    <template v-else-if="summaries">
      <ReadinessCard :profile="profile" />
      <ProfileProbeCard :profile="profile" />
      <div class="flex flex-col gap-3.5 min-[900px]:grid min-[900px]:grid-cols-2 min-[900px]:items-start min-[900px]:gap-x-12">
        <div class="flex flex-col">
          <ProfileSection v-model:open="open.about" title="About you" icon="i-lucide-pencil" :summary="summaries.about">
            <AboutFields :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.contact" title="Contact data" icon="i-lucide-contact" :summary="summaries.contact">
            <ContactFields :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.preferences" title="Work preferences" icon="i-lucide-sliders-horizontal" :summary="summaries.preferences">
            <PreferencesFields :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.skills" title="Skills" icon="i-lucide-sparkle" :summary="summaries.skills">
            <SkillsEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.additional" title="Additional information" icon="i-lucide-list-plus" :summary="summaries.additional">
            <AdditionalInfoEditor :profile="profile" />
          </ProfileSection>
        </div>
        <div class="flex flex-col">
          <ProfileSection v-model:open="open.experience" title="Experience" icon="i-lucide-briefcase" :summary="summaries.experience">
            <ExperienceEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.projects" title="Projects" icon="i-lucide-folder-git-2" :summary="summaries.projects">
            <ProjectsEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.education" title="Education" icon="i-lucide-graduation-cap" :summary="summaries.education">
            <EducationEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.certifications" title="Certifications" icon="i-lucide-badge-check" :summary="summaries.certifications">
            <CertificationsEditor :profile="profile" />
          </ProfileSection>
          <ProfileSection v-model:open="open.languages" title="Languages" icon="i-lucide-globe" :summary="summaries.languages">
            <LanguagesEditor :profile="profile" />
          </ProfileSection>
        </div>
      </div>
      <UButton :to="{ name: 'hoply-job-add' }" label="Add a job" size="lg" block class="justify-center min-[900px]:hidden" />
    </template>
  </div>
</template>
