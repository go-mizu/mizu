import { useState, useRef, useEffect } from 'react'
import { Property, PropertyType, NumberFormat, PropertyConfig } from '../api/client'
import { ChevronRight } from 'lucide-react'

interface PropertyHeaderProps {
  property: Property
  onUpdate: (updates: Partial<Property>) => void
  onDelete: () => void
}

const TYPE_ICONS: Record<PropertyType, string> = {
  text: 'Aa',
  number: '#',
  select: '○',
  multi_select: '◎',
  date: '📅',
  person: '👤',
  checkbox: '☑',
  url: '🔗',
  email: '✉',
  phone: '📞',
  files: '📎',
  relation: '↔',
  rollup: '∑',
  formula: 'ƒ',
  created_time: '⏱',
  created_by: '👤',
  last_edited_time: '⏱',
  last_edited_by: '👤',
  status: '●',
}

const NUMBER_FORMAT_OPTIONS: { value: NumberFormat; label: string; symbol?: string }[] = [
  { value: 'number', label: 'Number' },
  { value: 'number_with_commas', label: 'Number with commas' },
  { value: 'percent', label: 'Percent', symbol: '%' },
  { value: 'dollar', label: 'US Dollar', symbol: '$' },
  { value: 'euro', label: 'Euro', symbol: '€' },
  { value: 'pound', label: 'British Pound', symbol: '£' },
  { value: 'yen', label: 'Japanese Yen', symbol: '¥' },
  { value: 'rupee', label: 'Indian Rupee', symbol: '₹' },
  { value: 'won', label: 'Korean Won', symbol: '₩' },
  { value: 'yuan', label: 'Chinese Yuan', symbol: '¥' },
  { value: 'peso', label: 'Mexican Peso', symbol: '$' },
  { value: 'franc', label: 'Swiss Franc', symbol: 'CHF' },
  { value: 'kroner', label: 'Danish Krone', symbol: 'kr' },
  { value: 'real', label: 'Brazilian Real', symbol: 'R$' },
  { value: 'ringgit', label: 'Malaysian Ringgit', symbol: 'RM' },
  { value: 'ruble', label: 'Russian Ruble', symbol: '₽' },
  { value: 'rupiah', label: 'Indonesian Rupiah', symbol: 'Rp' },
  { value: 'baht', label: 'Thai Baht', symbol: '฿' },
  { value: 'lira', label: 'Turkish Lira', symbol: '₺' },
  { value: 'shekel', label: 'Israeli Shekel', symbol: '₪' },
  { value: 'rand', label: 'South African Rand', symbol: 'R' },
]

