import { useState } from 'react';
import { useBaseStore } from '../../stores/baseStore';

export function TableTabs() {
  const { tables, currentTable, selectTable, createTable, deleteTable } = useBaseStore();
  const [showNewTable, setShowNewTable] = useState(false);
  const [newTableName, setNewTableName] = useState('');
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; tableId: string } | null>(null);

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
    <div className="bg-white border-b border-gray-200 px-4">
      <div className="flex items-center gap-1 overflow-x-auto">
        {tables.map((table) => (
          <button
            key={table.id}
            onClick={() => selectTable(table.id)}
            onContextMenu={(e) => handleContextMenu(e, table.id)}
            className={`px-4 py-2 text-sm font-medium border-b-2 whitespace-nowrap transition-colors ${
              currentTable?.id === table.id
                ? 'border-primary text-primary'
                : 'border-transparent text-gray-600 hover:text-gray-900 hover:border-gray-300'
            }`}
          >
            {table.icon && <span className="mr-2">{table.icon}</span>}
            {table.name}
          </button>
        ))}

        {showNewTable ? (
          <div className="flex items-center gap-2 px-2">
            <input
              type="text"
              value={newTableName}
              onChange={(e) => setNewTableName(e.target.value)}
              placeholder="Table name"
              className="px-2 py-1 text-sm border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-primary"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreateTable();
                if (e.key === 'Escape') setShowNewTable(false);
              }}
            />
            <button onClick={handleCreateTable} className="text-primary hover:underline text-sm">Create</button>
            <button onClick={() => setShowNewTable(false)} className="text-gray-500 hover:text-gray-700 text-sm">Cancel</button>
          </div>
        ) : (
          <button
            onClick={() => setShowNewTable(true)}
            className="px-4 py-2 text-sm text-gray-600 hover:text-gray-900 flex items-center gap-1"
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
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={() => setContextMenu(null)}
          />
          <div
            className="dropdown-menu animate-slide-in"
            style={{ left: contextMenu.x, top: contextMenu.y }}
          >
            <button onClick={handleDeleteTable} className="dropdown-item dropdown-item-danger w-full text-left">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              Delete table
            </button>
          </div>
        </>
      )}
    </div>
  );
}
