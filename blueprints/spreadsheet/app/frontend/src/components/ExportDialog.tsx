import React, { useState, useCallback } from 'react';
import type { ExportOptions } from '../utils/api';

export type ExportFormat = 'csv' | 'tsv' | 'xlsx' | 'json' | 'pdf' | 'html';

interface ExportDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onExport: (format: ExportFormat, options: ExportOptions) => Promise<void>;
  workbookName: string;
  initialFormat?: ExportFormat;
}

export const ExportDialog: React.FC<ExportDialogProps> = ({
  isOpen,
  onClose,
  onExport,
  workbookName,
  initialFormat,
}) => {
  const [format, setFormat] = useState<ExportFormat>(initialFormat || 'xlsx');
  const [options, setOptions] = useState<ExportOptions>({
    formatting: true,
    formulas: false,
    headers: false,
    gridlines: true,
    orientation: 'portrait',
    paperSize: 'letter',
    metadata: false,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleExport = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      await onExport(format, options);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Export failed');
    } finally {
      setLoading(false);
    }
  }, [format, options, onExport, onClose]);

  const formatInfo: Record<ExportFormat, { name: string; ext: string; description: string }> = {
    xlsx: {
      name: 'Microsoft Excel',
      ext: '.xlsx',
      description: 'Full formatting and formulas support',
    },
    csv: {
      name: 'CSV',
      ext: '.csv',
      description: 'Comma-separated values, plain text',
    },
    tsv: {
      name: 'TSV',
      ext: '.tsv',
      description: 'Tab-separated values, plain text',
    },
    json: {
      name: 'JSON',
      ext: '.json',
      description: 'Structured data with full metadata',
    },
    pdf: {
      name: 'PDF',
      ext: '.pdf',
      description: 'Printable document format',
    },
    html: {
      name: 'HTML',
      ext: '.html',
      description: 'Web page with table formatting',
    },
  };

  if (!isOpen) return null;

  return (
    <div className="dialog-overlay" onClick={onClose}>
      <div className="export-dialog" onClick={(e) => e.stopPropagation()}>
        <div className="dialog-header">
          <h2>Export Spreadsheet</h2>
          <button className="close-btn" onClick={onClose}>&times;</button>
        </div>

        <div className="dialog-content">
          {/* Format Selection */}
          <div className="format-selection">
            <h3>Export Format</h3>
            <div className="format-grid">
              {(Object.keys(formatInfo) as ExportFormat[]).map((fmt) => (
                <label
                  key={fmt}
                  className={`format-option ${format === fmt ? 'selected' : ''}`}
                >
                  <input
                    type="radio"
                    name="format"
                    value={fmt}
                    checked={format === fmt}
                    onChange={() => setFormat(fmt)}
                  />
                  <div className="format-info">
                    <span className="format-name">{formatInfo[fmt].name}</span>
                    <span className="format-ext">{formatInfo[fmt].ext}</span>
                    <span className="format-desc">{formatInfo[fmt].description}</span>
                  </div>
                </label>
              ))}
            </div>
          </div>

          {/* Format-specific Options */}
          <div className="export-options">
            <h3>Export Options</h3>

            {(format === 'xlsx' || format === 'html') && (
              <label className="checkbox-option">
                <input
                  type="checkbox"
                  checked={options.formatting}
                  onChange={(e) => setOptions({ ...options, formatting: e.target.checked })}
                />
                <span>Include cell formatting</span>
              </label>
            )}

            {format === 'xlsx' && (
              <label className="checkbox-option">
                <input
                  type="checkbox"
                  checked={options.formulas}
                  onChange={(e) => setOptions({ ...options, formulas: e.target.checked })}
                />
                <span>Export formulas (instead of values)</span>
              </label>
            )}

            {(format === 'csv' || format === 'tsv' || format === 'html') && (
              <label className="checkbox-option">
                <input
                  type="checkbox"
                  checked={options.headers}
                  onChange={(e) => setOptions({ ...options, headers: e.target.checked })}
                />
                <span>Include column headers (A, B, C...)</span>
              </label>
            )}

            {format === 'pdf' && (
              <>
                <label className="checkbox-option">
                  <input
                    type="checkbox"
                    checked={options.gridlines}
                    onChange={(e) => setOptions({ ...options, gridlines: e.target.checked })}
                  />
                  <span>Include gridlines</span>
                </label>

                <div className="select-option">
                  <label>Orientation:</label>
                  <select
                    value={options.orientation}
                    onChange={(e) => setOptions({ ...options, orientation: e.target.value as 'portrait' | 'landscape' })}
                  >
                    <option value="portrait">Portrait</option>
                    <option value="landscape">Landscape</option>
                  </select>
                </div>

                <div className="select-option">
                  <label>Paper Size:</label>
                  <select
                    value={options.paperSize}
                    onChange={(e) => setOptions({ ...options, paperSize: e.target.value as 'letter' | 'a4' | 'legal' })}
                  >
                    <option value="letter">Letter</option>
                    <option value="a4">A4</option>
                    <option value="legal">Legal</option>
                  </select>
                </div>
              </>
            )}

            {format === 'json' && (
              <>
                <label className="checkbox-option">
                  <input
                    type="checkbox"
                    checked={options.metadata}
                    onChange={(e) => setOptions({ ...options, metadata: e.target.checked })}
                  />
                  <span>Include workbook metadata</span>
                </label>

                <label className="checkbox-option">
                  <input
                    type="checkbox"
                    checked={options.compact}
                    onChange={(e) => setOptions({ ...options, compact: e.target.checked })}
                  />
                  <span>Compact JSON (minified)</span>
                </label>
              </>
            )}
          </div>

          {/* Preview filename */}
          <div className="export-preview">
            <span className="preview-label">File name:</span>
            <span className="preview-filename">
              {workbookName}{formatInfo[format].ext}
            </span>
          </div>

          {/* Error */}
          {error && (
            <div className="export-error">
              {error}
            </div>
          )}
        </div>

        <div className="dialog-footer">
          <button className="btn-secondary" onClick={onClose}>
            Cancel
          </button>
          <button
            className="btn-primary"
            onClick={handleExport}
            disabled={loading}
          >
            {loading ? 'Exporting...' : 'Export'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ExportDialog;
