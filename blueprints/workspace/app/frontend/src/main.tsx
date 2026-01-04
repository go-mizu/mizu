import React, { useState, useEffect } from 'react'
import { createRoot } from 'react-dom/client'
import { BlockEditor } from './editor/BlockEditor'
import { DatabaseView } from './database/DatabaseView'
import { IconPicker } from './components/IconPicker'
import { PageExport } from './pages/PageExport'
import { initApp } from './app'
import { devTestBlocks } from './dev'
import './styles/main.css'

// Initialize the app
initApp()

// Check if we're in Vite dev mode
const isDevMode = import.meta.env.DEV

// Mount React components where needed
const editorContainer = document.getElementById('editor-root')
if (editorContainer) {
  const pageId = editorContainer.dataset.pageId || ''
  const initialBlocksData = editorContainer.dataset.blocks

  // Use dev test blocks if in dev mode and no blocks provided
  let initialBlocks = []
  if (initialBlocksData && initialBlocksData.trim()) {
    try {
      initialBlocks = JSON.parse(initialBlocksData)
    } catch (e) {
      console.warn('Failed to parse initial blocks:', e)
    }
  }

  // In dev mode with empty blocks, use comprehensive test data
  if (isDevMode && initialBlocks.length === 0) {
    console.log('Dev mode: Loading comprehensive test blocks')
    initialBlocks = devTestBlocks
  }

  const root = createRoot(editorContainer)
  root.render(
    <React.StrictMode>
      <BlockEditor
        pageId={pageId}
        initialBlocks={initialBlocks}
      />
    </React.StrictMode>
  )
}

// Mount database view
const databaseContainer = document.getElementById('database-root')
if (databaseContainer) {
  const databaseId = databaseContainer.dataset.databaseId || ''
  const viewType = databaseContainer.dataset.viewType || 'table'
  const initialData = databaseContainer.dataset.data

  const root = createRoot(databaseContainer)
  root.render(
    <React.StrictMode>
      <DatabaseView
        databaseId={databaseId}
        viewType={viewType as 'table' | 'board' | 'list' | 'calendar' | 'gallery'}
        initialData={initialData ? JSON.parse(initialData) : { rows: [], properties: [] }}
      />
    </React.StrictMode>
  )
}

// Mount icon pickers
document.querySelectorAll('[data-icon-picker]').forEach((el) => {
  const container = el as HTMLElement
  const target = container.dataset.target || ''
  const currentIcon = container.dataset.currentIcon || ''

  const root = createRoot(container)
  root.render(
    <React.StrictMode>
      <IconPicker
        currentIcon={currentIcon}
        onSelect={(icon) => {
          const event = new CustomEvent('iconSelected', { detail: { target, icon } })
          document.dispatchEvent(event)
        }}
      />
    </React.StrictMode>
  )
})

// Export Modal wrapper component for dev mode
function ExportModalWrapper() {
  const [isOpen, setIsOpen] = useState(false)
  const [pageInfo, setPageInfo] = useState({ pageId: '', pageTitle: '' })

  useEffect(() => {
    const handleOpenExport = (e: CustomEvent<{ pageId: string; pageTitle: string }>) => {
      setPageInfo(e.detail)
      setIsOpen(true)
    }

    window.addEventListener('open-export-modal', handleOpenExport as EventListener)
    return () => {
      window.removeEventListener('open-export-modal', handleOpenExport as EventListener)
    }
  }, [])

  return (
    <PageExport
      pageId={pageInfo.pageId}
      pageTitle={pageInfo.pageTitle}
      isOpen={isOpen}
      onClose={() => setIsOpen(false)}
    />
  )
}

// Mount export modal in dev mode
const exportModalContainer = document.getElementById('export-modal-root')
if (exportModalContainer && isDevMode) {
  const root = createRoot(exportModalContainer)
  root.render(
    <React.StrictMode>
      <ExportModalWrapper />
    </React.StrictMode>
  )
}
