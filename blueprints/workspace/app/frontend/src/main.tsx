import React, { useState, useEffect, Suspense, lazy } from 'react'
import { createRoot } from 'react-dom/client'
import { initApp } from './app'
import { devTestBlocks, devDatabaseProperties, devDatabaseRows, devDatabaseViews } from './dev'
import './styles/main.css'

// Lazy load heavy components for better initial bundle size
const BlockEditor = lazy(() => import('./editor/BlockEditor').then(m => ({ default: m.BlockEditor })))
const DatabaseView = lazy(() => import('./database/DatabaseView').then(m => ({ default: m.DatabaseView })))
const IconPicker = lazy(() => import('./components/IconPicker').then(m => ({ default: m.IconPicker })))
const PageExport = lazy(() => import('./pages/PageExport').then(m => ({ default: m.PageExport })))
const DatabaseViewShowcase = lazy(() => import('./dev/DatabaseViewShowcase').then(m => ({ default: m.DatabaseViewShowcase })))

// Loading fallback component
function LoadingFallback() {
  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '40px',
      color: 'var(--text-secondary, #6b7280)',
      fontSize: '14px',
    }}>
      Loading...
    </div>
  )
}

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

  // Expose dev blocks to window for export functionality
  if (isDevMode) {
    (window as unknown as { __DEV_BLOCKS__: typeof initialBlocks }).__DEV_BLOCKS__ = initialBlocks
  }

  const root = createRoot(editorContainer)
  root.render(
    <React.StrictMode>
      <Suspense fallback={<LoadingFallback />}>
        <BlockEditor
          pageId={pageId}
          initialBlocks={initialBlocks}
        />
      </Suspense>
    </React.StrictMode>
  )
}

// Mount database view
const databaseContainer = document.getElementById('database-root')
if (databaseContainer) {
  const databaseId = databaseContainer.dataset.databaseId || ''
  const viewType = databaseContainer.dataset.viewType || 'table'
  const initialData = databaseContainer.dataset.data

  // In dev mode with empty data, use comprehensive test data
  let dbInitialData: { rows: unknown[]; properties: unknown[]; views: unknown[] } = { rows: [], properties: [], views: [] }
  if (initialData && initialData.trim()) {
    try {
      dbInitialData = JSON.parse(initialData)
    } catch (e) {
      console.warn('Failed to parse initial database data:', e)
    }
  }

  // Use dev data in dev mode when no data provided
  if (isDevMode && dbInitialData.rows.length === 0) {
    console.log('Dev mode: Loading comprehensive database test data')
    dbInitialData = {
      rows: devDatabaseRows as unknown[],
      properties: devDatabaseProperties as unknown[],
      views: devDatabaseViews as unknown[],
    }
  }

  const root = createRoot(databaseContainer)
  root.render(
    <React.StrictMode>
      <Suspense fallback={<LoadingFallback />}>
        <DatabaseView
          databaseId={databaseId || 'dev-database'}
          viewType={viewType as 'table' | 'board' | 'list' | 'calendar' | 'gallery' | 'timeline' | 'chart'}
          initialData={dbInitialData as Parameters<typeof DatabaseView>[0]['initialData']}
        />
      </Suspense>
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
      <Suspense fallback={<LoadingFallback />}>
        <IconPicker
          currentIcon={currentIcon}
          onSelect={(icon) => {
            const event = new CustomEvent('iconSelected', { detail: { target, icon } })
            document.dispatchEvent(event)
          }}
        />
      </Suspense>
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
      // Set page title for dev mode export
      if (isDevMode) {
        (window as unknown as { __DEV_PAGE_TITLE__: string }).__DEV_PAGE_TITLE__ = e.detail.pageTitle
      }
    }

    window.addEventListener('open-export-modal', handleOpenExport as EventListener)
    return () => {
      window.removeEventListener('open-export-modal', handleOpenExport as EventListener)
    }
  }, [])

  return (
    <Suspense fallback={null}>
      <PageExport
        pageId={pageInfo.pageId}
        pageTitle={pageInfo.pageTitle}
        isOpen={isOpen}
        onClose={() => setIsOpen(false)}
      />
    </Suspense>
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

// Mount views showcase in dev mode
const viewsShowcaseContainer = document.getElementById('views-showcase-root')
if (viewsShowcaseContainer && isDevMode) {
  const root = createRoot(viewsShowcaseContainer)
  root.render(
    <React.StrictMode>
      <Suspense fallback={<LoadingFallback />}>
        <DatabaseViewShowcase />
      </Suspense>
    </React.StrictMode>
  )
}
