import { useEffect, useState, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useActiveCollaborators, CollaboratorPresence } from './CollaborationProvider'

interface CursorPosition {
  x: number
  y: number
  visible: boolean
}

// Collaborator cursor component
function CollaboratorCursor({ collaborator }: { collaborator: CollaboratorPresence }) {
  const [position, setPosition] = useState<CursorPosition>({ x: 0, y: 0, visible: false })
  const [isVisible, setIsVisible] = useState(true)

  // Update cursor position based on block ID
  useEffect(() => {
    if (!collaborator.cursor) {
      setPosition((p) => ({ ...p, visible: false }))
      return
    }

    // Find the block element
    const blockElement = document.querySelector(`[data-block-id="${collaborator.cursor.blockId}"]`)
    if (!blockElement) {
      setPosition((p) => ({ ...p, visible: false }))
      return
    }

    const rect = blockElement.getBoundingClientRect()

    // Calculate approximate position within the block based on offset
    // This is a simplified version - real implementation would need text measurement
    const charWidth = 8 // approximate
    const x = rect.left + Math.min(collaborator.cursor.offset * charWidth, rect.width - 10)
    const y = rect.top

    setPosition({ x, y, visible: true })
  }, [collaborator.cursor])

  // Hide cursor after inactivity
  useEffect(() => {
    const timeout = setTimeout(() => {
      const timeSinceActive = Date.now() - new Date(collaborator.lastActive).getTime()
      if (timeSinceActive > 10000) {
        setIsVisible(false)
      }
    }, 10000)

    return () => clearTimeout(timeout)
  }, [collaborator.lastActive])

  if (!position.visible || !isVisible) return null

  return (
    <motion.div
      initial={{ opacity: 0, scale: 0.8 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.8 }}
      style={{
        position: 'fixed',
        left: position.x,
        top: position.y,
        pointerEvents: 'none',
        zIndex: 1000,
        transform: 'translate(-2px, -2px)',
      }}
    >
      {/* Cursor caret */}
      <div
        style={{
          width: '2px',
          height: '20px',
          background: collaborator.color,
          borderRadius: '1px',
          boxShadow: `0 0 4px ${collaborator.color}`,
          animation: 'cursor-blink 1s ease-in-out infinite',
        }}
      />

      {/* Name label */}
      <motion.div
        initial={{ opacity: 0, y: -4 }}
        animate={{ opacity: 1, y: 0 }}
        style={{
          position: 'absolute',
          top: '-24px',
          left: '-2px',
          padding: '2px 6px',
          background: collaborator.color,
          color: 'white',
          fontSize: '11px',
          fontWeight: 500,
          borderRadius: '4px',
          whiteSpace: 'nowrap',
          boxShadow: '0 2px 4px rgba(0,0,0,0.2)',
        }}
      >
        {collaborator.name}
      </motion.div>

      <style>{`
        @keyframes cursor-blink {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.5; }
        }
      `}</style>
    </motion.div>
  )
}

