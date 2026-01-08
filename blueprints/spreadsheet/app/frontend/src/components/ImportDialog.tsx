import React, { useState, useCallback, useRef } from 'react';
import type { ImportOptions, ImportResult, ImportProgress } from '../utils/api';

interface ImportDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onImport: (file: File, options: ImportOptions, onProgress?: (progress: ImportProgress) => void) => Promise<ImportResult>;
  onSuccess?: (result: ImportResult) => void;
  format?: string;
}

// Format bytes to human readable string
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// Format speed to human readable string
function formatSpeed(bytesPerSecond: number): string {
  return formatBytes(bytesPerSecond) + '/s';
}

export const ImportDialog: React.FC<ImportDialogProps> = ({
  isOpen,
  onClose,
  onImport,
  onSuccess,
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
  const [progress, setProgress] = useState<ImportProgress | null>(null);
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
    }
  }, []);

  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
      setError(null);
    }
  }, []);

  const handleImport = useCallback(async () => {
    if (!file) return;

    setLoading(true);
    setError(null);
    setProgress(null);

    try {
      const importResult = await onImport(
        file,
        {
          ...options,
          format: format || undefined,
        },
        setProgress
      );

      // On success, call onSuccess callback and close dialog
      setLoading(false);
      setFile(null);
      setProgress(null);
      onClose();

      if (onSuccess) {
        onSuccess(importResult);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Import failed');
      setLoading(false);
      setProgress(null);
    }
  }, [file, options, format, onImport, onSuccess, onClose]);

  const handleClose = useCallback(() => {
    setFile(null);
    setError(null);
    setProgress(null);
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

          {/* Progress */}
          {loading && progress && (
            <div className="import-progress">
              <div className="progress-header">
                <span className="progress-phase">
                  {progress.phase === 'uploading' ? 'Uploading...' : 'Processing...'}
                </span>
                {progress.phase === 'uploading' && progress.speed > 0 && (
                  <span className="progress-speed">{formatSpeed(progress.speed)}</span>
                )}
              </div>
              <div className="progress-bar-container">
                <div
                  className={`progress-bar ${progress.phase === 'processing' ? 'indeterminate' : ''}`}
                  style={{
                    width: progress.phase === 'uploading'
                      ? `${Math.round((progress.loaded / progress.total) * 100)}%`
                      : '100%',
                  }}
                />
              </div>
              <div className="progress-details">
                {progress.phase === 'uploading' ? (
                  <span>
                    {formatBytes(progress.loaded)} / {formatBytes(progress.total)}
                    {' '}({Math.round((progress.loaded / progress.total) * 100)}%)
                  </span>
                ) : (
                  <span>Processing imported data...</span>
                )}
              </div>
            </div>
          )}

          {/* Error */}
          {error && (
            <div className="import-error">
              {error}
            </div>
          )}
        </div>

        <div className="dialog-footer">
          <button className="btn-secondary" onClick={handleClose} disabled={loading}>
            Cancel
          </button>
          <button
            className="btn-primary"
            onClick={handleImport}
            disabled={!file || loading}
          >
            {loading ? 'Importing...' : 'Import'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ImportDialog;
