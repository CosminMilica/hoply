/** The languages a CV is written in (cvdoc.English, cvdoc.Romanian). */
export const CV_LANGUAGES = [{ id: 'en', label: 'English' }, { id: 'ro', label: 'Română' }] as const
export type CvLanguage = (typeof CV_LANGUAGES)[number]['id']

/** A CV's section titles in its language (cvdoc.HeadingsFor). */
export const HEADINGS: Record<CvLanguage, Record<'summary' | 'skills' | 'experience' | 'projects' | 'education' | 'certifications' | 'languages' | 'additional', string>> = {
  en: { summary: 'Summary', skills: 'Skills', experience: 'Experience', projects: 'Projects', education: 'Education', certifications: 'Certifications', languages: 'Languages', additional: 'Additional information' },
  ro: { summary: 'Profil', skills: 'Competențe', experience: 'Experiență profesională', projects: 'Proiecte', education: 'Educație', certifications: 'Certificări', languages: 'Limbi străine', additional: 'Informații suplimentare' },
}

const ROMANIAN = /[ăâîșțşţ]|\b(?:și|si|pentru|cu|în|din|care|experiență|experienta|cerințe|cerinte|responsabilități|oferim|ani|lucru)\b/gi
const ENGLISH = /\b(?:the|and|with|for|you|we|our|will|experience|requirements|responsibilities|years)\b/gi

/** The language an ad is written in: Romanian when its words read as Romanian more than English. */
export function adLanguage(text: string): CvLanguage {
  const romanian = text.match(ROMANIAN)?.length ?? 0
  const english = text.match(ENGLISH)?.length ?? 0
  return romanian > english ? 'ro' : 'en'
}
