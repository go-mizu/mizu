import { Outlet, Link } from 'react-router-dom'

function Layout() {
  return (
    <div className="app">
      <header className="header">
        <nav>
          <Link to="/">Home</Link>
          <Link to="/about">About</Link>
        </nav>
      </header>
      <main className="main">
        <Outlet />
      </main>
      <footer className="footer">
        <p>Built with Mizu + React</p>
      </footer>
    </div>
  )
}

export default Layout
