import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { Search } from 'lucide-react'

export default function Header() {
  const [query, setQuery] = useState('')
  const navigate = useNavigate()
  const location = useLocation()

  const handleSubmit = (e: FormEvent) => {
    e.preventDefault()
    const q = query.trim()
    if (q) {
      navigate(`/search?q=${encodeURIComponent(q)}`)
    }
  }

  const navLinks = [
    { to: '/', label: 'Home' },
    { to: '/my-books', label: 'My Books' },
    { to: '/browse', label: 'Browse' },
    { to: '/community', label: 'Community' },
  ]

  return (
    <header className="header">
      <div className="header-inner">
        <Link to="/" className="header-logo">
          goodreads
        </Link>

        <nav className="header-nav">
          {navLinks.map((link) => (
            <Link
              key={link.to}
              to={link.to}
              className={location.pathname === link.to ? 'active' : ''}
            >
              {link.label}
            </Link>
          ))}
        </nav>

        <form className="header-search" onSubmit={handleSubmit}>
          <input
            type="text"
            placeholder="Search books"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <button type="submit" className="search-icon" aria-label="Search">
            <Search size={18} />
          </button>
        </form>
      </div>
    </header>
  )
}
