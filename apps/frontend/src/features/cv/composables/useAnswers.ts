import { reactive } from 'vue'
import { serviceErrorMessage } from '@/features/common/errors'
import type { ProbeAnswer } from '../components/ProbeQuestionCard.vue'
import type { Place, Remembered, useNotes } from './useNotes'

export const MIN_ANSWER = 10

/** A fact saved in the profile, a line of the profile picked, or no answer. */
export type Answer = { status: 'added', remembered: Remembered } | { status: 'picked', text: string } | { status: 'skipped' }

type Notes = Pick<ReturnType<typeof useNotes>, 'remember' | 'forget' | 'whereLabel'>

/** Answers to a page's questions, by key: a draft, a save that keeps its error next to the question, and an undo. */
export function useAnswers(notes: Notes) {
  const state = reactive({
    drafts: {} as Record<string, string>,
    answers: {} as Record<string, Answer>,
    saving: {} as Record<string, boolean>,
    errors: {} as Record<string, string | undefined>,
  })
  const draft = (key: string) => state.drafts[key] ?? ''
  const canAdd = (key: string) => draft(key).trim().length >= MIN_ANSWER && !state.saving[key]

  async function saving(key: string, work: () => Promise<void>) {
    state.saving[key] = true
    state.errors[key] = undefined
    try {
      await work()
    }
    catch (error) {
      state.errors[key] = await serviceErrorMessage(error)
    }
    finally {
      state.saving[key] = false
    }
  }

  return {
    state,
    draft,
    canAdd,
    saving,
    setDraft: (key: string, value: string) => { state.drafts[key] = value },
    answer: (key: string): Answer | undefined => state.answers[key],
    /** Saves the draft as a fact under place, tagged with the skills it shows. */
    async add(key: string, place: Place, skillsOf: (text: string) => string[] = () => []) {
      if (!canAdd(key)) return
      const text = draft(key).trim()
      await saving(key, async () => {
        state.answers[key] = { status: 'added', remembered: await notes.remember(place, text, skillsOf(text)) }
      })
    },
    pick: (key: string, text: string) => { state.answers[key] = { status: 'picked', text } },
    skip: (key: string) => { state.answers[key] = { status: 'skipped' } },
    /** Takes the answer back, and the fact it saved with it. */
    async undo(key: string) {
      const kept = state.answers[key]
      if (kept?.status !== 'added') {
        delete state.answers[key]
        return
      }
      await saving(key, async () => {
        await notes.forget(kept.remembered)
        delete state.answers[key]
      })
    },
    textOf(key: string) {
      const kept = state.answers[key]
      return kept?.status === 'added' ? kept.remembered.text : kept?.status === 'picked' ? kept.text : ''
    },
    savedUnder(key: string) {
      const kept = state.answers[key]
      return kept?.status === 'added' ? notes.whereLabel(kept.remembered.place) : ''
    },
    /** The answer as a question card shows it. */
    shown(key: string): ProbeAnswer {
      const kept = state.answers[key]
      if (!kept) return undefined
      return kept.status === 'added' ? { status: 'added', text: kept.remembered.text, where: notes.whereLabel(kept.remembered.place) } : { status: 'skipped' }
    },
  }
}
