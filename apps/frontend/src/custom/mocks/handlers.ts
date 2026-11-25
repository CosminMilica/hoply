import { handlers as generated } from '@/shared/mocks/handlers'
import { createCvHandlers } from '@/features/cv/mocks/handlers'

/**
 * The services that do not exist yet, mocked in the browser with in-memory stores: only cv now.
 * The profile, jobs and applications services are real and have no handler: their requests pass through. Delete a service's handlers
 * here once it ships; the pages only know its typed client.
 */
export const mockedServiceHandlers = () => [
  ...createCvHandlers(),
]

export const handlers = () => [...generated, ...mockedServiceHandlers()]
