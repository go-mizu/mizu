import { Link, Outlet } from "react-router";

export default function Layout() {
  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <h1 className="logo">{{.Name}}</h1>
          <nav className="nav">
            <Link to="/" className="nav-link">Home</Link>
            <Link to="/about" className="nav-link">About</Link>
            <Link to="/users" className="nav-link">Users</Link>
          </nav>
        </div>
      </header>

      <main className="main">
        <Outlet />
      </main>

      <footer className="footer">
        <p>Built with Mizu + React Router v7</p>
      </footer>
    </div>
  );
}
