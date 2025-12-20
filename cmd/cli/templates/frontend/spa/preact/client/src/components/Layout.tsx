import { ComponentChildren } from 'preact'

interface LayoutProps {
  children: ComponentChildren
}

function Layout({ children }: LayoutProps) {
  return (
    <div className="app">
      <header className="header">
        <nav>
          <a href="/">Home</a>
          <a href="/about">About</a>
        </nav>
      </header>
      <main className="main">
        {children}
      </main>
      <footer className="footer">
        <p>Built with Mizu + Preact</p>
      </footer>
    </div>
  )
}

export default Layout
