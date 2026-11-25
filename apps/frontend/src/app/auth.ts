// The authentication contract the guards, the API client and the shell use: forge's
// runtime (`useAuth` against the gateway's session), as auth.runtime: forge asks.
import { forgeAuth } from '@/shared/application/forgeAuth'

export const auth = forgeAuth()
