'use client'

import { useState, useEffect } from 'react'

export default function Home() {
  const [message, setMessage] = useState<string>('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/hello')
      .then(res => res.json())
      .then(data => {
        setMessage(data.message)
        setLoading(false)
      })
      .catch(() => setLoading(false))
  }, [])

  return (
    <div className="space-y-6">
      <h1 className="text-4xl font-bold">Welcome to {{.Name}}</h1>
      {loading ? (
        <p className="text-slate-500">Loading...</p>
      ) : (
        <div className="bg-white rounded-lg border border-slate-200 p-4">
          <p className="text-lg">{message}</p>
        </div>
      )}
    </div>
  )
}
