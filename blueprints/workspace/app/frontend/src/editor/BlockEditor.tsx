import { useCallback, useEffect, useMemo, useState, useRef } from 'react'
import {
  BlockNoteEditor,
  BlockNoteSchema,
  defaultBlockSpecs,
  defaultInlineContentSpecs,
  defaultStyleSpecs,
  filterSuggestionItems,
  PartialBlock,
  Block as BNBlock,
  createCodeBlockSpec,
} from '@blocknote/core'
import { codeBlockOptions } from '@blocknote/code-block'
// Note: @blocknote/code-block styles are bundled in the main mantine package
import {
  SuggestionMenuController,
  getDefaultReactSlashMenuItems,
  useCreateBlockNote,
  FormattingToolbar,
  FormattingToolbarController,
  BlockTypeSelect,
  BasicTextStyleButton,
  TextAlignButton,
  ColorStyleButton,
  NestBlockButton,
  UnnestBlockButton,
  CreateLinkButton,
  DefaultReactSuggestionItem,
  SideMenu,
  SideMenuController,
  DragHandleMenu,
  DragHandleButton,
  AddBlockButton,
  RemoveBlockItem,
  BlockColorsItem,
  useComponentsContext,
  useBlockNoteEditor,
  useSelectedBlocks,
} from '@blocknote/react'
import { combineByGroup } from '@blocknote/core'
import {
  withMultiColumn,
  multiColumnDropCursor,
  getMultiColumnSlashMenuItems,
  locales as multiColumnLocales,
} from '@blocknote/xl-multi-column'
import { BlockNoteView } from '@blocknote/mantine'
import '@blocknote/mantine/style.css'
// xl-multi-column styles are bundled with the main package
import { MantineProvider } from '@mantine/core'
import {
  Type,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  CheckSquare,
  Quote,
  Code,
  Minus,
  Image,
  Video,
  Music,
  FileText,
  Link2,
  Table,
  Calculator,
  ListTree,
  MessageSquare,
  ToggleLeft,
  Bookmark,
  FileUp,
  AtSign,
  ExternalLink,
  ChevronRight,
  Sparkles,
  User,
  Calendar,
  Database,
  Highlighter,
  Palette,
  ChevronDown,
  Check,
} from 'lucide-react'

import { api } from '../api/client'
import { CalloutBlock } from './blocks/CalloutBlock'
import { BookmarkBlock } from './blocks/BookmarkBlock'
import { EquationBlock } from './blocks/EquationBlock'
import { TableOfContentsBlock } from './blocks/TableOfContentsBlock'
import { PDFBlock } from './blocks/PDFBlock'
import { LinkToPageBlock } from './blocks/LinkToPageBlock'
import { SyncedBlock } from './blocks/SyncedBlock'
import { TemplateBlock } from './blocks/TemplateBlock'
import { InlineDatabaseBlock } from './blocks/InlineDatabaseBlock'
import { SimpleTableBlock } from './blocks/SimpleTableBlock'
import { MentionInline } from './InlineMention'

// Type for drag handle menu item props
interface DragHandleItemProps {
  block: any
  children: React.ReactNode
}

// Custom Drag Handle Menu Items
function DuplicateBlockItem({ block, children }: DragHandleItemProps) {
  const editor = useBlockNoteEditor()
  const Components = useComponentsContext()!

  return (
    <Components.Generic.Menu.Item
      onClick={() => {
        const blockCopy = { ...block, id: undefined }
        editor.insertBlocks([blockCopy], block, 'after')
      }}
    >
      {children}
    </Components.Generic.Menu.Item>
  )
}

function CopyLinkItem({ block, children }: DragHandleItemProps) {
  const Components = useComponentsContext()!

  return (
    <Components.Generic.Menu.Item
      onClick={() => {
        const url = `${window.location.origin}${window.location.pathname}#block-${block.id}`
        navigator.clipboard.writeText(url)
      }}
    >
      {children}
    </Components.Generic.Menu.Item>
  )
}

function TurnIntoItem({ block, children }: DragHandleItemProps) {
  const editor = useBlockNoteEditor()
  const Components = useComponentsContext()!

  const turnIntoOptions = [
    { type: 'paragraph', label: 'Text' },
    { type: 'heading', label: 'Heading 1', props: { level: 1 } },
    { type: 'heading', label: 'Heading 2', props: { level: 2 } },
    { type: 'heading', label: 'Heading 3', props: { level: 3 } },
    { type: 'bulletListItem', label: 'Bulleted list' },
    { type: 'numberedListItem', label: 'Numbered list' },
    { type: 'checkListItem', label: 'To-do list' },
    { type: 'quote', label: 'Quote' },
    { type: 'toggleListItem', label: 'Toggle list' },
    { type: 'callout', label: 'Callout' },
  ]

  return (
    <Components.Generic.Menu.Item
      subTrigger={true}
    >
      {children}
      <Components.Generic.Menu.Dropdown>
        {turnIntoOptions.map((option) => (
          <Components.Generic.Menu.Item
            key={`${option.type}-${option.label}`}
            onClick={() => {
              editor.updateBlock(block, {
                type: option.type as any,
                props: option.props as any,
              })
            }}
          >
            {option.label}
          </Components.Generic.Menu.Item>
        ))}
      </Components.Generic.Menu.Dropdown>
    </Components.Generic.Menu.Item>
  )
}

// =============================================================================
// Enhanced Color Picker Component (Notion-style)
// =============================================================================

