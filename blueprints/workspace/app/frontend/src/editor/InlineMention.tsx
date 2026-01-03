import { createReactInlineContentSpec } from '@blocknote/react'
import { AtSign, FileText, Calendar } from 'lucide-react'

// Inline mention component for rendering @mentions
export const MentionInline = createReactInlineContentSpec(
  {
    type: 'mention',
    propSchema: {
      mentionType: {
        default: 'user' as 'user' | 'page' | 'date',
      },
      id: {
        default: '',
      },
      label: {
        default: '',
      },
    },
    content: 'none',
  },
  {
    render: ({ inlineContent }) => {
      const mentionType = inlineContent.props.mentionType as 'user' | 'page' | 'date'
      const label = inlineContent.props.label as string
      const id = inlineContent.props.id as string

      const getIcon = () => {
        switch (mentionType) {
          case 'user':
            return <AtSign size={12} style={{ marginRight: '2px' }} />
          case 'page':
            return <FileText size={12} style={{ marginRight: '2px' }} />
          case 'date':
            return <Calendar size={12} style={{ marginRight: '2px' }} />
          default:
            return null
        }
      }

      const handleClick = () => {
        if (mentionType === 'user') {
          // Navigate to user profile or show user card
          console.log('User mention clicked:', id)
        } else if (mentionType === 'page') {
          // Navigate to page
          window.location.href = `/pages/${id}`
        } else if (mentionType === 'date') {
          // Could open date picker or show date info
          console.log('Date mention clicked:', id)
        }
      }

      return (
        <span
          onClick={handleClick}
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            padding: '1px 4px',
            borderRadius: '4px',
            background: 'var(--accent-bg, rgba(35, 131, 226, 0.1))',
            color: 'var(--accent-color, #2383e2)',
            fontSize: 'inherit',
            fontWeight: 500,
            cursor: 'pointer',
            transition: 'background 0.1s',
            verticalAlign: 'baseline',
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = 'rgba(35, 131, 226, 0.2)'
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = 'var(--accent-bg, rgba(35, 131, 226, 0.1))'
          }}
        >
          {getIcon()}
          {label}
        </span>
      )
    },
  }
)

// Export mention types for use elsewhere
export type MentionType = 'user' | 'page' | 'date'

export interface MentionData {
  type: MentionType
  id: string
  label: string
  data?: unknown
}
