import type { Field, CellValue } from '../../../types';

/**
 * Parse clipboard data for multi-cell paste
 */
export function parseClipboardData(text: string): string[][] {
  const lines = text.split(/\r?\n/).filter(line => line.length > 0);
  return lines.map(line => line.split('\t'));
}

/**
 * Convert cell value to clipboard-friendly string
 */
export function cellValueToString(value: CellValue, field: Field): string {
  if (value === null || value === undefined) return '';

  switch (field.type) {
    case 'multi_select':
      if (Array.isArray(value)) {
        const options = field.options?.choices || [];
        return (value as string[])
          .map(id => options.find(o => o.id === id)?.name || id)
          .join(', ');
      }
      return '';
    case 'single_select':
      const options = field.options?.choices || [];
      const opt = options.find(o => o.id === value);
      return opt?.name || String(value);
    case 'checkbox':
      return value ? 'true' : 'false';
    case 'date':
    case 'datetime':
      return value ? new Date(value as string).toLocaleDateString() : '';
    case 'user':
      if (Array.isArray(value)) {
        return (value as { name: string }[]).map(u => u.name).join(', ');
      }
      return '';
    case 'attachment':
      if (Array.isArray(value)) {
        return `${value.length} file(s)`;
      }
      return '';
    default:
      return String(value);
  }
}

/**
 * Parse string value to appropriate cell value based on field type
 */
export function stringToCellValue(text: string, field: Field): CellValue {
  const trimmed = text.trim();
  if (trimmed === '') return null;

  switch (field.type) {
    case 'number':
    case 'currency':
    case 'percent':
    case 'rating':
      const num = parseFloat(trimmed.replace(/[$%,]/g, ''));
      return isNaN(num) ? null : num;
    case 'checkbox':
      return trimmed.toLowerCase() === 'true' || trimmed === '1' || trimmed.toLowerCase() === 'yes';
    case 'date':
      const date = new Date(trimmed);
      return isNaN(date.getTime()) ? null : date.toISOString().split('T')[0];
    case 'datetime':
      const datetime = new Date(trimmed);
      return isNaN(datetime.getTime()) ? null : datetime.toISOString();
    case 'single_select':
      const options = field.options?.choices || [];
      const matchedOption = options.find(
        o => o.name.toLowerCase() === trimmed.toLowerCase() || o.id === trimmed
      );
      return matchedOption?.id || null;
    case 'multi_select':
      const multiOptions = field.options?.choices || [];
      const names = trimmed.split(',').map(s => s.trim());
      const ids = names
        .map(name => multiOptions.find(o => o.name.toLowerCase() === name.toLowerCase())?.id)
        .filter(Boolean) as string[];
      return ids.length > 0 ? ids : null;
    default:
      return trimmed;
  }
}

/**
 * Copy cells to clipboard
 */
export async function copyToClipboard(
  records: { id: string; values: Record<string, CellValue> }[],
  fields: Field[],
  selection: { startRow: number; startCol: number; endRow: number; endCol: number }
): Promise<void> {
  const minRow = Math.min(selection.startRow, selection.endRow);
  const maxRow = Math.max(selection.startRow, selection.endRow);
  const minCol = Math.min(selection.startCol, selection.endCol);
  const maxCol = Math.max(selection.startCol, selection.endCol);

  const lines: string[] = [];

  for (let r = minRow; r <= maxRow; r++) {
    const record = records[r];
    if (!record) continue;

    const cells: string[] = [];
    for (let c = minCol; c <= maxCol; c++) {
      const field = fields[c];
      if (!field) continue;

      const value = record.values[field.id];
      cells.push(cellValueToString(value, field));
    }
    lines.push(cells.join('\t'));
  }

  await navigator.clipboard.writeText(lines.join('\n'));
}

/**
 * Detect text pattern with incrementing numbers (e.g., "Item 1", "Item 2")
 */
function detectTextNumberPattern(values: string[]): { prefix: string; suffix: string; numbers: number[] } | null {
  if (values.length < 2) return null;

  // Try to find a common pattern with numbers
  const patterns = values.map(v => {
    const match = v.match(/^(.*?)(\d+)(.*?)$/);
    if (match) {
      return { prefix: match[1], num: parseInt(match[2], 10), suffix: match[3] };
    }
    return null;
  });

  // Check if all values have the same pattern
  if (patterns.every(p => p !== null)) {
    const first = patterns[0]!;
    if (patterns.every(p => p!.prefix === first.prefix && p!.suffix === first.suffix)) {
      const numbers = patterns.map(p => p!.num);
      // Check if it's an arithmetic sequence
      const diff = numbers[1] - numbers[0];
      let isArithmetic = true;
      for (let i = 2; i < numbers.length; i++) {
        if (numbers[i] - numbers[i - 1] !== diff) {
          isArithmetic = false;
          break;
        }
      }
      if (isArithmetic) {
        return { prefix: first.prefix, suffix: first.suffix, numbers };
      }
    }
  }

  return null;
}

