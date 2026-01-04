import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { Property, PropertyOption, api, FileAttachment } from '../api/client'
import { format, parseISO, isValid, parse } from 'date-fns'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Calendar,
  Clock,
  X,
  Plus,
  Check,
  ChevronDown,
  Upload,
  FileText,
  Link2,
  User,
  Search,
} from 'lucide-react'

interface PropertyCellProps {
  property: Property
  value: unknown
  onChange: (value: unknown) => void
  workspaceId?: string
}

interface RelationConfig {
  database_id: string
  type: string
}

interface RollupConfig {
  relation_property_id: string
  rollup_property_id: string
  function: string
}

interface FormulaConfig {
  expression: string
}

interface RelationValue {
  id: string
  title: string
}

interface WorkspaceMember {
  id: string
  name: string
  email: string
  avatar_url?: string
}

// Use FileAttachment from API client for file values
type FileValue = FileAttachment

export function PropertyCell({ property, value, onChange, workspaceId }: PropertyCellProps) {
  switch (property.type) {
    case 'text':
      return <TextCell value={value as string} onChange={onChange} />
    case 'number':
      return <NumberCell value={value as number} onChange={onChange} />
    case 'select':
      return <SelectCell value={value as string} options={property.options || []} onChange={onChange} />
    case 'multi_select':
      return <MultiSelectCell value={value as string[]} options={property.options || []} onChange={onChange} propertyId={property.id} />
    case 'date':
      return <DateCell value={value as string | DateValue} onChange={onChange} />
    case 'person':
      return <PersonCell value={value as string | string[]} onChange={onChange} workspaceId={workspaceId} />
    case 'checkbox':
      return <CheckboxCell value={value as boolean} onChange={onChange} />
    case 'url':
      return <UrlCell value={value as string} onChange={onChange} />
    case 'email':
      return <EmailCell value={value as string} onChange={onChange} />
    case 'phone':
      return <PhoneCell value={value as string} onChange={onChange} />
    case 'files':
      return <FilesCell value={value as FileValue[]} onChange={onChange} />
    case 'status':
      return <SelectCell value={value as string} options={property.options || []} onChange={onChange} />
    case 'relation':
      return <RelationCell value={value as RelationValue[]} config={property.config as unknown as RelationConfig} onChange={onChange} />
    case 'rollup':
      return <RollupCell value={value} config={property.config as unknown as RollupConfig} />
    case 'formula':
      return <FormulaCell value={value} config={property.config as unknown as FormulaConfig} />
    case 'created_time':
    case 'last_edited_time':
      return <ReadOnlyDateCell value={value as string} />
    case 'created_by':
    case 'last_edited_by':
      return <ReadOnlyPersonCell value={value as string} />
    default:
      return <TextCell value={value as string} onChange={onChange} />
  }
}

// Text cell
function TextCell({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [editing, setEditing] = useState(false)
  const [localValue, setLocalValue] = useState(value || '')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setLocalValue(value || '')
  }, [value])

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [editing])

  const handleBlur = useCallback(() => {
    setEditing(false)
    if (localValue !== value) {
      onChange(localValue)
    }
  }, [localValue, value, onChange])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleBlur()
    } else if (e.key === 'Escape') {
      setLocalValue(value || '')
      setEditing(false)
    }
  }, [handleBlur, value])

  if (editing) {
    return (
      <input
        ref={inputRef}
        className="cell-input text"
        value={localValue}
        onChange={(e) => setLocalValue(e.target.value)}
        onBlur={handleBlur}
        onKeyDown={handleKeyDown}
        style={{
          width: '100%',
          padding: '4px 8px',
          border: '2px solid var(--accent-color)',
          borderRadius: 'var(--radius-sm)',
          fontSize: 14,
          outline: 'none',
          background: 'var(--bg-primary)',
          color: 'var(--text-primary)',
        }}
      />
    )
  }

  return (
    <div
      className="cell-display text"
      onClick={() => setEditing(true)}
      style={{
        minHeight: 24,
        padding: '2px 0',
        cursor: 'text',
      }}
    >
      {value || <span style={{ color: 'var(--text-placeholder)' }}>Empty</span>}
    </div>
  )
}

// Number cell
function NumberCell({ value, onChange }: { value: number; onChange: (v: number) => void }) {
  const [editing, setEditing] = useState(false)
  const [localValue, setLocalValue] = useState(value?.toString() || '')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setLocalValue(value?.toString() || '')
  }, [value])

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [editing])

  const handleBlur = useCallback(() => {
    setEditing(false)
    const numValue = parseFloat(localValue)
    if (!isNaN(numValue) && numValue !== value) {
      onChange(numValue)
    }
  }, [localValue, value, onChange])

  if (editing) {
    return (
      <input
        ref={inputRef}
        type="number"
        className="cell-input number"
        value={localValue}
        onChange={(e) => setLocalValue(e.target.value)}
        onBlur={handleBlur}
        onKeyDown={(e) => e.key === 'Enter' && handleBlur()}
        style={{
          width: '100%',
          padding: '4px 8px',
          border: '2px solid var(--accent-color)',
          borderRadius: 'var(--radius-sm)',
          fontSize: 14,
          outline: 'none',
          background: 'var(--bg-primary)',
          color: 'var(--text-primary)',
        }}
      />
    )
  }

  return (
    <div
      className="cell-display number"
      onClick={() => setEditing(true)}
      style={{
        minHeight: 24,
        padding: '2px 0',
        cursor: 'text',
      }}
    >
      {value ?? <span style={{ color: 'var(--text-placeholder)' }}>Empty</span>}
    </div>
  )
}

