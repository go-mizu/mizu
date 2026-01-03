/**
 * @deprecated This component is deprecated. Block menu functionality is now
 * integrated directly into BlockEditor.tsx via BlockNote's SideMenuController
 * and CustomDragHandleMenu components.
 *
 * The following features are now available through the side menu:
 * - Drag handle for block reordering (built into BlockNote)
 * - Delete block (RemoveBlockItem)
 * - Duplicate block (DuplicateBlockItem)
 * - Turn into another block type (TurnIntoItem)
 * - Block colors (BlockColorsItem)
 * - Copy link to block (CopyLinkItem)
 * - Move up/down (MoveUpItem, MoveDownItem)
 *
 * Keyboard shortcuts are also available:
 * - Cmd/Ctrl + Shift + Up/Down: Move block
 * - Cmd/Ctrl + D: Duplicate block
 * - Cmd/Ctrl + Shift + Backspace: Delete block
 * - Tab/Shift+Tab: Nest/unnest block
 */

// Re-export types for backwards compatibility if needed
export interface BlockMenuProps {
  blockId: string
  onDelete: () => void
  onDuplicate: () => void
  onCopyLink: () => void
  onTurnInto: (type: string, props?: Record<string, unknown>) => void
  onAddComment?: () => void
  position?: { x: number; y: number }
}

// This component is deprecated - use BlockNote's built-in SideMenuController instead
export function BlockMenu(_props: BlockMenuProps) {
  console.warn(
    'BlockMenu is deprecated. Block menu functionality is now integrated into BlockEditor via SideMenuController.'
  )
  return null
}

export function BlockWrapper({
  children,
}: {
  blockId: string
  children: React.ReactNode
  onDelete: () => void
  onDuplicate: () => void
  onCopyLink: () => void
  onTurnInto: (type: string, props?: Record<string, unknown>) => void
  onAddComment?: () => void
}) {
  console.warn(
    'BlockWrapper is deprecated. Block wrapping is now handled by BlockNote internally.'
  )
  return <>{children}</>
}
