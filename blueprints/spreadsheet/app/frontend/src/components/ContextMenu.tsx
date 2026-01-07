import React, { useEffect, useRef } from 'react';
import type { Selection } from '../types';

export interface ContextMenuItem {
  id: string;
  label: string;
  icon?: React.ReactNode;
  shortcut?: string;
  action: () => void;
  disabled?: boolean;
  divider?: boolean;
}

export interface ContextMenuProps {
  position: { x: number; y: number };
  selection: Selection | null;
  onClose: () => void;
  onCut: () => void;
  onCopy: () => void;
  onPaste: () => void;
  onPasteValuesOnly: () => void;
  onClearContents: () => void;
  onInsertRowAbove: () => void;
  onInsertRowBelow: () => void;
  onInsertColLeft: () => void;
  onInsertColRight: () => void;
  onDeleteRow: () => void;
  onDeleteCol: () => void;
  onMergeCells: () => void;
  onUnmergeCells: () => void;
  hasMergedCells: boolean;
  canMerge: boolean;
}

const CutIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="6" cy="6" r="3" />
    <circle cx="6" cy="18" r="3" />
    <line x1="20" y1="4" x2="8.12" y2="15.88" />
    <line x1="14.47" y1="14.48" x2="20" y2="20" />
    <line x1="8.12" y1="8.12" x2="12" y2="12" />
  </svg>
);

const CopyIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="9" y="9" width="13" height="13" rx="2" />
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
  </svg>
);

const PasteIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2" />
    <rect x="8" y="2" width="8" height="4" rx="1" />
  </svg>
);

const DeleteIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M3 6h18" />
    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
    <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
  </svg>
);

const InsertRowIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <line x1="3" y1="12" x2="21" y2="12" />
    <line x1="12" y1="8" x2="12" y2="16" />
  </svg>
);

const InsertColIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <line x1="12" y1="3" x2="12" y2="21" />
    <line x1="8" y1="12" x2="16" y2="12" />
  </svg>
);

const MergeIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <path d="M9 3v18" />
    <path d="M3 9h18" />
  </svg>
);

export const ContextMenu: React.FC<ContextMenuProps> = ({
  position,
  onClose,
  onCut,
  onCopy,
  onPaste,
  onPasteValuesOnly,
  onClearContents,
  onInsertRowAbove,
  onInsertRowBelow,
  onInsertColLeft,
  onInsertColRight,
  onDeleteRow,
  onDeleteCol,
  onMergeCells,
  onUnmergeCells,
  hasMergedCells,
  canMerge,
}) => {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        onClose();
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [onClose]);

  // Adjust position to keep menu in viewport
  useEffect(() => {
    if (menuRef.current) {
      const rect = menuRef.current.getBoundingClientRect();
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      if (rect.right > viewportWidth) {
        menuRef.current.style.left = `${position.x - rect.width}px`;
      }
      if (rect.bottom > viewportHeight) {
        menuRef.current.style.top = `${position.y - rect.height}px`;
      }
    }
  }, [position]);

  const handleItemClick = (action: () => void) => {
    action();
    onClose();
  };

  return (
    <div
      ref={menuRef}
      className="context-menu"
      style={{
        position: 'fixed',
        left: position.x,
        top: position.y,
        zIndex: 1000,
      }}
    >
      <div className="context-menu-section">
        <button className="context-menu-item" onClick={() => handleItemClick(onCut)}>
          <span className="context-menu-icon"><CutIcon /></span>
          <span className="context-menu-label">Cut</span>
          <span className="context-menu-shortcut">Ctrl+X</span>
        </button>
        <button className="context-menu-item" onClick={() => handleItemClick(onCopy)}>
          <span className="context-menu-icon"><CopyIcon /></span>
          <span className="context-menu-label">Copy</span>
          <span className="context-menu-shortcut">Ctrl+C</span>
        </button>
        <button className="context-menu-item" onClick={() => handleItemClick(onPaste)}>
          <span className="context-menu-icon"><PasteIcon /></span>
          <span className="context-menu-label">Paste</span>
          <span className="context-menu-shortcut">Ctrl+V</span>
        </button>
        <button className="context-menu-item" onClick={() => handleItemClick(onPasteValuesOnly)}>
          <span className="context-menu-icon"></span>
          <span className="context-menu-label">Paste values only</span>
          <span className="context-menu-shortcut">Ctrl+Shift+V</span>
        </button>
      </div>

      <div className="context-menu-divider" />

      <div className="context-menu-section">
        <button className="context-menu-item" onClick={() => handleItemClick(onClearContents)}>
          <span className="context-menu-icon"><DeleteIcon /></span>
          <span className="context-menu-label">Clear contents</span>
          <span className="context-menu-shortcut">Delete</span>
        </button>
      </div>

      <div className="context-menu-divider" />

      <div className="context-menu-section">
        <button className="context-menu-item" onClick={() => handleItemClick(onInsertRowAbove)}>
          <span className="context-menu-icon"><InsertRowIcon /></span>
          <span className="context-menu-label">Insert row above</span>
          <span className="context-menu-shortcut"></span>
        </button>
        <button className="context-menu-item" onClick={() => handleItemClick(onInsertRowBelow)}>
          <span className="context-menu-icon"><InsertRowIcon /></span>
          <span className="context-menu-label">Insert row below</span>
          <span className="context-menu-shortcut"></span>
        </button>
        <button className="context-menu-item" onClick={() => handleItemClick(onDeleteRow)}>
          <span className="context-menu-icon"><DeleteIcon /></span>
          <span className="context-menu-label">Delete row</span>
          <span className="context-menu-shortcut"></span>
        </button>
      </div>

      <div className="context-menu-divider" />

      <div className="context-menu-section">
        <button className="context-menu-item" onClick={() => handleItemClick(onInsertColLeft)}>
          <span className="context-menu-icon"><InsertColIcon /></span>
          <span className="context-menu-label">Insert column left</span>
          <span className="context-menu-shortcut"></span>
        </button>
        <button className="context-menu-item" onClick={() => handleItemClick(onInsertColRight)}>
          <span className="context-menu-icon"><InsertColIcon /></span>
          <span className="context-menu-label">Insert column right</span>
          <span className="context-menu-shortcut"></span>
        </button>
        <button className="context-menu-item" onClick={() => handleItemClick(onDeleteCol)}>
          <span className="context-menu-icon"><DeleteIcon /></span>
          <span className="context-menu-label">Delete column</span>
          <span className="context-menu-shortcut"></span>
        </button>
      </div>

      {canMerge && (
        <>
          <div className="context-menu-divider" />
          <div className="context-menu-section">
            {hasMergedCells ? (
              <button className="context-menu-item" onClick={() => handleItemClick(onUnmergeCells)}>
                <span className="context-menu-icon"><MergeIcon /></span>
                <span className="context-menu-label">Unmerge cells</span>
                <span className="context-menu-shortcut"></span>
              </button>
            ) : (
              <button className="context-menu-item" onClick={() => handleItemClick(onMergeCells)}>
                <span className="context-menu-icon"><MergeIcon /></span>
                <span className="context-menu-label">Merge cells</span>
                <span className="context-menu-shortcut"></span>
              </button>
            )}
          </div>
        </>
      )}
    </div>
  );
};

export default ContextMenu;
