interface AboutProps {
  path: string
}

function About(_props: AboutProps) {
  return (
    <div className="page about">
      <h1>About</h1>
      <p>
        This is a Preact SPA powered by Mizu, a lightweight Go web framework.
      </p>
      <ul>
        <li>Preact 10 with TypeScript (~3kB gzipped)</li>
        <li>Vite for fast development and optimized builds</li>
        <li>preact-router for client-side routing</li>
        <li>Mizu backend with the frontend middleware</li>
      </ul>
    </div>
  )
}

export default About