// Select cell with improved styling
function SelectCell({
  value,
  options,
  onChange,
}: {
  value: string
  options: PropertyOption[]
  onChange: (v: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  useEffect(() => {
    if (open && inputRef.current) {
      inputRef.current.focus()
    }
  }, [open])

  const selectedOption = options.find((o) => o.id === value)
  const filteredOptions = options.filter(o =>
    o.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="cell-select" ref={ref} style={{ position: 'relative' }}>
      <div
        className="select-display"
        onClick={() => setOpen(!open)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          minHeight: 24,
          padding: '2px 0',
          cursor: 'pointer',
        }}
      >
        {selectedOption ? (
          <span
            className="select-tag"
            style={{
              display: 'inline-block',
              padding: '2px 8px',
              borderRadius: 3,
              fontSize: 13,
              backgroundColor: `var(--tag-${selectedOption.color || 'gray'})`,
            }}
          >
            {selectedOption.name}
          </span>
        ) : (
          <span style={{ color: 'var(--text-placeholder)' }}>Select...</span>
        )}
      </div>

      <AnimatePresence>
        {open && (
          <motion.div
            className="select-dropdown"
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.15 }}
            style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              minWidth: 200,
              maxWidth: 300,
              background: 'var(--bg-primary)',
              borderRadius: 'var(--radius-md)',
              boxShadow: 'var(--shadow-lg)',
              padding: 6,
              zIndex: 100,
              marginTop: 4,
            }}
          >
            <input
              ref={inputRef}
              type="text"
              placeholder="Search..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              style={{
                width: '100%',
                padding: '6px 8px',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-sm)',
                fontSize: 13,
                marginBottom: 4,
                outline: 'none',
                background: 'var(--bg-primary)',
                color: 'var(--text-primary)',
              }}
            />
            <div style={{ maxHeight: 200, overflowY: 'auto' }}>
              <button
                className="select-option"
                onClick={() => { onChange(''); setOpen(false); setSearch('') }}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  width: '100%',
                  padding: '6px 8px',
                  background: 'none',
                  border: 'none',
                  borderRadius: 'var(--radius-sm)',
                  cursor: 'pointer',
                  textAlign: 'left',
                }}
              >
                <span style={{ color: 'var(--text-placeholder)' }}>None</span>
              </button>
              {filteredOptions.map((option) => (
                <button
                  key={option.id}
                  className="select-option"
                  onClick={() => { onChange(option.id); setOpen(false); setSearch('') }}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    width: '100%',
                    padding: '6px 8px',
                    background: value === option.id ? 'var(--accent-bg)' : 'none',
                    border: 'none',
                    borderRadius: 'var(--radius-sm)',
                    cursor: 'pointer',
                    textAlign: 'left',
                  }}
                >
                  <span
                    className="select-tag"
                    style={{
                      display: 'inline-block',
                      padding: '2px 8px',
                      borderRadius: 3,
                      fontSize: 13,
                      backgroundColor: `var(--tag-${option.color || 'gray'})`,
                    }}
                  >
                    {option.name}
                  </span>
                  {value === option.id && (
                    <Check size={14} style={{ marginLeft: 'auto', color: 'var(--accent-color)' }} />
                  )}
                </button>
              ))}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

