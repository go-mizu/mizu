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
      .then(c => setChallenge(c))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [year])

  const handleSet = () => {
    const g = parseInt(goal)
    if (g <= 0) return
    booksApi.setChallenge(year, g)
      .then(c => setChallenge(c))
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
      <div className="page-container" style={{ maxWidth: 600, margin: '0 auto', padding: 40 }}>
        <div className="challenge-card">
          <div className="challenge-year">{year} Reading Challenge</div>

          {!challenge || challenge.goal === 0 ? (
            <>
              <div className="my-6">
                <Trophy size={48} className="mx-auto text-gr-orange mb-4" />
                <h2 className="challenge-title">Set Your Reading Goal</h2>
                <p className="text-gr-light mt-2 mb-6">How many books do you want to read this year?</p>
              </div>
              <div className="flex items-center justify-center gap-3 mb-4">
                <input
                  type="number"
                  min="1"
                  value={goal}
                  onChange={e => setGoal(e.target.value)}
                  placeholder="e.g. 24"
                  className="form-input text-center text-2xl font-bold"
                  style={{ width: 120 }}
                />
                <span className="text-gr-light text-lg">books</span>
              </div>
              <button className="btn btn-primary btn-lg" onClick={handleSet}>
                Start Challenge
              </button>
            </>
          ) : (
            <>
              <h2 className="challenge-title">
                <Trophy size={28} className="inline text-gr-orange mr-2" />
                Reading Challenge
              </h2>
              <div className="challenge-progress">
                {challenge.progress} <span className="text-2xl text-gr-light">/ {challenge.goal}</span>
              </div>
              <div className="challenge-goal">books read</div>

              <div className="mt-6 mx-auto" style={{ maxWidth: 300 }}>
                <div className="progress-bar" style={{ height: 12 }}>
                  <div className="progress-fill" style={{ width: `${pct}%` }} />
                </div>
                <div className="progress-label mt-2">{pct}% complete</div>
              </div>

              {pct >= 100 && (
                <p className="mt-6 text-gr-green font-bold text-lg">
                  Congratulations! You've reached your goal!
                </p>
              )}

              <div className="mt-8">
                <p className="text-sm text-gr-light mb-2">Update your goal</p>
                <div className="flex items-center justify-center gap-3">
                  <input
                    type="number"
                    min="1"
                    value={goal}
                    onChange={e => setGoal(e.target.value)}
                    placeholder={String(challenge.goal)}
                    className="form-input text-center"
                    style={{ width: 100 }}
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
