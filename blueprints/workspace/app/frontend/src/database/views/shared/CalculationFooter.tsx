import { useMemo, useState, useCallback } from 'react'
import { DatabaseRow, Property } from '../../../api/client'
import { ChevronDown, Calculator } from 'lucide-react'

export type CalculationType =
  | 'none'
  | 'count_all'
  | 'count_values'
  | 'count_unique'
  | 'count_empty'
  | 'count_not_empty'
  | 'percent_empty'
  | 'percent_not_empty'
  | 'sum'
  | 'average'
  | 'median'
  | 'min'
  | 'max'
  | 'range'
  | 'earliest_date'
  | 'latest_date'
  | 'date_range'

export interface ColumnCalculation {
  propertyId: string
  type: CalculationType
}

interface CalculationFooterProps {
  rows: DatabaseRow[]
  properties: Property[]
  calculations: ColumnCalculation[]
  onCalculationChange: (propertyId: string, type: CalculationType) => void
  columnWidths?: Record<string, number>
}

const CALCULATION_LABELS: Record<CalculationType, string> = {
  none: 'None',
  count_all: 'Count all',
  count_values: 'Count values',
  count_unique: 'Count unique',
  count_empty: 'Count empty',
  count_not_empty: 'Count not empty',
  percent_empty: '% Empty',
  percent_not_empty: '% Not empty',
  sum: 'Sum',
  average: 'Average',
  median: 'Median',
  min: 'Min',
  max: 'Max',
  range: 'Range',
  earliest_date: 'Earliest',
  latest_date: 'Latest',
  date_range: 'Date range',
}

// Calculation types available for each property type
const CALCULATION_OPTIONS: Record<string, CalculationType[]> = {
  common: ['none', 'count_all', 'count_values', 'count_unique', 'count_empty', 'count_not_empty', 'percent_empty', 'percent_not_empty'],
  number: ['sum', 'average', 'median', 'min', 'max', 'range'],
  date: ['earliest_date', 'latest_date', 'date_range'],
}

function getCalculationOptions(property: Property): CalculationType[] {
  const options = [...CALCULATION_OPTIONS.common]
  if (property.type === 'number') {
    options.push(...CALCULATION_OPTIONS.number)
  }
  if (property.type === 'date' || property.type === 'created_time' || property.type === 'last_edited_time') {
    options.push(...CALCULATION_OPTIONS.date)
  }
  return options
}

function calculateValue(
  rows: DatabaseRow[],
  propertyId: string,
  type: CalculationType
): string {
  const values = rows.map((row) => row.properties[propertyId])
  const nonEmptyValues = values.filter((v) => v !== null && v !== undefined && v !== '')
  const total = rows.length

  switch (type) {
    case 'none':
      return ''
    case 'count_all':
      return String(total)
    case 'count_values':
    case 'count_not_empty':
      return String(nonEmptyValues.length)
    case 'count_empty':
      return String(total - nonEmptyValues.length)
    case 'count_unique':
      return String(new Set(nonEmptyValues.map((v) => JSON.stringify(v))).size)
    case 'percent_empty':
      return total === 0 ? '0%' : `${Math.round(((total - nonEmptyValues.length) / total) * 100)}%`
    case 'percent_not_empty':
      return total === 0 ? '0%' : `${Math.round((nonEmptyValues.length / total) * 100)}%`
    case 'sum': {
      const nums = nonEmptyValues.filter((v) => typeof v === 'number') as number[]
      return nums.length === 0 ? '-' : String(nums.reduce((a, b) => a + b, 0))
    }
    case 'average': {
      const nums = nonEmptyValues.filter((v) => typeof v === 'number') as number[]
      if (nums.length === 0) return '-'
      const avg = nums.reduce((a, b) => a + b, 0) / nums.length
      return avg.toFixed(2)
    }
    case 'median': {
      const nums = (nonEmptyValues.filter((v) => typeof v === 'number') as number[]).sort(
        (a, b) => a - b
      )
      if (nums.length === 0) return '-'
      const mid = Math.floor(nums.length / 2)
      return nums.length % 2 ? String(nums[mid]) : String((nums[mid - 1] + nums[mid]) / 2)
    }
    case 'min': {
      const nums = nonEmptyValues.filter((v) => typeof v === 'number') as number[]
      return nums.length === 0 ? '-' : String(Math.min(...nums))
    }
    case 'max': {
      const nums = nonEmptyValues.filter((v) => typeof v === 'number') as number[]
      return nums.length === 0 ? '-' : String(Math.max(...nums))
    }
    case 'range': {
      const nums = nonEmptyValues.filter((v) => typeof v === 'number') as number[]
      if (nums.length === 0) return '-'
      return String(Math.max(...nums) - Math.min(...nums))
    }
    case 'earliest_date': {
      const dates = nonEmptyValues
        .map((v) => new Date(String(v)))
        .filter((d) => !isNaN(d.getTime()))
      if (dates.length === 0) return '-'
      const earliest = new Date(Math.min(...dates.map((d) => d.getTime())))
      return earliest.toLocaleDateString()
    }
    case 'latest_date': {
      const dates = nonEmptyValues
        .map((v) => new Date(String(v)))
        .filter((d) => !isNaN(d.getTime()))
      if (dates.length === 0) return '-'
      const latest = new Date(Math.max(...dates.map((d) => d.getTime())))
      return latest.toLocaleDateString()
    }
    case 'date_range': {
      const dates = nonEmptyValues
        .map((v) => new Date(String(v)))
        .filter((d) => !isNaN(d.getTime()))
      if (dates.length < 2) return '-'
      const min = Math.min(...dates.map((d) => d.getTime()))
      const max = Math.max(...dates.map((d) => d.getTime()))
      const days = Math.round((max - min) / (1000 * 60 * 60 * 24))
      return `${days} days`
    }
    default:
      return ''
  }
}