// Multi-select cell with tag creation
function MultiSelectCell({
  value,
  options,
  onChange,
  propertyId,
}: {
  value: string[]
  options: PropertyOption[]
  onChange: (v: string[]) => void
  propertyId?: string
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [isCreating, setIsCreating] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const selected = new Set(value || [])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  useEffect(() => {
    if (open && inputRef.current) {
      inputRef.current.focus()
    }
  }, [open])

  const toggleOption = (optionId: string) => {
    const next = new Set(selected)
    if (next.has(optionId)) {
      next.delete(optionId)
    } else {
      next.add(optionId)
    }
    onChange(Array.from(next))
  }

  const handleCreateOption = async () => {
    if (!search.trim() || !propertyId) return

    setIsCreating(true)
    try {
      // In a real implementation, this would call the API to create the option
      // For now, we just add it locally
      const newOptionId = `temp-${Date.now()}`
      const next = new Set(selected)
      next.add(newOptionId)
      onChange(Array.from(next))
      setSearch('')
    } catch (err) {
      console.error('Failed to create option:', err)
    } finally {
      setIsCreating(false)
    }
  }

  const selectedOptions = options.filter((o) => selected.has(o.id))
  const filteredOptions = options.filter(o =>
    o.name.toLowerCase().includes(search.toLowerCase())
  )
  const showCreateOption = search.trim() && !filteredOptions.some(o =>
    o.name.toLowerCase() === search.toLowerCase()
  )

  return (
    <div className="cell-multi-select" ref={ref} style={{ position: 'relative' }}>
      <div
        className="multi-select-display"
        onClick={() => setOpen(!open)}
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: 4,
          minHeight: 24,
          padding: '2px 0',
          cursor: 'pointer',
          alignItems: 'center',
        }}
      >
        {selectedOptions.length > 0 ? (
          selectedOptions.map((option) => (
            <span
              key={option.id}
              className="select-tag"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 4,
                padding: '2px 8px',
                borderRadius: 3,
                fontSize: 13,
                backgroundColor: `var(--tag-${option.color || 'gray'})`,
              }}
            >
              {option.name}
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  toggleOption(option.id)
                }}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  width: 14,
                  height: 14,
                  padding: 0,
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  borderRadius: 2,
                  color: 'inherit',
                  opacity: 0.7,
                }}
              >
                <X size={10} />
              </button>
            </span>
          ))
        ) : (
          <span style={{ color: 'var(--text-placeholder)' }}>Select...</span>
        )}
      </div>

      <AnimatePresence>
        {open && (
          <motion.div
            className="select-dropdown multi"
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.15 }}
            style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              minWidth: 200,
              maxWidth: 300,
              background: 'var(--bg-primary)',
              borderRadius: 'var(--radius-md)',
              boxShadow: 'var(--shadow-lg)',
              padding: 6,
              zIndex: 100,
              marginTop: 4,
            }}
          >
            <input
              ref={inputRef}
              type="text"
              placeholder="Search or create..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && showCreateOption) {
                  handleCreateOption()
                }
              }}
              style={{
                width: '100%',
                padding: '6px 8px',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-sm)',
                fontSize: 13,
                marginBottom: 4,
                outline: 'none',
                background: 'var(--bg-primary)',
                color: 'var(--text-primary)',
              }}
            />
            <div style={{ maxHeight: 200, overflowY: 'auto' }}>
              {filteredOptions.map((option) => (
                <button
                  key={option.id}
                  className={`select-option ${selected.has(option.id) ? 'selected' : ''}`}
                  onClick={() => toggleOption(option.id)}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    width: '100%',
                    padding: '6px 8px',
                    background: selected.has(option.id) ? 'var(--accent-bg)' : 'none',
                    border: 'none',
                    borderRadius: 'var(--radius-sm)',
                    cursor: 'pointer',
                    textAlign: 'left',
                  }}
                >
                  <span
                    style={{
                      width: 16,
                      height: 16,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      color: 'var(--accent-color)',
                    }}
                  >
                    {selected.has(option.id) && <Check size={14} />}
                  </span>
                  <span
                    className="select-tag"
                    style={{
                      display: 'inline-block',
                      padding: '2px 8px',
                      borderRadius: 3,
                      fontSize: 13,
                      backgroundColor: `var(--tag-${option.color || 'gray'})`,
                    }}
                  >
                    {option.name}
                  </span>
                </button>
              ))}
              {showCreateOption && (
                <button
                  onClick={handleCreateOption}
                  disabled={isCreating}
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    width: '100%',
                    padding: '6px 8px',
                    background: 'none',
                    border: 'none',
                    borderRadius: 'var(--radius-sm)',
                    cursor: 'pointer',
                    textAlign: 'left',
                    color: 'var(--accent-color)',
                  }}
                >
                  <Plus size={14} />
                  Create "{search}"
                </button>
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

// Enhanced date cell with time picker
interface DateValue {
  start: string
  end?: string
  include_time?: boolean
}

