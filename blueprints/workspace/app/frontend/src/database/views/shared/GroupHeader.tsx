import { ChevronRight, Plus } from 'lucide-react'

export interface GroupHeaderProps {
  name: string
  count: number
  color?: string
  isExpanded: boolean
  onToggle: () => void
  onAddItem?: () => void
  showCount?: boolean
  showAddButton?: boolean
  className?: string
}

export function GroupHeader({
  name,
  count,
  color,
  isExpanded,
  onToggle,
  onAddItem,
  showCount = true,
  showAddButton = true,
  className = '',
}: GroupHeaderProps) {
  return (
    <div className={`group-header-container ${className}`}>
      <button
        type="button"
        className="group-header-toggle"
        onClick={onToggle}
      >
        <ChevronRight
          size={14}
          className={`chevron ${isExpanded ? 'expanded' : ''}`}
        />
        {color && (
          <span className="group-color-dot" style={{ backgroundColor: color }} />
        )}
        <span className="group-name">{name}</span>
        {showCount && (
          <span className="group-count">{count}</span>
        )}
      </button>

      {showAddButton && onAddItem && (
        <button
          type="button"
          className="group-add-btn"
          onClick={(e) => {
            e.stopPropagation()
            onAddItem()
          }}
          title={`Add new item to ${name}`}
        >
          <Plus size={14} />
        </button>
      )}

      <style>{`
        .group-header-container {
          display: flex;
          align-items: center;
          gap: 4px;
          padding: 8px 12px;
          background: rgba(55, 53, 47, 0.02);
          border-bottom: 1px solid rgba(55, 53, 47, 0.09);
        }

        .group-header-toggle {
          display: flex;
          align-items: center;
          gap: 6px;
          flex: 1;
          padding: 0;
          border: none;
          background: none;
          cursor: pointer;
          font-size: 13px;
          color: #37352f;
          text-align: left;
        }

        .group-header-toggle:hover {
          opacity: 0.8;
        }

        .chevron {
          color: #9a9a97;
          transition: transform 0.15s;
        }

        .chevron.expanded {
          transform: rotate(90deg);
        }

        .group-color-dot {
          width: 8px;
          height: 8px;
          border-radius: 50%;
          flex-shrink: 0;
        }

        .group-name {
          font-weight: 500;
        }

        .group-count {
          margin-left: auto;
          padding: 1px 6px;
          background: rgba(55, 53, 47, 0.06);
          border-radius: 4px;
          font-size: 11px;
          color: #787774;
        }

        .group-add-btn {
          padding: 4px;
          border: none;
          background: none;
          cursor: pointer;
          color: #9a9a97;
          border-radius: 4px;
          display: flex;
          align-items: center;
          justify-content: center;
          opacity: 0;
          transition: all 0.15s;
        }

        .group-header-container:hover .group-add-btn {
          opacity: 1;
        }

        .group-add-btn:hover {
          background: rgba(55, 53, 47, 0.08);
          color: #37352f;
        }
      `}</style>
    </div>
  )
}

export default GroupHeader
