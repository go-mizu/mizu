/**
 * Number formatting utilities for spreadsheet cells
 */

export type NumberFormatType =
  | 'general'
  | 'number'
  | 'currency'
  | 'percent'
  | 'scientific'
  | 'date'
  | 'time'
  | 'datetime'
  | 'text';

export interface NumberFormatOptions {
  type: NumberFormatType;
  decimalPlaces?: number;
  useThousandsSeparator?: boolean;
  currencySymbol?: string;
  negativeStyle?: 'minus' | 'parentheses' | 'red' | 'redParentheses';
  locale?: string;
}

const DEFAULT_OPTIONS: NumberFormatOptions = {
  type: 'general',
  decimalPlaces: 2,
  useThousandsSeparator: true,
  currencySymbol: '$',
  negativeStyle: 'minus',
  locale: 'en-US',
};

/**
 * Parse a number format pattern string into options
 */
export function parseFormatPattern(pattern: string): NumberFormatOptions {
  const options: NumberFormatOptions = { ...DEFAULT_OPTIONS };

  if (!pattern) return options;

  // Currency pattern: $#,##0.00
  if (pattern.includes('$') || pattern.includes('EUR') || pattern.includes('GBP')) {
    options.type = 'currency';
    const currencyMatch = pattern.match(/(\$|EUR|GBP|£|€)/);
    if (currencyMatch) {
      options.currencySymbol = currencyMatch[1];
    }
  }
  // Percent pattern: 0.00%
  else if (pattern.includes('%')) {
    options.type = 'percent';
  }
  // Scientific notation: 0.00E+00
  else if (pattern.toLowerCase().includes('e+') || pattern.toLowerCase().includes('e-')) {
    options.type = 'scientific';
  }
  // Number pattern: #,##0.00
  else if (pattern.includes('#') || pattern.includes('0')) {
    options.type = 'number';
  }

  // Count decimal places
  const decimalMatch = pattern.match(/\.([0#]+)/);
  if (decimalMatch) {
    options.decimalPlaces = decimalMatch[1].length;
  } else if (!pattern.includes('.')) {
    options.decimalPlaces = 0;
  }

  // Check for thousands separator
  options.useThousandsSeparator = pattern.includes(',');

  // Check for parentheses (negative style)
  if (pattern.includes('(') || pattern.includes(')')) {
    options.negativeStyle = 'parentheses';
  }

  return options;
}

/**
 * Format a numeric value according to the specified options
 */
export function formatNumber(
  value: number | string | boolean | null,
  optionsOrPattern?: NumberFormatOptions | string
): string {
  if (value === null || value === undefined || value === '') {
    return '';
  }

  // Convert to number
  let numValue: number;
  if (typeof value === 'boolean') {
    return value ? 'TRUE' : 'FALSE';
  } else if (typeof value === 'string') {
    numValue = parseFloat(value);
    if (isNaN(numValue)) {
      return value; // Return original string if not a number
    }
  } else {
    numValue = value;
  }

  // Parse options
  const options: NumberFormatOptions =
    typeof optionsOrPattern === 'string'
      ? parseFormatPattern(optionsOrPattern)
      : { ...DEFAULT_OPTIONS, ...optionsOrPattern };

  const {
    type,
    decimalPlaces = 2,
    useThousandsSeparator = true,
    currencySymbol = '$',
    negativeStyle = 'minus',
    locale = 'en-US',
  } = options;

  const isNegative = numValue < 0;
  const absValue = Math.abs(numValue);

  let formatted: string;

  switch (type) {
    case 'currency':
      formatted = formatCurrency(absValue, decimalPlaces, currencySymbol, useThousandsSeparator, locale);
      break;

    case 'percent':
      formatted = formatPercent(absValue, decimalPlaces, useThousandsSeparator, locale);
      break;

    case 'scientific':
      formatted = absValue.toExponential(decimalPlaces);
      break;

    case 'number':
      formatted = formatPlainNumber(absValue, decimalPlaces, useThousandsSeparator, locale);
      break;

    case 'date':
      formatted = formatDate(numValue, locale);
      break;

    case 'time':
      formatted = formatTime(numValue, locale);
      break;

    case 'datetime':
      formatted = formatDateTime(numValue, locale);
      break;

    case 'text':
      return String(numValue);

    case 'general':
    default:
      formatted = formatGeneral(absValue, locale);
      break;
  }

  // Apply negative styling
  if (isNegative) {
    switch (negativeStyle) {
      case 'parentheses':
        formatted = `(${formatted})`;
        break;
      case 'red':
        // CSS class would be applied at render time
        formatted = `-${formatted}`;
        break;
      case 'redParentheses':
        formatted = `(${formatted})`;
        break;
      case 'minus':
      default:
        formatted = `-${formatted}`;
        break;
    }
  }

  return formatted;
}

function formatPlainNumber(
  value: number,
  decimalPlaces: number,
  useThousandsSeparator: boolean,
  locale: string
): string {
  return value.toLocaleString(locale, {
    minimumFractionDigits: decimalPlaces,
    maximumFractionDigits: decimalPlaces,
    useGrouping: useThousandsSeparator,
  });
}

function formatCurrency(
  value: number,
  decimalPlaces: number,
  currencySymbol: string,
  useThousandsSeparator: boolean,
  locale: string
): string {
  const formatted = formatPlainNumber(value, decimalPlaces, useThousandsSeparator, locale);
  return `${currencySymbol}${formatted}`;
}

function formatPercent(
  value: number,
  decimalPlaces: number,
  useThousandsSeparator: boolean,
  locale: string
): string {
  const percentValue = value * 100;
  const formatted = formatPlainNumber(percentValue, decimalPlaces, useThousandsSeparator, locale);
  return `${formatted}%`;
}

function formatGeneral(value: number, locale: string): string {
  // Smart formatting: show minimal decimals needed
  if (Number.isInteger(value)) {
    return value.toLocaleString(locale, { useGrouping: true });
  }

  // Show up to 10 significant digits
  const formatted = value.toPrecision(10);
  const num = parseFloat(formatted);
  return num.toLocaleString(locale, {
    maximumFractionDigits: 10,
    useGrouping: true,
  });
}

function formatDate(excelDate: number, locale: string): string {
  // Excel date serial number (days since 1900-01-01)
  const date = excelSerialToDate(excelDate);
  return date.toLocaleDateString(locale);
}

function formatTime(excelTime: number, locale: string): string {
  // Excel time is the fractional part of a day
  const date = excelSerialToDate(excelTime);
  return date.toLocaleTimeString(locale);
}

function formatDateTime(excelDateTime: number, locale: string): string {
  const date = excelSerialToDate(excelDateTime);
  return date.toLocaleString(locale);
}

function excelSerialToDate(serial: number): Date {
  // Excel serial date starts from 1900-01-01 (serial = 1)
  // JavaScript Date starts from 1970-01-01
  const excelEpoch = new Date(1899, 11, 30); // Dec 30, 1899
  const msPerDay = 24 * 60 * 60 * 1000;
  return new Date(excelEpoch.getTime() + serial * msPerDay);
}

/**
 * Increase decimal places in a format
 */
export function increaseDecimalPlaces(currentFormat?: string): string {
  if (!currentFormat) {
    return '#,##0.0';
  }

  const options = parseFormatPattern(currentFormat);
  const newDecimals = (options.decimalPlaces || 0) + 1;

  return buildFormatPattern({
    ...options,
    decimalPlaces: Math.min(newDecimals, 15), // Max 15 decimals
  });
}

/**
 * Decrease decimal places in a format
 */
export function decreaseDecimalPlaces(currentFormat?: string): string {
  if (!currentFormat) {
    return '#,##0';
  }

  const options = parseFormatPattern(currentFormat);
  const newDecimals = Math.max((options.decimalPlaces || 0) - 1, 0);

  return buildFormatPattern({
    ...options,
    decimalPlaces: newDecimals,
  });
}

/**
 * Build a format pattern string from options
 */
export function buildFormatPattern(options: NumberFormatOptions): string {
  const { type, decimalPlaces = 2, useThousandsSeparator = true, currencySymbol = '$' } = options;

  let pattern = '';

  // Add currency symbol
  if (type === 'currency') {
    pattern += currencySymbol;
  }

  // Add number pattern
  if (useThousandsSeparator) {
    pattern += '#,##0';
  } else {
    pattern += '0';
  }

  // Add decimals
  if (decimalPlaces && decimalPlaces > 0) {
    pattern += '.' + '0'.repeat(decimalPlaces);
  }

  // Add percent sign
  if (type === 'percent') {
    pattern += '%';
  }

  return pattern;
}

/**
 * Get currency format string
 */
export function getCurrencyFormat(decimalPlaces: number = 2): string {
  return buildFormatPattern({
    type: 'currency',
    decimalPlaces,
    useThousandsSeparator: true,
    currencySymbol: '$',
  });
}

/**
 * Get percent format string
 */
export function getPercentFormat(decimalPlaces: number = 2): string {
  return buildFormatPattern({
    type: 'percent',
    decimalPlaces,
    useThousandsSeparator: true,
  });
}

/**
 * Check if a value looks like a number
 */
export function isNumeric(value: unknown): boolean {
  if (typeof value === 'number') return !isNaN(value) && isFinite(value);
  if (typeof value === 'string') {
    const num = parseFloat(value);
    return !isNaN(num) && isFinite(num);
  }
  return false;
}

/**
 * Common number formats for the format dropdown
 */
export const COMMON_FORMATS = [
  { label: 'Automatic', pattern: '' },
  { label: 'Plain text', pattern: '@' },
  { label: 'Number', pattern: '#,##0.00' },
  { label: 'Integer', pattern: '#,##0' },
  { label: 'Currency ($)', pattern: '$#,##0.00' },
  { label: 'Currency (no decimals)', pattern: '$#,##0' },
  { label: 'Percent', pattern: '0.00%' },
  { label: 'Percent (no decimals)', pattern: '0%' },
  { label: 'Scientific', pattern: '0.00E+00' },
  { label: 'Date', pattern: 'MM/DD/YYYY' },
  { label: 'Time', pattern: 'HH:MM:SS' },
  { label: 'Date Time', pattern: 'MM/DD/YYYY HH:MM' },
];

export default formatNumber;
