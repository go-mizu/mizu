import React, { useState, useRef, useEffect } from 'react';
import type { Border } from '../types';

export type BorderType =
  | 'all'
  | 'inner'
  | 'horizontal'
  | 'vertical'
  | 'outer'
  | 'left'
  | 'right'
  | 'top'
  | 'bottom'
  | 'clear';

export interface BordersDropdownProps {
  onApplyBorder: (type: BorderType, border: Border | null) => void;
  currentBorderColor: string;
  currentBorderStyle: Border['style'];
}

const BORDER_STYLES: { value: Border['style']; label: string }[] = [
  { value: 'thin', label: 'Thin' },
  { value: 'medium', label: 'Medium' },
  { value: 'thick', label: 'Thick' },
  { value: 'dashed', label: 'Dashed' },
  { value: 'dotted', label: 'Dotted' },
  { value: 'double', label: 'Double' },
];

const BORDER_COLORS = [
  '#000000', '#434343', '#666666', '#999999', '#b7b7b7', '#cccccc', '#d9d9d9', '#efefef', '#f3f3f3', '#ffffff',
  '#980000', '#ff0000', '#ff9900', '#ffff00', '#00ff00', '#00ffff', '#4a86e8', '#0000ff', '#9900ff', '#ff00ff',
  '#e6b8af', '#f4cccc', '#fce5cd', '#fff2cc', '#d9ead3', '#d0e0e3', '#c9daf8', '#cfe2f3', '#d9d2e9', '#ead1dc',
];

