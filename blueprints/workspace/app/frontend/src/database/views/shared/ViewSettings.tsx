import { useState, useRef, useEffect } from 'react'
import { Settings, X, Eye, EyeOff, GripVertical } from 'lucide-react'
import { Property } from '../../../api/client'

export interface ViewSettingsProps {
  // Common settings
  properties: Property[]
  visibleProperties: string[]
  propertyOrder: string[]
  onVisibilityChange: (propertyId: string, visible: boolean) => void
  onPropertyOrderChange: (newOrder: string[]) => void

  // Table-specific
  showCalculations?: boolean
  onShowCalculationsChange?: (show: boolean) => void
  rowHeight?: 'small' | 'medium' | 'tall' | 'extra_tall'
  onRowHeightChange?: (height: 'small' | 'medium' | 'tall' | 'extra_tall') => void
  wrapCells?: boolean
  onWrapCellsChange?: (wrap: boolean) => void

  // Board-specific
  colorColumns?: boolean
  onColorColumnsChange?: (color: boolean) => void
  hideEmptyGroups?: boolean
  onHideEmptyGroupsChange?: (hide: boolean) => void
  cardSize?: 'small' | 'medium' | 'large'
  onCardSizeChange?: (size: 'small' | 'medium' | 'large') => void
  cardPreview?: 'none' | 'page_cover' | 'page_content' | 'files'
  onCardPreviewChange?: (preview: 'none' | 'page_cover' | 'page_content' | 'files') => void

  // Gallery-specific
  fitImage?: boolean
  onFitImageChange?: (fit: boolean) => void
  showTitle?: boolean
  onShowTitleChange?: (show: boolean) => void

  // Timeline-specific
  showTablePanel?: boolean
  onShowTablePanelChange?: (show: boolean) => void
  showDependencies?: boolean
  onShowDependenciesChange?: (show: boolean) => void
  timeScale?: 'hours' | 'days' | 'weeks' | 'months' | 'quarters' | 'years'
  onTimeScaleChange?: (scale: 'hours' | 'days' | 'weeks' | 'months' | 'quarters' | 'years') => void

  // Calendar-specific
  calendarMode?: 'month' | 'week' | 'day'
  onCalendarModeChange?: (mode: 'month' | 'week' | 'day') => void
  startWeekOnMonday?: boolean
  onStartWeekOnMondayChange?: (start: boolean) => void
}

