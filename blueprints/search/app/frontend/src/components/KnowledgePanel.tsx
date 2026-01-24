import { ExternalLink } from 'lucide-react'
import type { KnowledgePanel as KnowledgePanelType } from '../types'

interface KnowledgePanelProps {
  panel: KnowledgePanelType
}

export function KnowledgePanel({ panel }: KnowledgePanelProps) {
  return (
    <div className="knowledge-panel">
      {/* Image */}
      {panel.image && (
        <img
          src={panel.image}
          alt={panel.title}
          className="knowledge-panel-image"
          onError={(e) => {
            (e.target as HTMLImageElement).style.display = 'none'
          }}
        />
      )}

      <div className="knowledge-panel-content">
        {/* Title */}
        <h2 className="knowledge-panel-title">{panel.title}</h2>

        {/* Subtitle */}
        {panel.subtitle && (
          <p className="knowledge-panel-subtitle">
            {panel.subtitle.charAt(0).toUpperCase() + panel.subtitle.slice(1).replace(/_/g, ' ')}
          </p>
        )}

        {/* Description */}
        <p className="knowledge-panel-description">{panel.description}</p>

        {/* Facts */}
        {panel.facts && panel.facts.length > 0 && (
          <div className="knowledge-panel-facts">
            {panel.facts.map((fact) => (
              <div key={fact.label} className="knowledge-panel-fact">
                <span className="knowledge-panel-fact-label">{fact.label}</span>
                <span className="knowledge-panel-fact-value">{fact.value}</span>
              </div>
            ))}
          </div>
        )}

        {/* Links */}
        {panel.links && panel.links.length > 0 && (
          <div className="mt-4 pt-4 border-t border-[#dadce0]">
            {panel.links.map((link) => (
              <a
                key={link.url}
                href={link.url}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 py-2 text-sm text-[#1a73e8] hover:underline"
              >
                <ExternalLink size={14} />
                {link.title}
              </a>
            ))}
          </div>
        )}

        {/* Source */}
        {panel.source && (
          <p className="mt-4 text-xs text-[#70757a]">
            Source: {panel.source}
          </p>
        )}
      </div>
    </div>
  )
}
