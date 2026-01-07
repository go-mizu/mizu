import React, { useState, useCallback, useRef } from 'react';
import type { ImportOptions, ImportResult } from '../utils/api';

interface ImportDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onImport: (file: File, options: ImportOptions) => Promise<ImportResult>;
  format?: string;
}

export const ImportDialog: React.FC<ImportDialogProps> = ({
  isOpen,
  onClose,
  onImport,
  format,
}) => {
  const [file, setFile] = useState<File | null>(null);
  const [options, setOptions] = useState<ImportOptions>({
    hasHeaders: false,
    skipEmptyRows: true,
    trimWhitespace: true,
    autoDetectTypes: true,
    importFormatting: true,
    importFormulas: true,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [dragActive, setDragActive] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      setDragActive(true);
    } else if (e.type === 'dragleave') {
      setDragActive(false);
    }
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setDragActive(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      setFile(e.dataTransfer.files[0]);
      setError(null);
      setResult(null);
    }
  }, []);

  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
      setError(null);
      setResult(null);
    }
  }, []);

  const handleImport = useCallback(async () => {
    if (!file) return;

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const importResult = await onImport(file, {
        ...options,
        format: format || undefined,
      });
      setResult(importResult);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Import failed');
    } finally {
      setLoading(false);
    }
  }, [file, options, format, onImport]);

  const handleClose = useCallback(() => {
    setFile(null);
    setError(null);
    setResult(null);
    setLoading(false);
    onClose();
  }, [onClose]);

  const getAcceptedFormats = () => {
    switch (format) {
      case 'csv':
        return '.csv';
      case 'tsv':
        return '.tsv,.tab';
      case 'xlsx':
        return '.xlsx,.xlsm';
      case 'json':
        return '.json';
      default:
        return '.csv,.tsv,.xlsx,.xlsm,.json';
    }
  };

  if (!isOpen) return null;

  return (
    <div className="dialog-overlay" onClick={handleClose}>
      <div className="import-dialog" onClick={(e) => e.stopPropagation()}>
        <div className="dialog-header">
          <h2>Import File</h2>
          <button className="close-btn" onClick={handleClose}>&times;</button>
        </div>

        <div className="dialog-content">
          {/* File Drop Zone */}
          <div
            className={`file-dropzone ${dragActive ? 'active' : ''} ${file ? 'has-file' : ''}`}
            onDragEnter={handleDrag}
            onDragLeave={handleDrag}
            onDragOver={handleDrag}
            onDrop={handleDrop}
            onClick={() => fileInputRef.current?.click()}
          >
            <input
              ref={fileInputRef}
              type="file"
              accept={getAcceptedFormats()}
              onChange={handleFileSelect}
              style={{ display: 'none' }}
            />
            {file ? (
              <div className="file-info">
                <span className="file-name">{file.name}</span>
                <span className="file-size">({(file.size / 1024).toFixed(1)} KB)</span>
              </div>
            ) : (
              <div className="dropzone-text">
                <p>Drop a file here or click to select</p>
                <p className="supported-formats">
                  Supported: CSV, TSV, XLSX, JSON
                </p>
              </div>
            )}
          </div>

          {/* Options */}
          <div className="import-options">
            <h3>Import Options</h3>

            <label className="checkbox-option">
              <input
                type="checkbox"
                checked={options.hasHeaders}
                onChange={(e) => setOptions({ ...options, hasHeaders: e.target.checked })}
              />
              <span>First row contains headers</span>
            </label>

            <label className="checkbox-option">
              <input
                type="checkbox"
                checked={options.skipEmptyRows}
                onChange={(e) => setOptions({ ...options, skipEmptyRows: e.target.checked })}
              />
              <span>Skip empty rows</span>
            </label>

            <label className="checkbox-option">
              <input
                type="checkbox"
                checked={options.trimWhitespace}
                onChange={(e) => setOptions({ ...options, trimWhitespace: e.target.checked })}
              />
              <span>Trim whitespace</span>
            </label>

            <label className="checkbox-option">
              <input
                type="checkbox"
                checked={options.autoDetectTypes}
                onChange={(e) => setOptions({ ...options, autoDetectTypes: e.target.checked })}
              />
              <span>Auto-detect data types</span>
            </label>

            {(format === 'xlsx' || !format) && (
              <>
                <label className="checkbox-option">
                  <input
                    type="checkbox"
                    checked={options.importFormatting}
                    onChange={(e) => setOptions({ ...options, importFormatting: e.target.checked })}
                  />
                  <span>Import cell formatting</span>
                </label>

                <label className="checkbox-option">
                  <input
                    type="checkbox"
                    checked={options.importFormulas}
                    onChange={(e) => setOptions({ ...options, importFormulas: e.target.checked })}
                  />
                  <span>Import formulas</span>
                </label>
              </>
            )}

            <div className="text-option">
              <label>New sheet name (optional):</label>
              <input
                type="text"
                value={options.sheetName || ''}
                onChange={(e) => setOptions({ ...options, sheetName: e.target.value })}
                placeholder="Auto-generated from filename"
              />
            </div>
          </div>

          {/* Error */}
          {error && (
            <div className="import-error">
              {error}
            </div>
          )}

          {/* Result */}
          {result && (
            <div className="import-result">
              <h4>Import Complete</h4>
              <p>Rows imported: {result.rowsImported}</p>
              <p>Columns imported: {result.colsImported}</p>
              <p>Total cells: {result.cellsImported}</p>
              {result.warnings && result.warnings.length > 0 && (
                <div className="import-warnings">
                  <h5>Warnings:</h5>
                  <ul>
                    {result.warnings.map((w, i) => (
                      <li key={i}>{w}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="dialog-footer">
          <button className="btn-secondary" onClick={handleClose}>
            {result ? 'Close' : 'Cancel'}
          </button>
          {!result && (
            <button
              className="btn-primary"
              onClick={handleImport}
              disabled={!file || loading}
            >
              {loading ? 'Importing...' : 'Import'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default ImportDialog;
