import { ArrowLeft } from 'lucide-react'
import { Link } from 'react-router-dom'

interface PageHeaderProps {
  title: string
  /** Where the back arrow navigates to (default: /) */
  backTo?: string
  /** Optional action buttons on the right side */
  actions?: React.ReactNode
}

export function PageHeader({ title, backTo = '/', actions }: PageHeaderProps) {
  return (
    <header className="sticky top-0 bg-white z-50 border-b border-[#e8eaed]">
      <div className="max-w-2xl mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Link
              to={backTo}
              className="p-2 text-[#5f6368] hover:bg-[#f1f3f4] rounded-full transition-colors"
            >
              <ArrowLeft size={20} />
            </Link>
            <h1 className="text-xl font-semibold text-[#202124]">
              {title}
            </h1>
          </div>
          {actions && <div>{actions}</div>}
        </div>
      </div>
    </header>
  )
}
