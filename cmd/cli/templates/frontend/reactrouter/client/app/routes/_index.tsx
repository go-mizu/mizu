import { useLoaderData } from "react-router";
import type { Route } from "./+types/_index";

export async function loader() {
  const res = await fetch("/api/health");
  const data = await res.json();
  return { health: data };
}

export function meta({}: Route.MetaArgs) {
  return [
    { title: "{{.Name}}" },
    { name: "description", content: "Welcome to React Router v7 with Mizu!" },
  ];
}

export default function Index({ loaderData }: Route.ComponentProps) {
  return (
    <div className="home">
      <h1>Welcome to React Router v7</h1>
      <p className="subtitle">
        Modern routing framework powered by Mizu backend
      </p>

      <div className="features">
        <div className="feature">
          <h3>🚀 React Router v7</h3>
          <p>Type-safe routing with file-based conventions</p>
        </div>
        <div className="feature">
          <h3>⚡ Vite</h3>
          <p>Lightning fast builds and hot module replacement</p>
        </div>
        <div className="feature">
          <h3>🔷 TypeScript</h3>
          <p>Full type safety from routes to data</p>
        </div>
        <div className="feature">
          <h3>💧 Mizu</h3>
          <p>Lightweight Go backend with powerful middleware</p>
        </div>
      </div>

      {loaderData?.health && (
        <div className="status">
          <span className="status-indicator"></span>
          <span>Backend Status: {loaderData.health.status}</span>
        </div>
      )}
    </div>
  );
}
