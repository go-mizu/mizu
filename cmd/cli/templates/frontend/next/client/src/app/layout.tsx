import type { Metadata } from 'next'
import '@/styles/globals.css'
import Navigation from '@/components/Navigation'

export const metadata: Metadata = {
  title: '{{.Name}}',
  description: 'Built with Next.js and Mizu',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-slate-50 text-slate-900">
        <div className="flex flex-col min-h-screen">
          <Navigation />
          <main className="flex-1 max-w-5xl mx-auto w-full p-8">
            {children}
          </main>
          <footer className="border-t border-slate-200 bg-white py-4 text-center text-slate-500">
            <p>Built with Mizu + Next.js</p>
          </footer>
        </div>
      </body>
    </html>
  )
}
