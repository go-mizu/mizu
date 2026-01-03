import { createReactBlockSpec } from '@blocknote/react'
import { useEffect, useState } from 'react'

interface HeadingInfo {
  id: string
  text: string
  level: number
}

export const TableOfContentsBlock = createReactBlockSpec(
  {
    type: 'tableOfContents',
    propSchema: {},
    content: 'none',
  },
  {
    render: ({ editor }) => {
      const [headings, setHeadings] = useState<HeadingInfo[]>([])

      useEffect(() => {
        const updateHeadings = () => {
          const blocks = editor.document
          const extractedHeadings: HeadingInfo[] = []

          const extractFromBlocks = (blocks: any[]) => {
            for (const block of blocks) {
              if (block.type === 'heading') {
                const text = block.content
                  ?.map((c: any) => (typeof c === 'string' ? c : c.text || ''))
                  .join('') || ''
                if (text.trim()) {
                  extractedHeadings.push({
                    id: block.id,
                    text,
                    level: block.props?.level || 1,
                  })
                }
              }
              if (block.children) {
                extractFromBlocks(block.children)
              }
            }
          }

          extractFromBlocks(blocks)
          setHeadings(extractedHeadings)
        }

        updateHeadings()
        editor.onEditorContentChange(updateHeadings)
      }, [editor])

      const scrollToHeading = (id: string) => {
        const element = document.querySelector(`[data-id="${id}"]`)
        if (element) {
          element.scrollIntoView({ behavior: 'smooth', block: 'center' })
        }
      }

      if (headings.length === 0) {
        return (
          <div className="toc-block empty">
            <div className="toc-placeholder">
              Add headings to the page to create a table of contents
            </div>
          </div>
        )
      }

      return (
        <div className="toc-block">
          <div className="toc-title">Table of Contents</div>
          <nav className="toc-list">
            {headings.map((heading) => (
              <button
                key={heading.id}
                className={`toc-item level-${heading.level}`}
                onClick={() => scrollToHeading(heading.id)}
                style={{ paddingLeft: `${(heading.level - 1) * 16 + 8}px` }}
              >
                {heading.text}
              </button>
            ))}
          </nav>
        </div>
      )
    },
  }
)
