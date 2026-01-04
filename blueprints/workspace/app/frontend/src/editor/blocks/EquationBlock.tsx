import { createReactBlockSpec } from '@blocknote/react'
import { useState, useRef, useEffect } from 'react'
import katex from 'katex'
import 'katex/dist/katex.min.css'

export const EquationBlock = createReactBlockSpec(
  {
    type: 'equation',
    propSchema: {
      latex: {
        default: '',
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [isEditing, setIsEditing] = useState(!block.props.latex)
      const [latexInput, setLatexInput] = useState(block.props.latex || '')
      const [error, setError] = useState<string | null>(null)
      const inputRef = useRef<HTMLTextAreaElement>(null)
      const renderedRef = useRef<HTMLDivElement>(null)
      const previewRef = useRef<HTMLDivElement>(null)

      useEffect(() => {
        if (isEditing && inputRef.current) {
          inputRef.current.focus()
        }
      }, [isEditing])

      // Render equation in view mode
      useEffect(() => {
        if (!isEditing && renderedRef.current && block.props.latex) {
          try {
            katex.render(block.props.latex, renderedRef.current, {
              displayMode: true,
              throwOnError: false,
              errorColor: '#e03e3e',
            })
            setError(null)
          } catch (e) {
            setError((e as Error).message)
          }
        }
      }, [block.props.latex, isEditing])

      // Render preview in edit mode
      useEffect(() => {
        if (isEditing && previewRef.current && latexInput) {
          try {
            katex.render(latexInput, previewRef.current, {
              displayMode: true,
              throwOnError: false,
            })
          } catch {
            if (previewRef.current) {
              previewRef.current.textContent = 'Invalid LaTeX'
            }
          }
        }
      }, [latexInput, isEditing])

      const handleSave = () => {
        editor.updateBlock(block, { props: { ...block.props, latex: latexInput } })
        setIsEditing(false)
      }

      const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
          e.preventDefault()
          handleSave()
        }
        if (e.key === 'Escape') {
          setLatexInput(block.props.latex || '')
          setIsEditing(false)
        }
      }

      if (isEditing) {
        return (
          <div className="equation-block editing">
            <div className="equation-editor">
              <textarea
                ref={inputRef}
                value={latexInput}
                onChange={(e) => setLatexInput(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Enter LaTeX (e.g., E = mc^2, \\frac{a}{b}, \\sqrt{x})"
                className="equation-input"
                rows={2}
              />
              {latexInput && (
                <div className="equation-preview">
                  <div ref={previewRef} />
                </div>
              )}
              <div className="equation-actions">
                <button className="equation-btn-cancel" onClick={() => {
                  setLatexInput(block.props.latex || '')
                  setIsEditing(false)
                }}>
                  Cancel
                </button>
                <button className="equation-btn-done" onClick={handleSave}>
                  Done
                </button>
              </div>
            </div>
          </div>
        )
      }

      return (
        <div
          className="equation-block"
          data-equation-latex={block.props.latex}
          onClick={() => setIsEditing(true)}
          title="Click to edit equation"
        >
          {block.props.latex ? (
            <div ref={renderedRef} className="equation-rendered" />
          ) : (
            <div className="equation-placeholder">
              Click to add an equation
            </div>
          )}
          {error && <div className="equation-error">{error}</div>}
        </div>
      )
    },
    // Parse HTML to recreate block when pasting or drag-dropping
    parse: (element: HTMLElement) => {
      if (element.classList.contains('equation-block') || element.hasAttribute('data-equation-latex')) {
        return {
          latex: element.getAttribute('data-equation-latex') || '',
        }
      }
      return undefined
    },
    // Convert to external HTML for clipboard/export
    toExternalHTML: ({ block }) => {
      const latex = (block.props.latex as string) || ''
      return (
        <div className="equation-block" data-equation-latex={latex}>
          {latex || 'Empty equation'}
        </div>
      )
    },
  }
)