// Notion-like color palette
const TEXT_COLORS = [
  { name: 'Default', value: 'default', color: '#37352f', darkColor: '#e6e6e6' },
  { name: 'Gray', value: 'gray', color: '#787774', darkColor: '#9b9a97' },
  { name: 'Brown', value: 'brown', color: '#9f6b53', darkColor: '#ba856f' },
  { name: 'Orange', value: 'orange', color: '#d9730d', darkColor: '#ffa344' },
  { name: 'Yellow', value: 'yellow', color: '#cb912f', darkColor: '#ffdc49' },
  { name: 'Green', value: 'green', color: '#448361', darkColor: '#4dab9a' },
  { name: 'Blue', value: 'blue', color: '#2383e2', darkColor: '#529cca' },
  { name: 'Purple', value: 'purple', color: '#9065b0', darkColor: '#9a6dd7' },
  { name: 'Pink', value: 'pink', color: '#c14c8a', darkColor: '#e255a1' },
  { name: 'Red', value: 'red', color: '#d44c47', darkColor: '#ff7369' },
]

const BACKGROUND_COLORS = [
  { name: 'Default', value: 'default', color: 'transparent', darkColor: 'transparent' },
  { name: 'Gray', value: 'gray', color: '#f1f1ef', darkColor: '#454b4e' },
  { name: 'Brown', value: 'brown', color: '#f4eeee', darkColor: '#594a3a' },
  { name: 'Orange', value: 'orange', color: '#fbecdd', darkColor: '#59413a' },
  { name: 'Yellow', value: 'yellow', color: '#fbf3db', darkColor: '#59563b' },
  { name: 'Green', value: 'green', color: '#edf3ec', darkColor: '#354c4b' },
  { name: 'Blue', value: 'blue', color: '#e7f3f8', darkColor: '#364954' },
  { name: 'Purple', value: 'purple', color: '#f6f3f9', darkColor: '#443f57' },
  { name: 'Pink', value: 'pink', color: '#faf1f5', darkColor: '#533b4c' },
  { name: 'Red', value: 'red', color: '#fdebec', darkColor: '#594141' },
]

