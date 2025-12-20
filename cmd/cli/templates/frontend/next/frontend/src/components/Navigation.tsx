'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

export default function Navigation() {
  const pathname = usePathname()

  const navLinks = [
    { href: '/', label: 'Home' },
    { href: '/about', label: 'About' },
  ]

  return (
    <header className="bg-white border-b border-slate-200">
      <nav className="max-w-5xl mx-auto px-8 py-4">
        <ul className="flex gap-6">
          {navLinks.map(({ href, label }) => (
            <li key={href}>
              <Link
                href={href}
                className={`font-medium transition-colors ${
                  pathname === href
                    ? 'text-blue-600'
                    : 'text-slate-600 hover:text-blue-600'
                }`}
              >
                {label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>
    </header>
  )
}
