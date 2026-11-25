import type { z } from 'zod'
import { getApiClient } from '@/shared/api/client'
import {
  certificationSchema,
  educationSchema,
  experienceSchema,
  languageSchema,
  profileSchema,
  projectSchema,
  type CertificationInput,
  type EducationInput,
  type ExperienceInput,
  type LanguageInput,
  type ProfilePatch,
  type ProjectInput,
} from './profile.schema'

const ME = '/api/profile/v1/profiles/me'

/** POST, PUT and DELETE of one of the profile's lists. */
function entries<S extends z.ZodType, Input>(path: string, schema: S) {
  const at = (id: string) => `${ME}/${path}/${encodeURIComponent(id)}`
  return {
    async add(body: Input): Promise<z.output<S>> {
      return schema.parse(await getApiClient().post(`${ME}/${path}`, { json: body }).json())
    },
    async update(id: string, body: Input): Promise<z.output<S>> {
      return schema.parse(await getApiClient().put(at(id), { json: body }).json())
    },
    async remove(id: string) {
      await getApiClient().delete(at(id))
    },
  }
}

export const profileApi = {
  async get() {
    return profileSchema.parse(await getApiClient().get(ME).json())
  },
  async patch(body: ProfilePatch) {
    return profileSchema.parse(await getApiClient().patch(ME, { json: body }).json())
  },
  async putSkills(skills: string[]) {
    return profileSchema.parse(await getApiClient().put(`${ME}/skills`, { json: { skills } }).json())
  },
  experience: entries<typeof experienceSchema, ExperienceInput>('experience', experienceSchema),
  projects: entries<typeof projectSchema, ProjectInput>('projects', projectSchema),
  education: entries<typeof educationSchema, EducationInput>('education', educationSchema),
  certifications: entries<typeof certificationSchema, CertificationInput>('certifications', certificationSchema),
  languages: {
    async add(body: LanguageInput) {
      return languageSchema.parse(await getApiClient().post(`${ME}/languages`, { json: body }).json())
    },
    async remove(id: string) {
      await getApiClient().delete(`${ME}/languages/${encodeURIComponent(id)}`)
    },
  },
}
