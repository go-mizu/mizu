import { createReactBlockSpec } from '@blocknote/react'
import { useState, useCallback } from 'react'
import { FileText, Plus } from 'lucide-react'
import { motion } from 'framer-motion'
import { api } from '../../api/client'

export const ChildPageBlock = createReactBlockSpec(
  {
    type: 'childPage',
    propSchema: {
      pageId: {
        default: '',
      },
      title: {
        default: 'Untitled',
      },
      icon: {
        default: '',
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [isHovered, setIsHovered] = useState(false)
      const [isCreating, setIsCreating] = useState(false)

      const pageId = block.props.pageId as string
      const title = block.props.title as string
      const icon = block.props.icon as string

      // Navigate to child page
      const handleClick = useCallback(() => {
        if (pageId) {
          window.location.href = `/pages/${pageId}`
        }
      }, [pageId])

      // Create a new child page if none exists
      const handleCreate = useCallback(async () => {
        if (pageId) return // Already has a page

        setIsCreating(true)
        try {
          // Get parent page ID from editor
          const editorRoot = document.getElementById('editor-root')
          const parentId = editorRoot?.dataset.pageId

          if (!parentId) {
            console.error('No parent page ID found')
            return
          }

          // Create new child page
          const response = await api.post<{ id: string; title: string }>('/pages', {
            parent_id: parentId,
            title: 'Untitled',
          })

          // Update block with new page ID
          editor.updateBlock(block, {
            props: {
              ...block.props,
              pageId: response.id,
              title: response.title,
            },
          })

          // Navigate to new page
          window.location.href = `/pages/${response.id}`
        } catch (err) {
          console.error('Failed to create child page:', err)
        } finally {
          setIsCreating(false)
        }
      }, [pageId, block, editor])

      // Empty state - no page linked yet
      if (!pageId) {
        return (
          <button
            className="child-page-block empty"
            onClick={handleCreate}
            disabled={isCreating}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
              width: '100%',
              padding: '8px 10px',
              background: 'none',
              border: '1px dashed var(--border-color)',
              borderRadius: '4px',
              fontSize: '14px',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
              textAlign: 'left',
              transition: 'all 0.1s',
              margin: '4px 0',
            }}
            onMouseEnter={(e) => {
              e.currentTarget.style.borderColor = 'var(--accent-color)'
              e.currentTarget.style.background = 'var(--bg-hover)'
            }}
            onMouseLeave={(e) => {
              e.currentTarget.style.borderColor = 'var(--border-color)'
              e.currentTarget.style.background = 'none'
            }}
          >
            <Plus size={18} style={{ color: 'var(--text-tertiary)' }} />
            <span>{isCreating ? 'Creating page...' : 'Add a sub-page'}</span>
          </button>
        )
      }

      return (
        <motion.div
          className="child-page-block"
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
          onClick={handleClick}
          whileHover={{ scale: 1.005 }}
          whileTap={{ scale: 0.995 }}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            padding: '6px 8px',
            margin: '4px 0',
            borderRadius: '3px',
            cursor: 'pointer',
            background: isHovered ? 'var(--bg-hover)' : 'transparent',
            transition: 'background 0.1s',
          }}
        >
          {/* Page icon */}
          <span
            style={{
              fontSize: '18px',
              width: '22px',
              height: '22px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            {icon || <FileText size={18} style={{ color: 'var(--text-tertiary)' }} />}
          </span>

          {/* Page title */}
          <span
            style={{
              fontSize: '14px',
              color: 'var(--text-primary)',
              flex: 1,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {title}
          </span>

          {/* Arrow indicator on hover */}
          {isHovered && (
            <motion.span
              initial={{ opacity: 0, x: -4 }}
              animate={{ opacity: 1, x: 0 }}
              style={{
                color: 'var(--text-tertiary)',
                fontSize: '12px',
              }}
            >
              →
            </motion.span>
          )}
        </motion.div>
      )
    },
    parse: (element: HTMLElement) => {
      if (element.classList.contains('child-page-block')) {
        return {
          pageId: element.getAttribute('data-page-id') || '',
          title: element.getAttribute('data-title') || 'Untitled',
          icon: element.getAttribute('data-icon') || '',
        }
      }
      return undefined
    },
    toExternalHTML: ({ block }) => {
      const { pageId, title, icon } = block.props as Record<string, string>
      return (
        <div
          className="child-page-block"
          data-page-id={pageId}
          data-title={title}
          data-icon={icon}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            padding: '6px 8px',
          }}
        >
          <span>{icon || '📄'}</span>
          <span>{title}</span>
        </div>
      )
    },
  }
)
