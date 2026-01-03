import { useState, useRef, useEffect, useCallback } from 'react'
import { Property, DatabaseRow } from '../api/client'
import { Plus, ChevronDown, Edit2, Trash2, Copy, MoreHorizontal, FileText } from 'lucide-react'

export interface DatabaseTemplate {
  id: string
  name: string
  icon?: string
  description?: string
  defaultProperties: Record<string, unknown>
  createdAt: string
}

interface DatabaseTemplatesProps {
  databaseId: string
  properties: Property[]
  templates: DatabaseTemplate[]
  onAddRow: (initialProperties?: Record<string, unknown>) => Promise<DatabaseRow | null>
  onCreateTemplate: (template: Omit<DatabaseTemplate, 'id' | 'createdAt'>) => Promise<void>
  onUpdateTemplate: (templateId: string, updates: Partial<DatabaseTemplate>) => Promise<void>
  onDeleteTemplate: (templateId: string) => Promise<void>
}

export function DatabaseTemplates({
  databaseId,
  properties,
  templates,
  onAddRow,
  onCreateTemplate,
  onUpdateTemplate,
  onDeleteTemplate,
}: DatabaseTemplatesProps) {
  const [showDropdown, setShowDropdown] = useState(false)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<DatabaseTemplate | null>(null)
  const [menuTemplate, setMenuTemplate] = useState<string | null>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)

  // Close dropdowns on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowDropdown(false)
      }
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuTemplate(null)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Handle create row from template
  const handleCreateFromTemplate = useCallback(async (template: DatabaseTemplate) => {
    await onAddRow(template.defaultProperties)
    setShowDropdown(false)
  }, [onAddRow])

  // Handle create blank row
  const handleCreateBlank = useCallback(async () => {
    await onAddRow()
    setShowDropdown(false)
  }, [onAddRow])

  return (
    <div className="database-templates" style={{ position: 'relative' }}>
      <div ref={dropdownRef}>
        <button
          className="new-row-btn"
          onClick={() => setShowDropdown(!showDropdown)}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 6,
            padding: '8px 16px',
            background: 'var(--accent-color)',
            color: 'white',
            border: 'none',
            borderRadius: 'var(--radius-md)',
            cursor: 'pointer',
            fontSize: 13,
            fontWeight: 500,
          }}
        >
          <Plus size={14} />
          <span>New</span>
          {templates.length > 0 && <ChevronDown size={14} style={{ marginLeft: 4 }} />}
        </button>

        {showDropdown && (
          <div style={{
            position: 'absolute',
            top: '100%',
            left: 0,
            marginTop: 4,
            background: 'var(--bg-primary)',
            border: '1px solid var(--border-color)',
            borderRadius: 'var(--radius-md)',
            boxShadow: 'var(--shadow-lg)',
            minWidth: 240,
            zIndex: 100,
          }}>
            {/* Blank row option */}
            <button
              onClick={handleCreateBlank}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                width: '100%',
                padding: '10px 14px',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                fontSize: 13,
                textAlign: 'left',
              }}
            >
              <div style={{
                width: 28,
                height: 28,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: 'var(--bg-secondary)',
                borderRadius: 'var(--radius-sm)',
                color: 'var(--text-tertiary)',
              }}>
                <Plus size={16} />
              </div>
              <div>
                <div style={{ fontWeight: 500 }}>Blank</div>
                <div style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>Empty row</div>
              </div>
            </button>

            {templates.length > 0 && (
              <>
                <div style={{
                  padding: '8px 14px 4px',
                  fontSize: 11,
                  fontWeight: 600,
                  color: 'var(--text-tertiary)',
                  textTransform: 'uppercase',
                  borderTop: '1px solid var(--border-color)',
                }}>
                  Templates
                </div>

                {templates.map((template) => (
                  <div
                    key={template.id}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      position: 'relative',
                    }}
                  >
                    <button
                      onClick={() => handleCreateFromTemplate(template)}
                      style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: 10,
                        flex: 1,
                        padding: '10px 14px',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        fontSize: 13,
                        textAlign: 'left',
                      }}
                    >
                      <div style={{
                        width: 28,
                        height: 28,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: 'var(--accent-bg)',
                        borderRadius: 'var(--radius-sm)',
                        fontSize: 14,
                      }}>
                        {template.icon || <FileText size={16} />}
                      </div>
                      <div>
                        <div style={{ fontWeight: 500 }}>{template.name}</div>
                        {template.description && (
                          <div style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>
                            {template.description}
                          </div>
                        )}
                      </div>
                    </button>
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        setMenuTemplate(menuTemplate === template.id ? null : template.id)
                      }}
                      style={{
                        width: 24,
                        height: 24,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        color: 'var(--text-tertiary)',
                        marginRight: 8,
                      }}
                    >
                      <MoreHorizontal size={14} />
                    </button>

                    {/* Template menu */}
                    {menuTemplate === template.id && (
                      <div
                        ref={menuRef}
                        style={{
                          position: 'absolute',
                          right: 0,
                          top: '100%',
                          marginTop: 2,
                          background: 'var(--bg-primary)',
                          border: '1px solid var(--border-color)',
                          borderRadius: 'var(--radius-md)',
                          boxShadow: 'var(--shadow-lg)',
                          minWidth: 150,
                          zIndex: 101,
                        }}
                      >
                        <button
                          onClick={() => {
                            setEditingTemplate(template)
                            setShowCreateModal(true)
                            setMenuTemplate(null)
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
                          }}
                        >
                          <Edit2 size={14} />
                          Edit
                        </button>
                        <button
                          onClick={async () => {
                            await onCreateTemplate({
                              name: `${template.name} (copy)`,
                              icon: template.icon,
                              description: template.description,
                              defaultProperties: { ...template.defaultProperties },
                            })
                            setMenuTemplate(null)
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
                          }}
                        >
                          <Copy size={14} />
                          Duplicate
                        </button>
                        <button
                          onClick={async () => {
                            if (confirm('Delete this template?')) {
                              await onDeleteTemplate(template.id)
                              setMenuTemplate(null)
                            }
                          }}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 8,
                            width: '100%',
                            padding: '8px 12px',
                            background: 'none',
                            border: 'none',
                            borderTop: '1px solid var(--border-color)',
                            cursor: 'pointer',
                            fontSize: 13,
                            textAlign: 'left',
                            color: 'var(--error-color)',
                          }}
                        >
                          <Trash2 size={14} />
                          Delete
                        </button>
                      </div>
                    )}
                  </div>
                ))}
              </>
            )}

            {/* Create template button */}
            <button
              onClick={() => {
                setEditingTemplate(null)
                setShowCreateModal(true)
                setShowDropdown(false)
              }}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                width: '100%',
                padding: '10px 14px',
                background: 'none',
                border: 'none',
                borderTop: '1px solid var(--border-color)',
                cursor: 'pointer',
                fontSize: 13,
                color: 'var(--accent-color)',
              }}
            >
              <Plus size={14} />
              New template
            </button>
          </div>
        )}
      </div>

      {/* Create/Edit Template Modal */}
      {showCreateModal && (
        <TemplateModal
          template={editingTemplate}
          properties={properties}
          onSave={async (data) => {
            if (editingTemplate) {
              await onUpdateTemplate(editingTemplate.id, data)
            } else {
              await onCreateTemplate(data)
            }
            setShowCreateModal(false)
            setEditingTemplate(null)
          }}
          onClose={() => {
            setShowCreateModal(false)
            setEditingTemplate(null)
          }}
        />
      )}
    </div>
  )
}

