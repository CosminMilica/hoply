<script setup lang="ts">
import type { CvDocument } from '../api/cv.schema'
import HighlightedText from './HighlightedText.vue'

defineProps<{ document: CvDocument, highlights: string[] }>()
</script>

<template>
  <article
    data-print-root
    aria-label="CV preview"
    class="flex flex-col gap-3 rounded-xl border border-[#d8d6ce] bg-white p-5 font-[Arial,Helvetica,sans-serif] text-[12.5px] leading-normal text-[#111111]"
  >
    <header class="flex flex-col gap-0.5">
      <span v-if="document.name" class="text-xl font-bold" data-test="cv-name">{{ document.name }}</span>
      <span :class="document.name ? 'text-[13px] font-bold' : 'text-xl font-bold'">{{ document.headline }}</span>
      <span v-if="document.contact.length">{{ document.contact.join(' · ') }}</span>
      <span v-if="document.links.length">{{ document.links.join(' · ') }}</span>
    </header>
    <section class="flex flex-col gap-1">
      <h2 class="cv-heading">Summary</h2>
      <p class="m-0"><HighlightedText :text="document.summary" :keywords="highlights" /></p>
    </section>
    <section class="flex flex-col gap-1">
      <h2 class="cv-heading">Skills</h2>
      <p class="m-0"><HighlightedText :text="document.skills.join(', ')" :keywords="highlights" /></p>
    </section>
    <section class="flex flex-col gap-2.5">
      <h2 class="cv-heading">Experience</h2>
      <div v-for="(role, index) in document.experience" :key="index" class="flex flex-col gap-[3px]">
        <h3 class="m-0 text-[12.5px] font-bold">{{ role.role }}</h3>
        <span>{{ role.meta }}</span>
        <ul class="flex flex-col gap-[3px]">
          <li v-for="(bullet, at) in role.bullets" :key="at" class="flex gap-2">
            <span aria-hidden="true">•</span><HighlightedText :text="bullet" :keywords="highlights" />
          </li>
        </ul>
      </div>
    </section>
    <section v-if="document.projects.length" class="flex flex-col gap-2.5">
      <h2 class="cv-heading">Projects</h2>
      <div v-for="(project, index) in document.projects" :key="index" class="flex flex-col gap-[3px]">
        <h3 class="m-0 text-[12.5px] font-bold">{{ project.name }}</h3>
        <span v-if="project.meta">{{ project.meta }}</span>
        <ul class="flex flex-col gap-[3px]">
          <li v-for="(bullet, at) in project.bullets" :key="at" class="flex gap-2">
            <span aria-hidden="true">•</span><HighlightedText :text="bullet" :keywords="highlights" />
          </li>
        </ul>
      </div>
    </section>
    <section class="flex flex-col gap-1">
      <h2 class="cv-heading">Education</h2>
      <div v-for="(item, index) in document.education" :key="index" class="flex flex-col">
        <span class="font-bold">{{ item.degree }}</span>
        <span>{{ item.meta }}</span>
      </div>
    </section>
    <section v-if="document.certifications.length" class="flex flex-col gap-1">
      <h2 class="cv-heading">Certifications</h2>
      <ul class="flex flex-col">
        <li v-for="(item, index) in document.certifications" :key="index" class="flex gap-2"><span aria-hidden="true">•</span><span>{{ item }}</span></li>
      </ul>
    </section>
    <section class="flex flex-col gap-1">
      <h2 class="cv-heading">Languages</h2>
      <p class="m-0">{{ document.languages }}</p>
    </section>
    <section v-if="document.additional.length" class="flex flex-col gap-1">
      <h2 class="cv-heading">Additional information</h2>
      <ul class="flex flex-col">
        <li v-for="(item, index) in document.additional" :key="index" class="flex gap-2"><span aria-hidden="true">•</span><span>{{ item }}</span></li>
      </ul>
    </section>
  </article>
</template>

<style scoped>
.cv-heading {
  margin: 0;
  border-bottom: 1px solid #111111;
  padding-bottom: 2px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

@media print {
  :deep(mark) {
    background: none;
    color: inherit;
    font-weight: inherit;
    padding: 0;
  }
}
</style>
