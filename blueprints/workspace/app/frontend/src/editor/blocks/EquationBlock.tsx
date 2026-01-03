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

      useEffect(() => {
        if (isEditing && inputRef.current) {
          inputRef.current.focus()
        }
      }, [isEditing])

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
                placeholder="Enter LaTeX equation (e.g., E = mc^2)"
                className="equation-input"
                rows={3}
              />
              <div className="equation-preview">
                {latexInput && (
                  <div
                    ref={(el) => {
                      if (el && latexInput) {
                        try {
                          katex.render(latexInput, el, {
                            displayMode: true,
                            throwOnError: false,
                          })
                        } catch {
                          el.textContent = 'Preview unavailable'
                        }
                      }
                    }}
                  />
                )}
              </div>
              <div className="equation-actions">
                <button className="btn-secondary" onClick={() => setIsEditing(false)}>
                  Cancel
                </button>
                <button className="btn-primary" onClick={handleSave}>
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
  }
)