// Collaborator selection highlight
function CollaboratorSelection({ collaborator }: { collaborator: CollaboratorPresence }) {
  const [highlights, setHighlights] = useState<Array<{ x: number; y: number; width: number; height: number }>>([])

  useEffect(() => {
    if (!collaborator.selection) {
      setHighlights([])
      return
    }

    // Find block elements for anchor and focus
    const anchorBlock = document.querySelector(`[data-block-id="${collaborator.selection.anchor.blockId}"]`)
    const focusBlock = document.querySelector(`[data-block-id="${collaborator.selection.focus.blockId}"]`)

    if (!anchorBlock || !focusBlock) {
      setHighlights([])
      return
    }

    // For simplicity, just highlight the blocks involved
    // Real implementation would need to calculate exact text ranges
    const rects: Array<{ x: number; y: number; width: number; height: number }> = []

    if (collaborator.selection.anchor.blockId === collaborator.selection.focus.blockId) {
      // Same block selection
      const rect = anchorBlock.getBoundingClientRect()
      rects.push({
        x: rect.left,
        y: rect.top,
        width: rect.width,
        height: rect.height,
      })
    } else {
      // Multi-block selection - highlight both blocks
      const anchorRect = anchorBlock.getBoundingClientRect()
      const focusRect = focusBlock.getBoundingClientRect()
      rects.push(
        { x: anchorRect.left, y: anchorRect.top, width: anchorRect.width, height: anchorRect.height },
        { x: focusRect.left, y: focusRect.top, width: focusRect.width, height: focusRect.height }
      )
    }

    setHighlights(rects)
  }, [collaborator.selection])

  if (highlights.length === 0) return null

  return (
    <>
      {highlights.map((rect, i) => (
        <div
          key={i}
          style={{
            position: 'fixed',
            left: rect.x,
            top: rect.y,
            width: rect.width,
            height: rect.height,
            background: `${collaborator.color}20`,
            border: `1px solid ${collaborator.color}40`,
            borderRadius: '2px',
            pointerEvents: 'none',
            zIndex: 999,
          }}
        />
      ))}
    </>
  )
}

// Main collaborator cursors overlay
export function CollaboratorCursors() {
  const collaborators = useActiveCollaborators()

  return (
    <div className="collaborator-cursors" style={{ position: 'fixed', inset: 0, pointerEvents: 'none', zIndex: 9999 }}>
      <AnimatePresence>
        {collaborators.map((collaborator) => (
          <div key={collaborator.id}>
            <CollaboratorCursor collaborator={collaborator} />
            <CollaboratorSelection collaborator={collaborator} />
          </div>
        ))}
      </AnimatePresence>
    </div>
  )
}

// Presence avatars component - shows who's viewing the page
export function PresenceAvatars() {
  const collaborators = useActiveCollaborators()

  if (collaborators.length === 0) return null

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '-8px',
      }}
    >
      {collaborators.slice(0, 5).map((collaborator, index) => (
        <motion.div
          key={collaborator.id}
          initial={{ opacity: 0, scale: 0.8 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.8 }}
          style={{
            width: '28px',
            height: '28px',
            borderRadius: '50%',
            background: collaborator.avatarUrl
              ? `url(${collaborator.avatarUrl}) center/cover`
              : `linear-gradient(135deg, ${collaborator.color} 0%, ${adjustColor(collaborator.color, -20)} 100%)`,
            border: '2px solid var(--bg-primary)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '11px',
            fontWeight: 600,
            color: 'white',
            marginLeft: index > 0 ? '-8px' : 0,
            zIndex: collaborators.length - index,
            cursor: 'pointer',
            position: 'relative',
          }}
          title={collaborator.name}
        >
          {!collaborator.avatarUrl && collaborator.name.charAt(0).toUpperCase()}

          {/* Active indicator */}
          <div
            style={{
              position: 'absolute',
              bottom: '-1px',
              right: '-1px',
              width: '8px',
              height: '8px',
              borderRadius: '50%',
              background: '#22c55e',
              border: '2px solid var(--bg-primary)',
            }}
          />
        </motion.div>
      ))}

      {/* Overflow indicator */}
      {collaborators.length > 5 && (
        <div
          style={{
            width: '28px',
            height: '28px',
            borderRadius: '50%',
            background: 'var(--bg-secondary)',
            border: '2px solid var(--bg-primary)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '10px',
            fontWeight: 600,
            color: 'var(--text-secondary)',
            marginLeft: '-8px',
          }}
        >
          +{collaborators.length - 5}
        </div>
      )}
    </div>
  )
}

// Helper to adjust color brightness
function adjustColor(color: string, amount: number): string {
  const hex = color.replace('#', '')
  const r = Math.max(0, Math.min(255, parseInt(hex.substring(0, 2), 16) + amount))
  const g = Math.max(0, Math.min(255, parseInt(hex.substring(2, 4), 16) + amount))
  const b = Math.max(0, Math.min(255, parseInt(hex.substring(4, 6), 16) + amount))
  return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`
}