interface TemplateModalProps {
  template: DatabaseTemplate | null
  properties: Property[]
  onSave: (data: Omit<DatabaseTemplate, 'id' | 'createdAt'>) => Promise<void>
  onClose: () => void
}

function TemplateModal({ template, properties, onSave, onClose }: TemplateModalProps) {
  const [name, setName] = useState(template?.name || '')
  const [description, setDescription] = useState(template?.description || '')
  const [icon, setIcon] = useState(template?.icon || '')
  const [defaultValues, setDefaultValues] = useState<Record<string, unknown>>(
    template?.defaultProperties || {}
  )
  const [isSaving, setIsSaving] = useState(false)

  const handleSave = async () => {
    if (!name.trim()) return

    setIsSaving(true)
    try {
      await onSave({
        name: name.trim(),
        description: description.trim() || undefined,
        icon: icon || undefined,
        defaultProperties: defaultValues,
      })
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <div
      className="modal-overlay"
      onClick={onClose}
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        background: 'rgba(0, 0, 0, 0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1000,
      }}
    >
      <div
        className="template-modal"
        onClick={(e) => e.stopPropagation()}
        style={{
          width: '100%',
          maxWidth: 500,
          maxHeight: '80vh',
          background: 'var(--bg-primary)',
          borderRadius: 'var(--radius-lg)',
          boxShadow: 'var(--shadow-xl)',
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <div className="modal-header" style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '16px 20px',
          borderBottom: '1px solid var(--border-color)',
        }}>
          <h3 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>
            {template ? 'Edit Template' : 'Create Template'}
          </h3>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              fontSize: 18,
              color: 'var(--text-tertiary)',
            }}
          >
            ×
          </button>
        </div>

        <div className="modal-body" style={{ flex: 1, overflow: 'auto', padding: 20 }}>
          {/* Template name */}
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 13, fontWeight: 500, marginBottom: 6 }}>
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Template name"
              autoFocus
              style={{
                width: '100%',
                padding: '10px 12px',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)',
                fontSize: 14,
                outline: 'none',
              }}
            />
          </div>

          {/* Template description */}
          <div style={{ marginBottom: 16 }}>
            <label style={{ display: 'block', fontSize: 13, fontWeight: 500, marginBottom: 6 }}>
              Description (optional)
            </label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description"
              style={{
                width: '100%',
                padding: '10px 12px',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)',
                fontSize: 14,
                outline: 'none',
              }}
            />
          </div>

          {/* Template icon */}
          <div style={{ marginBottom: 20 }}>
            <label style={{ display: 'block', fontSize: 13, fontWeight: 500, marginBottom: 6 }}>
              Icon (emoji)
            </label>
            <input
              type="text"
              value={icon}
              onChange={(e) => setIcon(e.target.value)}
              placeholder="📝"
              maxLength={2}
              style={{
                width: 60,
                padding: '10px 12px',
                border: '1px solid var(--border-color)',
                borderRadius: 'var(--radius-md)',
                fontSize: 18,
                textAlign: 'center',
                outline: 'none',
              }}
            />
          </div>

          {/* Default property values */}
          <div>
            <label style={{ display: 'block', fontSize: 13, fontWeight: 500, marginBottom: 10 }}>
              Default Values
            </label>
            <div style={{ fontSize: 12, color: 'var(--text-tertiary)', marginBottom: 12 }}>
              Set default values for properties. Leave empty to skip.
            </div>
            {properties.filter(p => !['created_time', 'created_by', 'last_edited_time', 'last_edited_by'].includes(p.type)).map((prop) => (
              <div
                key={prop.id}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '8px 0',
                  borderBottom: '1px solid var(--border-color-light)',
                }}
              >
                <span style={{ minWidth: 120, fontSize: 13, color: 'var(--text-secondary)' }}>
                  {prop.name}
                </span>
                <div style={{ flex: 1 }}>
                  <PropertyDefaultInput
                    property={prop}
                    value={defaultValues[prop.id]}
                    onChange={(value) => {
                      setDefaultValues({ ...defaultValues, [prop.id]: value })
                    }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="modal-footer" style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'flex-end',
          gap: 8,
          padding: '16px 20px',
          borderTop: '1px solid var(--border-color)',
        }}>
          <button
            onClick={onClose}
            style={{
              padding: '10px 20px',
              background: 'none',
              border: '1px solid var(--border-color)',
              borderRadius: 'var(--radius-md)',
              cursor: 'pointer',
              fontSize: 14,
            }}
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={!name.trim() || isSaving}
            style={{
              padding: '10px 20px',
              background: 'var(--accent-color)',
              color: 'white',
              border: 'none',
              borderRadius: 'var(--radius-md)',
              cursor: name.trim() && !isSaving ? 'pointer' : 'not-allowed',
              fontSize: 14,
              fontWeight: 500,
              opacity: name.trim() && !isSaving ? 1 : 0.5,
            }}
          >
            {isSaving ? 'Saving...' : template ? 'Save Changes' : 'Create Template'}
          </button>
        </div>
      </div>
    </div>
  )
}

