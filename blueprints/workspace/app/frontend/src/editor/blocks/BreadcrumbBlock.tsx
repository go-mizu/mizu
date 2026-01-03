import { createReactBlockSpec } from '@blocknote/react'
import React, { useState, useEffect } from 'react'
import { ChevronRight, FileText } from 'lucide-react'
import { api } from '../../api/client'

interface PageHierarchy {
  id: string
  title: string
  icon?: string
}

export const BreadcrumbBlock = createReactBlockSpec(
  {
    type: 'breadcrumb',
    propSchema: {},
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [path, setPath] = useState<PageHierarchy[]>([])
      const [isLoading, setIsLoading] = useState(true)

      useEffect(() => {
        // Get page ID from editor or document
        const fetchHierarchy = async () => {
          try {
            // Try to get the current page ID from the editor container
            const editorRoot = document.getElementById('editor-root')
            const pageId = editorRoot?.dataset.pageId

            if (!pageId) {
              setIsLoading(false)
              return
            }

            const response = await api.get<{ path: PageHierarchy[] }>(
              `/pages/${pageId}/hierarchy`
            )
            setPath(response.path || [])
          } catch (err) {
            console.error('Failed to fetch page hierarchy:', err)
            // Fallback to sample path for demo
            setPath([
              { id: '1', title: 'Workspace', icon: '🏠' },
              { id: '2', title: 'Projects', icon: '📁' },
              { id: '3', title: 'Current Page', icon: '📄' },
            ])
          } finally {
            setIsLoading(false)
          }
        }

        fetchHierarchy()
      }, [])

      if (isLoading) {
        return (
          <nav
            className="breadcrumb-block"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '4px',
              padding: '4px 0',
              fontSize: '14px',
              color: 'var(--text-tertiary)',
            }}
          >
            Loading...
          </nav>
        )
      }

      if (path.length === 0) {
        return (
          <nav
            className="breadcrumb-block"
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '4px',
              padding: '4px 0',
              fontSize: '14px',
              color: 'var(--text-tertiary)',
            }}
          >
            <FileText size={14} />
            <span>Page</span>
          </nav>
        )
      }

      return (
        <nav
          className="breadcrumb-block"
          style={{
            display: 'flex',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: '2px',
            padding: '4px 0',
            fontSize: '14px',
          }}
        >
          {path.map((page, index) => (
            <React.Fragment key={page.id}>
              {index > 0 && (
                <ChevronRight
                  size={12}
                  style={{
                    color: 'var(--text-tertiary)',
                    flexShrink: 0,
                  }}
                />
              )}
              <a
                href={`/pages/${page.id}`}
                onClick={(e) => {
                  e.preventDefault()
                  window.location.href = `/pages/${page.id}`
                }}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '4px',
                  padding: '2px 6px',
                  borderRadius: '3px',
                  color: index === path.length - 1 ? 'var(--text-primary)' : 'var(--text-secondary)',
                  textDecoration: 'none',
                  transition: 'background 0.1s',
                  fontWeight: index === path.length - 1 ? 500 : 400,
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-hover)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'transparent'
                }}
              >
                {page.icon && <span style={{ fontSize: '14px' }}>{page.icon}</span>}
                <span>{page.title}</span>
              </a>
            </React.Fragment>
          ))}
        </nav>
      )
    },
    parse: (element: HTMLElement) => {
      if (element.classList.contains('breadcrumb-block')) {
        return {}
      }
      return undefined
    },
    toExternalHTML: () => {
      return (
        <nav className="breadcrumb-block" style={{ fontSize: '14px', color: '#787774' }}>
          Breadcrumb
        </nav>
      )
    },
  }
)
