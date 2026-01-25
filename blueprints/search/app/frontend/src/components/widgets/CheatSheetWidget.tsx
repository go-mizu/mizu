import { useState } from 'react'
import { ChevronDown, ChevronUp, Code } from 'lucide-react'
import type { CheatSheet } from '../../types'

interface CheatSheetWidgetProps {
  sheet: CheatSheet
}

export function CheatSheetWidget({ sheet }: CheatSheetWidgetProps) {
  const [expandedSections, setExpandedSections] = useState<Set<number>>(new Set([0]))

  const toggleSection = (index: number) => {
    const newExpanded = new Set(expandedSections)
    if (newExpanded.has(index)) {
      newExpanded.delete(index)
    } else {
      newExpanded.add(index)
    }
    setExpandedSections(newExpanded)
  }

  return (
    <div className="bg-[#f8f9fa] border border-[#dadce0] rounded-lg p-4 mb-4">
      <div className="flex items-center gap-2 mb-3">
        <Code size={18} className="text-[#1a73e8]" />
        <h3 className="font-medium text-[#202124]">{sheet.title}</h3>
      </div>

      {sheet.description && (
        <p className="text-sm text-[#70757a] mb-3">{sheet.description}</p>
      )}

      <div className="space-y-2">
        {sheet.sections.map((section, index) => (
          <div key={index} className="border border-[#e8eaed] rounded bg-white">
            <button
              type="button"
              onClick={() => toggleSection(index)}
              className="w-full flex items-center justify-between px-3 py-2 text-sm font-medium text-[#202124] hover:bg-[#f1f3f4]"
            >
              <span>{section.title}</span>
              {expandedSections.has(index) ? (
                <ChevronUp size={16} className="text-[#70757a]" />
              ) : (
                <ChevronDown size={16} className="text-[#70757a]" />
              )}
            </button>

            {expandedSections.has(index) && (
              <div className="px-3 pb-3 space-y-2">
                {section.items.map((item, itemIndex) => (
                  <div key={itemIndex} className="flex gap-3 text-sm">
                    <code className="px-2 py-0.5 bg-[#f1f3f4] rounded text-[#1a73e8] font-mono text-xs flex-shrink-0">
                      {item.code}
                    </code>
                    <span className="text-[#5f6368]">{item.description}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