/**
 * Generate fill values for sequences
 */
export function generateFillSequence(
  values: CellValue[],
  field: Field,
  count: number
): CellValue[] {
  if (values.length === 0 || count === 0) return [];

  // For single value, try to detect a pattern in the value itself
  if (values.length === 1) {
    const val = values[0];

    // For text fields, try to detect a number in the value and increment
    if (field.type === 'text' && typeof val === 'string') {
      const match = val.match(/^(.*?)(\d+)(.*?)$/);
      if (match) {
        const prefix = match[1];
        const num = parseInt(match[2], 10);
        const suffix = match[3];
        return Array.from({ length: count }, (_, i) => `${prefix}${num + i + 1}${suffix}`);
      }
    }

    return Array(count).fill(values[0]);
  }

  // For multiple values, try to detect and continue a sequence
  if (field.type === 'number' || field.type === 'currency' || field.type === 'percent' || field.type === 'rating') {
    const nums = values.map(v => Number(v)).filter(n => !isNaN(n));
    if (nums.length >= 2) {
      const diff = nums[1] - nums[0];
      let isArithmetic = true;
      for (let i = 2; i < nums.length; i++) {
        if (Math.abs(nums[i] - nums[i - 1] - diff) > 0.0001) {
          isArithmetic = false;
          break;
        }
      }

      if (isArithmetic) {
        const result: CellValue[] = [];
        let last = nums[nums.length - 1];
        for (let i = 0; i < count; i++) {
          last += diff;
          // Round to avoid floating point issues
          result.push(Math.round(last * 1000000) / 1000000);
        }
        return result;
      }

      // Check for geometric sequence
      if (nums.every(n => n !== 0)) {
        const ratio = nums[1] / nums[0];
        let isGeometric = true;
        for (let i = 2; i < nums.length; i++) {
          if (Math.abs(nums[i] / nums[i - 1] - ratio) > 0.0001) {
            isGeometric = false;
            break;
          }
        }
        if (isGeometric && ratio !== 1) {
          const result: CellValue[] = [];
          let last = nums[nums.length - 1];
          for (let i = 0; i < count; i++) {
            last *= ratio;
            result.push(Math.round(last * 1000000) / 1000000);
          }
          return result;
        }
      }
    }
  }

  if (field.type === 'date' || field.type === 'datetime') {
    const dates = values.map(v => v ? new Date(v as string) : null).filter(d => d !== null) as Date[];
    if (dates.length >= 2) {
      const diffMs = dates[1].getTime() - dates[0].getTime();
      let isArithmetic = true;
      for (let i = 2; i < dates.length; i++) {
        if (Math.abs(dates[i].getTime() - dates[i - 1].getTime() - diffMs) > 1000) { // 1 second tolerance
          isArithmetic = false;
          break;
        }
      }

      if (isArithmetic) {
        const result: CellValue[] = [];
        let last = dates[dates.length - 1].getTime();

        // Detect common intervals for smarter incrementing
        const dayMs = 24 * 60 * 60 * 1000;
        const isMonthly = Math.abs(diffMs - 30 * dayMs) < dayMs;

        for (let i = 0; i < count; i++) {
          if (isMonthly) {
            // Use month-based increment to handle varying month lengths
            const lastDate = new Date(last);
            lastDate.setMonth(lastDate.getMonth() + 1);
            last = lastDate.getTime();
          } else {
            last += diffMs;
          }
          const d = new Date(last);
          result.push(field.type === 'date' ? d.toISOString().split('T')[0] : d.toISOString());
        }
        return result;
      }
    }
  }

  // For text fields, try to detect numbered patterns
  if (field.type === 'text') {
    const textValues = values.filter((v): v is string => typeof v === 'string');
    if (textValues.length >= 2) {
      const pattern = detectTextNumberPattern(textValues);
      if (pattern) {
        const result: CellValue[] = [];
        const diff = pattern.numbers[1] - pattern.numbers[0];
        let lastNum = pattern.numbers[pattern.numbers.length - 1];
        for (let i = 0; i < count; i++) {
          lastNum += diff;
          result.push(`${pattern.prefix}${lastNum}${pattern.suffix}`);
        }
        return result;
      }
    }
  }

  // Default: cycle through values
  const result: CellValue[] = [];
  for (let i = 0; i < count; i++) {
    result.push(values[i % values.length]);
  }
  return result;
}
