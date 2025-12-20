import type { Route } from "./+types/about";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "About - {{.Name}}" },
    { name: "description", content: "Learn more about our stack" },
  ];
}

export default function About() {
  return (
    <div className="page">
      <h1>About This App</h1>
      <p className="lead">
        This application demonstrates the power of React Router v7 combined with a Mizu backend.
      </p>

      <section className="section">
        <h2>Technology Stack</h2>
        <ul className="tech-list">
          <li><strong>React Router v7</strong> - Modern React framework with type-safe routing</li>
          <li><strong>React 19</strong> - Latest React with improved performance</li>
          <li><strong>Vite</strong> - Next generation frontend tooling</li>
          <li><strong>TypeScript</strong> - Type-safe development</li>
          <li><strong>Mizu</strong> - Lightweight Go web framework</li>
        </ul>
      </section>

      <section className="section">
        <h2>Key Features</h2>
        <ul className="feature-list">
          <li>File-based routing with type generation</li>
          <li>Built-in data loading with loaders</li>
          <li>Type-safe form actions</li>
          <li>Error boundaries per route</li>
          <li>Optimistic UI updates</li>
          <li>Static export for production</li>
          <li>Single binary deployment</li>
        </ul>
      </section>
    </div>
  );
}
