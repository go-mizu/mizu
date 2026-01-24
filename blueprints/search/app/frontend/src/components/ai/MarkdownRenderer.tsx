import { useState, useEffect, useRef, useCallback, memo } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import rehypeRaw from 'rehype-raw'
import { Copy, Check, ExternalLink } from 'lucide-react'
import type { Components } from 'react-markdown'
import type { Citation } from '../../types/ai'

interface MarkdownRendererProps {
  content: string
  citations?: Citation[]
  isStreaming?: boolean
  onCitationClick?: (index: number) => void
}

// Mermaid diagram component
function MermaidDiagram({ code }: { code: string }) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [svg, setSvg] = useState<string>('')
  const [error, setError] = useState<string>('')

  useEffect(() => {
    let mounted = true

    async function renderDiagram() {
      try {
        const mermaid = await import('mermaid')
        mermaid.default.initialize({
          startOnLoad: false,
          theme: 'neutral',
          securityLevel: 'strict',
        })

        const id = `mermaid-${Math.random().toString(36).slice(2, 11)}`
        const { svg: renderedSvg } = await mermaid.default.render(id, code)

        if (mounted) {
          setSvg(renderedSvg)
          setError('')
        }
      } catch (err) {
        if (mounted) {
          setError(err instanceof Error ? err.message : 'Failed to render diagram')
        }
      }
    }

    renderDiagram()
    return () => { mounted = false }
  }, [code])

  if (error) {
    return (
      <div className="mermaid-error">
        <pre>{code}</pre>
        <span className="error-message">{error}</span>
      </div>
    )
  }

  if (!svg) {
    return <div className="mermaid-loading">Rendering diagram...</div>
  }

  return (
    <div
      ref={containerRef}
      className="mermaid-diagram"
      dangerouslySetInnerHTML={{ __html: svg }}
    />
  )
}

// Code block component with copy button
function CodeBlock({
  language,
  children,
}: {
  language: string
  children: string
}) {
  const [copied, setCopied] = useState(false)

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(children)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [children])

  // Check if this is a mermaid diagram
  if (language === 'mermaid') {
    return <MermaidDiagram code={children} />
  }

  return (
    <div className="code-block">
      <div className="code-block-header">
        <span className="code-block-language">{language || 'text'}</span>
        <button
          type="button"
          onClick={handleCopy}
          className="code-block-copy"
          title="Copy code"
        >
          {copied ? <Check size={14} /> : <Copy size={14} />}
          {copied ? 'Copied' : 'Copy'}
        </button>
      </div>
      <pre className={language ? `language-${language}` : ''}>
        <code>{children}</code>
      </pre>
    </div>
  )
}

// Citation marker component
function CitationMarker({
  index,
  citation,
  onClick,
}: {
  index: number
  citation?: Citation
  onClick?: () => void
}) {
  return (
    <button
      type="button"
      className="citation-marker"
      onClick={onClick}
      title={citation?.title || `Source ${index}`}
    >
      [{index}]
    </button>
  )
}

export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
  citations = [],
  isStreaming = false,
  onCitationClick,
}: MarkdownRendererProps) {
  // Process content to handle citation markers
  // Convert [1], [2] etc to spans we can style
  const processedContent = content.replace(
    /\[(\d+)\]/g,
    (_, num) => `<cite data-index="${num}">[${num}]</cite>`
  )

  const components: Components = {
    // Custom code block rendering
    pre({ children }) {
      return <>{children}</>
    },
    code({ className, children, ...props }) {
      const match = /language-(\w+)/.exec(className || '')
      const language = match ? match[1] : ''
      const code = String(children).replace(/\n$/, '')

      // Inline code
      if (!match && !code.includes('\n')) {
        return (
          <code className="inline-code" {...props}>
            {children}
          </code>
        )
      }

      return <CodeBlock language={language}>{code}</CodeBlock>
    },

    // Custom link rendering
    a({ href, children }) {
      const isExternal = href?.startsWith('http')
      return (
        <a
          href={href}
          target={isExternal ? '_blank' : undefined}
          rel={isExternal ? 'noopener noreferrer' : undefined}
          className="markdown-link"
        >
          {children}
          {isExternal && <ExternalLink size={12} className="inline ml-1" />}
        </a>
      )
    },

    // Custom table rendering
    table({ children }) {
      return (
        <div className="table-wrapper">
          <table className="markdown-table">{children}</table>
        </div>
      )
    },

    // Citation handling via custom element
    cite({ ...props }) {
      const index = Number(props['data-index'])
      const citation = citations.find((c) => c.index === index)
      return (
        <CitationMarker
          index={index}
          citation={citation}
          onClick={() => onCitationClick?.(index)}
        />
      )
    },
  }

  return (
    <div className={`markdown-content ${isStreaming ? 'streaming' : ''}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight, rehypeRaw]}
        components={components}
      >
        {processedContent}
      </ReactMarkdown>
      {isStreaming && <span className="typing-cursor" />}
    </div>
  )
})
