import type { Field, TableRecord } from '../types';
import { cellValueToString } from '../components/views/grid/clipboardUtils';

/**
 * Escape a value for CSV format
 */
function escapeCSV(value: string): string {
  // If value contains comma, quote, or newline, wrap in quotes
  if (value.includes(',') || value.includes('"') || value.includes('\n') || value.includes('\r')) {
    // Escape any existing quotes by doubling them
    return `"${value.replace(/"/g, '""')}"`;
  }
  return value;
}

/**
 * Export records to CSV format
 */
export function exportToCSV(
  records: TableRecord[],
  fields: Field[],
  options: {
    includeHeaders?: boolean;
    filename?: string;
  } = {}
): void {
  const { includeHeaders = true, filename = 'export.csv' } = options;

  const lines: string[] = [];

  // Add header row
  if (includeHeaders) {
    const headers = fields.map((f) => escapeCSV(f.name));
    lines.push(headers.join(','));
  }

  // Add data rows
  records.forEach((record) => {
    const row = fields.map((field) => {
      const value = record.values[field.id];
      const stringValue = cellValueToString(value, field);
      return escapeCSV(stringValue);
    });
    lines.push(row.join(','));
  });

  const csvContent = lines.join('\n');

  // Create blob and download
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);

  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.style.display = 'none';

  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);

  URL.revokeObjectURL(url);
}

/**
 * Export to TSV (Tab-separated values) format
 */
export function exportToTSV(
  records: TableRecord[],
  fields: Field[],
  options: {
    includeHeaders?: boolean;
    filename?: string;
  } = {}
): void {
  const { includeHeaders = true, filename = 'export.tsv' } = options;

  const lines: string[] = [];

  // Add header row
  if (includeHeaders) {
    const headers = fields.map((f) => f.name.replace(/\t/g, ' '));
    lines.push(headers.join('\t'));
  }

  // Add data rows
  records.forEach((record) => {
    const row = fields.map((field) => {
      const value = record.values[field.id];
      const stringValue = cellValueToString(value, field);
      // Replace tabs and newlines with spaces
      return stringValue.replace(/[\t\n\r]/g, ' ');
    });
    lines.push(row.join('\t'));
  });

  const tsvContent = lines.join('\n');

  // Create blob and download
  const blob = new Blob([tsvContent], { type: 'text/tab-separated-values;charset=utf-8;' });
  const url = URL.createObjectURL(blob);

  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  link.style.display = 'none';

  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);

  URL.revokeObjectURL(url);
}
