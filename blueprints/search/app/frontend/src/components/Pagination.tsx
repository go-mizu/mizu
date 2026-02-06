import { ChevronLeft, ChevronRight } from 'lucide-react'

interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

export function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null

  const windowSize = 10
  let start = Math.max(1, page - Math.floor(windowSize / 2))
  const end = Math.min(totalPages, start + windowSize - 1)
  if (end - start + 1 < windowSize) {
    start = Math.max(1, end - windowSize + 1)
  }

  const pageNumbers = Array.from({ length: end - start + 1 }, (_, i) => start + i)

  return (
    <div className="flex items-center justify-center gap-2 mt-10 py-4">
      <button
        type="button"
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        className="flex items-center gap-1 px-3 py-2 text-sm text-[#1a73e8] hover:bg-[#f1f3f4] rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
      >
        <ChevronLeft size={18} />
        Previous
      </button>

      {start > 1 && (
        <>
          <button
            type="button"
            onClick={() => onPageChange(1)}
            className="w-10 h-10 text-sm text-[#1a73e8] hover:bg-[#f1f3f4] rounded-lg transition-colors"
          >
            1
          </button>
          {start > 2 && <span className="text-[#70757a] px-1">...</span>}
        </>
      )}

      {pageNumbers.map((pageNum) => (
        <button
          key={pageNum}
          type="button"
          onClick={() => onPageChange(pageNum)}
          className={`w-10 h-10 text-sm rounded-lg transition-colors ${
            page === pageNum
              ? 'bg-[#1a73e8] text-white font-medium'
              : 'text-[#1a73e8] hover:bg-[#f1f3f4]'
          }`}
        >
          {pageNum}
        </button>
      ))}

      {end < totalPages && (
        <>
          {end < totalPages - 1 && <span className="text-[#70757a] px-1">...</span>}
          <button
            type="button"
            onClick={() => onPageChange(totalPages)}
            className="w-10 h-10 text-sm text-[#1a73e8] hover:bg-[#f1f3f4] rounded-lg transition-colors"
          >
            {totalPages}
          </button>
        </>
      )}

      <button
        type="button"
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        className="flex items-center gap-1 px-3 py-2 text-sm text-[#1a73e8] hover:bg-[#f1f3f4] rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
      >
        Next
        <ChevronRight size={18} />
      </button>
    </div>
  )
}
