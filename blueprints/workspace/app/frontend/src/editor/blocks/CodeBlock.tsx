import { createReactBlockSpec } from '@blocknote/react'
import { useState, useCallback, useRef, useEffect } from 'react'
import { Copy, Check, ChevronDown, Code, Hash } from 'lucide-react'

// Common programming languages with display names
const LANGUAGES = [
  { id: 'plaintext', name: 'Plain Text', aliases: ['text', 'plain'] },
  { id: 'javascript', name: 'JavaScript', aliases: ['js'] },
  { id: 'typescript', name: 'TypeScript', aliases: ['ts'] },
  { id: 'python', name: 'Python', aliases: ['py'] },
  { id: 'java', name: 'Java', aliases: [] },
  { id: 'c', name: 'C', aliases: [] },
  { id: 'cpp', name: 'C++', aliases: ['c++'] },
  { id: 'csharp', name: 'C#', aliases: ['cs', 'c#'] },
  { id: 'go', name: 'Go', aliases: ['golang'] },
  { id: 'rust', name: 'Rust', aliases: ['rs'] },
  { id: 'swift', name: 'Swift', aliases: [] },
  { id: 'kotlin', name: 'Kotlin', aliases: ['kt'] },
  { id: 'ruby', name: 'Ruby', aliases: ['rb'] },
  { id: 'php', name: 'PHP', aliases: [] },
  { id: 'html', name: 'HTML', aliases: [] },
  { id: 'css', name: 'CSS', aliases: [] },
  { id: 'scss', name: 'SCSS', aliases: ['sass'] },
  { id: 'json', name: 'JSON', aliases: [] },
  { id: 'yaml', name: 'YAML', aliases: ['yml'] },
  { id: 'xml', name: 'XML', aliases: [] },
  { id: 'markdown', name: 'Markdown', aliases: ['md'] },
  { id: 'sql', name: 'SQL', aliases: [] },
  { id: 'graphql', name: 'GraphQL', aliases: ['gql'] },
  { id: 'shell', name: 'Shell', aliases: ['bash', 'sh', 'zsh'] },
  { id: 'powershell', name: 'PowerShell', aliases: ['ps', 'ps1'] },
  { id: 'dockerfile', name: 'Dockerfile', aliases: ['docker'] },
  { id: 'lua', name: 'Lua', aliases: [] },
  { id: 'r', name: 'R', aliases: [] },
  { id: 'scala', name: 'Scala', aliases: [] },
  { id: 'elixir', name: 'Elixir', aliases: ['ex'] },
  { id: 'haskell', name: 'Haskell', aliases: ['hs'] },
  { id: 'clojure', name: 'Clojure', aliases: ['clj'] },
  { id: 'dart', name: 'Dart', aliases: [] },
  { id: 'vue', name: 'Vue', aliases: [] },
  { id: 'svelte', name: 'Svelte', aliases: [] },
  { id: 'jsx', name: 'JSX', aliases: ['react'] },
  { id: 'tsx', name: 'TSX', aliases: ['react-ts'] },
  { id: 'toml', name: 'TOML', aliases: [] },
  { id: 'ini', name: 'INI', aliases: [] },
  { id: 'diff', name: 'Diff', aliases: ['patch'] },
  { id: 'makefile', name: 'Makefile', aliases: ['make'] },
  { id: 'nginx', name: 'Nginx', aliases: [] },
  { id: 'apache', name: 'Apache', aliases: [] },
  { id: 'latex', name: 'LaTeX', aliases: ['tex'] },
  { id: 'matlab', name: 'MATLAB', aliases: [] },
  { id: 'julia', name: 'Julia', aliases: ['jl'] },
  { id: 'perl', name: 'Perl', aliases: ['pl'] },
  { id: 'vim', name: 'Vim Script', aliases: ['vimscript'] },
  { id: 'terraform', name: 'Terraform', aliases: ['tf', 'hcl'] },
  { id: 'protobuf', name: 'Protocol Buffers', aliases: ['proto'] },
  { id: 'solidity', name: 'Solidity', aliases: ['sol'] },
]

