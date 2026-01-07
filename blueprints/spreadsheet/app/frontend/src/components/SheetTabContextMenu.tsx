import React, { useEffect, useRef } from 'react';

export interface SheetTabContextMenuProps {
  position: { x: number; y: number };
  sheetId: string;
  sheetName: string;
  onClose: () => void;
  onDelete: (sheetId: string) => void;
  onDuplicate: (sheetId: string) => void;
  onRename: (sheetId: string) => void;
  onChangeColor: (sheetId: string, color: string) => void;
  onHide: (sheetId: string) => void;
  canDelete: boolean;
}

const TAB_COLORS = [
  { name: 'Red', value: '#ea4335' },
  { name: 'Orange', value: '#fa7b17' },
  { name: 'Yellow', value: '#f9ab00' },
  { name: 'Green', value: '#34a853' },
  { name: 'Cyan', value: '#24c1e0' },
  { name: 'Blue', value: '#4285f4' },
  { name: 'Purple', value: '#a142f4' },
  { name: 'Pink', value: '#f538a0' },
  { name: 'None', value: '' },
];

export const SheetTabContextMenu: React.FC<SheetTabContextMenuProps> = ({
  position,
  sheetId,
  onClose,
  onDelete,
  onDuplicate,
  onRename,
  onChangeColor,
  onHide,
  canDelete,
}) => {
  const menuRef = useRef<HTMLDivElement>(null);
  const [showColorPicker, setShowColorPicker] = React.useState(false);

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

  const handleAction = (action: () => void) => {
    action();
    onClose();
  };

  return (
    <div
      ref={menuRef}
      className="sheet-tab-context-menu"
      style={{
        position: 'fixed',
        left: position.x,
        top: position.y,
        zIndex: 1000,
      }}
    >
      <button
        className="context-menu-item"
        onClick={() => handleAction(() => onRename(sheetId))}
      >
        <span className="context-menu-icon"><RenameIcon /></span>
        <span className="context-menu-label">Rename</span>
      </button>

      <button
        className="context-menu-item"
        onClick={() => handleAction(() => onDuplicate(sheetId))}
      >
        <span className="context-menu-icon"><DuplicateIcon /></span>
        <span className="context-menu-label">Duplicate</span>
      </button>

      <div className="context-menu-divider" />

      <div
        className="context-menu-item has-submenu"
        onMouseEnter={() => setShowColorPicker(true)}
        onMouseLeave={() => setShowColorPicker(false)}
      >
        <span className="context-menu-icon"><ColorIcon /></span>
        <span className="context-menu-label">Change color</span>
        <span className="context-menu-arrow"><ArrowRightIcon /></span>

        {showColorPicker && (
          <div className="color-submenu">
            <div className="color-grid">
              {TAB_COLORS.map((color) => (
                <button
                  key={color.name}
                  className={`color-option ${color.value === '' ? 'no-color' : ''}`}
                  style={{ backgroundColor: color.value || '#ffffff' }}
                  title={color.name}
                  onClick={() => handleAction(() => onChangeColor(sheetId, color.value))}
                >
                  {color.value === '' && <NoColorIcon />}
                </button>
              ))}
            </div>
          </div>
        )}
      </div>

      <div className="context-menu-divider" />

      <button
        className="context-menu-item"
        onClick={() => handleAction(() => onHide(sheetId))}
      >
        <span className="context-menu-icon"><HideIcon /></span>
        <span className="context-menu-label">Hide sheet</span>
      </button>

      <button
        className={`context-menu-item ${!canDelete ? 'disabled' : ''}`}
        onClick={() => canDelete && handleAction(() => onDelete(sheetId))}
        disabled={!canDelete}
      >
        <span className="context-menu-icon"><DeleteIcon /></span>
        <span className="context-menu-label">Delete</span>
      </button>
    </div>
  );
};

const RenameIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
  </svg>
);

const DuplicateIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="9" y="9" width="13" height="13" rx="2" />
    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
  </svg>
);

const ColorIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10" />
    <path d="M12 2v10l8.5 5" />
  </svg>
);

const HideIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
    <line x1="1" y1="1" x2="23" y2="23" />
  </svg>
);

const DeleteIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M3 6h18" />
    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" />
    <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
  </svg>
);

const ArrowRightIcon = () => (
  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="9 18 15 12 9 6" />
  </svg>
);

const NoColorIcon = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#999" strokeWidth="2">
    <line x1="4" y1="4" x2="20" y2="20" />
  </svg>
);

export default SheetTabContextMenu;
