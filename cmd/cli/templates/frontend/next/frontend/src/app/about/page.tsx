export default function About() {
  return (
    <div className="space-y-6">
      <h1 className="text-4xl font-bold">About</h1>
      <p className="text-slate-600">
        This is a Next.js SPA powered by Mizu, a lightweight Go web framework.
      </p>
      <ul className="list-disc list-inside space-y-2 text-slate-600">
        <li>Next.js 15 with App Router and TypeScript</li>
        <li>Static export for SPA deployment</li>
        <li>Tailwind CSS for styling</li>
        <li>Mizu backend with the frontend middleware</li>
      </ul>
    </div>
  )
}
