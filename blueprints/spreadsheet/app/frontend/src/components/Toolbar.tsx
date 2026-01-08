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

// Google Sheets style icons - clean, simple, modern design
const UndoIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M12.5 8c-2.65 0-5.05.99-6.9 2.6L2 7v9h9l-3.62-3.62c1.39-1.16 3.16-1.88 5.12-1.88 3.54 0 6.55 2.31 7.6 5.5l2.37-.78C21.08 11.03 17.15 8 12.5 8z"/>
  </svg>
);

const RedoIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M18.4 10.6C16.55 8.99 14.15 8 11.5 8c-4.65 0-8.58 3.03-9.96 7.22L3.9 16c1.05-3.19 4.05-5.5 7.6-5.5 1.95 0 3.73.72 5.12 1.88L13 16h9V7l-3.6 3.6z"/>
  </svg>
);

const PrintIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M19 8H5c-1.66 0-3 1.34-3 3v6h4v4h12v-4h4v-6c0-1.66-1.34-3-3-3zm-3 11H8v-5h8v5zm3-7c-.55 0-1-.45-1-1s.45-1 1-1 1 .45 1 1-.45 1-1 1zm-1-9H6v4h12V3z"/>
  </svg>
);

const FormatPainterIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M18 4V3c0-.55-.45-1-1-1H5c-.55 0-1 .45-1 1v4c0 .55.45 1 1 1h12c.55 0 1-.45 1-1V6h1v4H9v11c0 .55.45 1 1 1h2c.55 0 1-.45 1-1v-9h8V4h-3z"/>
  </svg>
);

const CurrencyIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M11.8 10.9c-2.27-.59-3-1.2-3-2.15 0-1.09 1.01-1.85 2.7-1.85 1.78 0 2.44.85 2.5 2.1h2.21c-.07-1.72-1.12-3.3-3.21-3.81V3h-3v2.16c-1.94.42-3.5 1.68-3.5 3.61 0 2.31 1.91 3.46 4.7 4.13 2.5.6 3 1.48 3 2.41 0 .69-.49 1.79-2.7 1.79-2.06 0-2.87-.92-2.98-2.1h-2.2c.12 2.19 1.76 3.42 3.68 3.83V21h3v-2.15c1.95-.37 3.5-1.5 3.5-3.55 0-2.84-2.43-3.81-4.7-4.4z"/>
  </svg>
);

const PercentIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M7.5 11C9.43 11 11 9.43 11 7.5S9.43 4 7.5 4 4 5.57 4 7.5 5.57 11 7.5 11zm0-5C8.33 6 9 6.67 9 7.5S8.33 9 7.5 9 6 8.33 6 7.5 6.67 6 7.5 6zM4.41 19.5L19.5 4.42 20.91 5.84 5.83 20.92zM16.5 13c-1.93 0-3.5 1.57-3.5 3.5s1.57 3.5 3.5 3.5 3.5-1.57 3.5-3.5-1.57-3.5-3.5-3.5zm0 5c-.83 0-1.5-.67-1.5-1.5s.67-1.5 1.5-1.5 1.5.67 1.5 1.5-.67 1.5-1.5 1.5z"/>
  </svg>
);

const DecimalDecreaseIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M12 7l5 5H7z"/>
    <path d="M4 17h2v2H4zm4 0h2v2H8z"/>
    <circle cx="3" cy="18" r="1.5"/>
  </svg>
);

const DecimalIncreaseIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M12 17l-5-5h10z"/>
    <path d="M4 5h2v2H4zm4 0h2v2H8zm4 0h2v2h-2z"/>
    <circle cx="3" cy="6" r="1.5"/>
  </svg>
);

const AlignLeftIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M15 15H3v2h12v-2zm0-8H3v2h12V7zM3 13h18v-2H3v2zm0 8h18v-2H3v2zM3 3v2h18V3H3z"/>
  </svg>
);

const AlignCenterIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M7 15v2h10v-2H7zm-4 6h18v-2H3v2zm0-8h18v-2H3v2zm4-6v2h10V7H7zM3 3v2h18V3H3z"/>
  </svg>
);

const AlignRightIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M3 21h18v-2H3v2zm6-4h12v-2H9v2zm-6-4h18v-2H3v2zm6-4h12V7H9v2zM3 3v2h18V3H3z"/>
  </svg>
);

const WrapTextIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M4 19h6v-2H4v2zM20 5H4v2h16V5zm-3 6H4v2h13.25c1.1 0 2 .9 2 2s-.9 2-2 2H15v-2l-3 3 3 3v-2h2c2.21 0 4-1.79 4-4s-1.79-4-4-4z"/>
  </svg>
);

const MergeIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M5 10h2V8h10v2h2V8c0-1.1-.9-2-2-2H7c-1.1 0-2 .9-2 2v2zm0 4h2v2h10v-2h2v2c0 1.1-.9 2-2 2H7c-1.1 0-2-.9-2-2v-2zm12-3H7v2h10v-2z"/>
  </svg>
);

const SearchIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
  </svg>
);

const LinkIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M3.9 12c0-1.71 1.39-3.1 3.1-3.1h4V7H7c-2.76 0-5 2.24-5 5s2.24 5 5 5h4v-1.9H7c-1.71 0-3.1-1.39-3.1-3.1zM8 13h8v-2H8v2zm9-6h-4v1.9h4c1.71 0 3.1 1.39 3.1 3.1s-1.39 3.1-3.1 3.1h-4V17h4c2.76 0 5-2.24 5-5s-2.24-5-5-5z"/>
  </svg>
);

const CommentIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M21.99 4c0-1.1-.89-2-1.99-2H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h14l4 4-.01-18zM18 14H6v-2h12v2zm0-3H6V9h12v2zm0-3H6V6h12v2z"/>
  </svg>
);

const AlignTopIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M8 11h3v10h2V11h3l-4-4-4 4zM4 3v2h16V3H4z"/>
  </svg>
);

const AlignMiddleIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M8 19h3v4h2v-4h3l-4-4-4 4zm8-14h-3V1h-2v4H8l4 4 4-4zM4 11v2h16v-2H4z"/>
  </svg>
);

const AlignBottomIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M16 13h-3V3h-2v10H8l4 4 4-4zM4 19v2h16v-2H4z"/>
  </svg>
);

const BorderIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M13 7h-2v2h2V7zm0 4h-2v2h2v-2zm4 0h-2v2h2v-2zM3 3v18h18V3H3zm16 16H5V5h14v14zm-6-4h-2v2h2v-2zm-4-4H7v2h2v-2z"/>
  </svg>
);

const FillColorIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M16.56 8.94L7.62 0 6.21 1.41l2.38 2.38-5.15 5.15c-.59.59-.59 1.54 0 2.12l5.5 5.5c.29.29.68.44 1.06.44s.77-.15 1.06-.44l5.5-5.5c.59-.58.59-1.53 0-2.12zM5.21 10L10 5.21 14.79 10H5.21zM19 11.5s-2 2.17-2 3.5c0 1.1.9 2 2 2s2-.9 2-2c0-1.33-2-3.5-2-3.5z"/>
    <path d="M2 20h20v4H2z" fillOpacity="0.36"/>
  </svg>
);

const TextColorIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M11 3L5.5 17h2.25l1.12-3h6.25l1.12 3h2.25L13 3h-2zm-1.38 9L12 5.67 14.38 12H9.62z"/>
    <path d="M2 20h20v4H2z" fillOpacity="0.36"/>
  </svg>
);

// Text formatting icons (B, I, U, S) - Google style
const BoldIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M15.6 10.79c.97-.67 1.65-1.77 1.65-2.79 0-2.26-1.75-4-4-4H7v14h7.04c2.09 0 3.71-1.7 3.71-3.79 0-1.52-.86-2.82-2.15-3.42zM10 6.5h3c.83 0 1.5.67 1.5 1.5s-.67 1.5-1.5 1.5h-3v-3zm3.5 9H10v-3h3.5c.83 0 1.5.67 1.5 1.5s-.67 1.5-1.5 1.5z"/>
  </svg>
);

const ItalicIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M10 4v3h2.21l-3.42 8H6v3h8v-3h-2.21l3.42-8H18V4z"/>
  </svg>
);

const UnderlineIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M12 17c3.31 0 6-2.69 6-6V3h-2.5v8c0 1.93-1.57 3.5-3.5 3.5S8.5 12.93 8.5 11V3H6v8c0 3.31 2.69 6 6 6zm-7 2v2h14v-2H5z"/>
  </svg>
);

const StrikethroughIcon = () => (
  <svg viewBox="0 0 24 24" fill="currentColor">
    <path d="M10 19h4v-3h-4v3zM5 4v3h5v3h4V7h5V4H5zM3 14h18v-2H3v2z"/>
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

// Border Type Icons - Google Sheets style
const AllBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="currentColor">
    <path d="M1 1h16v16H1V1zm1 1v6h6V2H2zm7 0v6h6V2H9zM2 9v6h6V9H2zm7 0v6h6V9H9z"/>
  </svg>
);

const OuterBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="currentColor">
    <path d="M1 1h16v16H1V1zm1 1v14h14V2H2z"/>
  </svg>
);

const NoBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="currentColor">
    <path d="M1 1h2v2H1V1zm4 0h2v2H5V1zm4 0h2v2H9V1zm4 0h2v2h-2V1zm4 0h2v2h-2V1zM1 5h2v2H1V5zm16 0h2v2h-2V5zM1 9h2v2H1V9zm16 0h2v2h-2V9zM1 13h2v2H1v-2zm16 0h2v2h-2v-2zM1 17h2v-2H1v2zm4 0h2v-2H5v2zm4 0h2v-2H9v2zm4 0h2v-2h-2v2zm4 0h2v-2h-2v2z" opacity="0.3"/>
    <path d="M4 4l10 10M14 4L4 14" stroke="#ea4335" strokeWidth="2" fill="none"/>
  </svg>
);

const BottomBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="currentColor">
    <path d="M1 1h2v2H1V1zm4 0h2v2H5V1zm4 0h2v2H9V1zm4 0h2v2h-2V1zM1 5h2v2H1V5zm12 0h2v2h-2V5zM1 9h2v2H1V9zm12 0h2v2h-2V9zM1 13h2v2H1v-2zm12 0h2v2h-2v-2z" opacity="0.3"/>
    <path d="M1 17h16v-2H1v2z"/>
  </svg>
);

const TopBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="currentColor">
    <path d="M1 5h2v2H1V5zm12 0h2v2h-2V5zM1 9h2v2H1V9zm12 0h2v2h-2V9zM1 13h2v2H1v-2zm4 0h2v2H5v-2zm4 0h2v2H9v-2zm4 0h2v2h-2v-2z" opacity="0.3"/>
    <path d="M1 1h16v2H1V1z"/>
  </svg>
);

// Left and Right border icons for completeness
const LeftBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="currentColor">
    <path d="M5 1h2v2H5V1zm4 0h2v2H9V1zm4 0h2v2h-2V1zM13 5h2v2h-2V5zM13 9h2v2h-2V9zM5 13h2v2H5v-2zm4 0h2v2H9v-2zm4 0h2v2h-2v-2z" opacity="0.3"/>
    <path d="M1 1h2v16H1V1z"/>
  </svg>
);

const RightBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="currentColor">
    <path d="M1 1h2v2H1V1zm4 0h2v2H5V1zm4 0h2v2H9V1zM1 5h2v2H1V5zM1 9h2v2H1V9zM1 13h2v2H1v-2zm4 0h2v2H5v-2zm4 0h2v2H9v-2z" opacity="0.3"/>
    <path d="M15 1h2v16h-2V1z"/>
  </svg>
);

const InnerHorizontalIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="currentColor">
    <path d="M1 1h2v2H1V1zm4 0h2v2H5V1zm4 0h2v2H9V1zm4 0h2v2h-2V1zM1 5h2v2H1V5zm12 0h2v2h-2V5zM1 13h2v2H1v-2zm4 0h2v2H5v-2zm4 0h2v2H9v-2zm4 0h2v2h-2v-2z" opacity="0.3"/>
    <path d="M1 9h16v2H1V9z"/>
  </svg>
);

const InnerVerticalIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="currentColor">
    <path d="M1 1h2v2H1V1zm12 0h2v2h-2V1zM1 5h2v2H1V5zm12 0h2v2h-2V5zM1 9h2v2H1V9zm12 0h2v2h-2V9zM1 13h2v2H1v-2zm12 0h2v2h-2v-2z" opacity="0.3"/>
    <path d="M8 1h2v16H8V1z"/>
  </svg>
);

// Dropdown arrow icon
const DropdownArrowIcon = () => (
  <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor">
    <path d="M7 10l5 5 5-5z"/>
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
    { type: 'inner', label: 'Inner borders', icon: <InnerHorizontalIcon /> },
    { type: 'horizontal', label: 'Horizontal', icon: <InnerHorizontalIcon /> },
    { type: 'vertical', label: 'Vertical', icon: <InnerVerticalIcon /> },
    { type: 'outer', label: 'Outer borders', icon: <OuterBordersIcon /> },
    { type: 'left', label: 'Left border', icon: <LeftBorderIcon /> },
    { type: 'right', label: 'Right border', icon: <RightBorderIcon /> },
    { type: 'top', label: 'Top border', icon: <TopBorderIcon /> },
    { type: 'bottom', label: 'Bottom border', icon: <BottomBorderIcon /> },
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
        style={{ display: 'flex', alignItems: 'center', gap: 2 }}
      >
        <BorderIcon />
        <DropdownArrowIcon />
      </button>
      {isOpen && (
        <>
          <div
            style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 99 }}
            onClick={() => setIsOpen(false)}
          />
          <div className="borders-menu" style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            background: 'white',
            border: '1px solid #dadce0',
            borderRadius: 8,
            boxShadow: '0 2px 10px rgba(0,0,0,0.15)',
            zIndex: 100,
            minWidth: 180,
            padding: '4px 0',
          }}>
            {borderOptions.map(({ type, label, icon }, index) => (
              <React.Fragment key={type}>
                {index === 5 && <div style={{ height: 1, background: '#dadce0', margin: '4px 0' }} />}
                {index === 9 && <div style={{ height: 1, background: '#dadce0', margin: '4px 0' }} />}
                <button
                  className="borders-option"
                  onClick={() => handleSelect(type)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 12,
                    width: '100%',
                    padding: '8px 12px',
                    border: 'none',
                    background: 'none',
                    cursor: 'pointer',
                    fontSize: 13,
                    color: '#202124',
                    textAlign: 'left',
                  }}
                  onMouseEnter={(e) => (e.currentTarget.style.background = '#f1f3f4')}
                  onMouseLeave={(e) => (e.currentTarget.style.background = 'none')}
                >
                  <span style={{ display: 'flex', alignItems: 'center', width: 18 }}>{icon}</span>
                  <span>{label}</span>
                </button>
              </React.Fragment>
            ))}
          </div>
        </>
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
          className={currentFormat?.bold ? 'active' : ''}
          onClick={toggleBold}
        >
          <BoldIcon />
        </button>
        <button
          title="Italic (Ctrl+I)"
          className={currentFormat?.italic ? 'active' : ''}
          onClick={toggleItalic}
        >
          <ItalicIcon />
        </button>
        <button
          title="Underline (Ctrl+U)"
          className={currentFormat?.underline ? 'active' : ''}
          onClick={toggleUnderline}
        >
          <UnderlineIcon />
        </button>
        <button
          title="Strikethrough (Alt+Shift+5)"
          className={currentFormat?.strikethrough ? 'active' : ''}
          onClick={toggleStrikethrough}
        >
          <StrikethroughIcon />
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
