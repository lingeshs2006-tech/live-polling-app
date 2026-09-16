import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import './Dashboard.css'

export default function Dashboard() {
  const [polls, setPolls] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = useCallback(() => {
    api('/polls')
      .then(setPolls)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  useEffect(load, [load])

  async function remove(id) {
    if (!window.confirm('Delete this poll and all its results?')) return
    try {
      await api(`/polls/${id}`, { method: 'DELETE' })
      load()
    } catch (e) {
      setError(e.message)
    }
  }

  if (loading) return <div className="page center muted">Loading…</div>

  return (
    <div className="dash">
      <div className="dash-head">
        <h2>My Polls</h2>
        <Link to="/" className="btn btn-primary">+ New poll</Link>
      </div>

      {error && <p className="error">{error}</p>}

      {polls.length === 0 ? (
        <div className="empty">
          <p className="empty-title">No polls yet</p>
          <p className="empty-sub">Create your first poll and share the link with your audience.</p>
          <Link to="/" className="btn btn-primary">Create a poll</Link>
        </div>
      ) : (
        <div className="poll-grid">
          {polls.map((p) => {
            const total = p.options.reduce((s, o) => s + o.votes, 0)
            return (
              <div key={p.id} className="poll-card">
                <h3 className="poll-card-question">{p.question}</h3>
                <p className="poll-card-meta">
                  {p.options.length} options · {total} vote{total === 1 ? '' : 's'}
                </p>
                <div className="poll-card-actions">
                  <Link to={`/poll/${p.id}`} className="btn btn-ghost btn-sm">Open</Link>
                  <button className="btn btn-danger btn-sm" onClick={() => remove(p.id)}>
                    Delete
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}