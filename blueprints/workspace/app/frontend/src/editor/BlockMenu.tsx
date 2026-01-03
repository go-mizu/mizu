import { useState, useRef, useEffect, useCallback } from 'react'
import {
  GripVertical,
  Plus,
  Trash2,
  Copy,
  Link2,
  MessageSquare,
  ArrowRight,
  Type,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  CheckSquare,
  Quote,
  Code,
  ToggleLeft,
} from 'lucide-react'

interface BlockMenuProps {
  blockId: string
  onDelete: () => void
  onDuplicate: () => void
  onCopyLink: () => void
  onTurnInto: (type: string, props?: Record<string, unknown>) => void
  onAddComment?: () => void
  position?: { x: number; y: number }
}

const TURN_INTO_OPTIONS = [
  { type: 'paragraph', label: 'Text', icon: Type },
  { type: 'heading', label: 'Heading 1', icon: Heading1, props: { level: 1 } },
  { type: 'heading', label: 'Heading 2', icon: Heading2, props: { level: 2 } },
  { type: 'heading', label: 'Heading 3', icon: Heading3, props: { level: 3 } },
  { type: 'bulletListItem', label: 'Bulleted list', icon: List },
  { type: 'numberedListItem', label: 'Numbered list', icon: ListOrdered },
  { type: 'checkListItem', label: 'To-do list', icon: CheckSquare },
  { type: 'quote', label: 'Quote', icon: Quote },
  { type: 'codeBlock', label: 'Code', icon: Code },
  { type: 'toggle', label: 'Toggle list', icon: ToggleLeft },
]

export function BlockMenu({
  blockId,
  onDelete,
  onDuplicate,
  onCopyLink,
  onTurnInto,
  onAddComment,
}: BlockMenuProps) {
  const [showMenu, setShowMenu] = useState(false)
  const [showTurnInto, setShowTurnInto] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)

  // Close menu on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false)
        setShowTurnInto(false)
      }
    }

    if (showMenu) {
      document.addEventListener('mousedown', handleClickOutside)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [showMenu])

  // Handle keyboard navigation
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      setShowMenu(false)
      setShowTurnInto(false)
    }
  }, [])

  const handleMenuAction = (action: () => void) => {
    action()
    setShowMenu(false)
    setShowTurnInto(false)
  }

  return (
    <div className="block-handle" ref={menuRef} onKeyDown={handleKeyDown}>
      {/* Add block button */}
      <button
        className="add-block-btn"
        title="Click to add block below"
        onClick={() => {
          // This could trigger slash menu at this position
        }}
      >
        <Plus size={14} />
      </button>

      {/* Drag handle / Menu trigger */}
      <button
        ref={buttonRef}
        className="drag-handle-btn"
        title="Drag to move. Click for options."
        onClick={() => setShowMenu(!showMenu)}
        draggable
        onDragStart={(e) => {
          e.dataTransfer.setData('text/plain', blockId)
          e.dataTransfer.effectAllowed = 'move'
        }}
      >
        <GripVertical size={14} />
      </button>

      {/* Block menu dropdown */}
      {showMenu && (
        <div className="block-menu">
          <button
            className="block-menu-item"
            onClick={() => handleMenuAction(onDelete)}
          >
            <Trash2 size={16} className="icon" />
            <span>Delete</span>
          </button>

          <button
            className="block-menu-item"
            onClick={() => handleMenuAction(onDuplicate)}
          >
            <Copy size={16} className="icon" />
            <span>Duplicate</span>
          </button>

          <div
            className="block-menu-item"
            onMouseEnter={() => setShowTurnInto(true)}
            onMouseLeave={() => setShowTurnInto(false)}
          >
            <ArrowRight size={16} className="icon" />
            <span>Turn into</span>
            <ArrowRight size={14} style={{ marginLeft: 'auto' }} />

            {/* Turn into submenu */}
            {showTurnInto && (
              <div className="turn-into-menu">
                {TURN_INTO_OPTIONS.map((option) => (
                  <button
                    key={`${option.type}-${option.label}`}
                    className="block-menu-item"
                    onClick={() =>
                      handleMenuAction(() => onTurnInto(option.type, option.props))
                    }
                  >
                    <option.icon size={16} className="icon" />
                    <span>{option.label}</span>
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="block-menu-divider" />

          <button
            className="block-menu-item"
            onClick={() => handleMenuAction(onCopyLink)}
          >
            <Link2 size={16} className="icon" />
            <span>Copy link to block</span>
          </button>

          {onAddComment && (
            <button
              className="block-menu-item"
              onClick={() => handleMenuAction(onAddComment)}
            >
              <MessageSquare size={16} className="icon" />
              <span>Comment</span>
            </button>
          )}

          <div className="block-menu-divider" />

          <button
            className="block-menu-item danger"
            onClick={() => handleMenuAction(onDelete)}
          >
            <Trash2 size={16} className="icon" />
            <span>Delete</span>
          </button>
        </div>
      )}
    </div>
  )
}

// Wrapper component for blocks that adds the drag handle
export function BlockWrapper({
  blockId,
  children,
  onDelete,
  onDuplicate,
  onCopyLink,
  onTurnInto,
  onAddComment,
}: {
  blockId: string
  children: React.ReactNode
  onDelete: () => void
  onDuplicate: () => void
  onCopyLink: () => void
  onTurnInto: (type: string, props?: Record<string, unknown>) => void
  onAddComment?: () => void
}) {
  return (
    <div className="block-wrapper" data-block-id={blockId}>
      <BlockMenu
        blockId={blockId}
        onDelete={onDelete}
        onDuplicate={onDuplicate}
        onCopyLink={onCopyLink}
        onTurnInto={onTurnInto}
        onAddComment={onAddComment}
      />
      {children}
    </div>
  )
}