export function ViewSettings({
  properties,
  visibleProperties,
  propertyOrder,
  onVisibilityChange,
  onPropertyOrderChange,
  showCalculations,
  onShowCalculationsChange,
  rowHeight,
  onRowHeightChange,
  wrapCells,
  onWrapCellsChange,
  colorColumns,
  onColorColumnsChange,
  hideEmptyGroups,
  onHideEmptyGroupsChange,
  cardSize,
  onCardSizeChange,
  cardPreview,
  onCardPreviewChange,
  fitImage,
  onFitImageChange,
  showTitle,
  onShowTitleChange,
  showTablePanel,
  onShowTablePanelChange,
  showDependencies,
  onShowDependenciesChange,
  timeScale,
  onTimeScaleChange,
  calendarMode,
  onCalendarModeChange,
  startWeekOnMonday,
  onStartWeekOnMondayChange,
}: ViewSettingsProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [activeTab, setActiveTab] = useState<'properties' | 'layout'>('properties')
  const panelRef = useRef<HTMLDivElement>(null)
  const [draggingId, setDraggingId] = useState<string | null>(null)

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (panelRef.current && !panelRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
    }
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [isOpen])

  // Order properties
  const orderedProperties = [...properties].sort((a, b) => {
    const aIndex = propertyOrder.indexOf(a.id)
    const bIndex = propertyOrder.indexOf(b.id)
    if (aIndex === -1 && bIndex === -1) return 0
    if (aIndex === -1) return 1
    if (bIndex === -1) return -1
    return aIndex - bIndex
  })

  const handleDragStart = (propertyId: string) => {
    setDraggingId(propertyId)
  }

  const handleDragOver = (e: React.DragEvent, targetId: string) => {
    e.preventDefault()
    if (!draggingId || draggingId === targetId) return

    const newOrder = [...propertyOrder]
    const draggingIndex = newOrder.indexOf(draggingId)
    const targetIndex = newOrder.indexOf(targetId)

    if (draggingIndex === -1 || targetIndex === -1) return

    newOrder.splice(draggingIndex, 1)
    newOrder.splice(targetIndex, 0, draggingId)
    onPropertyOrderChange(newOrder)
  }

  const handleDragEnd = () => {
    setDraggingId(null)
  }

  return (
    <div className="view-settings-container" ref={panelRef}>
      <button
        type="button"
        className={`view-settings-trigger ${isOpen ? 'active' : ''}`}
        onClick={() => setIsOpen(!isOpen)}
      >
        <Settings size={14} />
      </button>

      {isOpen && (
        <div className="view-settings-panel">
          <div className="settings-header">
            <div className="settings-tabs">
              <button
                type="button"
                className={`tab ${activeTab === 'properties' ? 'active' : ''}`}
                onClick={() => setActiveTab('properties')}
              >
                Properties
              </button>
              <button
                type="button"
                className={`tab ${activeTab === 'layout' ? 'active' : ''}`}
                onClick={() => setActiveTab('layout')}
              >
                Layout
              </button>
            </div>
            <button
              type="button"
              className="close-btn"
              onClick={() => setIsOpen(false)}
            >
              <X size={16} />
            </button>
          </div>

          <div className="settings-content">
            {activeTab === 'properties' && (
              <div className="properties-list">
                {orderedProperties.map((property) => {
                  const isVisible = visibleProperties.includes(property.id)
                  return (
                    <div
                      key={property.id}
                      className={`property-item ${draggingId === property.id ? 'dragging' : ''}`}
                      draggable
                      onDragStart={() => handleDragStart(property.id)}
                      onDragOver={(e) => handleDragOver(e, property.id)}
                      onDragEnd={handleDragEnd}
                    >
                      <span className="drag-handle">
                        <GripVertical size={12} />
                      </span>
                      <span className="property-name">{property.name}</span>
                      <button
                        type="button"
                        className="visibility-toggle"
                        onClick={() => onVisibilityChange(property.id, !isVisible)}
                      >
                        {isVisible ? (
                          <Eye size={14} className="visible" />
                        ) : (
                          <EyeOff size={14} className="hidden" />
                        )}
                      </button>
                    </div>
                  )
                })}
              </div>
            )}

            {activeTab === 'layout' && (
              <div className="layout-options">
                {/* Row Height - Table */}
                {onRowHeightChange && rowHeight !== undefined && (
                  <div className="option-group">
                    <label>Row height</label>
                    <div className="button-group">
                      {(['small', 'medium', 'tall', 'extra_tall'] as const).map((h) => (
                        <button
                          key={h}
                          type="button"
                          className={rowHeight === h ? 'active' : ''}
                          onClick={() => onRowHeightChange(h)}
                        >
                          {h === 'small' ? 'S' : h === 'medium' ? 'M' : h === 'tall' ? 'L' : 'XL'}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {/* Card Size - Board/Gallery */}
                {onCardSizeChange && cardSize !== undefined && (
                  <div className="option-group">
                    <label>Card size</label>
                    <div className="button-group">
                      {(['small', 'medium', 'large'] as const).map((s) => (
                        <button
                          key={s}
                          type="button"
                          className={cardSize === s ? 'active' : ''}
                          onClick={() => onCardSizeChange(s)}
                        >
                          {s === 'small' ? 'S' : s === 'medium' ? 'M' : 'L'}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {/* Time Scale - Timeline */}
                {onTimeScaleChange && timeScale !== undefined && (
                  <div className="option-group">
                    <label>Time scale</label>
                    <select
                      value={timeScale}
                      onChange={(e) =>
                        onTimeScaleChange(
                          e.target.value as 'hours' | 'days' | 'weeks' | 'months' | 'quarters' | 'years'
                        )
                      }
                    >
                      <option value="hours">Hours</option>
                      <option value="days">Days</option>
                      <option value="weeks">Weeks</option>
                      <option value="months">Months</option>
                      <option value="quarters">Quarters</option>
                      <option value="years">Years</option>
                    </select>
                  </div>
                )}

                {/* Calendar Mode */}
                {onCalendarModeChange && calendarMode !== undefined && (
                  <div className="option-group">
                    <label>Calendar view</label>
                    <div className="button-group">
                      {(['month', 'week', 'day'] as const).map((m) => (
                        <button
                          key={m}
                          type="button"
                          className={calendarMode === m ? 'active' : ''}
                          onClick={() => onCalendarModeChange(m)}
                        >
                          {m.charAt(0).toUpperCase() + m.slice(1)}
                        </button>
                      ))}
                    </div>
                  </div>
                )}

                {/* Card Preview - Board */}
                {onCardPreviewChange && cardPreview !== undefined && (
                  <div className="option-group">
                    <label>Card preview</label>
                    <select
                      value={cardPreview}
                      onChange={(e) =>
                        onCardPreviewChange(
                          e.target.value as 'none' | 'page_cover' | 'page_content' | 'files'
                        )
                      }
                    >
                      <option value="none">None</option>
                      <option value="page_cover">Page cover</option>
                      <option value="page_content">Page content</option>
                      <option value="files">Files property</option>
                    </select>
                  </div>
                )}

                {/* Toggle Options */}
                <div className="toggle-options">
                  {onShowCalculationsChange && showCalculations !== undefined && (
                    <label className="toggle-option">
                      <input
                        type="checkbox"
                        checked={showCalculations}
                        onChange={(e) => onShowCalculationsChange(e.target.checked)}
                      />
                      <span>Show calculations</span>
                    </label>
                  )}

                  {onWrapCellsChange && wrapCells !== undefined && (
                    <label className="toggle-option">
                      <input
                        type="checkbox"
                        checked={wrapCells}
                        onChange={(e) => onWrapCellsChange(e.target.checked)}
                      />
                      <span>Wrap all cells</span>
                    </label>
                  )}

                  {onColorColumnsChange && colorColumns !== undefined && (
                    <label className="toggle-option">
                      <input
                        type="checkbox"
                        checked={colorColumns}
                        onChange={(e) => onColorColumnsChange(e.target.checked)}
                      />
                      <span>Color columns</span>
                    </label>
                  )}

                  {onHideEmptyGroupsChange && hideEmptyGroups !== undefined && (
                    <label className="toggle-option">
                      <input
                        type="checkbox"
                        checked={hideEmptyGroups}
                        onChange={(e) => onHideEmptyGroupsChange(e.target.checked)}
                      />
                      <span>Hide empty groups</span>
                    </label>
                  )}

                  {onFitImageChange && fitImage !== undefined && (
                    <label className="toggle-option">
                      <input
                        type="checkbox"
                        checked={fitImage}
                        onChange={(e) => onFitImageChange(e.target.checked)}
                      />
                      <span>Fit image</span>
                    </label>
                  )}

                  {onShowTitleChange && showTitle !== undefined && (
                    <label className="toggle-option">
                      <input
                        type="checkbox"
                        checked={showTitle}
                        onChange={(e) => onShowTitleChange(e.target.checked)}
                      />
                      <span>Show title</span>
                    </label>
                  )}

                  {onShowTablePanelChange && showTablePanel !== undefined && (
                    <label className="toggle-option">
                      <input
                        type="checkbox"
                        checked={showTablePanel}
                        onChange={(e) => onShowTablePanelChange(e.target.checked)}
                      />
                      <span>Show table</span>
                    </label>
                  )}

                  {onShowDependenciesChange && showDependencies !== undefined && (
                    <label className="toggle-option">
                      <input
                        type="checkbox"
                        checked={showDependencies}
                        onChange={(e) => onShowDependenciesChange(e.target.checked)}
                      />
                      <span>Show dependencies</span>
                    </label>
                  )}

                  {onStartWeekOnMondayChange && startWeekOnMonday !== undefined && (
                    <label className="toggle-option">
                      <input
                        type="checkbox"
                        checked={startWeekOnMonday}
                        onChange={(e) => onStartWeekOnMondayChange(e.target.checked)}
                      />
                      <span>Start week on Monday</span>
                    </label>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      <style>{`
        .view-settings-container {
          position: relative;
        }

        .view-settings-trigger {
          display: flex;
          align-items: center;
          justify-content: center;
          padding: 4px 8px;
          background: none;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          color: #9a9a97;
          transition: all 0.15s;
        }

        .view-settings-trigger:hover,
        .view-settings-trigger.active {
          background: #f7f6f3;
          color: #37352f;
        }

        .view-settings-panel {
          position: absolute;
          top: 100%;
          right: 0;
          margin-top: 4px;
          background: #fff;
          border: 1px solid rgba(55, 53, 47, 0.09);
          border-radius: 8px;
          box-shadow: rgba(15, 15, 15, 0.05) 0px 0px 0px 1px,
            rgba(15, 15, 15, 0.1) 0px 3px 6px, rgba(15, 15, 15, 0.2) 0px 9px 24px;
          width: 280px;
          z-index: 100;
          overflow: hidden;
        }

        .settings-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 8px 8px 0;
          border-bottom: 1px solid rgba(55, 53, 47, 0.09);
        }

        .settings-tabs {
          display: flex;
          gap: 2px;
        }

        .settings-tabs .tab {
          padding: 6px 12px;
          border: none;
          background: none;
          cursor: pointer;
          font-size: 13px;
          color: #787774;
          border-bottom: 2px solid transparent;
          margin-bottom: -1px;
          transition: all 0.15s;
        }

        .settings-tabs .tab:hover {
          color: #37352f;
        }

        .settings-tabs .tab.active {
          color: #37352f;
          border-bottom-color: #37352f;
        }

        .close-btn {
          padding: 4px;
          border: none;
          background: none;
          cursor: pointer;
          color: #787774;
          border-radius: 4px;
          display: flex;
        }

        .close-btn:hover {
          background: rgba(55, 53, 47, 0.08);
          color: #37352f;
        }

        .settings-content {
          max-height: 360px;
          overflow-y: auto;
        }

        .properties-list {
          padding: 8px 0;
        }

        .property-item {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 6px 12px;
          cursor: grab;
          transition: background 0.1s;
        }

        .property-item:hover {
          background: rgba(55, 53, 47, 0.04);
        }

        .property-item.dragging {
          opacity: 0.5;
          background: rgba(35, 131, 226, 0.1);
        }

        .drag-handle {
          color: #9a9a97;
          cursor: grab;
        }

        .property-name {
          flex: 1;
          font-size: 13px;
          color: #37352f;
        }

        .visibility-toggle {
          padding: 4px;
          border: none;
          background: none;
          cursor: pointer;
          border-radius: 4px;
          display: flex;
        }

        .visibility-toggle .visible {
          color: #2383e2;
        }

        .visibility-toggle .hidden {
          color: #9a9a97;
        }

        .layout-options {
          padding: 12px;
        }

        .option-group {
          margin-bottom: 16px;
        }

        .option-group label {
          display: block;
          font-size: 11px;
          font-weight: 500;
          color: #787774;
          text-transform: uppercase;
          letter-spacing: 0.5px;
          margin-bottom: 6px;
        }

        .option-group select {
          width: 100%;
          padding: 6px 8px;
          border: 1px solid rgba(55, 53, 47, 0.16);
          border-radius: 4px;
          font-size: 13px;
          background: #fff;
          outline: none;
        }

        .option-group select:focus {
          border-color: #2383e2;
        }

        .button-group {
          display: flex;
          gap: 2px;
          background: rgba(55, 53, 47, 0.06);
          padding: 2px;
          border-radius: 6px;
        }

        .button-group button {
          flex: 1;
          padding: 4px 8px;
          border: none;
          background: none;
          cursor: pointer;
          font-size: 12px;
          color: #787774;
          border-radius: 4px;
          transition: all 0.15s;
        }

        .button-group button:hover {
          color: #37352f;
        }

        .button-group button.active {
          background: #fff;
          color: #37352f;
          box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
        }

        .toggle-options {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .toggle-option {
          display: flex;
          align-items: center;
          gap: 8px;
          cursor: pointer;
          font-size: 13px;
          color: #37352f;
        }

        .toggle-option input {
          width: 16px;
          height: 16px;
          accent-color: #2383e2;
        }
      `}</style>
    </div>
  )
}

export default ViewSettings
