import { useState, useEffect } from 'react'

function Home() {
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
    <div className="page home">
      <h1>Welcome</h1>
      {loading ? (
        <p>Loading...</p>
      ) : (
        <p className="api-message">{message}</p>
      )}
    </div>
  )
}

export default Home
