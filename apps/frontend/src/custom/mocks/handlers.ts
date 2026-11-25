import { handlers as generated } from '@/shared/mocks/handlers'

/**
 * The services that do not exist yet, mocked in the browser with in-memory stores: none now. The profile, jobs,
 * applications and cv services are real and have no handler: their requests pass through. Add a service's handlers
 * here while it is being built; the pages only know its typed client.
 */
export const mockedServiceHandlers = () => []

export const handlers = () => [...generated, ...mockedServiceHandlers()]
