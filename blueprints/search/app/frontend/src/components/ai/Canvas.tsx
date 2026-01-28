import { useState } from 'react'
import {
  Plus,
  Download,
  Trash2,
  GripVertical,
  Type,
  Heading1,
  Code,
  StickyNote,
  Minus,
} from 'lucide-react'
import { aiApi } from '../../api/ai'
import type { Canvas as CanvasType, CanvasBlock, BlockType, ExportFormat } from '../../types/ai'

interface CanvasProps {
  canvas: CanvasType
  onUpdate: (canvas: CanvasType) => void
}

const blockTypeIcons: Record<BlockType, React.ReactNode> = {
  text: <Type size={16} />,
  ai_response: <Type size={16} />,
  note: <StickyNote size={16} />,
  citation: <Type size={16} />,
  heading: <Heading1 size={16} />,
  divider: <Minus size={16} />,
  code: <Code size={16} />,
}

interface BlockEditorProps {
  block: CanvasBlock
  sessionId: string
  onUpdate: (block: CanvasBlock) => void
  onDelete: () => void
}

function BlockEditor({ block, sessionId, onUpdate, onDelete }: BlockEditorProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [content, setContent] = useState(block.content)

  const handleSave = async () => {
    if (content !== block.content) {
      try {
        await aiApi.updateBlock(sessionId, block.id, { content })
        onUpdate({ ...block, content })
      } catch (err) {
        console.error('Failed to update block:', err)
      }
    }
    setIsEditing(false)
  }

  const handleDelete = async () => {
    try {
      await aiApi.deleteBlock(sessionId, block.id)
      onDelete()
    } catch (err) {
      console.error('Failed to delete block:', err)
    }
  }

  return (
    <div className={`canvas-block ${block.type}`}>
      <div className="canvas-block-drag">
        <GripVertical size={16} />
      </div>
      <div className="canvas-block-content">
        {isEditing ? (
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            onBlur={handleSave}
            autoFocus
            className="canvas-block-editor"
          />
        ) : (
          <div
            onClick={() => setIsEditing(true)}
            className="canvas-block-text"
          >
            {block.type === 'heading' ? (
              <h2>{block.content}</h2>
            ) : block.type === 'divider' ? (
              <hr />
            ) : block.type === 'code' ? (
              <pre><code>{block.content}</code></pre>
            ) : (
              <p>{block.content}</p>
            )}
          </div>
        )}
      </div>
      <button
        type="button"
        onClick={handleDelete}
        className="canvas-block-delete"
      >
        <Trash2 size={14} />
      </button>
    </div>
  )
}

export function Canvas({ canvas, onUpdate }: CanvasProps) {
  const [isAddingBlock, setIsAddingBlock] = useState(false)
  const blocks = canvas.blocks || []

  const handleAddBlock = async (type: BlockType) => {
    try {
      const { block } = await aiApi.addBlock(
        canvas.session_id,
        type,
        type === 'divider' ? '' : 'New block',
        blocks.length
      )
      onUpdate({
        ...canvas,
        blocks: [...blocks, block],
      })
      setIsAddingBlock(false)
    } catch (err) {
      console.error('Failed to add block:', err)
    }
  }

  const handleUpdateBlock = (index: number, updatedBlock: CanvasBlock) => {
    const newBlocks = [...blocks]
    newBlocks[index] = updatedBlock
    onUpdate({ ...canvas, blocks: newBlocks })
  }

  const handleDeleteBlock = (index: number) => {
    const newBlocks = blocks.filter((_, i) => i !== index)
    onUpdate({ ...canvas, blocks: newBlocks })
  }

  const handleExport = async (format: ExportFormat) => {
    try {
      const blob = await aiApi.exportCanvas(canvas.session_id, format)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${canvas.title || 'canvas'}.${format === 'markdown' ? 'md' : format}`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    } catch (err) {
      console.error('Failed to export canvas:', err)
    }
  }

  return (
    <div className="canvas">
      {/* Header */}
      <div className="canvas-header">
        <h3 className="canvas-title">{canvas.title || 'Research Canvas'}</h3>
        <div className="canvas-actions">
          <div className="canvas-export-dropdown">
            <button type="button" className="canvas-export-button">
              <Download size={16} />
              Export
            </button>
            <div className="canvas-export-menu">
              <button type="button" onClick={() => handleExport('markdown')}>
                Markdown
              </button>
              <button type="button" onClick={() => handleExport('html')}>
                HTML
              </button>
              <button type="button" onClick={() => handleExport('json')}>
                JSON
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Blocks */}
      <div className="canvas-blocks">
        {blocks.map((block, index) => (
          <BlockEditor
            key={block.id}
            block={block}
            sessionId={canvas.session_id}
            onUpdate={(b) => handleUpdateBlock(index, b)}
            onDelete={() => handleDeleteBlock(index)}
          />
        ))}

        {/* Add block button */}
        {isAddingBlock ? (
          <div className="canvas-add-menu">
            <button type="button" onClick={() => handleAddBlock('text')}>
              {blockTypeIcons.text} Text
            </button>
            <button type="button" onClick={() => handleAddBlock('heading')}>
              {blockTypeIcons.heading} Heading
            </button>
            <button type="button" onClick={() => handleAddBlock('note')}>
              {blockTypeIcons.note} Note
            </button>
            <button type="button" onClick={() => handleAddBlock('code')}>
              {blockTypeIcons.code} Code
            </button>
            <button type="button" onClick={() => handleAddBlock('divider')}>
              {blockTypeIcons.divider} Divider
            </button>
            <button type="button" onClick={() => setIsAddingBlock(false)}>
              Cancel
            </button>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setIsAddingBlock(true)}
            className="canvas-add-button"
          >
            <Plus size={16} />
            Add block
          </button>
        )}
      </div>
    </div>
  )
}
