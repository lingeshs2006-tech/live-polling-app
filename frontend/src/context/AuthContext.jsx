import { createContext, useContext, useEffect, useState } from 'react'
import { api } from '../api/client'

const AuthContext = createContext(null)

export { AuthContext }

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null)
  // loading starts true only when a token already exists (bootstrapping).
  const [loading, setLoading] = useState(() => !!localStorage.getItem('token'))

  // On first load with a stored token, verify it against /api/me.
  useEffect(() => {
    const token = localStorage.getItem('token')
    if (!token) return
    api('/me')
      .then(setUser)
      .catch(() => localStorage.removeItem('token'))
      .finally(() => setLoading(false))
  }, [])

  async function login(username, password) {
    const res = await api('/auth/login', {
      method: 'POST',
      body: { username, password },
      auth: false,
    })
    localStorage.setItem('token', res.token)
    setUser(res.user)
  }

  async function signup(username, password) {
    const res = await api('/auth/signup', {
      method: 'POST',
      body: { username, password },
      auth: false,
    })
    localStorage.setItem('token', res.token)
    setUser(res.user)
  }

  function logout() {
    localStorage.removeItem('token')
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, signup, logout }}>
      {children}
    </AuthContext.Provider>
  )
}