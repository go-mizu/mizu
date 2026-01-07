import React, { useCallback, useState } from 'react';
import type { CellFormat } from '../types';
import { getCurrencyFormat, getPercentFormat, increaseDecimalPlaces, decreaseDecimalPlaces } from '../utils/numberFormat';

interface ToolbarProps {
  onUndo: () => void;
  onRedo: () => void;
  canUndo: boolean;
  canRedo: boolean;
  onFormatChange: (format: Partial<CellFormat>) => void;
  currentFormat: CellFormat | undefined;
  onFind: () => void;
  onMergeCells: () => void;
  onUnmergeCells: () => void;
  canMerge: boolean;
  hasMergedCells: boolean;
  zoom?: number;
  onZoomChange?: (zoom: number) => void;
  onPrint?: () => void;
  onFormatPainter?: () => void;
  isFormatPainterActive?: boolean;
  onInsertLink?: () => void;
  onInsertComment?: () => void;
  onApplyBorder?: (type: string, border: { style: string; color: string } | null) => void;
}

// Icons
const UndoIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M3 10h10a5 5 0 0 1 5 5v2M3 10l5-5M3 10l5 5" />
  </svg>
);

const RedoIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M21 10H11a5 5 0 0 0-5 5v2M21 10l-5-5M21 10l-5 5" />
  </svg>
);

const PrintIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M6 9V2h12v7" />
    <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2" />
    <rect x="6" y="14" width="12" height="8" />
  </svg>
);

const FormatPainterIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M4 20h4l10.5 -10.5a2.828 2.828 0 1 0 -4 -4l-10.5 10.5v4" />
    <path d="M13.5 6.5l4 4" />
  </svg>
);

const CurrencyIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="12" y1="1" x2="12" y2="23" />
    <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
  </svg>
);

const PercentIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="19" y1="5" x2="5" y2="19" />
    <circle cx="6.5" cy="6.5" r="2.5" />
    <circle cx="17.5" cy="17.5" r="2.5" />
  </svg>
);

const DecimalDecreaseIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <text x="2" y="16" fontSize="10" fill="currentColor">.0</text>
    <path d="M16 12l4-4m0 0l4 4m-4-4v8" />
  </svg>
);

const DecimalIncreaseIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <text x="2" y="16" fontSize="10" fill="currentColor">.00</text>
    <path d="M16 16l4 4m0 0l4-4m-4 4v-8" />
  </svg>
);

const AlignLeftIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="3" y1="6" x2="21" y2="6" />
    <line x1="3" y1="12" x2="15" y2="12" />
    <line x1="3" y1="18" x2="18" y2="18" />
  </svg>
);

const AlignCenterIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="3" y1="6" x2="21" y2="6" />
    <line x1="6" y1="12" x2="18" y2="12" />
    <line x1="4" y1="18" x2="20" y2="18" />
  </svg>
);

const AlignRightIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="3" y1="6" x2="21" y2="6" />
    <line x1="9" y1="12" x2="21" y2="12" />
    <line x1="6" y1="18" x2="21" y2="18" />
  </svg>
);

const WrapTextIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="3" y1="6" x2="21" y2="6" />
    <path d="M3 12h15a3 3 0 1 1 0 6h-4" />
    <polyline points="12 15 15 18 12 21" />
  </svg>
);

const MergeIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="3" y="3" width="18" height="18" rx="2" />
    <path d="M9 3v18" />
    <path d="M3 9h18" />
    <path d="M7 12h10M12 7v10" />
  </svg>
);

const SearchIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="11" cy="11" r="8" />
    <line x1="21" y1="21" x2="16.65" y2="16.65" />
  </svg>
);

const LinkIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
    <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
  </svg>
);

const CommentIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
  </svg>
);

const AlignTopIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="4" y1="4" x2="20" y2="4" />
    <rect x="8" y="8" width="8" height="10" rx="1" />
  </svg>
);

const AlignMiddleIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="4" y1="12" x2="20" y2="12" />
    <rect x="8" y="6" width="8" height="12" rx="1" />
  </svg>
);

const AlignBottomIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <line x1="4" y1="20" x2="20" y2="20" />
    <rect x="8" y="6" width="8" height="10" rx="1" />
  </svg>
);

const BorderIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="3" y="3" width="18" height="18" rx="2" />
  </svg>
);

const FillColorIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M19 11l-8-8-8.6 8.6a2 2 0 0 0 0 2.8l5.2 5.2a2 2 0 0 0 2.8 0L19 11z" />
    <path d="M5 21h14" strokeWidth="3" stroke="currentColor" />
  </svg>
);

const TextColorIcon = () => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <path d="M4 20h16" strokeWidth="3" />
    <path d="M12 4L6 16h2l1.5-3h5l1.5 3h2L12 4z" />
  </svg>
);

const FontSizes = [6, 7, 8, 9, 10, 11, 12, 14, 16, 18, 20, 22, 24, 26, 28, 36, 48, 72];
const FontFamilies = [
  'Arial',
  'Helvetica',
  'Times New Roman',
  'Georgia',
  'Courier New',
  'Verdana',
  'Trebuchet MS',
  'Comic Sans MS',
];

// Border Type Icons
const AllBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" />
    <line x1="9" y1="1" x2="9" y2="17" />
    <line x1="1" y1="9" x2="17" y2="9" />
  </svg>
);

const OuterBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" />
  </svg>
);

const NoBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="4" y1="4" x2="14" y2="14" strokeWidth="2" stroke="#ea4335" />
    <line x1="14" y1="4" x2="4" y2="14" strokeWidth="2" stroke="#ea4335" />
  </svg>
);

const BottomBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="1" y1="17" x2="17" y2="17" strokeWidth="2" />
  </svg>
);

const TopBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="1" y1="1" x2="17" y2="1" strokeWidth="2" />
  </svg>
);

// Borders Menu Component
interface BordersMenuProps {
  onApplyBorder?: (type: string, border: { style: string; color: string } | null) => void;
}

