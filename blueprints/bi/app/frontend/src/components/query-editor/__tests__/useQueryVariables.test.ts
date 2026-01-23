import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import {
  parseVariables,
  inferVariableType,
  substituteVariables,
  useQueryVariables,
} from '../useQueryVariables'

describe('parseVariables', () => {
  it('parses single variable', () => {
    const sql = 'SELECT * FROM products WHERE category = {{category}}'
    const vars = parseVariables(sql)

    expect(vars).toHaveLength(1)
    expect(vars[0].name).toBe('category')
  })

  it('parses multiple variables', () => {
    const sql = 'SELECT * FROM orders WHERE date >= {{start_date}} AND date <= {{end_date}}'
    const vars = parseVariables(sql)

    expect(vars).toHaveLength(2)
    expect(vars[0].name).toBe('start_date')
    expect(vars[1].name).toBe('end_date')
  })

  it('deduplicates variables with same name', () => {
    const sql = 'SELECT * FROM orders WHERE {{category}} = 1 OR name = {{category}}'
    const vars = parseVariables(sql)

    expect(vars).toHaveLength(1)
    expect(vars[0].name).toBe('category')
  })

  it('returns empty array for no variables', () => {
    const sql = 'SELECT * FROM products LIMIT 10'
    const vars = parseVariables(sql)

    expect(vars).toHaveLength(0)
  })

  it('parses variables with underscores', () => {
    const sql = 'SELECT * FROM orders WHERE customer_id = {{customer_id}}'
    const vars = parseVariables(sql)

    expect(vars).toHaveLength(1)
    expect(vars[0].name).toBe('customer_id')
  })

  it('captures variable positions', () => {
    const sql = 'SELECT * FROM products WHERE id = {{product_id}}'
    const vars = parseVariables(sql)

    // Position 34 is where '{{' starts in "WHERE id = {{product_id}}"
    expect(vars[0].start).toBe(34)
    expect(vars[0].end).toBe(48)
  })
})

describe('inferVariableType', () => {
  it('infers date type from variable name containing date', () => {
    const sql = 'SELECT * FROM orders WHERE order_date = {{start_date}}'
    const type = inferVariableType(sql, 'start_date')

    expect(type).toBe('date')
  })

  it('infers date type from variable name containing timestamp', () => {
    const sql = 'SELECT * FROM events WHERE ts = {{event_timestamp}}'
    const type = inferVariableType(sql, 'event_timestamp')

    expect(type).toBe('date')
  })

  it('infers number type from variable name containing price', () => {
    const sql = 'SELECT * FROM products WHERE unit_price > {{min_price}}'
    const type = inferVariableType(sql, 'min_price')

    expect(type).toBe('number')
  })

  it('infers number type from variable name containing id', () => {
    const sql = 'SELECT * FROM orders WHERE customer_id = {{customer_id}}'
    const type = inferVariableType(sql, 'customer_id')

    expect(type).toBe('number')
  })

  it('infers number type from LIMIT context', () => {
    const sql = 'SELECT * FROM products LIMIT {{row_limit}}'
    const type = inferVariableType(sql, 'row_limit')

    expect(type).toBe('number')
  })

  it('defaults to text type', () => {
    const sql = 'SELECT * FROM products WHERE name = {{search}}'
    const type = inferVariableType(sql, 'search')

    expect(type).toBe('text')
  })
})

describe('substituteVariables', () => {
  it('substitutes single variable', () => {
    const sql = 'SELECT * FROM products WHERE category = {{category}}'
    const values = { category: 'Beverages' }

    const result = substituteVariables(sql, values)

    expect(result.sql).toBe('SELECT * FROM products WHERE category = ?')
    expect(result.params).toHaveLength(1)
    expect(result.params[0]).toBe('Beverages')
  })

  it('substitutes multiple variables', () => {
    const sql = 'SELECT * FROM orders WHERE date >= {{start}} AND date <= {{end}}'
    const values = { start: '2024-01-01', end: '2024-12-31' }

    const result = substituteVariables(sql, values)

    expect(result.sql).toBe('SELECT * FROM orders WHERE date >= ? AND date <= ?')
    expect(result.params).toHaveLength(2)
    expect(result.params[0]).toBe('2024-01-01')
    expect(result.params[1]).toBe('2024-12-31')
  })

  it('substitutes duplicate variables with same value', () => {
    const sql = 'SELECT * FROM orders WHERE {{cat}} = 1 OR name = {{cat}}'
    const values = { cat: 'Test' }

    const result = substituteVariables(sql, values)

    expect(result.sql).toBe('SELECT * FROM orders WHERE ? = 1 OR name = ?')
    expect(result.params).toHaveLength(2)
    expect(result.params[0]).toBe('Test')
    expect(result.params[1]).toBe('Test')
  })

  it('preserves SQL structure', () => {
    const sql = `SELECT
      p.name,
      c.name as category
    FROM products p
    JOIN categories c ON p.category_id = c.id
    WHERE c.name = {{category}}
    ORDER BY p.name`
    const values = { category: 'Seafood' }

    const result = substituteVariables(sql, values)

    expect(result.sql).toContain('JOIN categories c')
    expect(result.sql).toContain('WHERE c.name = ?')
    expect(result.params[0]).toBe('Seafood')
  })
})

describe('useQueryVariables hook', () => {
  it('parses variables from SQL', () => {
    const { result } = renderHook(() =>
      useQueryVariables('SELECT * FROM products WHERE category = {{category}}')
    )

    expect(result.current.variables).toHaveLength(1)
    expect(result.current.variables[0].name).toBe('category')
  })

  it('updates variable values', () => {
    const { result } = renderHook(() =>
      useQueryVariables('SELECT * FROM products WHERE category = {{category}}')
    )

    act(() => {
      result.current.setValue('category', 'Beverages')
    })

    expect(result.current.values.category).toBe('Beverages')
  })

  it('clears variable value', () => {
    const { result } = renderHook(() =>
      useQueryVariables('SELECT * FROM products WHERE category = {{category}}')
    )

    act(() => {
      result.current.setValue('category', 'Beverages')
    })

    expect(result.current.values.category).toBe('Beverages')

    act(() => {
      result.current.clearValue('category')
    })

    expect(result.current.values.category).toBeUndefined()
  })

  it('reports all required filled correctly', () => {
    const { result } = renderHook(() =>
      useQueryVariables('SELECT * FROM products WHERE category = {{category}}')
    )

    expect(result.current.allRequiredFilled).toBe(false)

    act(() => {
      result.current.setValue('category', 'Beverages')
    })

    expect(result.current.allRequiredFilled).toBe(true)
  })

  it('getSubstitutedSql returns correct result', () => {
    const { result } = renderHook(() =>
      useQueryVariables('SELECT * FROM products WHERE category = {{category}}')
    )

    act(() => {
      result.current.setValue('category', 'Beverages')
    })

    const substituted = result.current.getSubstitutedSql()

    expect(substituted.sql).toBe('SELECT * FROM products WHERE category = ?')
    expect(substituted.params[0]).toBe('Beverages')
  })

  it('updates when SQL changes', () => {
    const { result, rerender } = renderHook(
      ({ sql }) => useQueryVariables(sql),
      { initialProps: { sql: 'SELECT * FROM products WHERE id = {{id}}' } }
    )

    expect(result.current.variables[0].name).toBe('id')

    rerender({ sql: 'SELECT * FROM orders WHERE date = {{order_date}}' })

    expect(result.current.variables[0].name).toBe('order_date')
  })
})