function DateCell({ value, onChange }: { value: string | DateValue; onChange: (v: string | DateValue) => void }) {
  const [open, setOpen] = useState(false)
  const [includeTime, setIncludeTime] = useState(false)
  const [isRange, setIsRange] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  // Parse value
  const dateValue = useMemo(() => {
    if (typeof value === 'string') {
      return { start: value, end: undefined, include_time: false }
    }
    return value || { start: '', end: undefined, include_time: false }
  }, [value])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const formatDisplayDate = (dateStr: string | undefined, showTime: boolean) => {
    if (!dateStr) return null
    try {
      const date = parseISO(dateStr)
      if (!isValid(date)) return dateStr
      return showTime ? format(date, 'MMM d, yyyy h:mm a') : format(date, 'MMM d, yyyy')
    } catch {
      return dateStr
    }
  }

  const handleDateChange = (field: 'start' | 'end', dateStr: string) => {
    const newValue: DateValue = {
      ...dateValue,
      [field]: dateStr ? new Date(dateStr).toISOString() : undefined,
      include_time: includeTime,
    }
    onChange(typeof value === 'string' ? newValue.start! : newValue)
  }

  const handleTimeChange = (field: 'start' | 'end', timeStr: string) => {
    const currentDate = dateValue[field] ? parseISO(dateValue[field]!) : new Date()
    const [hours, minutes] = timeStr.split(':').map(Number)
    currentDate.setHours(hours, minutes)
    handleDateChange(field, currentDate.toISOString())
  }

  const displayValue = formatDisplayDate(dateValue.start, dateValue.include_time || false)
  const displayEndValue = formatDisplayDate(dateValue.end, dateValue.include_time || false)

  return (
    <div className="cell-date" ref={ref} style={{ position: 'relative' }}>
      <div
        className="date-display"
        onClick={() => setOpen(!open)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 4,
          minHeight: 24,
          padding: '2px 0',
          cursor: 'pointer',
        }}
      >
        {displayValue ? (
          <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
            <Calendar size={14} style={{ color: 'var(--text-tertiary)' }} />
            {displayValue}
            {displayEndValue && ` → ${displayEndValue}`}
          </span>
        ) : (
          <span style={{ color: 'var(--text-placeholder)' }}>Empty</span>
        )}
      </div>

      <AnimatePresence>
        {open && (
          <motion.div
            className="date-picker-dropdown"
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.15 }}
            style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              minWidth: 280,
              background: 'var(--bg-primary)',
              borderRadius: 'var(--radius-md)',
              boxShadow: 'var(--shadow-lg)',
              padding: 12,
              zIndex: 100,
              marginTop: 4,
            }}
          >
            {/* Date input */}
            <div style={{ marginBottom: 12 }}>
              <label style={{ display: 'block', fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4 }}>
                {isRange ? 'Start date' : 'Date'}
              </label>
              <input
                type="date"
                value={dateValue.start?.split('T')[0] || ''}
                onChange={(e) => handleDateChange('start', e.target.value)}
                style={{
                  width: '100%',
                  padding: '8px 12px',
                  border: '1px solid var(--border-color)',
                  borderRadius: 'var(--radius-sm)',
                  fontSize: 14,
                  outline: 'none',
                  background: 'var(--bg-primary)',
                  color: 'var(--text-primary)',
                }}
              />
            </div>

            {/* Time input */}
            {includeTime && (
              <div style={{ marginBottom: 12 }}>
                <label style={{ display: 'block', fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4 }}>
                  Time
                </label>
                <input
                  type="time"
                  value={dateValue.start ? format(parseISO(dateValue.start), 'HH:mm') : ''}
                  onChange={(e) => handleTimeChange('start', e.target.value)}
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    border: '1px solid var(--border-color)',
                    borderRadius: 'var(--radius-sm)',
                    fontSize: 14,
                    outline: 'none',
                    background: 'var(--bg-primary)',
                    color: 'var(--text-primary)',
                  }}
                />
              </div>
            )}

            {/* End date for range */}
            {isRange && (
              <>
                <div style={{ marginBottom: 12 }}>
                  <label style={{ display: 'block', fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4 }}>
                    End date
                  </label>
                  <input
                    type="date"
                    value={dateValue.end?.split('T')[0] || ''}
                    onChange={(e) => handleDateChange('end', e.target.value)}
                    style={{
                      width: '100%',
                      padding: '8px 12px',
                      border: '1px solid var(--border-color)',
                      borderRadius: 'var(--radius-sm)',
                      fontSize: 14,
                      outline: 'none',
                      background: 'var(--bg-primary)',
                      color: 'var(--text-primary)',
                    }}
                  />
                </div>
                {includeTime && (
                  <div style={{ marginBottom: 12 }}>
                    <label style={{ display: 'block', fontSize: 12, color: 'var(--text-secondary)', marginBottom: 4 }}>
                      End time
                    </label>
                    <input
                      type="time"
                      value={dateValue.end ? format(parseISO(dateValue.end), 'HH:mm') : ''}
                      onChange={(e) => handleTimeChange('end', e.target.value)}
                      style={{
                        width: '100%',
                        padding: '8px 12px',
                        border: '1px solid var(--border-color)',
                        borderRadius: 'var(--radius-sm)',
                        fontSize: 14,
                        outline: 'none',
                        background: 'var(--bg-primary)',
                        color: 'var(--text-primary)',
                      }}
                    />
                  </div>
                )}
              </>
            )}

            {/* Options */}
            <div style={{ display: 'flex', gap: 8, paddingTop: 8, borderTop: '1px solid var(--border-color)' }}>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={includeTime}
                  onChange={(e) => setIncludeTime(e.target.checked)}
                />
                Include time
              </label>
              <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, cursor: 'pointer' }}>
                <input
                  type="checkbox"
                  checked={isRange}
                  onChange={(e) => setIsRange(e.target.checked)}
                />
                End date
              </label>
            </div>

            {/* Clear button */}
            {dateValue.start && (
              <button
                onClick={() => onChange('')}
                style={{
                  width: '100%',
                  marginTop: 8,
                  padding: '6px 12px',
                  background: 'none',
                  border: '1px solid var(--border-color)',
                  borderRadius: 'var(--radius-sm)',
                  cursor: 'pointer',
                  fontSize: 13,
                  color: 'var(--text-secondary)',
                }}
              >
                Clear
              </button>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

// Enhanced Person cell with workspace members picker
function PersonCell({
  value,
  onChange,
  workspaceId,
}: {
  value: string | string[]
  onChange: (v: string | string[]) => void
  workspaceId?: string
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [members, setMembers] = useState<WorkspaceMember[]>([])
  const [loading, setLoading] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const selectedIds = Array.isArray(value) ? value : value ? [value] : []
  const isMulti = Array.isArray(value)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Fetch workspace members
  useEffect(() => {
    if (!open || !workspaceId) return

    const fetchMembers = async () => {
      setLoading(true)
      try {
        const data = await api.get<{ members: WorkspaceMember[] }>(`/workspaces/${workspaceId}/members`)
        setMembers(data.members || [])
      } catch (err) {
        console.error('Failed to fetch members:', err)
        // Fallback data for testing
        setMembers([
          { id: '1', name: 'John Doe', email: 'john@example.com' },
          { id: '2', name: 'Jane Smith', email: 'jane@example.com' },
        ])
      } finally {
        setLoading(false)
      }
    }

    fetchMembers()
  }, [open, workspaceId])

  useEffect(() => {
    if (open && inputRef.current) {
      inputRef.current.focus()
    }
  }, [open])

  const toggleMember = (memberId: string) => {
    if (isMulti) {
      const next = selectedIds.includes(memberId)
        ? selectedIds.filter(id => id !== memberId)
        : [...selectedIds, memberId]
      onChange(next)
    } else {
      onChange(memberId)
      setOpen(false)
    }
  }

  const filteredMembers = members.filter(m =>
    m.name.toLowerCase().includes(search.toLowerCase()) ||
    m.email.toLowerCase().includes(search.toLowerCase())
  )

  const selectedMembers = members.filter(m => selectedIds.includes(m.id))

  const getInitials = (name: string) => {
    return name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)
  }

  return (
    <div className="cell-person" ref={ref} style={{ position: 'relative' }}>
      <div
        className="person-display"
        onClick={() => setOpen(!open)}
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: 4,
          minHeight: 24,
          padding: '2px 0',
          cursor: 'pointer',
        }}
      >
        {selectedMembers.length > 0 ? (
          selectedMembers.map((member) => (
            <div
              key={member.id}
              className="person-badge"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
              }}
            >
              <span
                className="person-avatar"
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                  color: 'white',
                  fontSize: 10,
                  fontWeight: 600,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                {member.avatar_url ? (
                  <img src={member.avatar_url} alt="" style={{ width: '100%', height: '100%', borderRadius: '50%' }} />
                ) : (
                  getInitials(member.name)
                )}
              </span>
              <span className="person-name" style={{ fontSize: 14 }}>{member.name}</span>
            </div>
          ))
        ) : selectedIds.length > 0 ? (
          // Fallback for IDs without loaded member data
          selectedIds.map((id) => (
            <div key={id} className="person-badge" style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
              <span
                className="person-avatar"
                style={{
                  width: 20,
                  height: 20,
                  borderRadius: '50%',
                  background: 'var(--bg-secondary)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <User size={12} />
              </span>
              <span style={{ fontSize: 14 }}>{id}</span>
            </div>
          ))
        ) : (
          <span style={{ color: 'var(--text-placeholder)' }}>Empty</span>
        )}
      </div>

      <AnimatePresence>
        {open && (
          <motion.div
            className="person-dropdown"
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.15 }}
            style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              minWidth: 250,
              maxWidth: 350,
              background: 'var(--bg-primary)',
              borderRadius: 'var(--radius-md)',
              boxShadow: 'var(--shadow-lg)',
              padding: 6,
              zIndex: 100,
              marginTop: 4,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 8px', marginBottom: 4 }}>
              <Search size={14} style={{ color: 'var(--text-tertiary)' }} />
              <input
                ref={inputRef}
                type="text"
                placeholder="Search people..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                style={{
                  flex: 1,
                  padding: '6px 0',
                  border: 'none',
                  fontSize: 13,
                  outline: 'none',
                  background: 'transparent',
                  color: 'var(--text-primary)',
                }}
              />
            </div>
            <div style={{ maxHeight: 200, overflowY: 'auto' }}>
              {loading ? (
                <div style={{ padding: 12, textAlign: 'center', color: 'var(--text-tertiary)' }}>
                  Loading...
                </div>
              ) : filteredMembers.length === 0 ? (
                <div style={{ padding: 12, textAlign: 'center', color: 'var(--text-tertiary)' }}>
                  No members found
                </div>
              ) : (
                filteredMembers.map((member) => (
                  <button
                    key={member.id}
                    onClick={() => toggleMember(member.id)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      width: '100%',
                      padding: '8px',
                      background: selectedIds.includes(member.id) ? 'var(--accent-bg)' : 'none',
                      border: 'none',
                      borderRadius: 'var(--radius-sm)',
                      cursor: 'pointer',
                      textAlign: 'left',
                    }}
                  >
                    <span
                      style={{
                        width: 24,
                        height: 24,
                        borderRadius: '50%',
                        background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                        color: 'white',
                        fontSize: 11,
                        fontWeight: 600,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}
                    >
                      {member.avatar_url ? (
                        <img src={member.avatar_url} alt="" style={{ width: '100%', height: '100%', borderRadius: '50%' }} />
                      ) : (
                        getInitials(member.name)
                      )}
                    </span>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: 14, color: 'var(--text-primary)' }}>{member.name}</div>
                      <div style={{ fontSize: 12, color: 'var(--text-tertiary)', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                        {member.email}
                      </div>
                    </div>
                    {selectedIds.includes(member.id) && (
                      <Check size={14} style={{ color: 'var(--accent-color)' }} />
                    )}
                  </button>
                ))
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

// Checkbox cell
function CheckboxCell({ value, onChange }: { value: boolean; onChange: (v: boolean) => void }) {
  return (
    <div className="cell-checkbox" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <button
        onClick={() => onChange(!value)}
        style={{
          width: 18,
          height: 18,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          border: value ? 'none' : '2px solid var(--border-color-strong)',
          borderRadius: 3,
          background: value ? 'var(--accent-color)' : 'var(--bg-primary)',
          cursor: 'pointer',
          transition: 'all 0.15s ease',
        }}
      >
        {value && <Check size={12} color="white" strokeWidth={3} />}
      </button>
    </div>
  )
}

// URL cell
function UrlCell({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [editing, setEditing] = useState(false)
  const [localValue, setLocalValue] = useState(value || '')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setLocalValue(value || '')
  }, [value])

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus()
    }
  }, [editing])

  const handleBlur = () => {
    setEditing(false)
    if (localValue !== value) {
      onChange(localValue)
    }
  }

  if (editing) {
    return (
      <input
        ref={inputRef}
        type="url"
        className="cell-input url"
        value={localValue}
        onChange={(e) => setLocalValue(e.target.value)}
        onBlur={handleBlur}
        onKeyDown={(e) => {
          if (e.key === 'Enter') handleBlur()
          if (e.key === 'Escape') { setLocalValue(value || ''); setEditing(false) }
        }}
        style={{
          width: '100%',
          padding: '4px 8px',
          border: '2px solid var(--accent-color)',
          borderRadius: 'var(--radius-sm)',
          fontSize: 14,
          outline: 'none',
          background: 'var(--bg-primary)',
          color: 'var(--text-primary)',
        }}
      />
    )
  }

  return (
    <div className="cell-display url" onClick={() => setEditing(true)} style={{ minHeight: 24, padding: '2px 0', cursor: 'text' }}>
      {value ? (
        <a
          href={value.startsWith('http') ? value : `https://${value}`}
          target="_blank"
          rel="noopener noreferrer"
          onClick={(e) => e.stopPropagation()}
          style={{ color: 'var(--accent-color)', textDecoration: 'none', display: 'flex', alignItems: 'center', gap: 4 }}
        >
          <Link2 size={12} />
          {value.replace(/^https?:\/\//, '').slice(0, 30)}
          {value.length > 30 && '...'}
        </a>
      ) : (
        <span style={{ color: 'var(--text-placeholder)' }}>Empty</span>
      )}
    </div>
  )
}

// Email cell
function EmailCell({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [editing, setEditing] = useState(false)
  const [localValue, setLocalValue] = useState(value || '')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setLocalValue(value || '')
  }, [value])

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus()
    }
  }, [editing])

  const handleBlur = () => {
    setEditing(false)
    if (localValue !== value) {
      onChange(localValue)
    }
  }

  if (editing) {
    return (
      <input
        ref={inputRef}
        type="email"
        className="cell-input email"
        value={localValue}
        onChange={(e) => setLocalValue(e.target.value)}
        onBlur={handleBlur}
        onKeyDown={(e) => {
          if (e.key === 'Enter') handleBlur()
          if (e.key === 'Escape') { setLocalValue(value || ''); setEditing(false) }
        }}
        style={{
          width: '100%',
          padding: '4px 8px',
          border: '2px solid var(--accent-color)',
          borderRadius: 'var(--radius-sm)',
          fontSize: 14,
          outline: 'none',
          background: 'var(--bg-primary)',
          color: 'var(--text-primary)',
        }}
      />
    )
  }

  return (
    <div className="cell-display email" onClick={() => setEditing(true)} style={{ minHeight: 24, padding: '2px 0', cursor: 'text' }}>
      {value ? (
        <a href={`mailto:${value}`} onClick={(e) => e.stopPropagation()} style={{ color: 'var(--accent-color)', textDecoration: 'none' }}>
          {value}
        </a>
      ) : (
        <span style={{ color: 'var(--text-placeholder)' }}>Empty</span>
      )}
    </div>
  )
}