export function PropertyHeader({ property, onUpdate, onDelete }: PropertyHeaderProps) {
  const [editing, setEditing] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const [showTypeMenu, setShowTypeMenu] = useState(false)
  const [showNumberFormat, setShowNumberFormat] = useState(false)
  const [name, setName] = useState(property.name)
  const inputRef = useRef<HTMLInputElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [editing])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false)
        setShowTypeMenu(false)
        setShowNumberFormat(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleSave = () => {
    if (name !== property.name) {
      onUpdate({ name })
    }
    setEditing(false)
    setShowMenu(false)
  }

  const handleTypeChange = (type: PropertyType) => {
    onUpdate({ type })
    setShowTypeMenu(false)
    setShowMenu(false)
  }

  const handleNumberFormatChange = (format: NumberFormat) => {
    const config: PropertyConfig = {
      ...property.config,
      numberFormat: format,
    }
    onUpdate({ config })
    setShowNumberFormat(false)
    setShowMenu(false)
  }

  const currentNumberFormat = property.config?.numberFormat || 'number'
  const currentFormatOption = NUMBER_FORMAT_OPTIONS.find(f => f.value === currentNumberFormat)

  return (
    <div className="property-header" style={{
      display: 'flex',
      alignItems: 'center',
      gap: 6,
      position: 'relative',
    }}>
      <span className="property-type-icon" style={{ fontSize: 12 }}>{TYPE_ICONS[property.type]}</span>

      {editing ? (
        <input
          ref={inputRef}
          className="property-name-input"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onBlur={handleSave}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleSave()
            if (e.key === 'Escape') {
              setName(property.name)
              setEditing(false)
            }
          }}
          style={{
            flex: 1,
            padding: '2px 4px',
            border: '1px solid var(--accent-color)',
            borderRadius: 'var(--radius-sm)',
            fontSize: 13,
            outline: 'none',
          }}
        />
      ) : (
        <span
          className="property-name"
          onDoubleClick={() => setEditing(true)}
          style={{
            flex: 1,
            fontSize: 13,
            fontWeight: 500,
            cursor: 'pointer',
          }}
        >
          {property.name}
        </span>
      )}

      <button
        className="property-menu-btn"
        onClick={() => setShowMenu(!showMenu)}
        style={{
          width: 20,
          height: 20,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          background: 'none',
          border: 'none',
          cursor: 'pointer',
          color: 'var(--text-tertiary)',
          borderRadius: 'var(--radius-sm)',
          opacity: 0,
          transition: 'opacity 0.15s',
        }}
        onMouseEnter={(e) => (e.currentTarget.style.opacity = '1')}
        onMouseLeave={(e) => (e.currentTarget.style.opacity = showMenu ? '1' : '0')}
      >
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <circle cx="7" cy="3" r="1.5" fill="currentColor" />
          <circle cx="7" cy="7" r="1.5" fill="currentColor" />
          <circle cx="7" cy="11" r="1.5" fill="currentColor" />
        </svg>
      </button>

      {showMenu && (
        <div
          className="property-menu"
          ref={menuRef}
          style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            marginTop: 4,
            background: 'var(--bg-primary)',
            border: '1px solid var(--border-color)',
            borderRadius: 'var(--radius-md)',
            boxShadow: 'var(--shadow-lg)',
            minWidth: 220,
            zIndex: 100,
          }}
        >
          {/* Rename section */}
          <div className="menu-section" style={{ padding: '4px' }}>
            <button
              className="menu-item"
              onClick={() => {
                setEditing(true)
                setShowMenu(false)
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                width: '100%',
                padding: '8px 12px',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                fontSize: 13,
                textAlign: 'left',
                borderRadius: 'var(--radius-sm)',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M10 2l2 2M2 10l6-6 2 2-6 6H2v-2z" stroke="currentColor" strokeWidth="1.5" />
              </svg>
              Rename
            </button>
          </div>

          {/* Type change section */}
          <div className="menu-section" style={{ borderTop: '1px solid var(--border-color)', padding: '4px' }}>
            <button
              className="menu-item"
              onClick={() => setShowTypeMenu(!showTypeMenu)}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                width: '100%',
                padding: '8px 12px',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                fontSize: 13,
                textAlign: 'left',
                borderRadius: 'var(--radius-sm)',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <span style={{ width: 14, textAlign: 'center' }}>{TYPE_ICONS[property.type]}</span>
                <span>Type: {formatTypeName(property.type)}</span>
              </div>
              <ChevronRight size={14} style={{ color: 'var(--text-tertiary)' }} />
            </button>

            {showTypeMenu && (
              <div style={{
                position: 'absolute',
                left: '100%',
                top: 0,
                marginLeft: 4,
                background: 'var(--bg-primary)',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)',
                boxShadow: 'var(--shadow-lg)',
                minWidth: 180,
                maxHeight: 400,
                overflowY: 'auto',
              }}>
                {(Object.keys(TYPE_ICONS) as PropertyType[]).map((type) => (
                  <button
                    key={type}
                    className={`menu-item ${property.type === type ? 'active' : ''}`}
                    onClick={() => handleTypeChange(type)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      width: '100%',
                      padding: '8px 12px',
                      background: property.type === type ? 'var(--accent-bg)' : 'none',
                      border: 'none',
                      cursor: 'pointer',
                      fontSize: 13,
                      textAlign: 'left',
                    }}
                  >
                    <span style={{ width: 20, textAlign: 'center' }}>{TYPE_ICONS[type]}</span>
                    <span>{formatTypeName(type)}</span>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Number format section - only for number type */}
          {property.type === 'number' && (
            <div className="menu-section" style={{ borderTop: '1px solid var(--border-color)', padding: '4px' }}>
              <button
                className="menu-item"
                onClick={() => setShowNumberFormat(!showNumberFormat)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  width: '100%',
                  padding: '8px 12px',
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  fontSize: 13,
                  textAlign: 'left',
                  borderRadius: 'var(--radius-sm)',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{ width: 14, textAlign: 'center' }}>{currentFormatOption?.symbol || '#'}</span>
                  <span>Format: {currentFormatOption?.label}</span>
                </div>
                <ChevronRight size={14} style={{ color: 'var(--text-tertiary)' }} />
              </button>

              {showNumberFormat && (
                <div style={{
                  position: 'absolute',
                  left: '100%',
                  top: 0,
                  marginLeft: 4,
                  background: 'var(--bg-primary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 'var(--radius-md)',
                  boxShadow: 'var(--shadow-lg)',
                  minWidth: 200,
                  maxHeight: 400,
                  overflowY: 'auto',
                }}>
                  <div style={{
                    padding: '8px 12px',
                    fontSize: 11,
                    fontWeight: 600,
                    color: 'var(--text-tertiary)',
                    textTransform: 'uppercase',
                    borderBottom: '1px solid var(--border-color)',
                  }}>
                    Number Format
                  </div>
                  {NUMBER_FORMAT_OPTIONS.map(({ value, label, symbol }) => (
                    <button
                      key={value}
                      className={`menu-item ${currentNumberFormat === value ? 'active' : ''}`}
                      onClick={() => handleNumberFormatChange(value)}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 8,
                        width: '100%',
                        padding: '8px 12px',
                        background: currentNumberFormat === value ? 'var(--accent-bg)' : 'none',
                        border: 'none',
                        cursor: 'pointer',
                        fontSize: 13,
                        textAlign: 'left',
                      }}
                    >
                      <span style={{ width: 24, textAlign: 'center', color: 'var(--text-tertiary)' }}>
                        {symbol || '#'}
                      </span>
                      <span>{label}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Delete section */}
          <div className="menu-section" style={{ borderTop: '1px solid var(--border-color)', padding: '4px' }}>
            <button
              className="menu-item danger"
              onClick={onDelete}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                width: '100%',
                padding: '8px 12px',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                fontSize: 13,
                textAlign: 'left',
                color: 'var(--error-color)',
                borderRadius: 'var(--radius-sm)',
              }}
            >
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                <path d="M2 4h10M5 4V2h4v2M3 4v8a1 1 0 001 1h6a1 1 0 001-1V4" stroke="currentColor" strokeWidth="1.5" />
              </svg>
              Delete property
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function formatTypeName(type: PropertyType): string {
  return type
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ')
}

// Utility function to format a number based on the configured format
export function formatNumber(value: number | null | undefined, config?: PropertyConfig): string {
  if (value === null || value === undefined) return ''

  const format = config?.numberFormat || 'number'
  const precision = config?.precision ?? 2

  switch (format) {
    case 'number':
      return value.toString()

    case 'number_with_commas':
      return value.toLocaleString('en-US')

    case 'percent':
      return `${(value * 100).toFixed(precision)}%`

    case 'dollar':
      return value.toLocaleString('en-US', { style: 'currency', currency: 'USD' })

    case 'euro':
      return value.toLocaleString('de-DE', { style: 'currency', currency: 'EUR' })

    case 'pound':
      return value.toLocaleString('en-GB', { style: 'currency', currency: 'GBP' })

    case 'yen':
      return value.toLocaleString('ja-JP', { style: 'currency', currency: 'JPY' })

    case 'rupee':
      return value.toLocaleString('en-IN', { style: 'currency', currency: 'INR' })

    case 'won':
      return value.toLocaleString('ko-KR', { style: 'currency', currency: 'KRW' })

    case 'yuan':
      return value.toLocaleString('zh-CN', { style: 'currency', currency: 'CNY' })

    case 'peso':
      return value.toLocaleString('es-MX', { style: 'currency', currency: 'MXN' })

    case 'franc':
      return value.toLocaleString('de-CH', { style: 'currency', currency: 'CHF' })

    case 'kroner':
      return value.toLocaleString('da-DK', { style: 'currency', currency: 'DKK' })

    case 'real':
      return value.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL' })

    case 'ringgit':
      return value.toLocaleString('ms-MY', { style: 'currency', currency: 'MYR' })

    case 'ruble':
      return value.toLocaleString('ru-RU', { style: 'currency', currency: 'RUB' })

    case 'rupiah':
      return value.toLocaleString('id-ID', { style: 'currency', currency: 'IDR' })

    case 'baht':
      return value.toLocaleString('th-TH', { style: 'currency', currency: 'THB' })

    case 'lira':
      return value.toLocaleString('tr-TR', { style: 'currency', currency: 'TRY' })

    case 'shekel':
      return value.toLocaleString('he-IL', { style: 'currency', currency: 'ILS' })

    case 'rand':
      return value.toLocaleString('en-ZA', { style: 'currency', currency: 'ZAR' })

    default:
      return value.toString()
  }
}
