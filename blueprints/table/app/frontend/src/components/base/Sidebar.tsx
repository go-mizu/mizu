import { useState } from 'react';
import { useAuthStore } from '../../stores/authStore';
import { useBaseStore } from '../../stores/baseStore';

interface SidebarProps {
  isOpen: boolean;
  onToggle: () => void;
}

export function Sidebar({ isOpen, onToggle }: SidebarProps) {
  const { user, logout } = useAuthStore();
  const { workspaces, currentWorkspace, bases, currentBase, selectWorkspace, selectBase, createBase } = useBaseStore();
  const [showNewBase, setShowNewBase] = useState(false);
  const [newBaseName, setNewBaseName] = useState('');

  const handleCreateBase = async () => {
    if (!newBaseName.trim()) return;
    await createBase(newBaseName.trim());
    setNewBaseName('');
    setShowNewBase(false);
  };

  if (!isOpen) {
    return (
      <button
        onClick={onToggle}
        className="fixed left-4 top-4 z-50 p-2 bg-white rounded-lg shadow-md hover:bg-gray-50 border border-gray-200"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
    );
  }

  return (
    <div className="fixed left-0 top-0 bottom-0 w-64 bg-[#f8fafc] border-r border-slate-200 flex flex-col z-40">
      {/* Header */}
      <div className="p-4 border-b border-slate-200 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="w-2.5 h-2.5 rounded-full bg-primary" />
          <h1 className="text-lg font-semibold text-gray-900">Table</h1>
        </div>
        <button onClick={onToggle} className="p-1.5 hover:bg-white rounded-md">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Workspace selector */}
      <div className="p-3 border-b border-slate-200">
        <div className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-2">
          Workspace
        </div>
        <select
          value={currentWorkspace?.id || ''}
          onChange={(e) => selectWorkspace(e.target.value)}
          className="w-full px-3 py-2 text-sm border border-slate-200 rounded-lg bg-white shadow-sm focus:outline-none focus:ring-2 focus:ring-primary"
        >
          {workspaces.map((ws) => (
            <option key={ws.id} value={ws.id}>{ws.name}</option>
          ))}
        </select>
      </div>

      {/* Bases list */}
      <div className="flex-1 overflow-y-auto p-3">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Bases</span>
          <button
            onClick={() => setShowNewBase(true)}
            className="text-xs font-medium text-primary hover:text-primary-600"
          >
            + New base
          </button>
        </div>

        {showNewBase && (
          <div className="mb-3 p-3 bg-white rounded-lg border border-slate-200 shadow-sm">
            <input
              type="text"
              value={newBaseName}
              onChange={(e) => setNewBaseName(e.target.value)}
              placeholder="Base name"
              className="w-full px-3 py-2 text-sm border border-slate-200 rounded-md mb-2"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreateBase();
                if (e.key === 'Escape') setShowNewBase(false);
              }}
            />
            <div className="flex gap-2">
              <button onClick={handleCreateBase} className="btn btn-primary btn-sm">Create</button>
              <button onClick={() => setShowNewBase(false)} className="btn btn-secondary btn-sm">Cancel</button>
            </div>
          </div>
        )}

        <div className="space-y-1">
          {bases.map((base) => (
            <button
              key={base.id}
              onClick={() => selectBase(base.id)}
              className={`w-full text-left px-3 py-2 rounded-lg text-sm flex items-center gap-2 transition-colors ${
                currentBase?.id === base.id ? 'bg-primary-50 text-primary-700 border border-primary-100' : 'hover:bg-white'
              }`}
            >
              <span
                className="w-4 h-4 rounded border border-white shadow-sm"
                style={{ backgroundColor: base.color }}
              />
              {base.name}
            </button>
          ))}
        </div>

        {bases.length === 0 && !showNewBase && (
          <p className="text-sm text-gray-500 text-center py-4">
            No bases yet. Create one to get started!
          </p>
        )}
      </div>

      {/* User section */}
      <div className="p-3 border-t border-slate-200 bg-white/80">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center text-primary font-semibold">
            {user?.name?.charAt(0).toUpperCase()}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-gray-900 truncate">{user?.name}</p>
            <p className="text-xs text-gray-500 truncate">{user?.email}</p>
          </div>
          <button
            onClick={logout}
            className="p-2 hover:bg-gray-100 rounded-md text-gray-500 hover:text-gray-700"
            title="Sign out"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
