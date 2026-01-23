import { CompletionContext, CompletionResult, Completion } from '@codemirror/autocomplete'
import type { Table, Column } from '../../api/types'

// SQL keywords for autocomplete
const SQL_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'IS', 'NULL',
  'JOIN', 'LEFT', 'RIGHT', 'INNER', 'OUTER', 'FULL', 'CROSS', 'ON',
  'GROUP', 'BY', 'HAVING', 'ORDER', 'ASC', 'DESC', 'LIMIT', 'OFFSET',
  'AS', 'DISTINCT', 'ALL', 'UNION', 'INTERSECT', 'EXCEPT',
  'INSERT', 'INTO', 'VALUES', 'UPDATE', 'SET', 'DELETE',
  'CREATE', 'ALTER', 'DROP', 'TABLE', 'INDEX', 'VIEW',
  'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
  'BETWEEN', 'LIKE', 'ILIKE', 'EXISTS',
  'TRUE', 'FALSE',
  'WITH', 'RECURSIVE',
  'NULLS', 'FIRST', 'LAST',
]

// SQL functions for autocomplete
const SQL_FUNCTIONS = [
  // Aggregate functions
  'COUNT', 'SUM', 'AVG', 'MIN', 'MAX',
  'COUNT_DISTINCT', 'STRING_AGG', 'ARRAY_AGG',
  // String functions
  'LOWER', 'UPPER', 'LENGTH', 'TRIM', 'LTRIM', 'RTRIM',
  'SUBSTRING', 'SUBSTR', 'CONCAT', 'REPLACE', 'COALESCE',
  // Date functions
  'DATE', 'TIME', 'DATETIME', 'TIMESTAMP',
  'DATE_TRUNC', 'DATE_PART', 'EXTRACT',
  'NOW', 'CURRENT_DATE', 'CURRENT_TIME', 'CURRENT_TIMESTAMP',
  'YEAR', 'MONTH', 'DAY', 'HOUR', 'MINUTE', 'SECOND',
  'DATE_ADD', 'DATE_SUB', 'DATEDIFF',
  // Numeric functions
  'ABS', 'ROUND', 'CEIL', 'FLOOR', 'MOD', 'POWER', 'SQRT',
  // Conditional functions
  'IF', 'IFNULL', 'NULLIF', 'CAST', 'CONVERT',
]

// Icons for different completion types
const ICONS: Record<string, string> = {
  table: '\ud83d\udcca',
  column: '\ud83d\udcdd',
  keyword: '\ud83d\udd35',
  function: '\ud83d\udd27',
  variable: '\ud83c\udfaf',
}

// Type icons for columns
const TYPE_ICONS: Record<string, string> = {
  string: '\ud83c\udd70\ufe0f',
  number: '#\ufe0f\u20e3',
  boolean: '\u2714\ufe0f',
  datetime: '\ud83d\udcc5',
  date: '\ud83d\udcc5',
  json: '{}',
}

export interface AutocompleteOptions {
  tables: Table[]
  getColumnsForTable: (tableId: string) => Column[] | undefined
  onFetchColumns?: (tableId: string) => Promise<Column[]>
}

/**
 * Create a CodeMirror autocomplete extension for SQL
 */