// Phone cell
function PhoneCell({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [editing, setEditing] = useState(false)
  const [localValue, setLocalValue] = useState(value || '')
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setLocalValue(value || '')
  }, [value])

  useEffect(() => {
    if (editing && inputRef.current) {
      inputRef.current.focus()
    }
  }, [editing])

  const handleBlur = () => {
    setEditing(false)
    if (localValue !== value) {
      onChange(localValue)
    }
  }

  if (editing) {
    return (
      <input
        ref={inputRef}
        type="tel"
        className="cell-input phone"
        value={localValue}
        onChange={(e) => setLocalValue(e.target.value)}
        onBlur={handleBlur}
        onKeyDown={(e) => {
          if (e.key === 'Enter') handleBlur()
          if (e.key === 'Escape') { setLocalValue(value || ''); setEditing(false) }
        }}
        style={{
          width: '100%',
          padding: '4px 8px',
          border: '2px solid var(--accent-color)',
          borderRadius: 'var(--radius-sm)',
          fontSize: 14,
          outline: 'none',
          background: 'var(--bg-primary)',
          color: 'var(--text-primary)',
        }}
      />
    )
  }

  return (
    <div className="cell-display phone" onClick={() => setEditing(true)} style={{ minHeight: 24, padding: '2px 0', cursor: 'text' }}>
      {value ? (
        <a href={`tel:${value}`} onClick={(e) => e.stopPropagation()} style={{ color: 'var(--accent-color)', textDecoration: 'none' }}>
          {value}
        </a>
      ) : (
        <span style={{ color: 'var(--text-placeholder)' }}>Empty</span>
      )}
    </div>
  )
}

