import { useState, useRef, useEffect } from 'react';
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
  const [showWorkspaceDropdown, setShowWorkspaceDropdown] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setShowWorkspaceDropdown(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleCreateBase = async () => {
    if (!newBaseName.trim()) return;
    await createBase(newBaseName.trim());
    setNewBaseName('');
    setShowNewBase(false);
  };

  // Filter bases by search query
  const filteredBases = bases.filter(base =>
    base.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  if (!isOpen) {
    return (
      <button
        onClick={onToggle}
        className="fixed left-4 top-4 z-50 p-2.5 bg-white rounded-lg hover:bg-[var(--at-surface-hover)] border border-[var(--at-border)] transition-all duration-150 group"
        style={{ boxShadow: 'var(--shadow-md)' }}
      >
        <svg className="w-5 h-5 text-[var(--at-text-secondary)] group-hover:text-[var(--at-text)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
    );
  }

  return (
    <div className="fixed left-0 top-0 bottom-0 w-[260px] bg-[#fbfbfb] border-r border-[var(--at-border)] flex flex-col z-40 animate-slide-in-right">
      {/* Header */}
      <div className="px-4 py-3.5 border-b border-[var(--at-border)] flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <div className="w-8 h-8 bg-primary rounded-lg flex items-center justify-center">
            <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
            </svg>
          </div>
          <span className="text-[15px] font-semibold text-[var(--at-text)]">Table</span>
        </div>
        <button
          onClick={onToggle}
          className="p-1.5 hover:bg-[var(--at-surface-hover)] rounded-md transition-colors"
        >
          <svg className="w-5 h-5 text-[var(--at-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 19l-7-7 7-7m8 14l-7-7 7-7" />
          </svg>
        </button>
      </div>

      {/* Workspace selector - Custom dropdown */}
      <div className="p-3 border-b border-[var(--at-border)]" ref={dropdownRef}>
        <div className="sidebar-section-title">
          <span>Workspace</span>
        </div>
        <button
          onClick={() => setShowWorkspaceDropdown(!showWorkspaceDropdown)}
          className="w-full flex items-center justify-between px-3 py-2.5 bg-white border border-[var(--at-border-strong)] rounded-lg hover:border-[var(--at-primary)] transition-colors group"
          style={{ boxShadow: 'var(--shadow-xs)' }}
        >
          <div className="flex items-center gap-2.5">
            <div className="w-6 h-6 rounded-md bg-gradient-to-br from-purple-500 to-purple-600 flex items-center justify-center">
              <span className="text-white text-xs font-semibold">
                {currentWorkspace?.name?.charAt(0).toUpperCase()}
              </span>
            </div>
            <span className="text-sm font-medium text-[var(--at-text)] truncate">
              {currentWorkspace?.name || 'Select workspace'}
            </span>
          </div>
          <svg
            className={`w-4 h-4 text-[var(--at-muted)] transition-transform ${showWorkspaceDropdown ? 'rotate-180' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {showWorkspaceDropdown && (
          <div className="dropdown-menu mt-1.5 w-full left-0 right-0" style={{ position: 'relative' }}>
            {workspaces.map((ws) => (
              <button
                key={ws.id}
                onClick={() => {
                  selectWorkspace(ws.id);
                  setShowWorkspaceDropdown(false);
                }}
                className={`dropdown-item w-full ${currentWorkspace?.id === ws.id ? 'bg-[var(--at-primary-soft)]' : ''}`}
              >
                <div className="w-6 h-6 rounded-md bg-gradient-to-br from-purple-500 to-purple-600 flex items-center justify-center flex-shrink-0">
                  <span className="text-white text-xs font-semibold">
                    {ws.name.charAt(0).toUpperCase()}
                  </span>
                </div>
                <span className="truncate">{ws.name}</span>
                {currentWorkspace?.id === ws.id && (
                  <svg className="w-4 h-4 text-primary ml-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                )}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Search */}
      <div className="px-3 pt-3">
        <div className="relative">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search bases..."
            className="w-full pl-9 pr-3 py-2 text-sm bg-white border border-[var(--at-border)] rounded-lg focus:outline-none focus:border-[var(--at-primary)] focus:ring-2 focus:ring-[var(--at-primary-muted)] transition-all"
          />
          <svg className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--at-muted)]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
      </div>

      {/* Bases list */}
      <div className="flex-1 overflow-y-auto px-3 pt-4 pb-3">
        <div className="flex items-center justify-between mb-3 px-1">
          <span className="text-[11px] font-semibold text-[var(--at-muted)] uppercase tracking-wider">
            Bases
          </span>
          <button
            onClick={() => setShowNewBase(true)}
            className="flex items-center gap-1 text-xs font-medium text-primary hover:text-primary-600 transition-colors"
          >
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            New
          </button>
        </div>

        {showNewBase && (
          <div className="mb-3 p-3 bg-white rounded-lg border border-[var(--at-border)] animate-scale-in" style={{ boxShadow: 'var(--shadow-md)' }}>
            <input
              type="text"
              value={newBaseName}
              onChange={(e) => setNewBaseName(e.target.value)}
              placeholder="Enter base name..."
              className="input input-sm mb-3"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreateBase();
                if (e.key === 'Escape') setShowNewBase(false);
              }}
            />
            <div className="flex gap-2">
              <button onClick={handleCreateBase} className="btn btn-primary btn-sm flex-1">
                Create base
              </button>
              <button onClick={() => setShowNewBase(false)} className="btn btn-secondary btn-sm">
                Cancel
              </button>
            </div>
          </div>
        )}

        <div className="space-y-1">
          {filteredBases.map((base) => (
            <button
              key={base.id}
              onClick={() => selectBase(base.id)}
              className={`sidebar-item w-full text-left group ${
                currentBase?.id === base.id ? 'sidebar-item-active' : ''
              }`}
            >
              <div
                className="w-5 h-5 rounded-md flex-shrink-0 flex items-center justify-center shadow-sm"
                style={{ backgroundColor: base.color || '#166ee1' }}
              >
                <svg className="w-3 h-3 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6z" />
                </svg>
              </div>
              <span className="flex-1 truncate">{base.name}</span>
              <svg
                className={`w-4 h-4 text-[var(--at-muted)] opacity-0 group-hover:opacity-100 transition-opacity ${
                  currentBase?.id === base.id ? 'opacity-100' : ''
                }`}
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </button>
          ))}
        </div>

        {filteredBases.length === 0 && !showNewBase && (
          <div className="empty-state py-8">
            <svg className="empty-state-icon w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
            <p className="empty-state-title text-sm">
              {searchQuery ? 'No bases found' : 'No bases yet'}
            </p>
            <p className="empty-state-description text-xs">
              {searchQuery
                ? `No bases matching "${searchQuery}"`
                : 'Create your first base to get started'}
            </p>
            {!searchQuery && (
              <button
                onClick={() => setShowNewBase(true)}
                className="btn btn-primary btn-sm mt-2"
              >
                Create base
              </button>
            )}
          </div>
        )}
      </div>

      {/* User section */}
      <div className="sidebar-user">
        <div className="sidebar-avatar">
          {user?.name?.charAt(0).toUpperCase()}
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-[var(--at-text)] truncate">{user?.name}</p>
          <p className="text-xs text-[var(--at-muted)] truncate">{user?.email}</p>
        </div>
        <button
          onClick={logout}
          className="p-2 hover:bg-[var(--at-surface-hover)] rounded-md text-[var(--at-muted)] hover:text-[var(--at-danger)] transition-colors"
          title="Sign out"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
          </svg>
        </button>
      </div>
    </div>
  );
}