// Enhanced Text Color Button
function TextColorButton() {
  const editor = useBlockNoteEditor()
  const Components = useComponentsContext()!
  const [isOpen, setIsOpen] = useState(false)
  const [activeTextColor, setActiveTextColor] = useState('default')
  const menuRef = useRef<HTMLDivElement>(null)

  const selectedBlocks = useSelectedBlocks(editor)
  const hasInlineContent = selectedBlocks.some(
    (block) => block.content !== undefined && Array.isArray(block.content)
  )

  // Update active color when selection changes
  useEffect(() => {
    const updateColor = () => {
      try {
        const styles = editor.getActiveStyles()
        const textColor = String(styles.textColor || 'default')
        setActiveTextColor(textColor)
      } catch {
        setActiveTextColor('default')
      }
    }
    updateColor()
    // Listen to selection changes
    const unsubscribe = editor.onSelectionChange(updateColor)
    return () => unsubscribe()
  }, [editor])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  if (!hasInlineContent) return null

  const currentColor = TEXT_COLORS.find((c) => c.value === activeTextColor) || TEXT_COLORS[0]

  return (
    <div className="color-picker-wrapper" ref={menuRef}>
      <Components.FormattingToolbar.Button
        mainTooltip="Text color"
        onClick={() => setIsOpen(!isOpen)}
        isSelected={isOpen}
      >
        <div className="color-button-content">
          <span style={{ color: currentColor.color, fontWeight: 600 }}>A</span>
          <div
            className="color-indicator"
            style={{ backgroundColor: currentColor.value === 'default' ? '#37352f' : currentColor.color }}
          />
        </div>
      </Components.FormattingToolbar.Button>

      {isOpen && (
        <div className="color-picker-menu text-color-menu">
          <div className="color-picker-label">Text color</div>
          <div className="color-picker-grid">
            {TEXT_COLORS.map((color) => (
              <button
                key={color.value}
                className={`color-option ${activeTextColor === color.value ? 'selected' : ''}`}
                onClick={() => {
                  editor.toggleStyles({ textColor: color.value })
                  setActiveTextColor(color.value)
                  setIsOpen(false)
                }}
                title={color.name}
              >
                <span style={{ color: color.color, fontWeight: 600 }}>A</span>
                {activeTextColor === color.value && <Check size={12} className="check-icon" />}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// Enhanced Background/Highlight Color Button
function HighlightColorButton() {
  const editor = useBlockNoteEditor()
  const Components = useComponentsContext()!
  const [isOpen, setIsOpen] = useState(false)
  const [activeBackgroundColor, setActiveBackgroundColor] = useState('default')
  const menuRef = useRef<HTMLDivElement>(null)

  const selectedBlocks = useSelectedBlocks(editor)
  const hasInlineContent = selectedBlocks.some(
    (block) => block.content !== undefined && Array.isArray(block.content)
  )

  // Update active color when selection changes
  useEffect(() => {
    const updateColor = () => {
      try {
        const styles = editor.getActiveStyles()
        const bgColor = String(styles.backgroundColor || 'default')
        setActiveBackgroundColor(bgColor)
      } catch {
        setActiveBackgroundColor('default')
      }
    }
    updateColor()
    // Listen to selection changes
    const unsubscribe = editor.onSelectionChange(updateColor)
    return () => unsubscribe()
  }, [editor])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  if (!hasInlineContent) return null

  const currentColor = BACKGROUND_COLORS.find((c) => c.value === activeBackgroundColor) || BACKGROUND_COLORS[0]

  return (
    <div className="color-picker-wrapper" ref={menuRef}>
      <Components.FormattingToolbar.Button
        mainTooltip="Highlight color"
        onClick={() => setIsOpen(!isOpen)}
        isSelected={isOpen}
      >
        <div className="color-button-content">
          <Highlighter size={16} />
          <div
            className="color-indicator"
            style={{
              backgroundColor: currentColor.value === 'default' ? '#fbf3db' : currentColor.color,
              border: currentColor.value === 'default' ? '1px solid #e3e2de' : 'none',
            }}
          />
        </div>
      </Components.FormattingToolbar.Button>

      {isOpen && (
        <div className="color-picker-menu highlight-color-menu">
          <div className="color-picker-label">Background</div>
          <div className="color-picker-grid">
            {BACKGROUND_COLORS.map((color) => (
              <button
                key={color.value}
                className={`color-option bg-option ${activeBackgroundColor === color.value ? 'selected' : ''}`}
                onClick={() => {
                  editor.toggleStyles({ backgroundColor: color.value })
                  setActiveBackgroundColor(color.value)
                  setIsOpen(false)
                }}
                title={color.name}
                style={{
                  backgroundColor: color.value === 'default' ? 'transparent' : color.color,
                  border: color.value === 'default' ? '1px dashed #ccc' : '1px solid transparent',
                }}
              >
                {color.value === 'default' && <span className="no-color-icon">∅</span>}
                {activeBackgroundColor === color.value && <Check size={12} className="check-icon" />}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// Toolbar Separator
function ToolbarSeparator() {
  return <div className="toolbar-separator" />
}

function MoveUpItem({ block, children }: DragHandleItemProps) {
  const editor = useBlockNoteEditor()
  const Components = useComponentsContext()!

  return (
    <Components.Generic.Menu.Item
      onClick={() => {
        // Set cursor to this block first, then move
        editor.setTextCursorPosition(block, 'start')
        editor.moveBlocksUp()
      }}
    >
      {children}
    </Components.Generic.Menu.Item>
  )
}

function MoveDownItem({ block, children }: DragHandleItemProps) {
  const editor = useBlockNoteEditor()
  const Components = useComponentsContext()!

  return (
    <Components.Generic.Menu.Item
      onClick={() => {
        // Set cursor to this block first, then move
        editor.setTextCursorPosition(block, 'start')
        editor.moveBlocksDown()
      }}
    >
      {children}
    </Components.Generic.Menu.Item>
  )
}

// Custom Drag Handle Menu component - receives block prop and passes to children
function CustomDragHandleMenu({ block }: { block: any }) {
  return (
    <DragHandleMenu block={block}>
      <RemoveBlockItem block={block}>Delete</RemoveBlockItem>
      <DuplicateBlockItem block={block}>Duplicate</DuplicateBlockItem>
      <TurnIntoItem block={block}>Turn into</TurnIntoItem>
      <BlockColorsItem block={block}>Colors</BlockColorsItem>
      <CopyLinkItem block={block}>Copy link to block</CopyLinkItem>
      <MoveUpItem block={block}>Move up</MoveUpItem>
      <MoveDownItem block={block}>Move down</MoveDownItem>
    </DragHandleMenu>
  )
}

// API Block types from backend
interface RichText {
  type: string
  text: string
  annotations?: {
    bold?: boolean
    italic?: boolean
    strikethrough?: boolean
    underline?: boolean
    code?: boolean
    color?: string
  }
  link?: string
  mention?: {
    type: string
    user_id?: string
    page_id?: string
    date?: string
  }
}

interface BlockContent {
  rich_text?: RichText[]
  text?: string
  checked?: boolean
  icon?: string
  color?: string
  language?: string
  url?: string
  caption?: RichText[]
  title?: string
  description?: string
  table_width?: number
  has_header?: boolean
  database_id?: string
  synced_from?: string
  page_id?: string
}

interface Block {
  id: string
  type: string
  content: BlockContent
  children?: Block[]
}

interface BlockEditorProps {
  pageId: string
  initialBlocks: Block[]
  theme?: 'light' | 'dark'
  onSave?: () => void
  workspaceId?: string
}

// Create code block with syntax highlighting support
// Type assertion needed due to shiki version differences in dependencies
const codeBlockWithHighlighting = createCodeBlockSpec(codeBlockOptions as Parameters<typeof createCodeBlockSpec>[0])

// Custom schema with our custom blocks - wrapped with withMultiColumn for column support
// BlockNote 0.42+ has built-in blocks for toggle (toggleListItem), divider, audio, codeBlock
const baseSchema = BlockNoteSchema.create({
  blockSpecs: {
    ...defaultBlockSpecs,
    // Override codeBlock with syntax highlighting support
    codeBlock: codeBlockWithHighlighting,
    // Custom blocks that extend default functionality
    callout: CalloutBlock(),
    bookmark: BookmarkBlock(),
    equation: EquationBlock(),
    tableOfContents: TableOfContentsBlock(),
    pdf: PDFBlock(),
    linkToPage: LinkToPageBlock(),
    syncedBlock: SyncedBlock(),
    templateButton: TemplateBlock(),
    inlineDatabase: InlineDatabaseBlock(),
    simpleTable: SimpleTableBlock(),
  },
  inlineContentSpecs: {
    ...defaultInlineContentSpecs,
    mention: MentionInline,
  },
  styleSpecs: defaultStyleSpecs,
})

// Wrap schema with multi-column support
const schema = withMultiColumn(baseSchema)

// Mention suggestion item interface
interface MentionSuggestionItem {
  id: string
  label: string
  type: 'user' | 'page' | 'date'
  icon?: React.ReactNode
  email?: string
  date?: Date
}

// Get mention suggestion items
async function getMentionItems(
  query: string,
  workspaceId?: string
): Promise<MentionSuggestionItem[]> {
  const items: MentionSuggestionItem[] = []

  // Add date options first (always available)
  const dateItems: MentionSuggestionItem[] = [
    { id: 'today', label: 'Today', type: 'date', date: new Date() },
    { id: 'tomorrow', label: 'Tomorrow', type: 'date', date: new Date(Date.now() + 86400000) },
    { id: 'yesterday', label: 'Yesterday', type: 'date', date: new Date(Date.now() - 86400000) },
  ]

  // Filter dates by query
  const filteredDates = dateItems.filter(d =>
    d.label.toLowerCase().includes(query.toLowerCase())
  )
  items.push(...filteredDates)

  // Fetch users if workspaceId is available
  if (workspaceId) {
    try {
      const response = await api.get<{ users: Array<{ id: string; name: string; email: string }> }>(
        `/workspaces/${workspaceId}/members?q=${encodeURIComponent(query)}&limit=5`
      )
      const users = (response.users || []).map(u => ({
        id: u.id,
        label: u.name,
        type: 'user' as const,
        email: u.email,
      }))
      items.push(...users)
    } catch (err) {
      console.error('Failed to fetch users:', err)
    }

    // Fetch pages
    try {
      const response = await api.get<{ results: Array<{ id: string; title: string }> }>(
        `/search?q=${encodeURIComponent(query)}&type=page&limit=5`
      )
      const pages = (response.results || []).map(p => ({
        id: p.id,
        label: p.title,
        type: 'page' as const,
      }))
      items.push(...pages)
    } catch (err) {
      console.error('Failed to fetch pages:', err)
    }
  }

  return items
}

type EditorType = typeof schema.BlockNoteEditor

// API block type to BlockNote type mapping
// BlockNote 0.42+ uses toggleListItem instead of toggle, codeBlock instead of code
const API_TO_BLOCKNOTE: Record<string, string | { type: string; props?: Record<string, unknown> }> = {
  paragraph: 'paragraph',
  heading_1: { type: 'heading', props: { level: 1 } },
  heading_2: { type: 'heading', props: { level: 2 } },
  heading_3: { type: 'heading', props: { level: 3 } },
  bulleted_list: 'bulletListItem',
  numbered_list: 'numberedListItem',
  to_do: 'checkListItem',
  todo: 'checkListItem',
  code: 'codeBlock',
  quote: 'quote',
  callout: 'callout',
  toggle: 'toggleListItem',
  divider: 'divider',
  horizontalRule: 'divider',
  image: 'image',
  video: 'video',
  file: 'file',
  bookmark: 'bookmark',
  table: 'table',
  simple_table: 'simpleTable',
  equation: 'equation',
  table_of_contents: 'tableOfContents',
  audio: 'audio',
  pdf: 'pdf',
  column_list: 'columnList',
  link_to_page: 'linkToPage',
}

// BlockNote type to API type mapping
// BlockNote 0.42+ uses toggleListItem and codeBlock as built-in types
const BLOCKNOTE_TO_API: Record<string, string | ((props: Record<string, unknown>) => string)> = {
  paragraph: 'paragraph',
  heading: (props) => `heading_${props?.level || 1}`,
  bulletListItem: 'bulleted_list',
  numberedListItem: 'numbered_list',
  checkListItem: 'to_do',
  codeBlock: 'code',
  quote: 'quote',
  callout: 'callout',
  toggleListItem: 'toggle',
  divider: 'divider',
  image: 'image',
  video: 'video',
  file: 'file',
  bookmark: 'bookmark',
  table: 'table',
  simpleTable: 'simple_table',
  equation: 'equation',
  tableOfContents: 'table_of_contents',
  audio: 'audio',
  pdf: 'pdf',
  columnList: 'column_list',
  linkToPage: 'link_to_page',
}

// Convert rich text array to BlockNote inline content
function richTextToInlineContent(richText: RichText[] | undefined): Array<{
  type: 'text'
  text: string
  styles: Record<string, boolean | string>
}> | string {
  if (!richText || richText.length === 0) return ''

  return richText.map((rt) => ({
    type: 'text' as const,
    text: rt.text || '',
    styles: {
      ...(rt.annotations?.bold && { bold: true }),
      ...(rt.annotations?.italic && { italic: true }),
      ...(rt.annotations?.strikethrough && { strike: true }),
      ...(rt.annotations?.underline && { underline: true }),
      ...(rt.annotations?.code && { code: true }),
      ...(rt.annotations?.color && { textColor: rt.annotations.color }),
    },
  }))
}

// Convert API blocks to BlockNote format
function apiBlocksToBlockNote(blocks: Block[]): PartialBlock[] {
  return blocks.map((block) => {
    const mapping = API_TO_BLOCKNOTE[block.type]
    let type: string
    let props: Record<string, unknown> = {}

    if (typeof mapping === 'object') {
      type = mapping.type
      props = { ...mapping.props }
    } else {
      type = mapping || 'paragraph'
    }

    // Map content based on block type
    const content = block.content || {}

    // Handle specific props
    if (block.type === 'to_do' || block.type === 'todo') {
      props.checked = content.checked || false
    }
    if (block.type === 'code') {
      props.language = content.language || 'plaintext'
    }
    if (block.type === 'callout') {
      props.icon = content.icon || '💡'
      props.backgroundColor = content.color || 'default'
    }
    if (block.type === 'image') {
      props.url = content.url || ''
      props.caption = extractPlainText(content.caption)
    }
    if (block.type === 'video') {
      props.url = content.url || ''
    }
    if (block.type === 'file') {
      props.url = content.url || ''
      props.name = content.title || 'File'
    }
    if (block.type === 'bookmark') {
      props.url = content.url || ''
      props.title = content.title || ''
      props.description = content.description || ''
    }
    if (block.type === 'link_to_page') {
      props.pageId = content.page_id || ''
      props.title = content.title || ''
      props.icon = content.icon || ''
    }

    // Get inline content
    let inlineContent = richTextToInlineContent(content.rich_text)
    if (inlineContent === '' && content.text) {
      inlineContent = content.text
    }

    return {
      id: block.id,
      type: type as any,
      props,
      content: type === 'divider' || type === 'image' || type === 'video' || type === 'file'
        ? undefined
        : inlineContent || undefined,
      children: block.children ? apiBlocksToBlockNote(block.children) : [],
    }
  })
}

// Extract plain text from rich text array
function extractPlainText(richText: RichText[] | undefined): string {
  if (!richText) return ''
  return richText.map(rt => rt.text || '').join('')
}

// Convert inline content to rich text
function inlineContentToRichText(content: unknown): RichText[] {
  if (!content) return []
  if (typeof content === 'string') {
    return [{ type: 'text', text: content }]
  }
  if (!Array.isArray(content)) return []

  return content.map((item: any) => {
    if (typeof item === 'string') {
      return { type: 'text', text: item }
    }
    if (item.type === 'text') {
      const rt: RichText = {
        type: 'text',
        text: item.text || '',
      }
      if (item.styles && Object.keys(item.styles).length > 0) {
        rt.annotations = {
          bold: item.styles.bold || false,
          italic: item.styles.italic || false,
          strikethrough: item.styles.strike || false,
          underline: item.styles.underline || false,
          code: item.styles.code || false,
          color: item.styles.textColor || item.styles.backgroundColor,
        }
      }
      if (item.href) {
        rt.link = item.href
      }
      return rt
    }
    if (item.type === 'link') {
      return {
        type: 'text',
        text: item.content?.map((c: any) => c.text).join('') || '',
        link: item.href,
      }
    }
    return { type: 'text', text: '' }
  })
}

// Convert BlockNote blocks to API format (flattened - no children nesting)
function blockNoteToApiBlocks(blocks: PartialBlock[], parentId?: string): Block[] {
  const result: Block[] = []

  for (const block of blocks) {
    const type = block.type as string
    const props = (block.props || {}) as Record<string, unknown>
    const mapping = BLOCKNOTE_TO_API[type]

    let apiType: string
    if (typeof mapping === 'function') {
      apiType = mapping(props)
    } else {
      apiType = mapping || 'paragraph'
    }

    const content: BlockContent = {}

    // Convert inline content to rich_text
    if (block.content) {
      content.rich_text = inlineContentToRichText(block.content)
    }

    // Handle specific block props
    if (type === 'checkListItem') {
      content.checked = props.checked as boolean || false
    }
    if (type === 'codeBlock') {
      content.language = props.language as string || 'plaintext'
    }
    if (type === 'callout') {
      content.icon = props.icon as string || '💡'
      content.color = props.backgroundColor as string || 'default'
    }
    if (type === 'image') {
      content.url = props.url as string || ''
      if (props.caption) {
        content.caption = [{ type: 'text', text: props.caption as string }]
      }
    }
    if (type === 'video') {
      content.url = props.url as string || ''
    }
    if (type === 'file') {
      content.url = props.url as string || ''
      content.title = props.name as string || ''
    }
    if (type === 'bookmark') {
      content.url = props.url as string || ''
      content.title = props.title as string || ''
      content.description = props.description as string || ''
    }
    if (type === 'linkToPage') {
      content.page_id = props.pageId as string || ''
      content.title = props.title as string || ''
      content.icon = props.icon as string || ''
    }

    const apiBlock: Block = {
      id: block.id || crypto.randomUUID(),
      type: apiType,
      content,
    }

    // Add parent_id if this is a nested block
    if (parentId) {
      (apiBlock as any).parent_id = parentId
    }

    result.push(apiBlock)

    // Recursively add children as separate blocks with parent reference
    if (block.children && block.children.length > 0) {
      const childBlocks = blockNoteToApiBlocks(block.children as PartialBlock[], apiBlock.id)
      result.push(...childBlocks)
    }
  }

  return result
}

// Custom slash menu items - Comprehensive Notion-style menu
function getCustomSlashMenuItems(editor: EditorType): DefaultReactSuggestionItem[] {
  const defaultItems = getDefaultReactSlashMenuItems(editor)

  // Add comprehensive custom items matching Notion
  const customItems: DefaultReactSuggestionItem[] = [
    // Basic blocks
    {
      title: 'Text',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'paragraph' }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['text', 'p', 'paragraph'],
      group: 'Basic blocks',
      icon: <Type size={18} />,
      subtext: 'Just start writing with plain text',
    },
    {
      title: 'Heading 1',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'heading', props: { level: 1 } }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['h1', 'heading1', 'title'],
      group: 'Basic blocks',
      icon: <Heading1 size={18} />,
      subtext: 'Big section heading',
    },
    {
      title: 'Heading 2',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'heading', props: { level: 2 } }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['h2', 'heading2', 'subtitle'],
      group: 'Basic blocks',
      icon: <Heading2 size={18} />,
      subtext: 'Medium section heading',
    },
    {
      title: 'Heading 3',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'heading', props: { level: 3 } }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['h3', 'heading3'],
      group: 'Basic blocks',
      icon: <Heading3 size={18} />,
      subtext: 'Small section heading',
    },
    {
      title: 'Bulleted list',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'bulletListItem' }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['bullet', 'ul', 'list'],
      group: 'Basic blocks',
      icon: <List size={18} />,
      subtext: 'Create a simple bulleted list',
    },
    {
      title: 'Numbered list',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'numberedListItem' }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['numbered', 'ol', 'number'],
      group: 'Basic blocks',
      icon: <ListOrdered size={18} />,
      subtext: 'Create a list with numbering',
    },
    {
      title: 'To-do list',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'checkListItem' }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['todo', 'checkbox', 'task', 'check'],
      group: 'Basic blocks',
      icon: <CheckSquare size={18} />,
      subtext: 'Track tasks with a to-do list',
    },
    {
      title: 'Toggle list',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'toggleListItem' }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['toggle', 'collapsible', 'dropdown', 'expand', 'accordion'],
      group: 'Basic blocks',
      icon: <ToggleLeft size={18} />,
      subtext: 'Toggles can hide and show content inside',
    },
        {
      title: 'Callout',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'callout', props: { icon: '💡', backgroundColor: 'default' } }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['callout', 'info', 'note', 'warning', 'alert', 'tip'],
      group: 'Basic blocks',
      icon: <MessageSquare size={18} />,
      subtext: 'Make writing stand out',
    },
    {
      title: 'Divider',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'divider' } as any],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['divider', 'hr', 'line', 'separator', 'horizontal', '---'],
      group: 'Basic blocks',
      icon: <Minus size={18} />,
      subtext: 'Visually divide blocks',
    },
    {
      title: 'Code',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'codeBlock' }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['code', 'codeblock', 'pre', 'programming', 'syntax'],
      group: 'Basic blocks',
      icon: <Code size={18} />,
      subtext: 'Capture a code snippet with syntax highlighting',
    },

    // Media blocks
    {
      title: 'Image',
      onItemClick: () => {
        const input = document.createElement('input')
        input.type = 'file'
        input.accept = 'image/*'
        input.onchange = async (e) => {
          const file = (e.target as HTMLInputElement).files?.[0]
          if (file) {
            try {
              const result = await api.upload(file)
              editor.insertBlocks(
                [{ type: 'image', props: { url: result.url } }],
                editor.getTextCursorPosition().block,
                'after'
              )
            } catch (err) {
              console.error('Image upload failed:', err)
            }
          }
        }
        input.click()
      },
      aliases: ['image', 'picture', 'photo', 'img'],
      group: 'Media',
      icon: <Image size={18} />,
      subtext: 'Upload or embed with a link',
    },
    {
      title: 'Video',
      onItemClick: () => {
        const url = prompt('Enter video URL (YouTube, Vimeo, or direct link):')
        if (url) {
          editor.insertBlocks(
            [{ type: 'video', props: { url } }],
            editor.getTextCursorPosition().block,
            'after'
          )
        }
      },
      aliases: ['video', 'youtube', 'vimeo', 'movie'],
      group: 'Media',
      icon: <Video size={18} />,
      subtext: 'Embed a video',
    },
    {
      title: 'Audio',
      onItemClick: () => {
        const input = document.createElement('input')
        input.type = 'file'
        input.accept = 'audio/*'
        input.onchange = async (e) => {
          const file = (e.target as HTMLInputElement).files?.[0]
          if (file) {
            try {
              const result = await api.upload(file)
              editor.insertBlocks(
                [{ type: 'audio', props: { url: result.url, name: file.name } }],
                editor.getTextCursorPosition().block,
                'after'
              )
            } catch (err) {
              console.error('Audio upload failed:', err)
            }
          }
        }
        input.click()
      },
      aliases: ['audio', 'music', 'sound', 'mp3', 'podcast'],
      group: 'Media',
      icon: <Music size={18} />,
      subtext: 'Embed audio files',
    },
    {
      title: 'File',
      onItemClick: () => {
        const input = document.createElement('input')
        input.type = 'file'
        input.onchange = async (e) => {
          const file = (e.target as HTMLInputElement).files?.[0]
          if (file) {
            try {
              const result = await api.upload(file)
              editor.insertBlocks(
                [{ type: 'file', props: { url: result.url, name: file.name } }],
                editor.getTextCursorPosition().block,
                'after'
              )
            } catch (err) {
              console.error('File upload failed:', err)
            }
          }
        }
        input.click()
      },
      aliases: ['file', 'attachment', 'upload', 'document'],
      group: 'Media',
      icon: <FileUp size={18} />,
      subtext: 'Upload or embed a file',
    },
    {
      title: 'Bookmark',
      onItemClick: () => {
        const url = prompt('Enter URL to bookmark:')
        if (url) {
          editor.insertBlocks(
            [{ type: 'bookmark', props: { url, title: '', description: '' } }],
            editor.getTextCursorPosition().block,
            'after'
          )
        }
      },
      aliases: ['bookmark', 'link', 'web', 'url', 'weblink'],
      group: 'Media',
      icon: <Bookmark size={18} />,
      subtext: 'Save a link as a visual bookmark',
    },
    {
      title: 'PDF',
      onItemClick: () => {
        const input = document.createElement('input')
        input.type = 'file'
        input.accept = '.pdf'
        input.onchange = async (e) => {
          const file = (e.target as HTMLInputElement).files?.[0]
          if (file) {
            try {
              const result = await api.upload(file)
              editor.insertBlocks(
                [{ type: 'pdf', props: { url: result.url, name: file.name } }],
                editor.getTextCursorPosition().block,
                'after'
              )
            } catch (err) {
              console.error('PDF upload failed:', err)
            }
          }
        }
        input.click()
      },
      aliases: ['pdf', 'document'],
      group: 'Media',
      icon: <FileText size={18} />,
      subtext: 'Embed a PDF document',
    },

    // Advanced blocks
    {
      title: 'Table',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'table', content: { type: 'tableContent', rows: [
            { cells: [[], []] },
            { cells: [[], []] },
          ] } } as any],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['table', 'grid', 'spreadsheet'],
      group: 'Advanced blocks',
      icon: <Table size={18} />,
      subtext: 'Add a simple table',
    },
    {
      title: 'Simple Table',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'simpleTable' as const }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['simple table', 'basic table', 'data table'],
      group: 'Advanced blocks',
      icon: <Table size={18} />,
      subtext: 'Add a basic editable table',
    },
    // Column layouts are now provided by @blocknote/xl-multi-column via getMultiColumnSlashMenuItems
    {
      title: 'Equation',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'equation', props: { latex: '' } }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['equation', 'math', 'latex', 'formula', 'katex'],
      group: 'Advanced blocks',
      icon: <Calculator size={18} />,
      subtext: 'Write a mathematical equation',
    },
    {
      title: 'Table of contents',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'tableOfContents' }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['toc', 'table of contents', 'contents', 'outline', 'index'],
      group: 'Advanced blocks',
      icon: <ListTree size={18} />,
      subtext: 'Show an outline of the page',
    },
    {
      title: 'Link to page',
      onItemClick: () => {
        // This will open a page picker
        const pageId = prompt('Enter page ID to link:')
        if (pageId) {
          editor.insertBlocks(
            [{ type: 'linkToPage', props: { pageId, title: '', icon: '' } }],
            editor.getTextCursorPosition().block,
            'after'
          )
        }
      },
      aliases: ['link to page', 'page link', 'internal link'],
      group: 'Advanced blocks',
      icon: <ExternalLink size={18} />,
      subtext: 'Link to an existing page',
    },

    // Database blocks
    {
      title: 'Linked database',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'inlineDatabase' as const, props: { databaseId: '', viewId: '' } }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['linked database', 'embed database', 'inline database', 'database view'],
      group: 'Database',
      icon: <Database size={18} />,
      subtext: 'Embed an existing database',
    },
    {
      title: 'Synced block',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'syncedBlock' as const }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['synced', 'sync', 'synced block', 'shared block'],
      group: 'Advanced blocks',
      icon: <Link2 size={18} />,
      subtext: 'Sync content across pages',
    },
    {
      title: 'Template button',
      onItemClick: () => {
        editor.insertBlocks(
          [{ type: 'templateButton' as const }],
          editor.getTextCursorPosition().block,
          'after'
        )
      },
      aliases: ['template', 'template button', 'duplicate', 'repeat'],
      group: 'Advanced blocks',
      icon: <Sparkles size={18} />,
      subtext: 'Duplicate template content',
    },
  ]

  // Filter out default items that we're replacing with custom versions
  const customTitles = new Set(customItems.map(item => item.title.toLowerCase()))
  const filteredDefaultItems = defaultItems.filter(item =>
    !customTitles.has(item.title.toLowerCase())
  )

  return [...customItems, ...filteredDefaultItems]
}

