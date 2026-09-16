import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../context/useAuth'
import './Home.css'

export default function Home() {
  const { user } = useAuth()
  const navigate = useNavigate()

  if (!user) {
    return (
      <div className="hero">
        <h1 className="hero-title">
          Polls that update <span className="grad">live</span>.
        </h1>
        <p className="hero-sub">
          Create a poll, share the link, and watch results roll in — in real
          time, no refresh needed.
        </p>
        <div className="hero-actions">
          <Link to="/signup" className="btn btn-primary">Create a poll</Link>
          <Link to="/login" className="btn btn-ghost">Log in</Link>
        </div>
        <div className="hero-cards">
          <div className="hero-card"><strong>Create</strong><span>Any registered user can build a poll in seconds.</span></div>
          <div className="hero-card"><strong>Share</strong><span>Send the link to your audience. No login needed to vote.</span></div>
          <div className="hero-card"><strong>Watch live</strong><span>Results stream over WebSockets as votes arrive.</span></div>
        </div>
      </div>
    )
  }

  return <CreatePoll onCreated={(id) => navigate(`/poll/${id}`)} />
}

function CreatePoll({ onCreated }) {
  const [question, setQuestion] = useState('')
  const [options, setOptions] = useState(['', ''])
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')

  function updateOption(i, value) {
    setOptions((prev) => prev.map((o, idx) => (idx === i ? value : o)))
  }

  function addOption() {
    if (options.length < 8) setOptions((prev) => [...prev, ''])
  }

  function removeOption(i) {
    if (options.length > 2) setOptions((prev) => prev.filter((_, idx) => idx !== i))
  }

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')

    const cleanOptions = options.map((o) => o.trim()).filter(Boolean)
    if (!question.trim()) return setError('Add a question to get started.')
    if (cleanOptions.length < 2) return setError('Add at least two options.')

    setCreating(true)
    try {
      const res = await api('/polls', {
        method: 'POST',
        body: { question: question.trim(), options: cleanOptions },
      })
      onCreated(res.poll.id)
    } catch (err) {
      setError(err.message)
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="create-wrap">
      <div className="create-card">
        <h2 className="create-title">Create a live poll</h2>
        <form onSubmit={handleSubmit}>
          <label className="field">
            <span>Question</span>
            <input
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              placeholder="e.g. Which language should we learn next?"
              maxLength={200}
            />
          </label>

          <label className="field-label">Options</label>
          <div className="option-list">
            {options.map((opt, i) => (
              <div key={i} className="option-row">
                <input
                  value={opt}
                  onChange={(e) => updateOption(i, e.target.value)}
                  placeholder={`Option ${i + 1}`}
                  maxLength={120}
                />
                {options.length > 2 && (
                  <button type="button" className="icon-btn" onClick={() => removeOption(i)} aria-label="Remove option">
                    ×
                  </button>
                )}
              </div>
            ))}
          </div>

          {options.length < 8 && (
            <button type="button" className="add-option" onClick={addOption}>
              + Add an option
            </button>
          )}

          {error && <p className="error">{error}</p>}

          <button type="submit" className="btn btn-primary btn-block" disabled={creating}>
            {creating ? 'Creating…' : 'Create poll & get link'}
          </button>
        </form>
      </div>
    </div>
  )
}