function PropertyDefaultInput({
  property,
  value,
  onChange,
}: {
  property: Property
  value: unknown
  onChange: (value: unknown) => void
}) {
  const inputStyle = {
    width: '100%',
    padding: '6px 10px',
    border: '1px solid var(--border-color)',
    borderRadius: 'var(--radius-sm)',
    fontSize: 13,
    outline: 'none',
  }

  switch (property.type) {
    case 'text':
    case 'url':
    case 'email':
    case 'phone':
      return (
        <input
          type="text"
          value={(value as string) || ''}
          onChange={(e) => onChange(e.target.value)}
          placeholder={`Default ${property.name.toLowerCase()}`}
          style={inputStyle}
        />
      )

    case 'number':
      return (
        <input
          type="number"
          value={(value as number) || ''}
          onChange={(e) => onChange(e.target.value ? parseFloat(e.target.value) : null)}
          placeholder="0"
          style={inputStyle}
        />
      )

    case 'select':
    case 'status':
      return (
        <select
          value={(value as string) || ''}
          onChange={(e) => onChange(e.target.value || null)}
          style={inputStyle}
        >
          <option value="">No default</option>
          {property.options?.map((opt) => (
            <option key={opt.id} value={opt.id}>
              {opt.name}
            </option>
          ))}
        </select>
      )

    case 'checkbox':
      return (
        <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
          <input
            type="checkbox"
            checked={(value as boolean) || false}
            onChange={(e) => onChange(e.target.checked)}
          />
          <span style={{ fontSize: 13 }}>Checked by default</span>
        </label>
      )

    default:
      return (
        <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
          Not configurable
        </span>
      )
  }
}