// File type detection for display
const IMAGE_EXTS = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico', 'tiff', 'tif']
const VIDEO_EXTS = ['mp4', 'webm', 'mov', 'avi', 'mkv', 'wmv', 'flv']
const AUDIO_EXTS = ['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a', 'wma']
const DOC_EXTS = ['pdf', 'doc', 'docx', 'txt', 'rtf', 'odt']
const SHEET_EXTS = ['xls', 'xlsx', 'csv', 'ods']
const PRES_EXTS = ['ppt', 'pptx', 'odp']
const ARCHIVE_EXTS = ['zip', 'rar', '7z', 'tar', 'gz', 'bz2']
const CODE_EXTS = ['js', 'ts', 'tsx', 'jsx', 'py', 'go', 'rs', 'java', 'c', 'cpp', 'css', 'html', 'json', 'md']

function getFileExt(name: string): string {
  const parts = name.split('.')
  return parts.length > 1 ? parts.pop()?.toLowerCase() || '' : ''
}

function isImage(file: FileValue): boolean {
  if (file.type?.startsWith('image/')) return true
  return IMAGE_EXTS.includes(getFileExt(file.name))
}

function getFileIconForDisplay(file: FileValue): string {
  const ext = getFileExt(file.name)
  if (file.type?.startsWith('image/') || IMAGE_EXTS.includes(ext)) return '🖼️'
  if (file.type?.startsWith('video/') || VIDEO_EXTS.includes(ext)) return '🎬'
  if (file.type?.startsWith('audio/') || AUDIO_EXTS.includes(ext)) return '🎵'
  if (file.type?.includes('pdf') || ext === 'pdf') return '📄'
  if (DOC_EXTS.includes(ext)) return '📝'
  if (SHEET_EXTS.includes(ext)) return '📊'
  if (PRES_EXTS.includes(ext)) return '📽️'
  if (ARCHIVE_EXTS.includes(ext)) return '🗜️'
  if (CODE_EXTS.includes(ext)) return '💻'
  return '📎'
}

