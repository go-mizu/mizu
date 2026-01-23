import { useMemo, useState, useCallback } from 'react'

export type VariableType = 'text' | 'number' | 'date' | 'dropdown'

export interface Variable {
  name: string
  type: VariableType
  required: boolean
  defaultValue?: string | number | Date
  options?: string[]  // For dropdown type
}

export interface ParsedVariable {
  name: string
  start: number  // Position in SQL string
  end: number
}

// Regular expression to match {{variable_name}}
const VARIABLE_REGEX = /\{\{(\w+)\}\}/g

/**
 * Parse variables from SQL string
 */
export function parseVariables(sql: string): ParsedVariable[] {
  const variables: ParsedVariable[] = []
  let match: RegExpExecArray | null

  // Reset regex state
  VARIABLE_REGEX.lastIndex = 0

  while ((match = VARIABLE_REGEX.exec(sql)) !== null) {
    variables.push({
      name: match[1],
      start: match.index,
      end: match.index + match[0].length,
    })
  }

  // Return unique variables (by name)
  const seen = new Set<string>()
  return variables.filter(v => {
    if (seen.has(v.name)) return false
    seen.add(v.name)
    return true
  })
}

/**
 * Infer variable type from context
 */
export function inferVariableType(sql: string, variableName: string): VariableType {
  const lowerVarName = variableName.toLowerCase()

  // Check variable name for date patterns
  const datePatterns = ['date', 'time', 'timestamp', 'created', 'updated', '_at']
  for (const pattern of datePatterns) {
    if (lowerVarName.includes(pattern)) {
      return 'date'
    }
  }

  // Check variable name for number patterns
  const numberPatterns = ['price', 'amount', 'count', 'qty', 'quantity', 'id', 'limit', 'offset', 'min', 'max', 'total']
  for (const pattern of numberPatterns) {
    if (lowerVarName.includes(pattern)) {
      return 'number'
    }
  }

  // Look for context clues around the variable in the SQL
  const patterns = [
    // Date patterns in SQL context
    { regex: new RegExp(`(date|time|created|updated|_at).*\\{\\{${variableName}\\}\\}`, 'i'), type: 'date' as const },
    { regex: new RegExp(`\\{\\{${variableName}\\}\\}.*date`, 'i'), type: 'date' as const },
    { regex: new RegExp(`BETWEEN.*\\{\\{${variableName}\\}\\}.*AND`, 'i'), type: 'date' as const },
    // Number patterns in SQL context
    { regex: new RegExp(`\\{\\{${variableName}\\}\\}.*[<>]`, 'i'), type: 'number' as const },
    { regex: new RegExp(`LIMIT\\s+\\{\\{${variableName}\\}\\}`, 'i'), type: 'number' as const },
  ]

  for (const pattern of patterns) {
    if (pattern.regex.test(sql)) {
      return pattern.type
    }
  }

  // Default to text
  return 'text'
}

/**
 * Substitute variables in SQL with parameterized placeholders
 * Returns the modified SQL and ordered parameter values
 */
export function substituteVariables(
  sql: string,
  values: Record<string, string | number | Date | null>
): { sql: string; params: (string | number | Date | null)[] } {
  const params: (string | number | Date | null)[] = []
  let result = sql

  // Reset regex state
  VARIABLE_REGEX.lastIndex = 0

  // Replace each variable with a placeholder
  result = sql.replace(VARIABLE_REGEX, (_match, varName) => {
    const value = values[varName]
    params.push(value ?? null)
    return '?'
  })

  return { sql: result, params }
}

/**
 * Hook for managing query variables
 */
export function useQueryVariables(sql: string) {
  const [values, setValues] = useState<Record<string, string | number | Date | null>>({})

  // Parse variables from SQL
  const parsedVariables = useMemo(() => parseVariables(sql), [sql])

  // Build variable definitions with inferred types
  const variables = useMemo(() => {
    return parsedVariables.map(v => ({
      ...v,
      type: inferVariableType(sql, v.name),
      required: true, // Default to required
    }))
  }, [parsedVariables, sql])

  // Set a variable value
  const setValue = useCallback((name: string, value: string | number | Date | null) => {
    setValues(prev => ({
      ...prev,
      [name]: value,
    }))
  }, [])

  // Clear a variable value
  const clearValue = useCallback((name: string) => {
    setValues(prev => {
      const next = { ...prev }
      delete next[name]
      return next
    })
  }, [])

  // Clear all variable values
  const clearAll = useCallback(() => {
    setValues({})
  }, [])

  // Check if all required variables have values
  const allRequiredFilled = useMemo(() => {
    return variables.every(v => !v.required || values[v.name] != null)
  }, [variables, values])

  // Get substituted SQL
  const getSubstitutedSql = useCallback(() => {
    return substituteVariables(sql, values)
  }, [sql, values])

  return {
    variables,
    values,
    setValue,
    clearValue,
    clearAll,
    allRequiredFilled,
    getSubstitutedSql,
  }
}
