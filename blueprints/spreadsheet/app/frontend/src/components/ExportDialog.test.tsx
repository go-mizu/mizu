import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ExportDialog } from './ExportDialog';
import type { ExportFormat } from './ExportDialog';

describe('ExportDialog', () => {
  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    onExport: vi.fn(),
    workbookName: 'Test Workbook',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render when open', () => {
    render(<ExportDialog {...defaultProps} />);

    expect(screen.getByText('Export Spreadsheet')).toBeInTheDocument();
    expect(screen.getByText('Export Format')).toBeInTheDocument();
  });

  it('should not render when closed', () => {
    render(<ExportDialog {...defaultProps} isOpen={false} />);

    expect(screen.queryByText('Export Spreadsheet')).not.toBeInTheDocument();
  });

  it('should call onClose when close button clicked', () => {
    render(<ExportDialog {...defaultProps} />);

    const closeButton = screen.getByText('×');
    fireEvent.click(closeButton);

    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it('should call onClose when Cancel button clicked', () => {
    render(<ExportDialog {...defaultProps} />);

    const cancelButton = screen.getByText('Cancel');
    fireEvent.click(cancelButton);

    expect(defaultProps.onClose).toHaveBeenCalledTimes(1);
  });

  it('should display all supported formats', () => {
    render(<ExportDialog {...defaultProps} />);

    expect(screen.getByText('Microsoft Excel')).toBeInTheDocument();
    expect(screen.getByText('CSV')).toBeInTheDocument();
    expect(screen.getByText('TSV')).toBeInTheDocument();
    expect(screen.getByText('JSON')).toBeInTheDocument();
    expect(screen.getByText('PDF')).toBeInTheDocument();
    expect(screen.getByText('HTML')).toBeInTheDocument();
  });

  it('should default to XLSX format', () => {
    render(<ExportDialog {...defaultProps} />);

    const xlsxRadio = screen.getByRole('radio', { name: /Microsoft Excel/i }) as HTMLInputElement;
    expect(xlsxRadio.checked).toBe(true);
  });

  it('should select format when clicked', () => {
    render(<ExportDialog {...defaultProps} />);

    const csvRadio = screen.getByRole('radio', { name: /^CSV/i }) as HTMLInputElement;
    fireEvent.click(csvRadio);

    expect(csvRadio.checked).toBe(true);
  });

  it('should use initial format when provided', () => {
    render(<ExportDialog {...defaultProps} initialFormat="csv" />);

    const csvRadio = screen.getByRole('radio', { name: /^CSV/i }) as HTMLInputElement;
    expect(csvRadio.checked).toBe(true);
  });

  it('should show format-specific options for XLSX', () => {
    render(<ExportDialog {...defaultProps} initialFormat="xlsx" />);

    expect(screen.getByText('Include cell formatting')).toBeInTheDocument();
    expect(screen.getByText('Export formulas (instead of values)')).toBeInTheDocument();
  });

  it('should show format-specific options for CSV', () => {
    render(<ExportDialog {...defaultProps} initialFormat="csv" />);

    expect(screen.getByText('Include column headers (A, B, C...)')).toBeInTheDocument();
  });

  it('should show format-specific options for PDF', () => {
    render(<ExportDialog {...defaultProps} initialFormat="pdf" />);

    expect(screen.getByText('Include gridlines')).toBeInTheDocument();
    expect(screen.getByText('Orientation:')).toBeInTheDocument();
    expect(screen.getByText('Paper Size:')).toBeInTheDocument();
  });

  it('should show format-specific options for JSON', () => {
    render(<ExportDialog {...defaultProps} initialFormat="json" />);

    expect(screen.getByText('Include workbook metadata')).toBeInTheDocument();
    expect(screen.getByText('Compact JSON (minified)')).toBeInTheDocument();
  });

  it('should display preview filename with correct extension', () => {
    render(<ExportDialog {...defaultProps} />);

    expect(screen.getByText('Test Workbook.xlsx')).toBeInTheDocument();

    const csvRadio = screen.getByRole('radio', { name: /^CSV/i });
    fireEvent.click(csvRadio);

    expect(screen.getByText('Test Workbook.csv')).toBeInTheDocument();
  });

  it('should call onExport with correct format and options', async () => {
    const onExport = vi.fn().mockResolvedValue(undefined);
    render(<ExportDialog {...defaultProps} onExport={onExport} initialFormat="xlsx" />);

    const exportButton = screen.getByText('Export');
    fireEvent.click(exportButton);

    await waitFor(() => {
      expect(onExport).toHaveBeenCalledWith(
        'xlsx',
        expect.objectContaining({
          formatting: true,
          formulas: false,
        })
      );
    });
  });

  it('should toggle options and export with correct values', async () => {
    const onExport = vi.fn().mockResolvedValue(undefined);
    render(<ExportDialog {...defaultProps} onExport={onExport} initialFormat="xlsx" />);

    // Toggle formulas option
    const formulasCheckbox = screen.getByRole('checkbox', { name: /Export formulas/i });
    fireEvent.click(formulasCheckbox);

    const exportButton = screen.getByText('Export');
    fireEvent.click(exportButton);

    await waitFor(() => {
      expect(onExport).toHaveBeenCalledWith(
        'xlsx',
        expect.objectContaining({
          formulas: true,
        })
      );
    });
  });

  it('should show loading state during export', async () => {
    const onExport = vi.fn().mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));
    render(<ExportDialog {...defaultProps} onExport={onExport} />);

    const exportButton = screen.getByText('Export');
    fireEvent.click(exportButton);

    await waitFor(() => {
      expect(screen.getByText('Exporting...')).toBeInTheDocument();
    });
  });

  it('should disable Export button while loading', async () => {
    const onExport = vi.fn().mockImplementation(() => new Promise(resolve => setTimeout(resolve, 100)));
    render(<ExportDialog {...defaultProps} onExport={onExport} />);

    const exportButton = screen.getByText('Export');
    fireEvent.click(exportButton);

    await waitFor(() => {
      const loadingButton = screen.getByText('Exporting...');
      expect(loadingButton).toBeDisabled();
    });
  });

  it('should show error message on export failure', async () => {
    const onExport = vi.fn().mockRejectedValue(new Error('Export failed'));
    render(<ExportDialog {...defaultProps} onExport={onExport} />);

    const exportButton = screen.getByText('Export');
    fireEvent.click(exportButton);

    await waitFor(() => {
      expect(screen.getByText('Export failed')).toBeInTheDocument();
    });
  });

  it('should close dialog on successful export', async () => {
    const onExport = vi.fn().mockResolvedValue(undefined);
    const onClose = vi.fn();
    render(<ExportDialog {...defaultProps} onExport={onExport} onClose={onClose} />);

    const exportButton = screen.getByText('Export');
    fireEvent.click(exportButton);

    await waitFor(() => {
      expect(onClose).toHaveBeenCalled();
    });
  });

  it('should prevent dialog close when clicking inside dialog', () => {
    render(<ExportDialog {...defaultProps} />);

    const dialog = document.querySelector('.export-dialog');
    if (dialog) {
      fireEvent.click(dialog);
      expect(defaultProps.onClose).not.toHaveBeenCalled();
    }
  });

  it('should change paper size for PDF', () => {
    render(<ExportDialog {...defaultProps} initialFormat="pdf" />);

    // Find the select by its containing label text
    const selects = document.querySelectorAll('select');
    const paperSizeSelect = Array.from(selects).find(s =>
      s.parentElement?.textContent?.includes('Paper Size')
    ) as HTMLSelectElement;

    expect(paperSizeSelect).toBeTruthy();
    fireEvent.change(paperSizeSelect, { target: { value: 'a4' } });

    expect(paperSizeSelect.value).toBe('a4');
  });

  it('should change orientation for PDF', () => {
    render(<ExportDialog {...defaultProps} initialFormat="pdf" />);

    // Find the select by its containing label text
    const selects = document.querySelectorAll('select');
    const orientationSelect = Array.from(selects).find(s =>
      s.parentElement?.textContent?.includes('Orientation')
    ) as HTMLSelectElement;

    expect(orientationSelect).toBeTruthy();
    fireEvent.change(orientationSelect, { target: { value: 'landscape' } });

    expect(orientationSelect.value).toBe('landscape');
  });

  it('should update options when format changes', async () => {
    const onExport = vi.fn().mockResolvedValue(undefined);
    render(<ExportDialog {...defaultProps} onExport={onExport} />);

    // Select JSON format
    const jsonRadio = screen.getByRole('radio', { name: /^JSON/i });
    fireEvent.click(jsonRadio);

    // Check metadata option
    const metadataCheckbox = screen.getByRole('checkbox', { name: /Include workbook metadata/i });
    fireEvent.click(metadataCheckbox);

    const exportButton = screen.getByText('Export');
    fireEvent.click(exportButton);

    await waitFor(() => {
      expect(onExport).toHaveBeenCalledWith(
        'json',
        expect.objectContaining({
          metadata: true,
        })
      );
    });
  });
});