// Enhanced Files cell with upload and image preview
function FilesCell({ value, onChange }: { value: FileValue[]; onChange: (v: FileValue[]) => void }) {
  const [isDragging, setIsDragging] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const files = Array.isArray(value) ? value : []

  const handleUpload = async (uploadFiles: FileList) => {
    setIsUploading(true)
    try {
      const newFiles: FileValue[] = []
      for (const file of Array.from(uploadFiles)) {
        const result = await api.upload(file)
        newFiles.push({
          id: result.id,
          name: result.filename,
          url: result.url,
          size: file.size,
          type: result.type,
        })
      }
      onChange([...files, ...newFiles])
    } catch (err) {
      console.error('Upload failed:', err)
    } finally {
      setIsUploading(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
    if (e.dataTransfer.files.length > 0) {
      handleUpload(e.dataTransfer.files)
    }
  }

  const removeFile = (index: number) => {
    onChange(files.filter((_, i) => i !== index))
  }

  const formatSize = (bytes?: number) => {
    if (!bytes) return ''
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  return (
    <div className="cell-files">
      <input
        ref={fileInputRef}
        type="file"
        multiple
        onChange={(e) => e.target.files && handleUpload(e.target.files)}
        style={{ display: 'none' }}
      />

      {files.length > 0 ? (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
          {files.map((file, i) => (
            <div
              key={file.id || i}
              className="file-item"
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 6,
                padding: '4px 8px',
                background: 'var(--bg-secondary)',
                borderRadius: 'var(--radius-sm)',
                fontSize: 13,
              }}
            >
              {/* Image thumbnail or file icon */}
              {isImage(file) && file.url ? (
                <img
                  src={file.thumbnailUrl || file.url}
                  alt={file.name}
                  style={{
                    width: 24,
                    height: 24,
                    objectFit: 'cover',
                    borderRadius: 3,
                    flexShrink: 0,
                  }}
                />
              ) : (
                <span style={{ fontSize: 16, flexShrink: 0 }}>{getFileIconForDisplay(file)}</span>
              )}
              <a
                href={file.url}
                target="_blank"
                rel="noopener noreferrer"
                style={{
                  color: 'var(--text-primary)',
                  textDecoration: 'none',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  maxWidth: 120,
                }}
                title={file.name}
              >
                {file.name}
              </a>
              {file.size && (
                <span style={{ color: 'var(--text-tertiary)', fontSize: 11, flexShrink: 0 }}>
                  {formatSize(file.size)}
                </span>
              )}
              <button
                onClick={() => removeFile(i)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  padding: 2,
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  color: 'var(--text-tertiary)',
                  borderRadius: 2,
                  flexShrink: 0,
                }}
              >
                <X size={12} />
              </button>
            </div>
          ))}
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={isUploading}
            style={{
              display: 'inline-flex',
              alignItems: 'center',
              gap: 4,
              padding: '4px 8px',
              background: 'none',
              border: '1px dashed var(--border-color)',
              borderRadius: 'var(--radius-sm)',
              cursor: 'pointer',
              fontSize: 13,
              color: 'var(--text-tertiary)',
            }}
          >
            <Plus size={12} />
            {isUploading ? 'Uploading...' : 'Add'}
          </button>
        </div>
      ) : (
        <div
          onDragOver={(e) => { e.preventDefault(); setIsDragging(true) }}
          onDragLeave={() => setIsDragging(false)}
          onDrop={handleDrop}
          onClick={() => fileInputRef.current?.click()}
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            gap: 8,
            padding: 8,
            border: `2px dashed ${isDragging ? 'var(--accent-color)' : 'var(--border-color)'}`,
            borderRadius: 'var(--radius-sm)',
            cursor: 'pointer',
            color: 'var(--text-placeholder)',
            fontSize: 13,
            transition: 'border-color 0.15s ease',
          }}
        >
          <Upload size={14} />
          {isUploading ? 'Uploading...' : 'Add file'}
        </div>
      )}
    </div>
  )
}

// Read-only date cell
function ReadOnlyDateCell({ value }: { value: string }) {
  const displayValue = value ? (() => {
    try {
      const date = parseISO(value)
      return isValid(date) ? format(date, 'MMM d, yyyy HH:mm') : value
    } catch {
      return value
    }
  })() : '-'

  return <div className="cell-display readonly" style={{ color: 'var(--text-secondary)' }}>{displayValue}</div>
}

