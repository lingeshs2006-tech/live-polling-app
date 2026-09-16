import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { api, getVoterId, wsUrl } from '../api/client'
import LiveResults from '../components/LiveResults'
import './PollPage.css'

// PollPage is the shareable link target. Audience votes here and everyone
// watching sees results update live over a websocket — no refresh ever.
export default function PollPage() {
  const { id } = useParams()

  const [state, setState] = useState({ loading: true, error: '', poll: null })
  const [counts, setCounts] = useState([])
  const [total, setTotal] = useState(0)
  const [selected, setSelected] = useState(null)
  const [voting, setVoting] = useState(false)
  const [voteError, setVoteError] = useState('')
  const [voted, setVoted] = useState(false)
  const [reconnecting, setReconnecting] = useState(false)
  const [copyMsg, setCopyMsg] = useState(false)

  const wsRef = useRef(null)
  const pollRef = useRef(null)

  // Initial load: fetch poll + live counts (rest endpoint also hydrates
  // Redis for us). Then open the websocket for push updates.
  useEffect(() => {
    let mounted = true
    api(`/polls/${id}`, { auth: false })
      .then((res) => {
        if (!mounted) return
        pollRef.current = res.poll
        setState({ loading: false, error: '', poll: res.poll })
        setCounts(res.counts)
        setTotal(res.total)
      })
      .catch((e) => mounted && setState({ loading: false, error: e.message, poll: null }))
    return () => {
      mounted = false
    }
  }, [id])

  useEffect(() => {
    let ws
    let retryTimer

    // function declaration (hoisted) allows safe self-reference on reconnect
    function connectStream() {
      ws = new WebSocket(wsUrl(id))

      ws.onopen = () => {
        setReconnecting(false)
      }

      ws.onmessage = (e) => {
        let msg
        try {
          msg = JSON.parse(e.data)
        } catch {
          return
        }

        if (msg.type === 'poll-deleted') {
          setState({ loading: false, error: 'This poll was deleted by its creator.', poll: null })
          ws.close()
          return
        }

        // snapshot and vote events both carry the live counts.
        if (msg.type === 'vote' || msg.type === 'snapshot') {
          setCounts(msg.counts || [])
          setTotal(msg.totalVotes || 0)
        }
      }

      ws.onclose = () => {
        // Auto-reconnect so the stream recovers from blips.
        retryTimer = setTimeout(() => {
          if (pollRef.current && document.visibilityState !== 'hidden') {
            setReconnecting(true)
            connectStream()
          }
        }, 2000)
      }

      wsRef.current = ws
    }

    connectStream()
    return () => {
      clearTimeout(retryTimer)
      if (ws) ws.close()
    }
  }, [id])

  async function castVote() {
    if (selected === null || voting) return
    setVoting(true)
    setVoteError('')
    try {
      const res = await api(`/polls/${id}/vote`, {
        method: 'POST',
        body: { option: selected },
        auth: false,
        headers: { 'X-Voter-Id': getVoterId() },
      })
      setCounts(res.counts)
      setTotal(res.total)
      setVoted(true)
    } catch (e) {
      setVoteError(e.message)
      if (e.message.includes('already voted')) setVoted(true)
    } finally {
      setVoting(false)
    }
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(window.location.href)
      setCopyMsg(true)
      setTimeout(() => setCopyMsg(false), 2000)
    } catch {
      /* clipboard unavailable */
    }
  }

  if (state.loading) return <div className="page center muted">Loading poll…</div>

  if (state.error || !state.poll) {
    return (
      <div className="poll-error">
        <p className="poll-error-icon">◠◦◡</p>
        <p className="poll-error-text">{state.error || 'Poll not found'}</p>
      </div>
    )
  }

  const { poll } = state

  return (
    <div className="poll-page">
      <div className="poll-card-main">
        <div className="poll-head">
          <h1>{poll.question}</h1>
          <button className="share-btn" onClick={copyLink} title="Copy link">
            {copyMsg ? '✓ Copied!' : '🔗 Copy link'}
          </button>
        </div>

        {!voted ? (
          <div className="vote-box">
            <p className="vote-prompt">Pick an option:</p>
            <div className="vote-options">
              {poll.options.map((opt, i) => (
                <button
                  key={i}
                  className={`vote-option${selected === i ? ' selected' : ''}`}
                  onClick={() => setSelected(i)}
                >
                  {opt.text}
                </button>
              ))}
            </div>
            {voteError && <p className="error">{voteError}</p>}
            <button
              className="btn btn-primary btn-block"
              onClick={castVote}
              disabled={selected === null || voting}
            >
              {voting ? 'Casting vote…' : 'Vote'}
            </button>
            {!poll.totalVotes && <p className="live-wait">Waiting for the first vote…</p>}
          </div>
        ) : (
          <div className="share-prompt">
            <p className="vote-thanks">Your vote counts. Results are live below.</p>
            <button className="btn btn-ghost" onClick={copyLink}>
              Share this link with more people
            </button>
          </div>
        )}

        <div className="results-box">
          {reconnecting && <p className="reconnect-banner">Reconnecting to live stream…</p>}
          <LiveResults
            options={poll.options}
            counts={counts}
            total={total}
          />
        </div>
      </div>
    </div>
  )
}