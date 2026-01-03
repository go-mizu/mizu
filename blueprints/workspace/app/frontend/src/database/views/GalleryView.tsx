import { useState, useCallback, useMemo } from 'react'
import { DatabaseRow, Property } from '../../api/client'
import { PropertyCell } from '../PropertyCell'

interface GalleryViewProps {
  rows: DatabaseRow[]
  properties: Property[]
  groupBy: string | null
  onAddRow: () => void
  onUpdateRow: (rowId: string, updates: Record<string, unknown>) => void
  onDeleteRow: (rowId: string) => void
  onAddProperty: (property: Omit<Property, 'id'>) => void
  onUpdateProperty: (propertyId: string, updates: Partial<Property>) => void
  onDeleteProperty: (propertyId: string) => void
}

type CardSize = 'small' | 'medium' | 'large'
type ImageFit = 'cover' | 'contain'

export function GalleryView({
  rows,
  properties,
  onAddRow,
  onUpdateRow,
  onDeleteRow,
}: GalleryViewProps) {
  const [cardSize, setCardSize] = useState<CardSize>('medium')
  const [imageFit, setImageFit] = useState<ImageFit>('cover')
  const [hoveredCard, setHoveredCard] = useState<string | null>(null)

  // Find cover image property (files property)
  const coverProperty = useMemo(() => {
    return properties.find((p) => p.type === 'files' || p.name.toLowerCase().includes('cover') || p.name.toLowerCase().includes('image'))
  }, [properties])

  // Get title property
  const titleProperty = useMemo(() => {
    return properties.find((p) => p.type === 'text') || properties[0]
  }, [properties])

  // Get preview properties
  const previewProperties = useMemo(() => {
    return properties
      .filter((p) => p.id !== titleProperty?.id && p.id !== coverProperty?.id)
      .slice(0, 3)
  }, [properties, titleProperty, coverProperty])

  // Get card class based on size
  const getCardClass = () => {
    switch (cardSize) {
      case 'small':
        return 'gallery-card small'
      case 'large':
        return 'gallery-card large'
      default:
        return 'gallery-card medium'
    }
  }

  // Get cover image URL for a row
  const getCoverUrl = useCallback((row: DatabaseRow): string | null => {
    if (!coverProperty) return null

    const value = row.properties[coverProperty.id]
    if (typeof value === 'string') return value
    if (Array.isArray(value) && value.length > 0) {
      const first = value[0]
      if (typeof first === 'string') return first
      if (first && typeof first === 'object' && 'url' in first) return first.url as string
    }
    return null
  }, [coverProperty])

  return (
    <div className="gallery-view">
      {/* Gallery toolbar */}
      <div className="gallery-toolbar">
        <div className="size-options">
          <span>Card size:</span>
          <button
            className={cardSize === 'small' ? 'active' : ''}
            onClick={() => setCardSize('small')}
          >
            S
          </button>
          <button
            className={cardSize === 'medium' ? 'active' : ''}
            onClick={() => setCardSize('medium')}
          >
            M
          </button>
          <button
            className={cardSize === 'large' ? 'active' : ''}
            onClick={() => setCardSize('large')}
          >
            L
          </button>
        </div>

        <div className="fit-options">
          <span>Image fit:</span>
          <button
            className={imageFit === 'cover' ? 'active' : ''}
            onClick={() => setImageFit('cover')}
          >
            Fill
          </button>
          <button
            className={imageFit === 'contain' ? 'active' : ''}
            onClick={() => setImageFit('contain')}
          >
            Fit
          </button>
        </div>
      </div>

      {/* Gallery grid */}
      <div className={`gallery-grid ${cardSize}`}>
        {rows.map((row) => {
          const coverUrl = getCoverUrl(row)
          const isHovered = hoveredCard === row.id

          return (
            <div
              key={row.id}
              className={getCardClass()}
              onMouseEnter={() => setHoveredCard(row.id)}
              onMouseLeave={() => setHoveredCard(null)}
            >
              {/* Cover image */}
              <div className="card-cover" style={{ objectFit: imageFit }}>
                {coverUrl ? (
                  <img src={coverUrl} alt="" style={{ objectFit: imageFit }} />
                ) : (
                  <div className="cover-placeholder">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                      <rect x="3" y="3" width="18" height="18" rx="2" stroke="currentColor" strokeWidth="2" />
                      <circle cx="8" cy="8" r="2" fill="currentColor" />
                      <path d="M21 15l-5-5-6 6-3-3-4 4" stroke="currentColor" strokeWidth="2" />
                    </svg>
                  </div>
                )}
              </div>

              {/* Card content */}
              <div className="card-content">
                {/* Title */}
                <div className="card-title">
                  {titleProperty && (
                    <PropertyCell
                      property={titleProperty}
                      value={row.properties[titleProperty.id]}
                      onChange={(value) => onUpdateRow(row.id, { [titleProperty.id]: value })}
                    />
                  )}
                </div>

                {/* Properties */}
                <div className="card-properties">
                  {previewProperties.map((property) => (
                    <div key={property.id} className="card-property">
                      <span className="property-name">{property.name}</span>
                      <PropertyCell
                        property={property}
                        value={row.properties[property.id]}
                        onChange={(value) => onUpdateRow(row.id, { [property.id]: value })}
                      />
                    </div>
                  ))}
                </div>
              </div>

              {/* Actions overlay */}
              {isHovered && (
                <div className="card-actions-overlay">
                  <button
                    className="card-action"
                    onClick={() => {
                      // TODO: Open card detail
                    }}
                  >
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                      <path d="M7 1v12M1 7h12" stroke="currentColor" strokeWidth="2" />
                    </svg>
                    Open
                  </button>
                  <button className="card-action delete" onClick={() => onDeleteRow(row.id)}>
                    <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                      <path d="M2 4h10M5 4V2h4v2M3 4v8a1 1 0 001 1h6a1 1 0 001-1V4" stroke="currentColor" strokeWidth="1.5" />
                    </svg>
                    Delete
                  </button>
                </div>
              )}
            </div>
          )
        })}

        {/* Add card */}
        <div className="gallery-card add-card" onClick={onAddRow}>
          <div className="add-card-content">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M12 5v14M5 12h14" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
            </svg>
            <span>New</span>
          </div>
        </div>
      </div>
    </div>
  )
}
