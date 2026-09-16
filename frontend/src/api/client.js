const API_BASE = import.meta.env.VITE_API_BASE || '/api'

// Central fetch wrapper: injects the JWT, parses JSON or throws with the
// server-provided error message.
export async function api(
  path,
  { method = 'GET', body, auth = true, headers: extraHeaders = {} } = {}
) {
  const headers = { 'Content-Type': 'application/json', ...extraHeaders }

  if (auth) {
    const token = localStorage.getItem('token')
    if (token) headers.Authorization = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  let data = null
  const text = await res.text()

  try {
    data = text ? JSON.parse(text) : null
  } catch {
    data = null
  }

  if (!res.ok) {
    const message = (data && data.error) || `Request failed (${res.status})`
    const err = new Error(message)
    err.status = res.status
    throw err
  }

  return data
}

// Stable anonymous voter token stored in localStorage, used for the server
// side "one person = one vote" check. The backend only ever sees a hash of it.
export function getVoterId() {
  let id = localStorage.getItem('voterId')

  if (!id) {
    id = `anon-${Date.now()}-${Math.random().toString(36).slice(2)}`
    localStorage.setItem('voterId', id)
  }

  return id
}

// WebSocket URL for live poll updates.
export function wsUrl(pollId) {
  const base = import.meta.env.VITE_API_BASE || window.location.origin
  const protocol = base.startsWith('https://') ? 'wss' : 'ws'
  const host = base.replace(/^https?:\/\//, '')

  return `${protocol}://${host}/ws?poll=${pollId}`
}