export function BlockEditor({ pageId, initialBlocks, theme = 'light', onSave, workspaceId }: BlockEditorProps) {
  const [isSaving, setIsSaving] = useState(false)
  const [lastSaved, setLastSaved] = useState<Date | null>(null)
  const [saveError, setSaveError] = useState<string | null>(null)
  const editorRef = useRef<EditorType | null>(null)

  // Convert initial blocks to BlockNote format
  const initialContent = useMemo(() => {
    if (initialBlocks.length === 0) {
      return undefined
    }
    return apiBlocksToBlockNote(initialBlocks)
  }, [initialBlocks])

  // Create editor instance with custom schema and multi-column drop cursor
  const editor = useCreateBlockNote({
    schema,
    initialContent,
    dropCursor: multiColumnDropCursor,
    uploadFile: async (file: File) => {
      const result = await api.upload(file)
      return result.url
    },
  })

  // Store editor ref
  useEffect(() => {
    editorRef.current = editor
  }, [editor])

  // Save handler
  const saveBlocks = useCallback(async () => {
    if (!pageId) {
      console.warn('Cannot save: pageId is missing')
      return
    }

    setIsSaving(true)
    setSaveError(null)
    try {
      const document = editor.document
      if (!document || !Array.isArray(document)) {
        console.warn('Cannot save: document is invalid')
        return
      }
      const blocks = blockNoteToApiBlocks(document as PartialBlock[])
      await api.put(`/pages/${pageId}/blocks`, { blocks })
      setLastSaved(new Date())
      onSave?.()
    } catch (err) {
      console.error('Failed to save blocks:', err)
      const errorMessage = err instanceof Error ? err.message : 'Failed to save'
      setSaveError(errorMessage)
    } finally {
      setIsSaving(false)
    }
  }, [editor, pageId, onSave])

  // Debounced save on change - this is handled via onChange prop on BlockNoteView
  const saveTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const handleEditorChange = useCallback(() => {
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current)
    }
    saveTimeoutRef.current = setTimeout(saveBlocks, 1000)
  }, [saveBlocks])

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current)
      }
    }
  }, [])

  // Keyboard shortcuts for block manipulation and saving
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const isMod = e.metaKey || e.ctrlKey

      // Cmd/Ctrl + S to save immediately
      if (isMod && e.key === 's') {
        e.preventDefault()
        saveBlocks()
        return
      }

      // Get the current block for block-level operations
      const currentBlock = editor.getTextCursorPosition()?.block
      if (!currentBlock) return

      // Cmd/Ctrl + Shift + Up - Move block up
      if (isMod && e.shiftKey && e.key === 'ArrowUp') {
        e.preventDefault()
        editor.moveBlocksUp()
        return
      }

      // Cmd/Ctrl + Shift + Down - Move block down
      if (isMod && e.shiftKey && e.key === 'ArrowDown') {
        e.preventDefault()
        editor.moveBlocksDown()
        return
      }

      // Cmd/Ctrl + D - Duplicate block
      if (isMod && e.key === 'd') {
        e.preventDefault()
        const blockCopy = { ...currentBlock, id: undefined }
        editor.insertBlocks([blockCopy], currentBlock, 'after')
        return
      }

      // Cmd/Ctrl + Shift + Backspace - Delete block
      if (isMod && e.shiftKey && e.key === 'Backspace') {
        e.preventDefault()
        editor.removeBlocks([currentBlock])
        return
      }

      // Cmd/Ctrl + E - Toggle inline code
      if (isMod && e.key === 'e') {
        e.preventDefault()
        editor.toggleStyles({ code: true })
        return
      }

      // Cmd/Ctrl + Shift + S - Toggle strikethrough
      if (isMod && e.shiftKey && e.key === 's') {
        e.preventDefault()
        editor.toggleStyles({ strike: true })
        return
      }

      // Cmd/Ctrl + Shift + H - Toggle highlight (yellow background)
      if (isMod && e.shiftKey && e.key === 'h') {
        e.preventDefault()
        editor.toggleStyles({ backgroundColor: 'yellow' })
        return
      }

      // Cmd/Ctrl + Alt + 1/2/3 - Convert to Heading 1/2/3
      if (isMod && e.altKey && ['1', '2', '3'].includes(e.key)) {
        e.preventDefault()
        const level = parseInt(e.key) as 1 | 2 | 3
        editor.updateBlock(currentBlock, {
          type: 'heading',
          props: { level },
        })
        return
      }

      // Cmd/Ctrl + Alt + 0 - Convert to paragraph
      if (isMod && e.altKey && e.key === '0') {
        e.preventDefault()
        editor.updateBlock(currentBlock, { type: 'paragraph' })
        return
      }

      // Cmd/Ctrl + Shift + 7 - Toggle numbered list
      if (isMod && e.shiftKey && e.key === '7') {
        e.preventDefault()
        editor.updateBlock(currentBlock, { type: 'numberedListItem' })
        return
      }

      // Cmd/Ctrl + Shift + 8 - Toggle bullet list
      if (isMod && e.shiftKey && e.key === '8') {
        e.preventDefault()
        editor.updateBlock(currentBlock, { type: 'bulletListItem' })
        return
      }

      // Cmd/Ctrl + Shift + 9 - Toggle todo/checklist
      if (isMod && e.shiftKey && e.key === '9') {
        e.preventDefault()
        editor.updateBlock(currentBlock, { type: 'checkListItem' })
        return
      }

      // Cmd/Ctrl + Shift + . - Toggle quote block
      if (isMod && e.shiftKey && e.key === '.') {
        e.preventDefault()
        editor.updateBlock(currentBlock, { type: 'quote' })
        return
      }

      // Cmd/Ctrl + Shift + C - Toggle code block
      if (isMod && e.shiftKey && e.key === 'c') {
        e.preventDefault()
        editor.updateBlock(currentBlock, { type: 'codeBlock' })
        return
      }

      // Cmd/Ctrl + Shift + T - Toggle toggle list
      if (isMod && e.shiftKey && e.key === 't') {
        e.preventDefault()
        editor.updateBlock(currentBlock, { type: 'toggleListItem' })
        return
      }

      // Note: Tab/Shift+Tab for nesting is handled natively by BlockNote
      // Note: Cmd+K for links is handled by BlockNote's CreateLinkButton
      // Note: Escape is handled natively by BlockNote to close menus/clear selection
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [editor, saveBlocks])

  // Determine effective theme
  const effectiveTheme = theme === 'dark' ? 'dark' : 'light'

  return (
    <MantineProvider>
      <div className="block-editor">
        <div className="editor-status">
          {isSaving && <span className="save-indicator saving">Saving...</span>}
          {saveError && <span className="save-indicator error">{saveError}</span>}
          {!isSaving && !saveError && lastSaved && (
            <span className="save-indicator saved">
              Saved at {lastSaved.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
            </span>
          )}
        </div>
        <BlockNoteView
          editor={editor}
          theme={effectiveTheme}
          onChange={handleEditorChange}
          slashMenu={false}
          sideMenu={false}
          formattingToolbar={false}
        >
          {/* Side menu with add button and drag handle for block manipulation */}
          <SideMenuController
            sideMenu={(props) => (
              <SideMenu {...props}>
                <AddBlockButton {...props} />
                <DragHandleButton {...props} dragHandleMenu={CustomDragHandleMenu} />
              </SideMenu>
            )}
          />
          {/* Slash menu with multi-column items */}
          <SuggestionMenuController
            triggerCharacter="/"
            getItems={async (query) => {
              try {
                const customItems = getCustomSlashMenuItems(editor)
                const multiColumnItems = getMultiColumnSlashMenuItems(editor)
                const allItems = combineByGroup(customItems, multiColumnItems)
                return filterSuggestionItems(allItems, query)
              } catch (err) {
                console.error('Error loading slash menu items:', err)
                // Return default items as fallback
                return filterSuggestionItems(getDefaultReactSlashMenuItems(editor), query)
              }
            }}
          />
          {/* @ mention menu */}
          <SuggestionMenuController
            triggerCharacter="@"
            getItems={async (query) => {
              const items = await getMentionItems(query, workspaceId)
              return items.map((item) => ({
                title: item.label,
                onItemClick: () => {
                  editor.insertInlineContent([
                    {
                      type: 'mention',
                      props: {
                        mentionType: item.type,
                        id: item.id,
                        label: item.label,
                      },
                    },
                    ' ',
                  ])
                },
                icon: item.type === 'user' ? (
                  <User size={16} />
                ) : item.type === 'page' ? (
                  <FileText size={16} />
                ) : (
                  <Calendar size={16} />
                ),
                subtext: item.type === 'user' && item.email ? item.email :
                         item.type === 'date' && item.date ? item.date.toLocaleDateString() :
                         item.type === 'page' ? 'Page' : '',
                group: item.type === 'user' ? 'People' : item.type === 'page' ? 'Pages' : 'Dates',
              }))
            }}
          />
          <FormattingToolbarController
            formattingToolbar={() => (
              <FormattingToolbar>
                <BlockTypeSelect key="blockTypeSelect" />

                <ToolbarSeparator key="sep1" />

                {/* Text styling buttons */}
                <BasicTextStyleButton basicTextStyle="bold" key="boldStyleButton" />
                <BasicTextStyleButton basicTextStyle="italic" key="italicStyleButton" />
                <BasicTextStyleButton basicTextStyle="underline" key="underlineStyleButton" />
                <BasicTextStyleButton basicTextStyle="strike" key="strikeStyleButton" />
                <BasicTextStyleButton basicTextStyle="code" key="codeStyleButton" />

                <ToolbarSeparator key="sep2" />

                {/* Custom color pickers - Notion style */}
                <TextColorButton key="textColorButton" />
                <HighlightColorButton key="highlightColorButton" />

                <ToolbarSeparator key="sep3" />

                {/* Text alignment */}
                <TextAlignButton textAlignment="left" key="textAlignLeftButton" />
                <TextAlignButton textAlignment="center" key="textAlignCenterButton" />
                <TextAlignButton textAlignment="right" key="textAlignRightButton" />

                <ToolbarSeparator key="sep4" />

                {/* Block manipulation */}
                <NestBlockButton key="nestBlockButton" />
                <UnnestBlockButton key="unnestBlockButton" />
                <CreateLinkButton key="createLinkButton" />
              </FormattingToolbar>
            )}
          />
        </BlockNoteView>
      </div>
    </MantineProvider>
  )
}

export { schema }
