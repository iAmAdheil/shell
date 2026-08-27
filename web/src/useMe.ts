import * as Sentry from '@sentry/react'
import { useCallback, useEffect, useState } from 'react'
import { fetchMe, type Me } from './api'

type State =
  | { status: 'loading' }
  | { status: 'logged-out' }
  | { status: 'logged-in'; me: Me }
  | { status: 'error'; message: string }

/** Tracks who is logged in, and reloads it after a login or a logout. */
export function useMe() {
  const [state, setState] = useState<State>({ status: 'loading' })

  const reload = useCallback(async () => {
    try {
      const me = await fetchMe()
      // Sentry gets the Account ID only, never the User's name or avatar.
      Sentry.setUser(me ? { id: me.id } : null)
      setState(me ? { status: 'logged-in', me } : { status: 'logged-out' })
    } catch (err) {
      // The app shows this one, so nothing else would report it.
      Sentry.captureException(err)
      setState({ status: 'error', message: String(err) })
    }
  }, [])

  // Fetching who is logged in is exactly what an effect is for: the server is
  // the external system. The rule fires on the setState inside reload, which
  // runs after an await, not synchronously during this effect.
  useEffect(() => {
    void reload()
  }, [reload])

  return { state, reload }
}
