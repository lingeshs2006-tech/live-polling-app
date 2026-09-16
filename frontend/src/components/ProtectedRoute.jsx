import { Navigate } from 'react-router-dom'
import { useAuth } from '../context/useAuth'

// Guards create/dashboard routes: only logged-in users get in.
export default function ProtectedRoute({ children }) {
  const { user, loading } = useAuth()

  if (loading) {
    return <div className="page center muted">Loading…</div>
  }
  if (!user) {
    return <Navigate to="/login" replace />
  }
  return children
}