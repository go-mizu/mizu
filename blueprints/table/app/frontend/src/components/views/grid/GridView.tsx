import { useState, useRef, useEffect } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { TableRecord, CellValue } from '../../../types';
import { CellEditor } from './CellEditor';
import { FieldHeader } from './FieldHeader';
import { AddFieldButton } from './AddFieldButton';
import { RecordModal } from '../RecordModal';

export function GridView() {
  const {
    currentTable,
    fields,
    createRecord,
    deleteRecord,
    updateCellValue,
    getSortedRecords,
  } = useBaseStore();

  // Get filtered and sorted records
  const displayRecords = getSortedRecords();

  const [selectedCell, setSelectedCell] = useState<{ recordId: string; fieldId: string } | null>(null);
  const [editingCell, setEditingCell] = useState<{ recordId: string; fieldId: string } | null>(null);
  const [selectedRows, setSelectedRows] = useState<Set<string>>(new Set());
  const [expandedRecord, setExpandedRecord] = useState<TableRecord | null>(null);
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; recordId: string } | null>(null);
  const gridRef = useRef<HTMLDivElement>(null);

  // Handle bulk delete of selected rows
  const handleBulkDelete = async () => {
    if (selectedRows.size === 0) return;
    const count = selectedRows.size;
    if (!window.confirm(`Delete ${count} record${count > 1 ? 's' : ''}? This cannot be undone.`)) return;

    for (const recordId of selectedRows) {
      await deleteRecord(recordId);
    }
    setSelectedRows(new Set());
  };

  // Handle keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Handle bulk delete with Delete/Backspace when rows are selected
      if ((e.key === 'Delete' || e.key === 'Backspace') && selectedRows.size > 0 && !editingCell && !selectedCell) {
        e.preventDefault();
        handleBulkDelete();
        return;
      }

      // Handle Cmd/Ctrl+A for select all
      if ((e.metaKey || e.ctrlKey) && e.key === 'a' && !editingCell) {
        e.preventDefault();
        setSelectedRows(new Set(displayRecords.map(r => r.id)));
        return;
      }

      if (!selectedCell || editingCell) return;

      const currentRecordIndex = displayRecords.findIndex(r => r.id === selectedCell.recordId);
      const currentFieldIndex = fields.findIndex(f => f.id === selectedCell.fieldId);

      switch (e.key) {
        case 'ArrowUp':
          e.preventDefault();
          if (currentRecordIndex > 0) {
            setSelectedCell({ recordId: displayRecords[currentRecordIndex - 1].id, fieldId: selectedCell.fieldId });
          }
          break;
        case 'ArrowDown':
          e.preventDefault();
          if (currentRecordIndex < displayRecords.length - 1) {
            setSelectedCell({ recordId: displayRecords[currentRecordIndex + 1].id, fieldId: selectedCell.fieldId });
          }
          break;
        case 'ArrowLeft':
          e.preventDefault();
          if (currentFieldIndex > 0) {
            setSelectedCell({ recordId: selectedCell.recordId, fieldId: fields[currentFieldIndex - 1].id });
          }
          break;
        case 'ArrowRight':
          e.preventDefault();
          if (currentFieldIndex < fields.length - 1) {
            setSelectedCell({ recordId: selectedCell.recordId, fieldId: fields[currentFieldIndex + 1].id });
          }
          break;
        case 'Tab':
          e.preventDefault();
          if (e.shiftKey) {
            // Move to previous cell
            if (currentFieldIndex > 0) {
              setSelectedCell({ recordId: selectedCell.recordId, fieldId: fields[currentFieldIndex - 1].id });
            } else if (currentRecordIndex > 0) {
              // Go to last field of previous row
              setSelectedCell({ recordId: displayRecords[currentRecordIndex - 1].id, fieldId: fields[fields.length - 1].id });
            }
          } else {
            // Move to next cell
            if (currentFieldIndex < fields.length - 1) {
              setSelectedCell({ recordId: selectedCell.recordId, fieldId: fields[currentFieldIndex + 1].id });
            } else if (currentRecordIndex < displayRecords.length - 1) {
              // Go to first field of next row
              setSelectedCell({ recordId: displayRecords[currentRecordIndex + 1].id, fieldId: fields[0].id });
            }
          }
          break;
        case 'Enter':
          e.preventDefault();
          setEditingCell(selectedCell);
          break;
        case 'Escape':
          e.preventDefault();
          setSelectedCell(null);
          setSelectedRows(new Set());
          break;
        case 'Delete':
        case 'Backspace':
          if (!editingCell) {
            e.preventDefault();
            updateCellValue(selectedCell.recordId, selectedCell.fieldId, null);
          }
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedCell, editingCell, displayRecords, fields, updateCellValue, selectedRows]);

  const handleCellClick = (recordId: string, fieldId: string) => {
    setSelectedCell({ recordId, fieldId });
  };

  const handleCellDoubleClick = (recordId: string, fieldId: string) => {
    setEditingCell({ recordId, fieldId });
  };

  const handleCellChange = async (recordId: string, fieldId: string, value: CellValue) => {
    await updateCellValue(recordId, fieldId, value);
    setEditingCell(null);
  };

  const handleAddRow = async () => {
    await createRecord({});
  };

  const handleRowContextMenu = (e: React.MouseEvent, recordId: string) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, recordId });
  };

  const handleDeleteRow = async () => {
    if (contextMenu) {
      await deleteRecord(contextMenu.recordId);
      setContextMenu(null);
    }
  };

  const handleExpandRecord = (record: TableRecord) => {
    setExpandedRecord(record);
  };

  const getCellValue = (record: TableRecord, fieldId: string): CellValue => {
    return record.values[fieldId] ?? null;
  };

  const toggleRowSelection = (recordId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const newSelection = new Set(selectedRows);
    if (e.shiftKey && selectedRows.size > 0) {
      // Range selection
      const lastSelected = Array.from(selectedRows).pop()!;
      const lastIndex = displayRecords.findIndex(r => r.id === lastSelected);
      const currentIndex = displayRecords.findIndex(r => r.id === recordId);
      const start = Math.min(lastIndex, currentIndex);
      const end = Math.max(lastIndex, currentIndex);
      for (let i = start; i <= end; i++) {
        newSelection.add(displayRecords[i].id);
      }
    } else if (e.metaKey || e.ctrlKey) {
      // Toggle selection
      if (newSelection.has(recordId)) {
        newSelection.delete(recordId);
      } else {
        newSelection.add(recordId);
      }
    } else {
      // Single selection
      newSelection.clear();
      newSelection.add(recordId);
    }
    setSelectedRows(newSelection);
  };

  const toggleAllRows = () => {
    if (selectedRows.size === displayRecords.length) {
      setSelectedRows(new Set());
    } else {
      setSelectedRows(new Set(displayRecords.map(r => r.id)));
    }
  };

  if (!currentTable) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-500">
        Select a table to view records
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto flex flex-col" ref={gridRef}>
      {/* Selection bar */}
      {selectedRows.size > 0 && (
        <div className="flex items-center gap-3 px-4 py-2 bg-primary-50 border-b border-primary-200">
          <span className="text-sm font-medium text-primary-700">
            {selectedRows.size} row{selectedRows.size > 1 ? 's' : ''} selected
          </span>
          <button
            onClick={() => setSelectedRows(new Set())}
            className="text-sm text-primary-600 hover:text-primary-800"
          >
            Clear selection
          </button>
          <button
            onClick={handleBulkDelete}
            className="text-sm text-red-600 hover:text-red-800 flex items-center gap-1"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            Delete selected
          </button>
          <span className="text-xs text-gray-500 ml-auto">
            Press Delete or Backspace to delete • Escape to clear
          </span>
        </div>
      )}
      <div className="flex-1 overflow-auto">
      <table className="w-full border-collapse">
        <thead className="sticky top-0 z-10 bg-slate-50 shadow-sm">
          <tr>
            {/* Row number and checkbox column */}
            <th className="w-16 border-b border-r border-slate-200 bg-slate-50 p-0">
              <div className="flex items-center justify-center h-9">
                <input
                  type="checkbox"
                  checked={selectedRows.size === displayRecords.length && displayRecords.length > 0}
                  onChange={toggleAllRows}
                  className="w-4 h-4 rounded border-gray-300"
                />
              </div>
            </th>

            {/* Field headers */}
            {fields.map((field) => (
              <FieldHeader key={field.id} field={field} />
            ))}

            {/* Add field button */}
            <th className="w-32 border-b border-slate-200 bg-slate-50 p-0">
              <AddFieldButton />
            </th>
          </tr>
        </thead>
        <tbody>
          {displayRecords.map((record, rowIndex) => (
            <tr
              key={record.id}
              className={`group ${selectedRows.has(record.id) ? 'bg-primary-50' : 'hover:bg-slate-50'}`}
              onContextMenu={(e) => handleRowContextMenu(e, record.id)}
            >
              {/* Row number and checkbox */}
              <td className="border-b border-r border-slate-200 p-0">
                <div className="flex items-center justify-center h-9 gap-1">
                  <input
                    type="checkbox"
                    checked={selectedRows.has(record.id)}
                    onChange={(e) => toggleRowSelection(record.id, e as any)}
                    className="w-4 h-4 rounded border-gray-300 opacity-0 group-hover:opacity-100 checked:opacity-100"
                  />
                  <span className="text-xs text-gray-400 w-6 text-center group-hover:hidden">
                    {rowIndex + 1}
                  </span>
                  <button
                    onClick={() => handleExpandRecord(record)}
                    className="hidden group-hover:block text-gray-400 hover:text-gray-600"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                    </svg>
                  </button>
                </div>
              </td>

              {/* Cells */}
              {fields.map((field) => {
                const isSelected = selectedCell?.recordId === record.id && selectedCell?.fieldId === field.id;
                const isEditing = editingCell?.recordId === record.id && editingCell?.fieldId === field.id;
                const value = getCellValue(record, field.id);

                return (
                  <td
                    key={field.id}
                    className={`border-b border-r border-slate-200 p-0 ${
                      isSelected ? 'ring-2 ring-primary ring-inset' : ''
                    }`}
                    onClick={() => handleCellClick(record.id, field.id)}
                    onDoubleClick={() => handleCellDoubleClick(record.id, field.id)}
                  >
                    <CellEditor
                      field={field}
                      value={value}
                      isEditing={isEditing}
                      onChange={(newValue) => handleCellChange(record.id, field.id, newValue)}
                      onCancel={() => setEditingCell(null)}
                    />
                  </td>
                );
              })}

              {/* Empty cell for add field column */}
              <td className="border-b border-slate-200" />
            </tr>
          ))}

          {/* Add row button */}
          <tr>
            <td colSpan={fields.length + 2} className="border-b border-slate-200 p-0">
              <button
                onClick={handleAddRow}
                className="w-full h-9 text-left px-4 text-sm text-slate-500 hover:bg-slate-50 flex items-center gap-2"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add row
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      {/* Context menu */}
      {contextMenu && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setContextMenu(null)} />
          <div
            className="dropdown-menu animate-slide-in"
            style={{ left: contextMenu.x, top: contextMenu.y }}
          >
            <button
              onClick={() => {
                const record = displayRecords.find(r => r.id === contextMenu.recordId);
                if (record) handleExpandRecord(record);
                setContextMenu(null);
              }}
              className="dropdown-item w-full text-left"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
              </svg>
              Expand record
            </button>
            <button
              onClick={handleDeleteRow}
              className="dropdown-item dropdown-item-danger w-full text-left"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              Delete row
            </button>
          </div>
        </>
      )}

      {/* Record modal */}
      {expandedRecord && (
        <RecordModal
          record={expandedRecord}
          onClose={() => setExpandedRecord(null)}
        />
      )}
      </div>
    </div>
  );
}
