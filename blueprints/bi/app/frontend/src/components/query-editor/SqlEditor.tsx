import { useRef, useCallback, useMemo, useEffect } from 'react'
import CodeMirror, { ReactCodeMirrorRef } from '@uiw/react-codemirror'
import { sql, SQLite } from '@codemirror/lang-sql'
import { keymap, EditorView } from '@codemirror/view'
import { autocompletion } from '@codemirror/autocomplete'
import { indentWithTab } from '@codemirror/commands'
import { search, highlightSelectionMatches } from '@codemirror/search'
import { bracketMatching } from '@codemirror/language'
import { lintGutter } from '@codemirror/lint'
import { format as formatSql } from 'sql-formatter'
import { Box, Paper, Stack } from '@mantine/core'
import { editorTheme, sqlSyntaxHighlighting } from './SqlHighlighting'
import { createSqlAutocomplete } from './AutocompleteProvider'
import EditorToolbar from './EditorToolbar'
import VariableWidgets from './VariableWidgets'
import SchemaExplorer from './SchemaExplorer'
import { useQueryVariables } from './useQueryVariables'
import { useTables } from '../../api/hooks'
import type { Column } from '../../api/types'

interface SqlEditorProps {
  value: string
  onChange: (value: string) => void
  onRun: (sql: string, params?: (string | number | Date | null)[]) => void
  datasourceId: string | null
  isRunning?: boolean
  minHeight?: number
  showSchema?: boolean
  onSchemaToggle?: (visible: boolean) => void
}

