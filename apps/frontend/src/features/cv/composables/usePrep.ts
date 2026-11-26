import { computed, reactive, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import type { Job } from '@/features/jobs/api/jobs.schema'
import { keywordsIn, matchJob, requirementsOf } from '@/features/jobs/match'
import type { Bullet, Profile } from '@/features/profile/api/profile.schema'
import { monthRange } from '@/features/profile/format'
import { bulletLine, knownSkills } from '@/features/profile/skills'
import type { CvRequest, PrepQuestions, ProbeQuestion } from '../api/cv.schema'
import { useAnswers } from './useAnswers'
import { OTHER, useNotes, type Place } from './useNotes'

/** The keys of a page's answers: a gap by its skill, a responsibility by its index, one of Claude's questions by its. */
export const gapKey = (skill: string) => `g:${skill}`
export const claimKey = (index: number) => `c:${index}`
export const probeKey = (index: number) => `p:${index}`

const joinList = (items: string[]) =>
  items.length <= 1 ? items.join('') : `${items.slice(0, -1).join(', ')} and ${items[items.length - 1]}`

interface PrepInputs {
  job: MaybeRefOrGetter<Job | undefined>
  profile: MaybeRefOrGetter<Profile | undefined>
  questions: MaybeRefOrGetter<PrepQuestions | undefined>
  /** Claude's questions for the job, once ready. */
  probe?: MaybeRefOrGetter<ProbeQuestion[] | undefined>
}

/**
 * Prepare your CV for one job. The facts the person confirms are saved in their profile, where every
 * later CV finds them: a highlight under the role or project they pick, or a line of additional
 * information. What stays with this CV is the roles to feature, the skills up front and the
 * responsibilities confirmed.
 */
export type Prep = ReturnType<typeof usePrep>

export function usePrep({ job, profile, questions, probe }: PrepInputs) {
  const notes = useNotes(profile)
  const { experience, projects, entryOf, updateEntry, whereLabel, mutations } = notes
  const answers = useAnswers(notes)
  const state = reactive({
    skip: {} as Record<string, boolean>,
    focus: null as string[] | null,
    places: {} as Record<string, Place>,
  })

  const current = () => toValue(profile)
  const keywords = computed(() => { const current = toValue(job); return current ? requirementsOf(current) : [] })
  const known = computed(() => (current() ? knownSkills(current()!) : []))
  const match = computed(() => matchJob(toValue(job) ?? { must_have: [], nice_to_have: [] }, known.value))
  const includedRoles = computed(() => experience.value.filter(role => !state.skip[role.id]))

  const choices = computed(() => [
    ...includedRoles.value.map(role => ({ id: `experience:${role.id}`, label: role.company || role.role, where: `${role.role}${role.company ? ` at ${role.company}` : ''}` })),
    ...projects.value.map(project => ({ id: `projects:${project.id}`, label: project.name, where: `the ${project.name} project` })),
  ])
  function whereOf(key: string): Place {
    const chosen = state.places[key]
    if (chosen === OTHER || choices.value.some(choice => choice.id === chosen)) return chosen!
    return choices.value[0]?.id ?? OTHER
  }
  function roleChoices(key: string) {
    const where = whereOf(key)
    return [...choices.value, { id: OTHER, label: 'Elsewhere' }].map(choice => ({ id: choice.id, label: choice.label, pressed: choice.id === where }))
  }
  function pickRole(key: string, id: Place) {
    state.places[key] = id
  }

  const addGap = (skill: string) => answers.add(gapKey(skill), whereOf(gapKey(skill)), () => [skill])
  /** An answer to one of Claude's questions becomes a note of the role or project it is about. */
  const answerProbe = (index: number, question: ProbeQuestion) =>
    answers.add(probeKey(index), question.place, text => keywordsIn(text, keywords.value))
  const claimYes = (index: number) => answers.add(claimKey(index), whereOf(claimKey(index)), text => keywordsIn(text, keywords.value))

  /** The person's own highlights that share one of the job's keywords with a responsibility. */
  function suggestionsFor(responsibility: string) {
    const shared = keywordsIn(responsibility, keywords.value).map(keyword => keyword.toLowerCase())
    if (!shared.length) return []
    const entries: { label: string, bullets: Bullet[] }[] = [
      ...experience.value.map(role => ({ label: role.company || role.role, bullets: role.bullets })),
      ...projects.value.map(project => ({ label: project.name, bullets: project.bullets })),
    ]
    return entries
      .flatMap(entry => entry.bullets.map(bullet => ({ label: entry.label, bullet })))
      .filter(({ bullet }) => bullet.skills.some(skill => shared.includes(skill.toLowerCase()))
        || keywordsIn(bulletLine(bullet), shared).length > 0)
      .slice(0, 3)
      .map(({ label, bullet }) => ({ text: bulletLine(bullet), label }))
  }

  const baseScore = computed(() => toValue(questions)?.base_score ?? 0)
  const matchLabel = computed(() => (match.value.score !== baseScore.value ? `${baseScore.value}% → ${match.value.score}%` : `${match.value.score}%`))
  const progress = computed(() => {
    const gaps = toValue(questions)?.gaps ?? []
    const asked = toValue(probe) ?? []
    // Claude's questions cover the job's responsibilities; the rules' questions about them are asked only without Claude.
    const claims = asked.length ? [] : (toValue(questions)?.responsibilities ?? [])
    const answered = [...gaps.map(gap => gapKey(gap.skill)), ...claims.map(claim => claimKey(claim.index)), ...asked.map((_, index) => probeKey(index))]
      .filter(key => answers.answer(key)).length
    return `${answered} of ${gaps.length + claims.length + asked.length} questions answered`
  })

  // The highlights to strengthen are chosen once, so that a saved result does not move them.
  const impactRows = ref<{ key: string, roleId: string, index: number, bullet: string }[] | null>(null)
  watch(() => includedRoles.value[0], (role) => {
    if (impactRows.value || !current()) return
    impactRows.value = (role?.bullets ?? [])
      .map((bullet, index) => ({ key: `${role!.id}-${index}`, roleId: role!.id, index, bullet: bullet.text, result: bullet.result }))
      .filter(row => !/\d/.test(row.bullet) && !row.result.trim())
      .slice(0, 3)
      .map(({ result: _result, ...row }) => row)
  }, { immediate: true })
  async function saveImpact(key: string, value: string) {
    const row = impactRows.value?.find(item => item.key === key)
    const target = row && entryOf(`experience:${row.roleId}`)
    const bullet = target?.entry.bullets[row!.index]
    if (!target || !bullet || bullet.text !== row!.bullet || bullet.result === value.trim()) return
    await answers.saving(`i:${key}`, () => updateEntry(`experience:${row!.roleId}`, entry => ({
      bullets: entry.bullets.map((item, index) => (index === row!.index && item.text === row!.bullet ? { ...item, result: value.trim() } : item)),
    })))
  }

  const note = ref('')
  watch(() => current()?.additional_info.join('\n') ?? '', (value) => { note.value = value }, { immediate: true })
  async function saveNote() {
    const lines = note.value.split('\n').map(line => line.trim()).filter(Boolean).slice(0, 10)
    if (lines.join('\n') === (current()?.additional_info ?? []).join('\n')) return
    await answers.saving('note', async () => {
      await mutations.patch.mutateAsync({ additional_info: lines })
    })
  }

  const roles = computed(() => experience.value.map((role) => {
    const tools = role.tools.map(tool => tool.toLowerCase())
    const overlap = match.value.matched.filter(keyword => tools.includes(keyword.toLowerCase()))
    return {
      id: role.id,
      role: role.role,
      meta: `${role.company} · ${monthRange(role.start, role.end) || 'dates not set'}`,
      overlap: overlap.length ? `Uses ${joinList(overlap)}` : 'No tools in common with this job',
      included: !state.skip[role.id],
    }
  }))
  function toggleRole(id: string) {
    state.skip[id] = !state.skip[id]
  }

  const focusSelection = computed(() => (state.focus?.length ? state.focus : match.value.matched.slice(0, 3)))
  const focusChips = computed(() => match.value.matched.map(skill => ({ label: skill, pressed: focusSelection.value.includes(skill) })))
  function toggleFocus(skill: string) {
    const selected = focusSelection.value
    if (selected.includes(skill)) state.focus = selected.filter(item => item !== skill)
    else if (selected.length < 3) state.focus = [...selected, skill]
  }

  function toRequest(jobId: string): CvRequest {
    return {
      job_id: jobId,
      skip_roles: Object.keys(state.skip).filter(id => state.skip[id]),
      focus: state.focus ?? [],
      confirmed: (toValue(questions)?.responsibilities ?? [])
        .filter((claim) => {
          const kept = answers.answer(claimKey(claim.index))
          return kept && kept.status !== 'skipped'
        })
        .map(claim => claim.index),
    }
  }

  return {
    state,
    answers,
    whereLabel,
    roleChoices,
    pickRole,
    score: computed(() => match.value.score),
    matched: computed(() => match.value.matched),
    known,
    matchLabel,
    addGap,
    answerProbe,
    claimYes,
    suggestionsFor,
    progress,
    impactRows: computed(() => impactRows.value ?? []),
    saveImpact,
    note,
    saveNote,
    roles,
    toggleRole,
    focusChips,
    toggleFocus,
    toRequest,
  }
}