export function createSqlAutocomplete(options: AutocompleteOptions) {
  const { tables, getColumnsForTable } = options

  return async function sqlAutocomplete(context: CompletionContext): Promise<CompletionResult | null> {
    // Get the word before cursor
    const word = context.matchBefore(/[\w.]*/)
    if (!word) return null

    // Don't autocomplete if we're inside a string
    const line = context.state.doc.lineAt(context.pos)
    const beforeCursor = line.text.slice(0, context.pos - line.from)
    const singleQuotes = (beforeCursor.match(/'/g) || []).length
    const doubleQuotes = (beforeCursor.match(/"/g) || []).length
    if (singleQuotes % 2 === 1 || doubleQuotes % 2 === 1) return null

    // Check if we're completing a variable
    const varMatch = context.matchBefore(/\{\{[\w]*/)
    if (varMatch) {
      return {
        from: varMatch.from + 2,
        options: getVariableSuggestions(),
      }
    }

    // Determine context
    const sqlContext = getSqlContext(context)

    const completions: Completion[] = []

    // Add table suggestions after FROM, JOIN, etc.
    if (sqlContext === 'table' || sqlContext === 'any') {
      for (const table of tables) {
        completions.push({
          label: table.name,
          type: 'class',
          detail: 'table',
          info: `${ICONS.table} ${table.display_name || table.name}`,
          boost: sqlContext === 'table' ? 10 : 0,
        })
      }
    }

    // Add column suggestions
    if (sqlContext === 'column' || sqlContext === 'any') {
      // Get columns from all available tables
      for (const table of tables) {
        const columns = getColumnsForTable(table.id)
        if (columns) {
          for (const col of columns) {
            const typeIcon = TYPE_ICONS[col.mapped_type || 'string'] || ''
            completions.push({
              label: col.name,
              type: 'property',
              detail: `${table.name}.${col.mapped_type || 'unknown'}`,
              info: `${typeIcon} ${col.display_name || col.name}`,
              boost: sqlContext === 'column' ? 5 : 0,
            })
            // Also add fully qualified name
            completions.push({
              label: `${table.name}.${col.name}`,
              type: 'property',
              detail: col.mapped_type || 'unknown',
              info: `${typeIcon} ${col.display_name || col.name}`,
              boost: sqlContext === 'column' ? 3 : 0,
            })
          }
        }
      }
    }

    // Add SQL keywords
    if (sqlContext === 'keyword' || sqlContext === 'any') {
      for (const kw of SQL_KEYWORDS) {
        completions.push({
          label: kw,
          type: 'keyword',
          detail: 'keyword',
          boost: sqlContext === 'keyword' ? 2 : -5,
        })
      }
    }

    // Add SQL functions
    if (sqlContext === 'function' || sqlContext === 'any') {
      for (const fn of SQL_FUNCTIONS) {
        completions.push({
          label: fn,
          type: 'function',
          detail: 'function',
          apply: `${fn}()`,
          boost: sqlContext === 'function' ? 3 : -3,
        })
      }
    }

    // Filter by prefix if we have one
    const prefix = word.text.toLowerCase()
    const filtered = prefix
      ? completions.filter(c => c.label.toLowerCase().startsWith(prefix))
      : completions

    if (filtered.length === 0) return null

    return {
      from: word.from,
      options: filtered,
      validFor: /^[\w.]*$/,
    }
  }
}

/**
 * Determine the SQL context at cursor position
 */
function getSqlContext(context: CompletionContext): 'table' | 'column' | 'keyword' | 'function' | 'any' {
  const line = context.state.doc.lineAt(context.pos)
  const beforeCursor = line.text.slice(0, context.pos - line.from).toUpperCase()

  // Check for table context (after FROM, JOIN, INTO, UPDATE)
  const tableKeywords = ['FROM ', 'JOIN ', 'INTO ', 'UPDATE ', 'TABLE ']
  for (const kw of tableKeywords) {
    const idx = beforeCursor.lastIndexOf(kw)
    if (idx !== -1) {
      const afterKeyword = beforeCursor.slice(idx + kw.length).trim()
      // If there's no comma after the keyword and no other keyword, we're still in table context
      if (!afterKeyword.includes(',') && !afterKeyword.match(/\b(WHERE|ON|SET|VALUES)\b/)) {
        return 'table'
      }
    }
  }

  // Check for column context (after SELECT, WHERE, ON, ORDER BY, GROUP BY)
  const columnKeywords = ['SELECT ', 'WHERE ', 'ON ', 'ORDER BY ', 'GROUP BY ', 'HAVING ', 'AND ', 'OR ']
  for (const kw of columnKeywords) {
    const idx = beforeCursor.lastIndexOf(kw)
    if (idx !== -1) {
      const afterKeyword = beforeCursor.slice(idx + kw.length).trim()
      if (!afterKeyword.match(/\b(FROM|WHERE|GROUP|ORDER|HAVING|LIMIT)\b/)) {
        return 'column'
      }
    }
  }

  // Check for function context (after SELECT, aggregate position)
  if (beforeCursor.match(/SELECT\s+$/i) || beforeCursor.match(/,\s*$/)) {
    return 'function'
  }

  // Default to any
  return 'any'
}

/**
 * Get variable name suggestions
 */
function getVariableSuggestions(): Completion[] {
  // Common variable names
  const suggestions = [
    { name: 'start_date', type: 'date' },
    { name: 'end_date', type: 'date' },
    { name: 'category', type: 'text' },
    { name: 'customer_id', type: 'number' },
    { name: 'min_price', type: 'number' },
    { name: 'max_price', type: 'number' },
    { name: 'search', type: 'text' },
    { name: 'limit', type: 'number' },
    { name: 'country', type: 'text' },
    { name: 'status', type: 'text' },
  ]

  return suggestions.map(s => ({
    label: s.name,
    type: 'variable',
    detail: `${s.type} variable`,
    info: `${ICONS.variable} {{${s.name}}}`,
    apply: s.name + '}}',
  }))
}

/**
 * Get all unique columns from tables
 */
export function getAllColumns(tables: Table[], getColumnsForTable: (id: string) => Column[] | undefined): Column[] {
  const allColumns: Column[] = []
  const seen = new Set<string>()

  for (const table of tables) {
    const columns = getColumnsForTable(table.id)
    if (columns) {
      for (const col of columns) {
        const key = `${table.name}.${col.name}`
        if (!seen.has(key)) {
          seen.add(key)
          allColumns.push(col)
        }
      }
    }
  }

  return allColumns
}
