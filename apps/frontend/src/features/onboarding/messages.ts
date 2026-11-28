import { countEn, countRo } from '@/features/common/i18n/format'

/** The onboarding and preferences words, in each language the app speaks. */
export const en = {
  stepOf: (step: number, total: number) => `Step ${step} of ${total}`,
  welcome: 'Welcome to crashr',
  languageIntro: 'Pick the language of the app. We use it in menus and messages, and as the first language of your CVs and letters.',
  languageLater: 'You can change it any time in Preferences, in your account menu.',
  languageGroup: 'App language',
  /** Each language's line, written in that language. */
  languageHints: { en: 'Menus, messages and the default CV in English.', ro: 'Meniuri, mesaje și CV-ul implicit în română.' },
  domainsTitle: 'Which fields interest you?',
  domainsIntro: 'Pick one or more. Find jobs shows you the jobs in the fields you pick, each with its own filters.',
  domainsGroup: 'Fields of work',
  filters: 'Filters',
  picked: (n: number) => (n === 0 ? 'Pick at least one field' : `${countEn(n, 'field', 'fields')} picked`),
  finish: 'Done, show me the jobs',
  saveFailed: 'We could not save your choices. Try again.',
  loadFailed: 'We could not load your profile.',
  preferences: {
    title: 'Preferences',
    intro: 'The language of the app and the fields whose jobs Find jobs shows you.',
    atLeastOne: 'Keep at least one field.',
  },
}

export const ro: typeof en = {
  stepOf: (step: number, total: number) => `Pasul ${step} din ${total}`,
  welcome: 'Bine ai venit în crashr',
  languageIntro: 'Alege limba aplicației. O folosim în meniuri și mesaje, și ca limbă implicită pentru CV-uri și scrisori.',
  languageLater: 'Poți schimba limba oricând din Preferințe, în meniul contului.',
  languageGroup: 'Limba aplicației',
  languageHints: { en: 'Menus, messages and the default CV in English.', ro: 'Meniuri, mesaje și CV-ul implicit în română.' },
  domainsTitle: 'Ce domenii te interesează?',
  domainsIntro: 'Alege unul sau mai multe. La Caută joburi vei vedea doar joburile din domeniile alese, fiecare cu filtrele lui.',
  domainsGroup: 'Domenii',
  filters: 'Filtre',
  picked: (n: number) => (n === 0 ? 'Alege cel puțin un domeniu' : n === 1 ? '1 domeniu ales' : `${countRo(n, 'domeniu', 'domenii')} alese`),
  finish: 'Gata, arată-mi joburile',
  saveFailed: 'Nu am putut salva alegerile. Încearcă din nou.',
  loadFailed: 'Nu am putut încărca profilul.',
  preferences: {
    title: 'Preferințe',
    intro: 'Limba aplicației și domeniile din care îți arătăm joburi la Caută joburi.',
    atLeastOne: 'Păstrează cel puțin un domeniu.',
  },
}
