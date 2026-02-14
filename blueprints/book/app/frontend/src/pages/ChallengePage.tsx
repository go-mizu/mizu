import { useState, useEffect } from 'react'
import { Trophy } from 'lucide-react'
import Header from '../components/Header'
import { booksApi } from '../api/books'
import type { ReadingChallenge } from '../types'

export default function ChallengePage() {
  const [challenge, setChallenge] = useState<ReadingChallenge | null>(null)
  const [loading, setLoading] = useState(true)
  const [goal, setGoal] = useState('')
  const year = new Date().getFullYear()

  useEffect(() => {
    booksApi.getChallenge(year)
      .then((c) => setChallenge(c))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [year])

  const handleSet = () => {
    const g = parseInt(goal)
    if (g <= 0) return
    booksApi.setChallenge(year, g)
      .then((c) => setChallenge(c))
      .catch(() => {})
  }

  if (loading) {
    return (
      <>
        <Header />
        <div className="loading-spinner"><div className="spinner" /></div>
      </>
    )
  }

  const pct = challenge && challenge.goal > 0
    ? Math.min(100, Math.round((challenge.progress / challenge.goal) * 100))
    : 0

  return (
    <>
      <Header />
      <div className="page-container page-narrow challenge-page">
        <div className="challenge-card">
          <div className="challenge-year">{year} Reading Challenge</div>

          {!challenge || challenge.goal === 0 ? (
            <>
              <div className="challenge-empty-head">
                <Trophy size={48} className="challenge-icon" />
                <h2 className="challenge-title">Set Your Reading Goal</h2>
                <p className="challenge-hint">How many books do you want to read this year?</p>
              </div>
              <div className="challenge-goal-input-row">
                <input
                  type="number"
                  min="1"
                  value={goal}
                  onChange={(e) => setGoal(e.target.value)}
                  placeholder="e.g. 24"
                  className="form-input challenge-goal-input"
                />
                <span className="challenge-goal-unit">books</span>
              </div>
              <button className="btn btn-primary btn-lg" onClick={handleSet}>
                Start Challenge
              </button>
            </>
          ) : (
            <>
              <h2 className="challenge-title challenge-title-with-icon">
                <Trophy size={28} className="challenge-icon" />
                Reading Challenge
              </h2>
              <div className="challenge-progress">
                {challenge.progress} <span className="challenge-progress-total">/ {challenge.goal}</span>
              </div>
              <div className="challenge-goal">books read</div>

              <div className="challenge-progress-wrap challenge-progress-wrap-wide">
                <div className="progress-bar progress-bar-lg">
                  <div className="progress-fill" style={{ width: `${pct}%` }} />
                </div>
                <div className="progress-label">{pct}% complete</div>
              </div>

              {pct >= 100 && (
                <p className="challenge-success">
                  Congratulations! You've reached your goal!
                </p>
              )}

              <div className="challenge-update">
                <p className="challenge-update-label">Update your goal</p>
                <div className="challenge-goal-input-row">
                  <input
                    type="number"
                    min="1"
                    value={goal}
                    onChange={(e) => setGoal(e.target.value)}
                    placeholder={String(challenge.goal)}
                    className="form-input challenge-goal-input challenge-goal-input-sm"
                  />
                  <button className="btn btn-secondary btn-sm" onClick={handleSet}>
                    Update
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </>
  )
}
