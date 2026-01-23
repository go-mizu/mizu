import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { tags } from '@lezer/highlight'
import { EditorView } from '@codemirror/view'

// Metabase-aligned color scheme
const colors = {
  keyword: '#509EE3',      // SELECT, FROM, WHERE, JOIN
  function: '#84BB4C',     // COUNT, SUM, AVG, DATE_TRUNC
  string: '#ED6E6E',       // 'string values'
  number: '#F9A825',       // 123, 45.67
  comment: '#93A1A1',      // -- comments
  variable: '#7172AD',     // {{variable}}
  operator: '#509EE3',     // =, >, <, AND, OR
  bracket: '#509EE3',      // (, ), [, ]
  identifier: '#2E2E2E',   // table/column names
}

// Syntax highlighting styles
export const sqlHighlightStyle = HighlightStyle.define([
  { tag: tags.keyword, color: colors.keyword, fontWeight: '600' },
  { tag: tags.operatorKeyword, color: colors.keyword, fontWeight: '600' },
  { tag: tags.function(tags.variableName), color: colors.function },
  { tag: tags.string, color: colors.string },
  { tag: tags.number, color: colors.number },
  { tag: tags.comment, color: colors.comment, fontStyle: 'italic' },
  { tag: tags.lineComment, color: colors.comment, fontStyle: 'italic' },
  { tag: tags.blockComment, color: colors.comment, fontStyle: 'italic' },
  { tag: tags.operator, color: colors.operator },
  { tag: tags.punctuation, color: colors.bracket },
  { tag: tags.bracket, color: colors.bracket },
  { tag: tags.variableName, color: colors.identifier },
  { tag: tags.special(tags.variableName), color: colors.variable },
  { tag: tags.bool, color: colors.keyword },
  { tag: tags.null, color: colors.keyword },
])

// Editor theme (gutter, selection, etc.)
export const editorTheme = EditorView.theme({
  '&': {
    fontSize: '14px',
    fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace',
  },
  '.cm-content': {
    padding: '12px 0',
    minHeight: '200px',
  },
  '.cm-scroller': {
    overflow: 'auto',
  },
  '.cm-gutters': {
    backgroundColor: '#F8F9FA',
    borderRight: '1px solid #E9ECEF',
    color: '#93A1A1',
  },
  '.cm-lineNumbers .cm-gutterElement': {
    padding: '0 12px 0 8px',
    minWidth: '40px',
  },
  '.cm-activeLineGutter': {
    backgroundColor: '#E6F2FF',
  },
  '.cm-activeLine': {
    backgroundColor: '#F8F9FA',
  },
  '.cm-selectionBackground': {
    backgroundColor: '#E6F2FF !important',
  },
  '&.cm-focused .cm-selectionBackground': {
    backgroundColor: '#CCE5FF !important',
  },
  '.cm-cursor': {
    borderLeftColor: '#509EE3',
    borderLeftWidth: '2px',
  },
  '.cm-matchingBracket': {
    backgroundColor: '#CCE5FF',
    outline: '1px solid #509EE3',
  },
  '.cm-tooltip': {
    border: '1px solid #E9ECEF',
    borderRadius: '4px',
    boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
  },
  '.cm-tooltip-autocomplete': {
    '& > ul': {
      fontFamily: 'inherit',
      maxHeight: '300px',
    },
    '& > ul > li': {
      padding: '4px 12px',
    },
    '& > ul > li[aria-selected]': {
      backgroundColor: '#E6F2FF',
      color: '#2E2E2E',
    },
  },
  // Variable highlighting ({{variable}})
  '.cm-variable-highlight': {
    backgroundColor: '#EFE6F7',
    color: colors.variable,
    borderRadius: '3px',
    padding: '0 2px',
  },
  // Error highlighting
  '.cm-lintRange-error': {
    backgroundImage: 'none',
    borderBottom: '2px wavy #ED6E6E',
  },
  '.cm-lintRange-warning': {
    backgroundImage: 'none',
    borderBottom: '2px wavy #F9A825',
  },
})

// Combined extensions for syntax highlighting
export const sqlSyntaxHighlighting = syntaxHighlighting(sqlHighlightStyle)
