import { useState, useRef, useEffect } from 'react';
import { useBaseStore } from '../../stores/baseStore';

export function TableTabs() {
  const { tables, currentTable, selectTable, createTable, deleteTable } = useBaseStore();
  const [showNewTable, setShowNewTable] = useState(false);
  const [newTableName, setNewTableName] = useState('');
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; tableId: string } | null>(null);
  const contextMenuRef = useRef<HTMLDivElement>(null);

  // Close context menu on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
        setContextMenu(null);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleCreateTable = async () => {
    if (!newTableName.trim()) return;
    const table = await createTable(newTableName.trim());
    setNewTableName('');
    setShowNewTable(false);
    await selectTable(table.id);
  };

  const handleContextMenu = (e: React.MouseEvent, tableId: string) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, tableId });
  };

  const handleDeleteTable = async () => {
    if (contextMenu) {
      await deleteTable(contextMenu.tableId);
      setContextMenu(null);
    }
  };

  return (
    <div className="bg-white border-b border-[var(--at-border)]">
      <div className="flex items-center gap-0.5 px-4 overflow-x-auto scrollbar-hide">
        {tables.map((table) => (
          <button
            key={table.id}
            onClick={() => selectTable(table.id)}
            onContextMenu={(e) => handleContextMenu(e, table.id)}
            className={`group relative px-4 py-2.5 text-sm font-medium whitespace-nowrap transition-all rounded-t-lg ${
              currentTable?.id === table.id
                ? 'text-primary'
                : 'text-[var(--at-text-secondary)] hover:text-[var(--at-text)] hover:bg-[var(--at-surface-hover)]'
            }`}
          >
            {/* Tab content */}
            <span className="flex items-center gap-2">
              {table.icon && <span>{table.icon}</span>}
              {table.name}
            </span>

            {/* Active indicator - thicker bottom border */}
            {currentTable?.id === table.id && (
              <span className="absolute bottom-0 left-2 right-2 h-[3px] bg-primary rounded-t-full" />
            )}

            {/* Hover indicator */}
            {currentTable?.id !== table.id && (
              <span className="absolute bottom-0 left-2 right-2 h-[2px] bg-transparent group-hover:bg-[var(--at-border-strong)] rounded-t-full transition-colors" />
            )}
          </button>
        ))}

        {/* New table input/button */}
        {showNewTable ? (
          <div className="flex items-center gap-2 px-3 py-1 animate-scale-in">
            <div className="relative">
              <input
                type="text"
                value={newTableName}
                onChange={(e) => setNewTableName(e.target.value)}
                placeholder="Table name"
                className="input input-sm w-36"
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleCreateTable();
                  if (e.key === 'Escape') {
                    setShowNewTable(false);
                    setNewTableName('');
                  }
                }}
              />
            </div>
            <button
              onClick={handleCreateTable}
              disabled={!newTableName.trim()}
              className="btn btn-primary btn-sm"
            >
              Create
            </button>
            <button
              onClick={() => {
                setShowNewTable(false);
                setNewTableName('');
              }}
              className="btn btn-ghost btn-sm"
            >
              Cancel
            </button>
          </div>
        ) : (
          <button
            onClick={() => setShowNewTable(true)}
            className="flex items-center gap-1.5 px-3 py-2 text-sm font-medium text-[var(--at-muted)] hover:text-primary hover:bg-[var(--at-surface-hover)] rounded-lg transition-colors ml-1"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add table
          </button>
        )}
      </div>

      {/* Context menu */}
      {contextMenu && (
        <div
          ref={contextMenuRef}
          className="dropdown-menu fixed z-50"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <button
            onClick={handleDeleteTable}
            className="dropdown-item dropdown-item-danger w-full text-left"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            Delete table
          </button>
          <button
            onClick={() => setContextMenu(null)}
            className="dropdown-item w-full text-left"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            Duplicate table
          </button>
          <div className="dropdown-divider" />
          <button
            onClick={() => setContextMenu(null)}
            className="dropdown-item w-full text-left"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
            </svg>
            Rename table
          </button>
        </div>
      )}
    </div>
  );
}
