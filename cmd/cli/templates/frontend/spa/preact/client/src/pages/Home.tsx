import { useState, useEffect } from 'preact/hooks'

interface HomeProps {
  path: string
}

function Home(_props: HomeProps) {
  const [message, setMessage] = useState('')
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
