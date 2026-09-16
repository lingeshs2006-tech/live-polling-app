import './PollResults.css'

// LiveResults shows each option as a bar whose width is driven by the vote
// share. The data updates in place from websocket events; no page refresh.
export default function LiveResults({ options, counts, total }) {
  const max = Math.max(1, total || 1)

  return (
    <div className="results">
      {options.map((opt, i) => {
        const count = counts[i] || 0
        const pct = total > 0 ? (count / max) * 100 : 0
        const share = total > 0 ? Math.round((count / total) * 1000) / 10 : 0
        return (
          <div key={i} className="result-row">
            <div className="result-label">
              <span className="result-option">{opt.text}</span>
              <span className="result-meta">
                <strong>{count.toLocaleString()}</strong> vote{count === 1 ? '' : 's'} · {share}%
              </span>
            </div>
            <div className="result-track">
              <div
                className="result-bar"
                style={{ width: `${pct}%` }}
                data-color={i % 5}
              />
            </div>
          </div>
        )
      })}
      <div className="results-footer">
        {total.toLocaleString()} total vote{total === 1 ? '' : 's'} · live
      </div>
    </div>
  )
}