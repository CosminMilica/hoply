import { HTTPError } from 'ky'
import { unpackApiErrorMessage } from '@/shared/api/errors'

/**
 * The service's own message for a refused request: the services' envelope `{ error: { message } }`,
 * or a top-level `message`. ky 2 reads an error's body into `error.data` before it throws, so the
 * response body is no longer there to parse.
 */
export async function serviceErrorMessage(error: unknown): Promise<string> {
  if (error instanceof HTTPError) {
    const data = error.data as { message?: unknown, error?: { message?: unknown } } | undefined
    const message = data?.error?.message ?? data?.message
    if (typeof message === 'string' && message.trim()) return message
  }
  return unpackApiErrorMessage(error)
}