// Read-only person cell
function ReadOnlyPersonCell({ value }: { value: string }) {
  return (
    <div className="cell-display readonly" style={{ color: 'var(--text-secondary)' }}>
      {value ? (
        <div className="person-badge" style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          <span
            style={{
              width: 20,
              height: 20,
              borderRadius: '50%',
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
              color: 'white',
              fontSize: 10,
              fontWeight: 600,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            {value.charAt(0).toUpperCase()}
          </span>
          <span>{value}</span>
        </div>
      ) : (
        '-'
      )}
    </div>
  )
}

// Relation cell
function RelationCell({
  value,
  config,
  onChange,
}: {
  value: RelationValue[]
  config: RelationConfig
  onChange: (v: RelationValue[]) => void
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [searchResults, setSearchResults] = useState<RelationValue[]>([])
  const [loading, setLoading] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const relations = value || []

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  useEffect(() => {
    if (open && inputRef.current) {
      inputRef.current.focus()
    }
  }, [open])

  // Search for related items
  useEffect(() => {
    if (!open || !config.database_id) return

    const fetchRelated = async () => {
      setLoading(true)
      try {
        const data = await api.get<{ rows: RelationValue[] }>(
          `/databases/${config.database_id}/rows?q=${encodeURIComponent(search)}`
        )
        setSearchResults(data.rows?.map(r => ({ id: r.id, title: (r as any).properties?.title || 'Untitled' })) || [])
      } catch (err) {
        console.error('Failed to fetch related items:', err)
        // Fallback data
        setSearchResults([
          { id: '1', title: 'Item 1' },
          { id: '2', title: 'Item 2' },
          { id: '3', title: 'Item 3' },
        ])
      } finally {
        setLoading(false)
      }
    }

    const timeout = setTimeout(fetchRelated, 150)
    return () => clearTimeout(timeout)
  }, [open, config.database_id, search])

  const toggleRelation = (item: RelationValue) => {
    const exists = relations.some(r => r.id === item.id)
    if (exists) {
      onChange(relations.filter(r => r.id !== item.id))
    } else {
      onChange([...relations, item])
    }
  }

  return (
    <div className="cell-relation" ref={ref} style={{ position: 'relative' }}>
      <div
        className="relation-display"
        onClick={() => setOpen(!open)}
        style={{
          display: 'flex',
          flexWrap: 'wrap',
          alignItems: 'center',
          gap: 4,
          minHeight: 24,
          padding: '2px 0',
          cursor: 'pointer',
        }}
      >
        {relations.length > 0 ? (
          relations.map(rel => (
            <span
              key={rel.id}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 4,
                padding: '2px 8px',
                background: 'var(--bg-secondary)',
                borderRadius: 3,
                fontSize: 13,
              }}
            >
              {rel.title}
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  onChange(relations.filter(r => r.id !== rel.id))
                }}
                style={{
                  display: 'flex',
                  padding: 0,
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  color: 'var(--text-tertiary)',
                }}
              >
                <X size={12} />
              </button>
            </span>
          ))
        ) : (
          <span style={{ color: 'var(--text-placeholder)' }}>Add relation...</span>
        )}
      </div>

      <AnimatePresence>
        {open && (
          <motion.div
            className="relation-dropdown"
            initial={{ opacity: 0, y: -4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.15 }}
            style={{
              position: 'absolute',
              top: '100%',
              left: 0,
              minWidth: 250,
              background: 'var(--bg-primary)',
              borderRadius: 'var(--radius-md)',
              boxShadow: 'var(--shadow-lg)',
              padding: 6,
              zIndex: 100,
              marginTop: 4,
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 8px', marginBottom: 4 }}>
              <Search size={14} style={{ color: 'var(--text-tertiary)' }} />
              <input
                ref={inputRef}
                type="text"
                placeholder="Search..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                style={{
                  flex: 1,
                  padding: '6px 0',
                  border: 'none',
                  fontSize: 13,
                  outline: 'none',
                  background: 'transparent',
                  color: 'var(--text-primary)',
                }}
              />
            </div>
            <div style={{ maxHeight: 200, overflowY: 'auto' }}>
              {loading ? (
                <div style={{ padding: 12, textAlign: 'center', color: 'var(--text-tertiary)' }}>Loading...</div>
              ) : searchResults.length === 0 ? (
                <div style={{ padding: 12, textAlign: 'center', color: 'var(--text-tertiary)' }}>No results</div>
              ) : (
                searchResults.map(item => (
                  <button
                    key={item.id}
                    onClick={() => toggleRelation(item)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      width: '100%',
                      padding: '8px',
                      background: relations.some(r => r.id === item.id) ? 'var(--accent-bg)' : 'none',
                      border: 'none',
                      borderRadius: 'var(--radius-sm)',
                      cursor: 'pointer',
                      textAlign: 'left',
                    }}
                  >
                    <span style={{ width: 16 }}>
                      {relations.some(r => r.id === item.id) && <Check size={14} style={{ color: 'var(--accent-color)' }} />}
                    </span>
                    <span style={{ fontSize: 14 }}>{item.title}</span>
                  </button>
                ))
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}

// Rollup cell (read-only computed value)
function RollupCell({
  value,
  config,
}: {
  value: unknown
  config: RollupConfig
}) {
  const formatValue = () => {
    if (value === null || value === undefined) return '-'

    switch (config.function) {
      case 'count':
      case 'count_values':
      case 'count_unique_values':
        return typeof value === 'number' ? value.toString() : '-'
      case 'sum':
      case 'average':
      case 'min':
      case 'max':
        return typeof value === 'number' ? value.toLocaleString() : '-'
      case 'percent_empty':
      case 'percent_not_empty':
        return typeof value === 'number' ? `${(value * 100).toFixed(1)}%` : '-'
      case 'show_original':
        if (Array.isArray(value)) {
          return value.join(', ')
        }
        return String(value)
      default:
        return String(value)
    }
  }

  return (
    <div className="cell-display rollup" title={`Rollup: ${config.function}`} style={{ color: 'var(--text-secondary)' }}>
      {formatValue()}
    </div>
  )
}

// Formula cell (read-only computed value)
function FormulaCell({
  value,
  config,
}: {
  value: unknown
  config: FormulaConfig
}) {
  const formatValue = () => {
    if (value === null || value === undefined) return '-'

    if (typeof value === 'boolean') {
      return value ? '✓' : '✗'
    }
    if (typeof value === 'number') {
      return value.toLocaleString()
    }
    if (value instanceof Date) {
      return format(value, 'MMM d, yyyy')
    }
    return String(value)
  }

  return (
    <div className="cell-display formula" title={`Formula: ${config.expression}`} style={{ color: 'var(--text-secondary)' }}>
      {formatValue()}
    </div>
  )
}
