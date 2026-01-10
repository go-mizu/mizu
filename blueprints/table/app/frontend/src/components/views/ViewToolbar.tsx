import { useState, useRef, useEffect } from 'react';
import { useBaseStore } from '../../stores/baseStore';
import type { ViewType } from '../../types';
import { FilterBuilder } from '../common/FilterBuilder';
import { SortBuilder } from '../common/SortBuilder';
import { GroupBuilder } from '../common/GroupBuilder';

const VIEW_TYPES: { type: ViewType; label: string; icon: string }[] = [
  { type: 'grid', label: 'Grid', icon: 'M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z' },
  { type: 'kanban', label: 'Kanban', icon: 'M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2' },
  { type: 'calendar', label: 'Calendar', icon: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z' },
  { type: 'gallery', label: 'Gallery', icon: 'M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z' },
  { type: 'timeline', label: 'Timeline', icon: 'M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
  { type: 'form', label: 'Form', icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
  { type: 'list', label: 'List', icon: 'M4 6h16M4 10h16M4 14h16M4 18h16' },
];

export function ViewToolbar() {
  const { views, currentView, selectView, createView, filters, sorts, groupBy } = useBaseStore();
  const [showViewMenu, setShowViewMenu] = useState(false);
  const [showNewView, setShowNewView] = useState(false);
  const [showFilter, setShowFilter] = useState(false);
  const [showSort, setShowSort] = useState(false);
  const [showGroup, setShowGroup] = useState(false);
  const [newViewName, setNewViewName] = useState('');
  const [newViewType, setNewViewType] = useState<ViewType>('grid');

  const filterRef = useRef<HTMLDivElement>(null);
  const sortRef = useRef<HTMLDivElement>(null);
  const groupRef = useRef<HTMLDivElement>(null);

  // Close dropdowns on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (filterRef.current && !filterRef.current.contains(e.target as Node)) {
        setShowFilter(false);
      }
      if (sortRef.current && !sortRef.current.contains(e.target as Node)) {
        setShowSort(false);
      }
      if (groupRef.current && !groupRef.current.contains(e.target as Node)) {
        setShowGroup(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleCreateView = async () => {
    if (!newViewName.trim()) return;
    const view = await createView(newViewName.trim(), newViewType);
    setNewViewName('');
    setNewViewType('grid');
    setShowNewView(false);
    await selectView(view.id);
  };

  const currentViewType = VIEW_TYPES.find(v => v.type === currentView?.type);
  const viewLabel = currentView?.name || (views.length > 0 ? 'Select view' : 'Views');
  const hasFilters = filters.length > 0;
  const hasSorts = sorts.length > 0;
  const hasGroup = groupBy !== null;

  return (
    <div className="bg-white border-b border-gray-200 px-4 py-2 flex items-center gap-3">
      {/* View selector */}
      <div className="relative">
        <button
          onClick={() => setShowViewMenu(!showViewMenu)}
          data-testid="view-selector"
          className="flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-100 rounded-md"
        >
          {currentViewType && (
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={currentViewType.icon} />
            </svg>
          )}
          {viewLabel}
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {showViewMenu && (
          <>
            <div className="fixed inset-0 z-40" onClick={() => setShowViewMenu(false)} />
            <div className="dropdown-menu animate-slide-in mt-1">
              {views.length === 0 ? (
                <div className="px-3 py-2 text-sm text-gray-500">No views yet</div>
              ) : (
                views.map((view) => {
                  const viewType = VIEW_TYPES.find(v => v.type === view.type);
                  return (
                    <button
                      key={view.id}
                      onClick={() => {
                        selectView(view.id);
                        setShowViewMenu(false);
                      }}
                      className={`dropdown-item w-full text-left ${view.id === currentView?.id ? 'bg-primary-50 text-primary' : ''}`}
                    >
                      {viewType && (
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={viewType.icon} />
                        </svg>
                      )}
                      {view.name}
                    </button>
                  );
                })
              )}
              <hr className="my-1" />
              <button
                onClick={() => {
                  setShowViewMenu(false);
                  setShowNewView(true);
                }}
                className="dropdown-item w-full text-left text-primary"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Create view
              </button>
            </div>
          </>
        )}
      </div>

      <div className="flex-1" />

      {/* Filter button */}
      <div className="relative" ref={filterRef}>
        <button
          onClick={() => setShowFilter(!showFilter)}
          className={`px-3 py-1.5 text-sm rounded-md flex items-center gap-1 ${
            hasFilters
              ? 'bg-primary-50 text-primary-700'
              : 'text-gray-600 hover:bg-gray-100'
          }`}
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
          </svg>
          Filter
          {hasFilters && (
            <span className="ml-1 bg-primary text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
              {filters.length}
            </span>
          )}
        </button>

        {showFilter && (
          <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-xl border border-gray-200 z-50 animate-slide-in">
            <FilterBuilder onClose={() => setShowFilter(false)} />
          </div>
        )}
      </div>

      {/* Sort button */}
      <div className="relative" ref={sortRef}>
        <button
          onClick={() => setShowSort(!showSort)}
          className={`px-3 py-1.5 text-sm rounded-md flex items-center gap-1 ${
            hasSorts
              ? 'bg-primary-50 text-primary-700'
              : 'text-gray-600 hover:bg-gray-100'
          }`}
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
          </svg>
          Sort
          {hasSorts && (
            <span className="ml-1 bg-primary text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
              {sorts.length}
            </span>
          )}
        </button>

        {showSort && (
          <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-xl border border-gray-200 z-50 animate-slide-in">
            <SortBuilder onClose={() => setShowSort(false)} />
          </div>
        )}
      </div>

      {/* Group button */}
      <div className="relative" ref={groupRef}>
        <button
          onClick={() => setShowGroup(!showGroup)}
          className={`px-3 py-1.5 text-sm rounded-md flex items-center gap-1 ${
            hasGroup
              ? 'bg-primary-50 text-primary-700'
              : 'text-gray-600 hover:bg-gray-100'
          }`}
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          Group
          {hasGroup && (
            <span className="ml-1 w-2 h-2 rounded-full bg-primary" />
          )}
        </button>

        {showGroup && (
          <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-xl border border-gray-200 z-50 animate-slide-in">
            <GroupBuilder onClose={() => setShowGroup(false)} />
          </div>
        )}
      </div>

      {/* New view modal */}
      {showNewView && (
        <div className="modal-overlay" onClick={() => setShowNewView(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="text-lg font-semibold">Create view</h3>
              <button onClick={() => setShowNewView(false)} className="text-gray-400 hover:text-gray-600">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="modal-body space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input
                  type="text"
                  value={newViewName}
                  onChange={(e) => setNewViewName(e.target.value)}
                  className="input"
                  placeholder="View name"
                  autoFocus
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">Type</label>
                <div className="grid grid-cols-3 gap-2">
                  {VIEW_TYPES.map((viewType) => (
                    <button
                      key={viewType.type}
                      onClick={() => setNewViewType(viewType.type)}
                      className={`p-3 rounded-md border-2 text-center ${
                        newViewType === viewType.type
                          ? 'border-primary bg-primary-50'
                          : 'border-gray-200 hover:border-gray-300'
                      }`}
                    >
                      <svg className="w-6 h-6 mx-auto mb-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={viewType.icon} />
                      </svg>
                      <span className="text-xs">{viewType.label}</span>
                    </button>
                  ))}
                </div>
              </div>
            </div>
            <div className="modal-footer">
              <button onClick={() => setShowNewView(false)} className="btn btn-secondary">Cancel</button>
              <button onClick={handleCreateView} className="btn btn-primary">Create view</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
