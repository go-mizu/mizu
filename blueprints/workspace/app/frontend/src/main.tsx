import React from 'react'
import { createRoot } from 'react-dom/client'
import { BlockEditor } from './editor/BlockEditor'
import { DatabaseView } from './database/DatabaseView'
import { IconPicker } from './components/IconPicker'
import { initApp } from './app'
import './styles/main.css'

// Initialize the app
initApp()

// Mount React components where needed
const editorContainer = document.getElementById('editor-root')
if (editorContainer) {
  const pageId = editorContainer.dataset.pageId || ''
  const initialBlocks = editorContainer.dataset.blocks

  const root = createRoot(editorContainer)
  root.render(
    <React.StrictMode>
      <BlockEditor
        pageId={pageId}
        initialBlocks={initialBlocks ? JSON.parse(initialBlocks) : []}
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