export default function SqlEditor({
  value,
  onChange,
  onRun,
  datasourceId,
  isRunning = false,
  minHeight = 200,
  showSchema: initialShowSchema = true,
  onSchemaToggle,
}: SqlEditorProps) {
  const editorRef = useRef<ReactCodeMirrorRef>(null)
  const columnsCache = useRef<Map<string, Column[]>>(new Map())

  // Track schema visibility
  const [showSchema, setShowSchema] = React.useState(initialShowSchema)

  // Variable handling
  const { variables, values, setValue, clearValue, getSubstitutedSql } = useQueryVariables(value)

  // Fetch tables for autocomplete
  const { data: tables = [] } = useTables(datasourceId || '')

  // Create a function to get columns for a table
  const getColumnsForTable = useCallback((tableId: string): Column[] | undefined => {
    return columnsCache.current.get(tableId)
  }, [])

  // Fetch columns for expanded tables
  useEffect(() => {
    // Fetch columns for the first few tables for better autocomplete
    const tablesToFetch = tables.slice(0, 10)
    tablesToFetch.forEach(async (table) => {
      if (!columnsCache.current.has(table.id)) {
        try {
          const response = await fetch(`/api/tables/${table.id}/columns`)
          if (response.ok) {
            const cols = await response.json()
            columnsCache.current.set(table.id, cols)
          }
        } catch {
          // Ignore errors
        }
      }
    })
  }, [tables])

  // Create autocomplete extension
  const autocompleteExtension = useMemo(() => {
    const completionSource = createSqlAutocomplete({
      tables,
      getColumnsForTable,
    })

    return autocompletion({
      override: [completionSource],
      defaultKeymap: true,
      activateOnTyping: true,
      maxRenderedOptions: 50,
    })
  }, [tables, getColumnsForTable])

  // Handle running the query
  const handleRun = useCallback(() => {
    const view = editorRef.current?.view
    if (!view) return

    // Check if there's a selection
    const selection = view.state.selection.main
    let queryToRun = value

    if (!selection.empty) {
      // Run only the selected text
      queryToRun = view.state.sliceDoc(selection.from, selection.to)
    }

    // Substitute variables if any
    if (variables.length > 0) {
      const { sql: substituted, params } = getSubstitutedSql()
      // If running selection, we need to substitute in that too
      if (!selection.empty) {
        // For selection, just run as-is (variables in selection should work)
        onRun(queryToRun)
      } else {
        onRun(substituted, params)
      }
    } else {
      onRun(queryToRun)
    }
  }, [value, variables, getSubstitutedSql, onRun])

  // Handle formatting
  const handleFormat = useCallback(() => {
    try {
      const formatted = formatSql(value, {
        language: 'sqlite',
        keywordCase: 'upper',
        indentStyle: 'standard',
        linesBetweenQueries: 2,
      })
      onChange(formatted)
    } catch {
      // If formatting fails, do nothing
      console.warn('SQL formatting failed')
    }
  }, [value, onChange])

  // Handle copy
  const handleCopy = useCallback(() => {
    navigator.clipboard.writeText(value)
  }, [value])

  // Handle schema toggle
  const handleSchemaToggle = useCallback(() => {
    const newValue = !showSchema
    setShowSchema(newValue)
    onSchemaToggle?.(newValue)
  }, [showSchema, onSchemaToggle])

  // Handle inserting text from schema explorer
  const handleInsertText = useCallback((text: string) => {
    const view = editorRef.current?.view
    if (!view) return

    const pos = view.state.selection.main.head
    view.dispatch({
      changes: { from: pos, insert: text },
      selection: { anchor: pos + text.length },
    })
    view.focus()
  }, [])

  // Custom keymap for running queries
  const runKeymap = useMemo(() => keymap.of([
    {
      key: 'Mod-Enter',
      run: () => {
        handleRun()
        return true
      },
    },
    {
      key: 'Mod-Shift-f',
      run: () => {
        handleFormat()
        return true
      },
    },
  ]), [handleRun, handleFormat])

  // Check if there's a selection
  const hasSelection = useMemo(() => {
    const view = editorRef.current?.view
    if (!view) return false
    return !view.state.selection.main.empty
  }, [value])

  // All extensions
  const extensions = useMemo(() => [
    sql({ dialect: SQLite }),
    editorTheme,
    sqlSyntaxHighlighting,
    autocompleteExtension,
    runKeymap,
    keymap.of([indentWithTab]),
    search(),
    highlightSelectionMatches(),
    bracketMatching(),
    lintGutter(),
    EditorView.lineWrapping,
  ], [autocompleteExtension, runKeymap])

  return (
    <Stack gap={0} h="100%">
      {/* Variable widgets */}
      <VariableWidgets
        variables={variables}
        values={values}
        onValueChange={setValue}
        onClearValue={clearValue}
      />

      {/* Toolbar */}
      <EditorToolbar
        onRun={handleRun}
        onFormat={handleFormat}
        onCopy={handleCopy}
        onToggleSchema={handleSchemaToggle}
        isRunning={isRunning}
        schemaVisible={showSchema}
        hasSelection={hasSelection}
      />

      {/* Main editor area */}
      <Box
        flex={1}
        style={{
          display: 'flex',
          overflow: 'hidden',
        }}
      >
        {/* Schema explorer sidebar */}
        {showSchema && (
          <Paper
            w={220}
            style={{
              borderRight: '1px solid var(--mantine-color-gray-2)',
              overflow: 'hidden',
              flexShrink: 0,
            }}
          >
            <SchemaExplorer
              datasourceId={datasourceId}
              onInsertText={handleInsertText}
            />
          </Paper>
        )}

        {/* Editor */}
        <Box
          flex={1}
          style={{
            overflow: 'auto',
            minHeight,
          }}
        >
          <CodeMirror
            ref={editorRef}
            value={value}
            onChange={onChange}
            extensions={extensions}
            placeholder="-- Write your SQL query here
-- Use {{variable_name}} for parameters
-- Press Cmd/Ctrl + Enter to run

SELECT * FROM products LIMIT 10"
            basicSetup={{
              lineNumbers: true,
              highlightActiveLineGutter: true,
              highlightActiveLine: true,
              foldGutter: true,
              dropCursor: true,
              allowMultipleSelections: true,
              indentOnInput: true,
              closeBrackets: true,
              autocompletion: false, // We use custom
              rectangularSelection: true,
              crosshairCursor: true,
              highlightSelectionMatches: false, // We add it separately
            }}
            style={{
              fontSize: '14px',
              height: '100%',
            }}
          />
        </Box>
      </Box>
    </Stack>
  )
}

// Need to import React for useState
import React from 'react'