const BordersMenu: React.FC<BordersMenuProps> = ({ onApplyBorder }) => {
  const [isOpen, setIsOpen] = useState(false);

  const borderOptions = [
    { type: 'all', label: 'All borders', icon: <AllBordersIcon /> },
    { type: 'outer', label: 'Outer borders', icon: <OuterBordersIcon /> },
    { type: 'bottom', label: 'Bottom border', icon: <BottomBorderIcon /> },
    { type: 'top', label: 'Top border', icon: <TopBorderIcon /> },
    { type: 'clear', label: 'Clear borders', icon: <NoBordersIcon /> },
  ];

  const handleSelect = (type: string) => {
    if (onApplyBorder) {
      if (type === 'clear') {
        onApplyBorder(type, null);
      } else {
        onApplyBorder(type, { style: 'thin', color: '#000000' });
      }
    }
    setIsOpen(false);
  };

  return (
    <div className="borders-dropdown" style={{ position: 'relative' }}>
      <button
        title="Borders"
        onClick={() => setIsOpen(!isOpen)}
        className={isOpen ? 'active' : ''}
      >
        <BorderIcon />
        <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" style={{ marginLeft: 2 }}>
          <polyline points="6 9 12 15 18 9" />
        </svg>
      </button>
      {isOpen && (
        <div className="borders-menu" style={{
          position: 'absolute',
          top: '100%',
          left: 0,
          background: 'white',
          border: '1px solid #dadce0',
          borderRadius: 4,
          boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
          zIndex: 100,
          minWidth: 150,
          padding: '4px 0',
        }}>
          {borderOptions.map(({ type, label, icon }) => (
            <button
              key={type}
              className="borders-option"
              onClick={() => handleSelect(type)}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                width: '100%',
                padding: '6px 12px',
                border: 'none',
                background: 'none',
                cursor: 'pointer',
                fontSize: 13,
              }}
              onMouseEnter={(e) => (e.currentTarget.style.background = '#f1f3f4')}
              onMouseLeave={(e) => (e.currentTarget.style.background = 'none')}
            >
              {icon}
              <span>{label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export const Toolbar: React.FC<ToolbarProps> = ({
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onFormatChange,
  currentFormat,
  onFind,
  onMergeCells,
  onUnmergeCells,
  canMerge,
  hasMergedCells,
  zoom = 100,
  onZoomChange,
  onPrint,
  onFormatPainter,
  isFormatPainterActive = false,
  onInsertLink,
  onInsertComment,
  onApplyBorder,
}) => {
  const handleFontFamilyChange = useCallback((e: React.ChangeEvent<HTMLSelectElement>) => {
    onFormatChange({ fontFamily: e.target.value });
  }, [onFormatChange]);

  const handleFontSizeChange = useCallback((e: React.ChangeEvent<HTMLSelectElement>) => {
    onFormatChange({ fontSize: parseInt(e.target.value, 10) });
  }, [onFormatChange]);

  const toggleBold = useCallback(() => {
    onFormatChange({ bold: !currentFormat?.bold });
  }, [onFormatChange, currentFormat]);

  const toggleItalic = useCallback(() => {
    onFormatChange({ italic: !currentFormat?.italic });
  }, [onFormatChange, currentFormat]);

  const toggleUnderline = useCallback(() => {
    onFormatChange({ underline: !currentFormat?.underline });
  }, [onFormatChange, currentFormat]);

  const toggleStrikethrough = useCallback(() => {
    onFormatChange({ strikethrough: !currentFormat?.strikethrough });
  }, [onFormatChange, currentFormat]);

  const setAlignment = useCallback((align: 'left' | 'center' | 'right') => {
    onFormatChange({ hAlign: align });
  }, [onFormatChange]);

  const setVerticalAlignment = useCallback((align: 'top' | 'middle' | 'bottom') => {
    onFormatChange({ vAlign: align });
  }, [onFormatChange]);

  const toggleWrapText = useCallback(() => {
    onFormatChange({ wrapText: !currentFormat?.wrapText });
  }, [onFormatChange, currentFormat]);

  const handleTextColor = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    onFormatChange({ fontColor: e.target.value });
  }, [onFormatChange]);

  const handleFillColor = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    onFormatChange({ backgroundColor: e.target.value });
  }, [onFormatChange]);

  // Number formatting handlers
  const handleCurrency = useCallback(() => {
    onFormatChange({ numberFormat: getCurrencyFormat() });
  }, [onFormatChange]);

  const handlePercent = useCallback(() => {
    onFormatChange({ numberFormat: getPercentFormat() });
  }, [onFormatChange]);

  const handleDecimalIncrease = useCallback(() => {
    onFormatChange({ numberFormat: increaseDecimalPlaces(currentFormat?.numberFormat) });
  }, [onFormatChange, currentFormat]);

  const handleDecimalDecrease = useCallback(() => {
    onFormatChange({ numberFormat: decreaseDecimalPlaces(currentFormat?.numberFormat) });
  }, [onFormatChange, currentFormat]);

  const handleZoomSelect = useCallback((e: React.ChangeEvent<HTMLSelectElement>) => {
    if (onZoomChange) {
      onZoomChange(parseInt(e.target.value, 10));
    }
  }, [onZoomChange]);

  return (
    <div className="toolbar">
      {/* Undo/Redo */}
      <div className="toolbar-group">
        <button
          title="Undo (Ctrl+Z)"
          onClick={onUndo}
          disabled={!canUndo}
          className={!canUndo ? 'disabled' : ''}
        >
          <UndoIcon />
        </button>
        <button
          title="Redo (Ctrl+Y)"
          onClick={onRedo}
          disabled={!canRedo}
          className={!canRedo ? 'disabled' : ''}
        >
          <RedoIcon />
        </button>
        <button title="Print (Ctrl+P)" onClick={onPrint}>
          <PrintIcon />
        </button>
        <button
          title="Format painter"
          onClick={onFormatPainter}
          className={isFormatPainterActive ? 'active' : ''}
        >
          <FormatPainterIcon />
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Zoom */}
      <div className="toolbar-group">
        <select className="zoom-select" value={zoom} onChange={handleZoomSelect}>
          <option value="50">50%</option>
          <option value="75">75%</option>
          <option value="90">90%</option>
          <option value="100">100%</option>
          <option value="125">125%</option>
          <option value="150">150%</option>
          <option value="200">200%</option>
        </select>
      </div>

      <div className="toolbar-divider" />

      {/* Font */}
      <div className="toolbar-group">
        <select
          className="font-select"
          value={currentFormat?.fontFamily || 'Arial'}
          onChange={handleFontFamilyChange}
        >
          {FontFamilies.map(font => (
            <option key={font} value={font}>{font}</option>
          ))}
        </select>
        <select
          className="size-select"
          value={currentFormat?.fontSize || 10}
          onChange={handleFontSizeChange}
        >
          {FontSizes.map(size => (
            <option key={size} value={size}>{size}</option>
          ))}
        </select>
      </div>

      <div className="toolbar-divider" />

      {/* Text formatting */}
      <div className="toolbar-group">
        <button
          title="Bold (Ctrl+B)"
          className={`format-btn ${currentFormat?.bold ? 'active' : ''}`}
          onClick={toggleBold}
        >
          <strong>B</strong>
        </button>
        <button
          title="Italic (Ctrl+I)"
          className={`format-btn ${currentFormat?.italic ? 'active' : ''}`}
          onClick={toggleItalic}
        >
          <em>I</em>
        </button>
        <button
          title="Underline (Ctrl+U)"
          className={`format-btn ${currentFormat?.underline ? 'active' : ''}`}
          onClick={toggleUnderline}
        >
          <u>U</u>
        </button>
        <button
          title="Strikethrough (Ctrl+5)"
          className={`format-btn ${currentFormat?.strikethrough ? 'active' : ''}`}
          onClick={toggleStrikethrough}
        >
          <s>S</s>
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Colors */}
      <div className="toolbar-group">
        <label className="color-picker-btn" title="Text color">
          <TextColorIcon />
          <input
            type="color"
            value={currentFormat?.fontColor || '#000000'}
            onChange={handleTextColor}
            className="color-input"
          />
          <div
            className="color-indicator"
            style={{ backgroundColor: currentFormat?.fontColor || '#000000' }}
          />
        </label>
        <label className="color-picker-btn" title="Fill color">
          <FillColorIcon />
          <input
            type="color"
            value={currentFormat?.backgroundColor || '#ffffff'}
            onChange={handleFillColor}
            className="color-input"
          />
          <div
            className="color-indicator"
            style={{ backgroundColor: currentFormat?.backgroundColor || '#ffffff' }}
          />
        </label>
        <BordersMenu onApplyBorder={onApplyBorder} />
      </div>

      <div className="toolbar-divider" />

      {/* Number format */}
      <div className="toolbar-group">
        <button title="Format as currency" onClick={handleCurrency}>
          <CurrencyIcon />
        </button>
        <button title="Format as percent" onClick={handlePercent}>
          <PercentIcon />
        </button>
        <button title="Decrease decimal places" onClick={handleDecimalDecrease}>
          <DecimalDecreaseIcon />
        </button>
        <button title="Increase decimal places" onClick={handleDecimalIncrease}>
          <DecimalIncreaseIcon />
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Horizontal Alignment */}
      <div className="toolbar-group">
        <button
          title="Align left"
          className={currentFormat?.hAlign === 'left' ? 'active' : ''}
          onClick={() => setAlignment('left')}
        >
          <AlignLeftIcon />
        </button>
        <button
          title="Align center"
          className={currentFormat?.hAlign === 'center' ? 'active' : ''}
          onClick={() => setAlignment('center')}
        >
          <AlignCenterIcon />
        </button>
        <button
          title="Align right"
          className={currentFormat?.hAlign === 'right' ? 'active' : ''}
          onClick={() => setAlignment('right')}
        >
          <AlignRightIcon />
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Vertical Alignment */}
      <div className="toolbar-group">
        <button
          title="Align top"
          className={currentFormat?.vAlign === 'top' ? 'active' : ''}
          onClick={() => setVerticalAlignment('top')}
        >
          <AlignTopIcon />
        </button>
        <button
          title="Align middle"
          className={currentFormat?.vAlign === 'middle' ? 'active' : ''}
          onClick={() => setVerticalAlignment('middle')}
        >
          <AlignMiddleIcon />
        </button>
        <button
          title="Align bottom"
          className={currentFormat?.vAlign === 'bottom' ? 'active' : ''}
          onClick={() => setVerticalAlignment('bottom')}
        >
          <AlignBottomIcon />
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Text operations */}
      <div className="toolbar-group">
        <button
          title="Wrap text"
          className={currentFormat?.wrapText ? 'active' : ''}
          onClick={toggleWrapText}
        >
          <WrapTextIcon />
        </button>
        <button
          title={hasMergedCells ? 'Unmerge cells' : 'Merge cells'}
          className={hasMergedCells ? 'active' : ''}
          onClick={hasMergedCells ? onUnmergeCells : onMergeCells}
          disabled={!canMerge && !hasMergedCells}
        >
          <MergeIcon />
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Insert */}
      <div className="toolbar-group">
        <button title="Insert link" onClick={onInsertLink}>
          <LinkIcon />
        </button>
        <button title="Insert comment" onClick={onInsertComment}>
          <CommentIcon />
        </button>
      </div>

      <div className="toolbar-divider" />

      {/* Find */}
      <div className="toolbar-group">
        <button title="Find and replace (Ctrl+F)" onClick={onFind}>
          <SearchIcon />
        </button>
      </div>
    </div>
  );
};

export default Toolbar;
