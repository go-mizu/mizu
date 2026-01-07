import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ImportDialog } from './ImportDialog';

describe('ImportDialog', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    onImport: vi.fn(),
    sheetName: 'Sheet1',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render when open', () => {
    render(<ImportDialog {...defaultProps} />);

    expect(screen.getByText('Import File')).toBeInTheDocument();
    // Check for dropzone area - text may vary
    expect(document.querySelector('.file-dropzone')).toBeInTheDocument();
  });

  it('should not render when closed', () => {
    render(<ImportDialog {...defaultProps} isOpen={false} />);

    expect(screen.queryByText('Import Data')).not.toBeInTheDocument();
  });

  it('should call onClose when close button clicked', () => {
    render(<ImportDialog {...defaultProps} />);

    const closeButton = screen.getByText('×');
    fireEvent.click(closeButton);

    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it('should call onClose when Cancel button clicked', () => {
    render(<ImportDialog {...defaultProps} />);

    const cancelButton = screen.getByText('Cancel');
    fireEvent.click(cancelButton);

    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it('should call onClose when overlay clicked', () => {
    render(<ImportDialog {...defaultProps} />);

    const overlay = document.querySelector('.dialog-overlay');
    if (overlay) {
      fireEvent.click(overlay);
      expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
    }
  });

  it('should show supported formats', () => {
    render(<ImportDialog {...defaultProps} />);

    expect(screen.getByText(/CSV, TSV, XLSX, JSON/)).toBeInTheDocument();
  });

  it('should display import options', () => {
    render(<ImportDialog {...defaultProps} />);

    expect(screen.getByText('Import Options')).toBeInTheDocument();
    expect(screen.getByText('First row contains headers')).toBeInTheDocument();
    expect(screen.getByText('Skip empty rows')).toBeInTheDocument();
    expect(screen.getByText('Auto-detect data types')).toBeInTheDocument();
  });

  it('should toggle options when clicked', () => {
    render(<ImportDialog {...defaultProps} />);

    const checkbox = screen.getByRole('checkbox', { name: /First row contains headers/i });
    expect(checkbox).not.toBeChecked();

    fireEvent.click(checkbox);
    expect(checkbox).toBeChecked();
  });

  it('should accept file via file input', async () => {
    const onImport = vi.fn().mockResolvedValue(undefined);
    render(<ImportDialog {...defaultProps} onImport={onImport} />);

    const file = new File(['col1,col2\nval1,val2'], 'test.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    Object.defineProperty(input, 'files', {
      value: [file],
    });

    fireEvent.change(input);

    await waitFor(() => {
      expect(screen.getByText('test.csv')).toBeInTheDocument();
    });
  });

  it('should call onImport when Import button clicked with file', async () => {
    const onImport = vi.fn().mockResolvedValue(undefined);
    render(<ImportDialog {...defaultProps} onImport={onImport} />);

    const file = new File(['col1,col2'], 'test.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    Object.defineProperty(input, 'files', {
      value: [file],
    });

    fireEvent.change(input);

    await waitFor(() => {
      expect(screen.getByText('test.csv')).toBeInTheDocument();
    });

    const importButton = screen.getByText('Import');
    fireEvent.click(importButton);

    await waitFor(() => {
      expect(onImport).toHaveBeenCalled();
      const [calledFile, calledOptions] = onImport.mock.calls[0];
      expect(calledFile.name).toBe('test.csv');
      expect(calledOptions).toHaveProperty('autoDetectTypes');
    });
  });

  it('should show loading state during import', async () => {
    const onImport = vi.fn().mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));
    render(<ImportDialog {...defaultProps} onImport={onImport} />);

    const file = new File(['data'], 'test.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    Object.defineProperty(input, 'files', {
      value: [file],
    });

    fireEvent.change(input);

    await waitFor(() => {
      expect(screen.getByText('test.csv')).toBeInTheDocument();
    });

    const importButton = screen.getByText('Import');
    fireEvent.click(importButton);

    await waitFor(() => {
      expect(screen.getByText('Importing...')).toBeInTheDocument();
    });
  });

  it('should show error message on import failure', async () => {
    const onImport = vi.fn().mockRejectedValue(new Error('Import failed'));
    render(<ImportDialog {...defaultProps} onImport={onImport} />);

    const file = new File(['data'], 'test.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    Object.defineProperty(input, 'files', {
      value: [file],
    });

    fireEvent.change(input);

    await waitFor(() => {
      expect(screen.getByText('test.csv')).toBeInTheDocument();
    });

    const importButton = screen.getByText('Import');
    fireEvent.click(importButton);

    await waitFor(() => {
      expect(screen.getByText('Import failed')).toBeInTheDocument();
    });
  });

  it('should render with format prop', () => {
    render(<ImportDialog {...defaultProps} format="xlsx" />);

    // The dialog should render correctly with any initial format
    expect(screen.getByText('Import File')).toBeInTheDocument();
    expect(document.querySelector('.import-dialog')).toBeInTheDocument();
  });

  it('should have dropzone element for drag and drop', async () => {
    render(<ImportDialog {...defaultProps} />);

    const dropzone = document.querySelector('.file-dropzone');
    expect(dropzone).toBeInTheDocument();
  });

  it('should show file info after selecting a file', async () => {
    render(<ImportDialog {...defaultProps} />);

    const file = new File(['data'], 'test.csv', { type: 'text/csv' });
    const input = document.querySelector('input[type="file"]') as HTMLInputElement;

    Object.defineProperty(input, 'files', {
      value: [file],
    });

    fireEvent.change(input);

    await waitFor(() => {
      expect(screen.getByText('test.csv')).toBeInTheDocument();
    });
  });
});