export const EnhancedCodeBlock = createReactBlockSpec(
  {
    type: 'enhancedCode',
    propSchema: {
      language: { default: 'plaintext' },
      code: { default: '' },
      showLineNumbers: { default: true },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [language, setLanguage] = useState(block.props.language as string || 'plaintext')
      const [code, setCode] = useState(block.props.code as string || '')
      const [showLineNumbers, setShowLineNumbers] = useState(block.props.showLineNumbers !== false)
      const [showLanguageMenu, setShowLanguageMenu] = useState(false)
      const [searchQuery, setSearchQuery] = useState('')
      const [copied, setCopied] = useState(false)
      const [highlightedIndex, setHighlightedIndex] = useState(0)
      const textareaRef = useRef<HTMLTextAreaElement>(null)
      const menuRef = useRef<HTMLDivElement>(null)
      const searchInputRef = useRef<HTMLInputElement>(null)

      // Filter languages by search
      const filteredLanguages = LANGUAGES.filter((lang) => {
        const query = searchQuery.toLowerCase()
        return (
          lang.name.toLowerCase().includes(query) ||
          lang.id.toLowerCase().includes(query) ||
          lang.aliases.some((a) => a.toLowerCase().includes(query))
        )
      })

      // Get display name for current language
      const currentLanguage = LANGUAGES.find((l) => l.id === language) || LANGUAGES[0]

      // Update block props
      const updateBlock = useCallback((updates: Record<string, unknown>) => {
        editor.updateBlock(block, {
          props: { ...block.props, ...updates },
        })
      }, [block, editor])

      // Handle language change
      const handleLanguageChange = useCallback((langId: string) => {
        setLanguage(langId)
        updateBlock({ language: langId })
        setShowLanguageMenu(false)
        setSearchQuery('')
        textareaRef.current?.focus()
      }, [updateBlock])

      // Handle code change
      const handleCodeChange = useCallback((value: string) => {
        setCode(value)
        updateBlock({ code: value })
      }, [updateBlock])

      // Handle copy
      const handleCopy = useCallback(() => {
        navigator.clipboard.writeText(code)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
      }, [code])

      // Toggle line numbers
      const handleToggleLineNumbers = useCallback(() => {
        setShowLineNumbers(!showLineNumbers)
        updateBlock({ showLineNumbers: !showLineNumbers })
      }, [showLineNumbers, updateBlock])

      // Handle keyboard navigation in menu
      const handleMenuKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'ArrowDown') {
          e.preventDefault()
          setHighlightedIndex((prev) => Math.min(prev + 1, filteredLanguages.length - 1))
        } else if (e.key === 'ArrowUp') {
          e.preventDefault()
          setHighlightedIndex((prev) => Math.max(prev - 1, 0))
        } else if (e.key === 'Enter') {
          e.preventDefault()
          if (filteredLanguages[highlightedIndex]) {
            handleLanguageChange(filteredLanguages[highlightedIndex].id)
          }
        } else if (e.key === 'Escape') {
          setShowLanguageMenu(false)
          setSearchQuery('')
        }
      }, [filteredLanguages, highlightedIndex, handleLanguageChange])

      // Close menu when clicking outside
      useEffect(() => {
        const handleClickOutside = (e: MouseEvent) => {
          if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
            setShowLanguageMenu(false)
            setSearchQuery('')
          }
        }
        if (showLanguageMenu) {
          document.addEventListener('mousedown', handleClickOutside)
          searchInputRef.current?.focus()
        }
        return () => document.removeEventListener('mousedown', handleClickOutside)
      }, [showLanguageMenu])

      // Reset highlighted index when search changes
      useEffect(() => {
        setHighlightedIndex(0)
      }, [searchQuery])

      // Calculate line numbers
      const lines = code.split('\n')
      const lineCount = lines.length

      return (
        <div
          className="enhanced-code-block"
          style={{
            margin: '8px 0',
            borderRadius: '6px',
            overflow: 'hidden',
            border: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
            background: 'var(--code-bg, #f7f6f3)',
            fontFamily: 'var(--font-mono, "SFMono-Regular", Menlo, Monaco, monospace)',
            fontSize: '13px',
          }}
        >
          {/* Header */}
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              padding: '6px 12px',
              borderBottom: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
              background: 'var(--bg-secondary, rgba(55, 53, 47, 0.04))',
            }}
          >
            {/* Language selector */}
            <div ref={menuRef} style={{ position: 'relative' }}>
              <button
                onClick={() => setShowLanguageMenu(!showLanguageMenu)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '4px',
                  padding: '4px 8px',
                  background: 'none',
                  border: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
                  borderRadius: '4px',
                  color: 'var(--text-secondary, #787774)',
                  fontSize: '12px',
                  cursor: 'pointer',
                  transition: 'background 0.15s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-hover, rgba(55, 53, 47, 0.08))'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'none'
                }}
              >
                <Code size={12} />
                <span>{currentLanguage.name}</span>
                <ChevronDown size={12} />
              </button>

              {/* Language menu */}
              {showLanguageMenu && (
                <div
                  style={{
                    position: 'absolute',
                    top: '100%',
                    left: 0,
                    marginTop: '4px',
                    width: '200px',
                    maxHeight: '300px',
                    background: 'var(--bg-primary, white)',
                    border: '1px solid var(--border-color, rgba(55, 53, 47, 0.16))',
                    borderRadius: '6px',
                    boxShadow: '0 4px 12px rgba(0, 0, 0, 0.15)',
                    zIndex: 100,
                    overflow: 'hidden',
                  }}
                >
                  {/* Search input */}
                  <div style={{ padding: '8px', borderBottom: '1px solid var(--border-color)' }}>
                    <input
                      ref={searchInputRef}
                      type="text"
                      placeholder="Search language..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      onKeyDown={handleMenuKeyDown}
                      style={{
                        width: '100%',
                        padding: '6px 8px',
                        border: '1px solid var(--border-color)',
                        borderRadius: '4px',
                        fontSize: '12px',
                        outline: 'none',
                        background: 'var(--bg-primary)',
                        color: 'var(--text-primary)',
                      }}
                    />
                  </div>

                  {/* Language list */}
                  <div
                    style={{
                      maxHeight: '240px',
                      overflowY: 'auto',
                    }}
                  >
                    {filteredLanguages.map((lang, index) => (
                      <button
                        key={lang.id}
                        onClick={() => handleLanguageChange(lang.id)}
                        style={{
                          display: 'block',
                          width: '100%',
                          padding: '8px 12px',
                          background: index === highlightedIndex
                            ? 'var(--bg-hover, rgba(55, 53, 47, 0.08))'
                            : lang.id === language
                              ? 'var(--accent-bg, rgba(35, 131, 226, 0.1))'
                              : 'none',
                          border: 'none',
                          textAlign: 'left',
                          fontSize: '13px',
                          color: lang.id === language
                            ? 'var(--accent-color, #2383e2)'
                            : 'var(--text-primary, #37352f)',
                          cursor: 'pointer',
                        }}
                      >
                        {lang.name}
                      </button>
                    ))}
                    {filteredLanguages.length === 0 && (
                      <div
                        style={{
                          padding: '12px',
                          textAlign: 'center',
                          color: 'var(--text-tertiary)',
                          fontSize: '12px',
                        }}
                      >
                        No languages found
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>

            {/* Actions */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
              {/* Line numbers toggle */}
              <button
                onClick={handleToggleLineNumbers}
                title={showLineNumbers ? 'Hide line numbers' : 'Show line numbers'}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  width: '28px',
                  height: '28px',
                  background: 'none',
                  border: 'none',
                  borderRadius: '4px',
                  color: showLineNumbers
                    ? 'var(--accent-color, #2383e2)'
                    : 'var(--text-tertiary, #9b9a97)',
                  cursor: 'pointer',
                  transition: 'background 0.15s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-hover, rgba(55, 53, 47, 0.08))'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'none'
                }}
              >
                <Hash size={14} />
              </button>

              {/* Copy button */}
              <button
                onClick={handleCopy}
                title="Copy code"
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  width: '28px',
                  height: '28px',
                  background: 'none',
                  border: 'none',
                  borderRadius: '4px',
                  color: copied
                    ? 'var(--success-color, #0f7b6c)'
                    : 'var(--text-tertiary, #9b9a97)',
                  cursor: 'pointer',
                  transition: 'background 0.15s, color 0.15s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-hover, rgba(55, 53, 47, 0.08))'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'none'
                }}
              >
                {copied ? <Check size={14} /> : <Copy size={14} />}
              </button>
            </div>
          </div>

          {/* Code area */}
          <div
            style={{
              display: 'flex',
              padding: '12px 0',
              overflowX: 'auto',
            }}
          >
            {/* Line numbers */}
            {showLineNumbers && (
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  padding: '0 12px',
                  borderRight: '1px solid var(--border-color, rgba(55, 53, 47, 0.09))',
                  color: 'var(--text-tertiary, #9b9a97)',
                  fontSize: '13px',
                  lineHeight: '1.5',
                  textAlign: 'right',
                  userSelect: 'none',
                }}
              >
                {Array.from({ length: lineCount }, (_, i) => (
                  <span key={i + 1}>{i + 1}</span>
                ))}
              </div>
            )}

            {/* Code input */}
            <textarea
              ref={textareaRef}
              value={code}
              onChange={(e) => handleCodeChange(e.target.value)}
              placeholder="// Write your code here..."
              spellCheck={false}
              style={{
                flex: 1,
                minHeight: '100px',
                padding: '0 12px',
                border: 'none',
                outline: 'none',
                resize: 'vertical',
                background: 'transparent',
                color: 'var(--text-primary, #37352f)',
                fontSize: '13px',
                lineHeight: '1.5',
                fontFamily: 'inherit',
                whiteSpace: 'pre',
                overflowWrap: 'normal',
              }}
            />
          </div>
        </div>
      )
    },
  }
)
