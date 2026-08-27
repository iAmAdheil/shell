export type Me = {
  id: string
  name: string
  avatarUrl: string
}

/** Returns the logged-in User, or null when nobody is logged in. */
export async function fetchMe(): Promise<Me | null> {
  const res = await fetch('/api/me', { credentials: 'same-origin' })
  if (res.status === 401) return null
  if (!res.ok) throw new Error(`GET /api/me failed: ${res.status}`)
  return res.json()
}

export async function logOut(): Promise<void> {
  const res = await fetch('/api/auth/logout', {
    method: 'POST',
    credentials: 'same-origin',
  })
  if (!res.ok) throw new Error(`logout failed: ${res.status}`)
}

/** Starts a new Session and returns its Session Code. */
export async function createSession(): Promise<string> {
  const res = await fetch('/api/sessions', {
    method: 'POST',
    credentials: 'same-origin',
  })
  if (!res.ok) throw new Error(`could not start a Session: ${res.status}`)
  const body = (await res.json()) as { code: string }
  return body.code
}
