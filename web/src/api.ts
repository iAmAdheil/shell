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