export function CalculationFooter({
  rows,
  properties,
  calculations,
  onCalculationChange,
  columnWidths = {},
}: CalculationFooterProps) {
  const [openMenuFor, setOpenMenuFor] = useState<string | null>(null)

  const getCalculationType = useCallback(
    (propertyId: string): CalculationType => {
      const calc = calculations.find((c) => c.propertyId === propertyId)
      return calc?.type || 'none'
    },
    [calculations]
  )

  const calculatedValues = useMemo(() => {
    const result: Record<string, string> = {}
    for (const property of properties) {
      const calcType = getCalculationType(property.id)
      result[property.id] = calculateValue(rows, property.id, calcType)
    }
    return result
  }, [rows, properties, getCalculationType])

  return (
    <div className="calculation-footer">
      {properties.map((property, index) => {
        const calcType = getCalculationType(property.id)
        const value = calculatedValues[property.id]
        const width = columnWidths[property.id] || (index === 0 ? 280 : 150)

        return (
          <div
            key={property.id}
            className="calculation-cell"
            style={{ width, minWidth: width, maxWidth: width }}
          >
            <div
              className="calculation-trigger"
              onClick={() => setOpenMenuFor(openMenuFor === property.id ? null : property.id)}
            >
              {calcType === 'none' ? (
                <span className="calculation-placeholder">
                  <Calculator size={12} />
                  Calculate
                </span>
              ) : (
                <span className="calculation-value">
                  <span className="calc-label">{CALCULATION_LABELS[calcType]}</span>
                  <span className="calc-result">{value}</span>
                </span>
              )}
              <ChevronDown size={12} className="calc-chevron" />
            </div>

            {openMenuFor === property.id && (
              <>
                <div className="calculation-backdrop" onClick={() => setOpenMenuFor(null)} />
                <div className="calculation-menu">
                  {getCalculationOptions(property).map((option) => (
                    <button
                      key={option}
                      type="button"
                      className={`calculation-option ${calcType === option ? 'active' : ''}`}
                      onClick={() => {
                        onCalculationChange(property.id, option)
                        setOpenMenuFor(null)
                      }}
                    >
                      {CALCULATION_LABELS[option]}
                      {option !== 'none' && (
                        <span className="option-preview">
                          {calculateValue(rows, property.id, option)}
                        </span>
                      )}
                    </button>
                  ))}
                </div>
              </>
            )}
          </div>
        )
      })}

      <style>{`
        .calculation-footer {
          display: flex;
          border-top: 1px solid #e9e9e7;
          background: #fbfbfa;
          height: 34px;
          overflow: hidden;
        }

        .calculation-cell {
          position: relative;
          border-right: 1px solid #e9e9e7;
          flex-shrink: 0;
        }

        .calculation-trigger {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 0 8px;
          height: 100%;
          cursor: pointer;
          font-size: 12px;
          color: #787774;
          transition: background 0.1s;
        }

        .calculation-trigger:hover {
          background: rgba(55, 53, 47, 0.04);
        }

        .calculation-placeholder {
          display: flex;
          align-items: center;
          gap: 4px;
          opacity: 0;
          transition: opacity 0.1s;
        }

        .calculation-cell:hover .calculation-placeholder {
          opacity: 1;
        }

        .calculation-value {
          display: flex;
          align-items: center;
          gap: 6px;
        }

        .calc-label {
          font-size: 11px;
          color: #9a9a97;
        }

        .calc-result {
          font-weight: 500;
          color: #37352f;
        }

        .calc-chevron {
          opacity: 0;
          transition: opacity 0.1s;
        }

        .calculation-cell:hover .calc-chevron {
          opacity: 1;
        }

        .calculation-backdrop {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          z-index: 999;
        }

        .calculation-menu {
          position: absolute;
          bottom: 100%;
          left: 0;
          margin-bottom: 4px;
          background: #fff;
          border: 1px solid rgba(55, 53, 47, 0.09);
          border-radius: 6px;
          box-shadow: rgba(15, 15, 15, 0.05) 0px 0px 0px 1px,
            rgba(15, 15, 15, 0.1) 0px 3px 6px, rgba(15, 15, 15, 0.2) 0px 9px 24px;
          min-width: 180px;
          max-height: 320px;
          overflow-y: auto;
          padding: 4px 0;
          z-index: 1000;
        }

        .calculation-option {
          display: flex;
          align-items: center;
          justify-content: space-between;
          width: 100%;
          padding: 6px 12px;
          border: none;
          background: none;
          cursor: pointer;
          text-align: left;
          font-size: 13px;
          color: #37352f;
          transition: background 0.1s;
        }

        .calculation-option:hover {
          background: rgba(55, 53, 47, 0.04);
        }

        .calculation-option.active {
          background: rgba(35, 131, 226, 0.1);
          color: #2383e2;
        }

        .option-preview {
          font-size: 11px;
          color: #9a9a97;
        }

        .calculation-option.active .option-preview {
          color: rgba(35, 131, 226, 0.7);
        }
      `}</style>
    </div>
  )
}

export default CalculationFooter