export const BordersDropdown: React.FC<BordersDropdownProps> = ({
  onApplyBorder,
  currentBorderColor,
  currentBorderStyle,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [showColorPicker, setShowColorPicker] = useState(false);
  const [showStylePicker, setShowStylePicker] = useState(false);
  const [borderColor, setBorderColor] = useState(currentBorderColor || '#000000');
  const [borderStyle, setBorderStyle] = useState<Border['style']>(currentBorderStyle || 'thin');
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setShowColorPicker(false);
        setShowStylePicker(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleApplyBorder = (type: BorderType) => {
    if (type === 'clear') {
      onApplyBorder(type, null);
    } else {
      onApplyBorder(type, { style: borderStyle, color: borderColor });
    }
    setIsOpen(false);
  };

  const borderOptions: { type: BorderType; label: string; icon: React.ReactNode }[] = [
    { type: 'all', label: 'All borders', icon: <AllBordersIcon /> },
    { type: 'inner', label: 'Inner borders', icon: <InnerBordersIcon /> },
    { type: 'horizontal', label: 'Horizontal borders', icon: <HorizontalBordersIcon /> },
    { type: 'vertical', label: 'Vertical borders', icon: <VerticalBordersIcon /> },
    { type: 'outer', label: 'Outer borders', icon: <OuterBordersIcon /> },
  ];

  const singleBorderOptions: { type: BorderType; label: string; icon: React.ReactNode }[] = [
    { type: 'left', label: 'Left border', icon: <LeftBorderIcon /> },
    { type: 'right', label: 'Right border', icon: <RightBorderIcon /> },
    { type: 'top', label: 'Top border', icon: <TopBorderIcon /> },
    { type: 'bottom', label: 'Bottom border', icon: <BottomBorderIcon /> },
  ];

  return (
    <div className="borders-dropdown" ref={dropdownRef}>
      <button
        className="toolbar-btn borders-btn"
        onClick={() => setIsOpen(!isOpen)}
        title="Borders"
      >
        <BorderIcon />
        <DropdownArrowIcon />
      </button>

      {isOpen && (
        <div className="borders-menu">
          <div className="borders-section">
            {borderOptions.map(({ type, label, icon }) => (
              <button
                key={type}
                className="borders-option"
                onClick={() => handleApplyBorder(type)}
                title={label}
              >
                {icon}
                <span className="borders-option-label">{label}</span>
              </button>
            ))}
          </div>

          <div className="borders-divider" />

          <div className="borders-section">
            {singleBorderOptions.map(({ type, label, icon }) => (
              <button
                key={type}
                className="borders-option"
                onClick={() => handleApplyBorder(type)}
                title={label}
              >
                {icon}
                <span className="borders-option-label">{label}</span>
              </button>
            ))}
          </div>

          <div className="borders-divider" />

          <button
            className="borders-option"
            onClick={() => handleApplyBorder('clear')}
            title="Clear borders"
          >
            <ClearBordersIcon />
            <span className="borders-option-label">Clear borders</span>
          </button>

          <div className="borders-divider" />

          <div
            className="borders-option has-submenu"
            onMouseEnter={() => setShowColorPicker(true)}
            onMouseLeave={() => setShowColorPicker(false)}
          >
            <span className="borders-color-preview" style={{ backgroundColor: borderColor }} />
            <span className="borders-option-label">Border color</span>
            <ArrowRightIcon />

            {showColorPicker && (
              <div className="borders-color-picker">
                <div className="color-grid">
                  {BORDER_COLORS.map((color) => (
                    <button
                      key={color}
                      className={`color-option ${color === borderColor ? 'selected' : ''}`}
                      style={{ backgroundColor: color }}
                      onClick={() => {
                        setBorderColor(color);
                        setShowColorPicker(false);
                      }}
                    />
                  ))}
                </div>
              </div>
            )}
          </div>

          <div
            className="borders-option has-submenu"
            onMouseEnter={() => setShowStylePicker(true)}
            onMouseLeave={() => setShowStylePicker(false)}
          >
            <span className="borders-style-preview">
              <svg width="20" height="2" viewBox="0 0 20 2">
                {borderStyle === 'dotted' && (
                  <line x1="0" y1="1" x2="20" y2="1" stroke={borderColor} strokeWidth="1" strokeDasharray="1,2" />
                )}
                {borderStyle === 'dashed' && (
                  <line x1="0" y1="1" x2="20" y2="1" stroke={borderColor} strokeWidth="1" strokeDasharray="4,2" />
                )}
                {borderStyle === 'thin' && (
                  <line x1="0" y1="1" x2="20" y2="1" stroke={borderColor} strokeWidth="1" />
                )}
                {borderStyle === 'medium' && (
                  <line x1="0" y1="1" x2="20" y2="1" stroke={borderColor} strokeWidth="2" />
                )}
                {borderStyle === 'thick' && (
                  <line x1="0" y1="1" x2="20" y2="1" stroke={borderColor} strokeWidth="3" />
                )}
                {borderStyle === 'double' && (
                  <>
                    <line x1="0" y1="0" x2="20" y2="0" stroke={borderColor} strokeWidth="1" />
                    <line x1="0" y1="2" x2="20" y2="2" stroke={borderColor} strokeWidth="1" />
                  </>
                )}
              </svg>
            </span>
            <span className="borders-option-label">Border style</span>
            <ArrowRightIcon />

            {showStylePicker && (
              <div className="borders-style-picker">
                {BORDER_STYLES.map(({ value, label }) => (
                  <button
                    key={value}
                    className={`style-option ${value === borderStyle ? 'selected' : ''}`}
                    onClick={() => {
                      setBorderStyle(value);
                      setShowStylePicker(false);
                    }}
                  >
                    <svg width="40" height="8" viewBox="0 0 40 8">
                      {value === 'dotted' && (
                        <line x1="0" y1="4" x2="40" y2="4" stroke="#000" strokeWidth="1" strokeDasharray="1,2" />
                      )}
                      {value === 'dashed' && (
                        <line x1="0" y1="4" x2="40" y2="4" stroke="#000" strokeWidth="1" strokeDasharray="4,2" />
                      )}
                      {value === 'thin' && (
                        <line x1="0" y1="4" x2="40" y2="4" stroke="#000" strokeWidth="1" />
                      )}
                      {value === 'medium' && (
                        <line x1="0" y1="4" x2="40" y2="4" stroke="#000" strokeWidth="2" />
                      )}
                      {value === 'thick' && (
                        <line x1="0" y1="4" x2="40" y2="4" stroke="#000" strokeWidth="3" />
                      )}
                      {value === 'double' && (
                        <>
                          <line x1="0" y1="2" x2="40" y2="2" stroke="#000" strokeWidth="1" />
                          <line x1="0" y1="6" x2="40" y2="6" stroke="#000" strokeWidth="1" />
                        </>
                      )}
                    </svg>
                    <span>{label}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

// Border Icons
const BorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <rect x="3" y="3" width="18" height="18" rx="2" />
  </svg>
);

const DropdownArrowIcon = () => (
  <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3">
    <polyline points="6 9 12 15 18 9" />
  </svg>
);

const ArrowRightIcon = () => (
  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="9 18 15 12 9 6" />
  </svg>
);

const AllBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" />
    <line x1="9" y1="1" x2="9" y2="17" />
    <line x1="1" y1="9" x2="17" y2="9" />
  </svg>
);

const InnerBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="9" y1="1" x2="9" y2="17" />
    <line x1="1" y1="9" x2="17" y2="9" />
  </svg>
);

const HorizontalBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="1" y1="9" x2="17" y2="9" />
  </svg>
);

const VerticalBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="9" y1="1" x2="9" y2="17" />
  </svg>
);

const OuterBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" />
  </svg>
);

const LeftBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="1" y1="1" x2="1" y2="17" strokeWidth="2" />
  </svg>
);

const RightBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="17" y1="1" x2="17" y2="17" strokeWidth="2" />
  </svg>
);

const TopBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="1" y1="1" x2="17" y2="1" strokeWidth="2" />
  </svg>
);

const BottomBorderIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="1" y1="17" x2="17" y2="17" strokeWidth="2" />
  </svg>
);

const ClearBordersIcon = () => (
  <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.5">
    <rect x="1" y="1" width="16" height="16" strokeDasharray="2,2" />
    <line x1="4" y1="4" x2="14" y2="14" strokeWidth="2" stroke="#ea4335" />
    <line x1="14" y1="4" x2="4" y2="14" strokeWidth="2" stroke="#ea4335" />
  </svg>
);

export default BordersDropdown;
