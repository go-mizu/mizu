import { Search } from 'lucide-react'

interface RelatedSearchesWidgetProps {
  searches: string[]
  onSearch: (query: string) => void
}

export function RelatedSearchesWidget({ searches, onSearch }: RelatedSearchesWidgetProps) {
  if (!searches || searches.length === 0) return null

  return (
    <div className="bg-[#f8f9fa] border border-[#dadce0] rounded-lg p-4">
      <div className="flex items-center gap-2 mb-3">
        <Search size={18} className="text-[#1a73e8]" />
        <h3 className="font-medium text-[#202124]">Related Searches</h3>
      </div>

      <div className="grid grid-cols-2 gap-2">
        {searches.map((search, index) => (
          <button
            key={index}
            type="button"
            onClick={() => onSearch(search)}
            className="flex items-center gap-2 px-3 py-2 text-sm text-left text-[#1a0dab] hover:bg-[#e8f0fe] rounded transition-colors"
          >
            <Search size={14} className="text-[#70757a] flex-shrink-0" />
            <span className="truncate">{search}</span>
          </button>
        ))}
      </div>
    </div>
  )
}
