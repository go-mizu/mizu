import { useState, useRef, useEffect, useMemo } from 'react';
import { useBaseStore } from '../../stores/baseStore';
import type { ViewType } from '../../types';
import { FilterBuilder } from '../common/FilterBuilder';
import { SortBuilder } from '../common/SortBuilder';
import { GroupBuilder } from '../common/GroupBuilder';
import { RowColorConfig } from './grid/RowColorConfig';
import { normalizeFieldConfig } from './grid/fieldConfig';
import { exportToCSV, exportToTSV } from '../../utils/exportCsv';

const ROW_HEIGHT_OPTIONS = [
  { value: 'short', label: 'Short', height: 36 },
  { value: 'medium', label: 'Medium', height: 56 },
  { value: 'tall', label: 'Tall', height: 96 },
  { value: 'extra_tall', label: 'Extra Tall', height: 144 },
] as const;

const VIEW_TYPES: { type: ViewType; label: string; icon: string }[] = [
  { type: 'grid', label: 'Grid', icon: 'M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z' },
  { type: 'kanban', label: 'Kanban', icon: 'M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2' },
  { type: 'calendar', label: 'Calendar', icon: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z' },
  { type: 'gallery', label: 'Gallery', icon: 'M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z' },
  { type: 'timeline', label: 'Timeline', icon: 'M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
  { type: 'form', label: 'Form', icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' },
  { type: 'list', label: 'List', icon: 'M4 6h16M4 10h16M4 14h16M4 18h16' },
  { type: 'dashboard', label: 'Dashboard', icon: 'M4 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM14 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zM4 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1v-4zM14 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z' },
];

export function ViewToolbar() {
  const { views, currentView, selectView, createView, filters, sorts, groupBy, fields, updateViewFieldConfig, updateViewConfig, getSortedRecords, currentTable } = useBaseStore();
  const [showViewMenu, setShowViewMenu] = useState(false);
  const [showNewView, setShowNewView] = useState(false);
  const [showFilter, setShowFilter] = useState(false);
  const [showSort, setShowSort] = useState(false);
  const [showGroup, setShowGroup] = useState(false);
  const [showColor, setShowColor] = useState(false);
  const [showFields, setShowFields] = useState(false);
  const [showRowHeight, setShowRowHeight] = useState(false);
  const [showExport, setShowExport] = useState(false);
  const [newViewName, setNewViewName] = useState('');
  const [newViewType, setNewViewType] = useState<ViewType>('grid');

  const filterRef = useRef<HTMLDivElement>(null);
  const sortRef = useRef<HTMLDivElement>(null);
  const groupRef = useRef<HTMLDivElement>(null);
  const colorRef = useRef<HTMLDivElement>(null);
  const fieldsRef = useRef<HTMLDivElement>(null);
  const rowHeightRef = useRef<HTMLDivElement>(null);
  const exportRef = useRef<HTMLDivElement>(null);

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
      if (colorRef.current && !colorRef.current.contains(e.target as Node)) {
        setShowColor(false);
      }
      if (fieldsRef.current && !fieldsRef.current.contains(e.target as Node)) {
        setShowFields(false);
      }
      if (rowHeightRef.current && !rowHeightRef.current.contains(e.target as Node)) {
        setShowRowHeight(false);
      }
      if (exportRef.current && !exportRef.current.contains(e.target as Node)) {
        setShowExport(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Get current row height from view config
  const currentRowHeight = useMemo(() => {
    if (!currentView?.config) return 'short';
    const config = typeof currentView.config === 'string'
      ? JSON.parse(currentView.config)
      : currentView.config;
    return config.row_height || 'short';
  }, [currentView?.config]);

  // Handle row height change
  const handleRowHeightChange = async (height: string) => {
    await updateViewConfig({ row_height: height });
    setShowRowHeight(false);
  };

  // Handle export
  const handleExport = async (format: 'csv' | 'tsv') => {
    const records = getSortedRecords();
    const visibleFieldsList = fieldConfig
      .filter(c => c.visible)
      .map(c => fields.find(f => f.id === c.field_id))
      .filter((f): f is NonNullable<typeof f> => f !== undefined);

    const filename = `${currentTable?.name || 'export'}_${new Date().toISOString().split('T')[0]}`;

    if (format === 'csv') {
      exportToCSV(records, visibleFieldsList, { filename: `${filename}.csv` });
    } else {
      exportToTSV(records, visibleFieldsList, { filename: `${filename}.tsv` });
    }
    setShowExport(false);
  };

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
  const showFieldButton = currentView?.type === 'grid';

  // Check if row coloring is active
  const currentRowColorFieldId = useMemo(() => {
    if (!currentView?.config) return null;
    const config = typeof currentView.config === 'string'
      ? JSON.parse(currentView.config)
      : currentView.config;
    return config.row_color_field_id || null;
  }, [currentView?.config]);
  const hasRowColor = currentRowColorFieldId !== null;
  const fieldConfig = useMemo(() => normalizeFieldConfig(fields, currentView?.field_config), [fields, currentView?.field_config]);
  const hiddenCount = fieldConfig.filter((config) => !config.visible).length;

  const toggleFieldVisibility = (fieldId: string) => {
    const nextConfig = fieldConfig.map((config) => {
      if (config.field_id !== fieldId) return config;
      return { ...config, visible: !config.visible };
    });
    updateViewFieldConfig(nextConfig);
  };

  const showAllFields = () => {
    const nextConfig = fieldConfig.map((config) => ({ ...config, visible: true }));
    updateViewFieldConfig(nextConfig);
  };

  return (
    <div className="bg-white border-b border-slate-200 px-4 py-2 flex items-center gap-3">
      {/* View selector */}
      <div className="relative">
        <button
          onClick={() => setShowViewMenu(!showViewMenu)}
          data-testid="view-selector"
          className="flex items-center gap-2 px-3 py-1.5 text-sm font-semibold text-gray-700 hover:bg-slate-50 rounded-md border border-transparent hover:border-slate-200"
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
          className={`px-3 py-1.5 text-sm rounded-md flex items-center gap-1 border ${
            hasFilters
              ? 'bg-primary-50 text-primary-700 border-primary-100'
              : 'text-gray-600 hover:bg-slate-50 border-transparent hover:border-slate-200'
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
          className={`px-3 py-1.5 text-sm rounded-md flex items-center gap-1 border ${
            hasSorts
              ? 'bg-primary-50 text-primary-700 border-primary-100'
              : 'text-gray-600 hover:bg-slate-50 border-transparent hover:border-slate-200'
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
          className={`px-3 py-1.5 text-sm rounded-md flex items-center gap-1 border ${
            hasGroup
              ? 'bg-primary-50 text-primary-700 border-primary-100'
              : 'text-gray-600 hover:bg-slate-50 border-transparent hover:border-slate-200'
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

      {/* Color button (Grid only) */}
      {showFieldButton && (
        <div className="relative" ref={colorRef}>
          <button
            onClick={() => setShowColor(!showColor)}
            className={`px-3 py-1.5 text-sm rounded-md flex items-center gap-1 border ${
              hasRowColor
                ? 'bg-primary-50 text-primary-700 border-primary-100'
                : 'text-gray-600 hover:bg-slate-50 border-transparent hover:border-slate-200'
            }`}
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
            </svg>
            Color
            {hasRowColor && (
              <span className="ml-1 w-2 h-2 rounded-full bg-primary" />
            )}
          </button>

          {showColor && (
            <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-xl border border-gray-200 z-50 animate-slide-in">
              <RowColorConfig onClose={() => setShowColor(false)} />
            </div>
          )}
        </div>
      )}

      {/* Fields button (Grid only) */}
      {showFieldButton && (
        <div className="relative" ref={fieldsRef}>
          <button
            onClick={() => setShowFields(!showFields)}
            className={`px-3 py-1.5 text-sm rounded-md flex items-center gap-1 border ${
              hiddenCount > 0
                ? 'bg-primary-50 text-primary-700 border-primary-100'
                : 'text-gray-600 hover:bg-slate-50 border-transparent hover:border-slate-200'
            }`}
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
            </svg>
            Fields
            {hiddenCount > 0 && (
              <span className="ml-1 bg-primary text-white text-xs rounded-full w-5 h-5 flex items-center justify-center">
                {hiddenCount}
              </span>
            )}
          </button>

          {showFields && (
            <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-xl border border-gray-200 z-50 animate-slide-in min-w-[240px]">
              <div className="p-3 border-b border-gray-200 flex items-center justify-between">
                <span className="text-sm font-semibold text-gray-900">Visible fields</span>
                <button
                  onClick={showAllFields}
                  className="text-xs text-primary hover:text-primary-600"
                >
                  Show all
                </button>
              </div>
              <div className="max-h-[320px] overflow-auto p-2 space-y-1">
                {fieldConfig.map((config) => {
                  const field = fields.find((f) => f.id === config.field_id);
                  if (!field) return null;
                  return (
                    <label
                      key={field.id}
                      className="flex items-center gap-2 px-2 py-1 rounded-md hover:bg-slate-50 text-sm text-gray-700"
                    >
                      <input
                        type="checkbox"
                        checked={config.visible}
                        onChange={() => toggleFieldVisibility(field.id)}
                        className="w-4 h-4 rounded border-gray-300"
                      />
                      <span className="truncate">{field.name}</span>
                    </label>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Row Height button (Grid only) */}
      {showFieldButton && (
        <div className="relative" ref={rowHeightRef}>
          <button
            onClick={() => setShowRowHeight(!showRowHeight)}
            className="px-3 py-1.5 text-sm rounded-md flex items-center gap-1 border text-gray-600 hover:bg-slate-50 border-transparent hover:border-slate-200"
            title="Row height"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
            </svg>
          </button>

          {showRowHeight && (
            <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-xl border border-gray-200 z-50 animate-slide-in min-w-[160px]">
              <div className="p-2 border-b border-gray-200">
                <span className="text-xs font-semibold text-gray-500 uppercase">Row Height</span>
              </div>
              <div className="p-1">
                {ROW_HEIGHT_OPTIONS.map((option) => (
                  <button
                    key={option.value}
                    onClick={() => handleRowHeightChange(option.value)}
                    className={`dropdown-item w-full text-left flex items-center justify-between ${
                      currentRowHeight === option.value ? 'bg-primary-50 text-primary-700' : ''
                    }`}
                  >
                    <span>{option.label}</span>
                    {currentRowHeight === option.value && (
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Export button */}
      <div className="relative" ref={exportRef}>
        <button
          onClick={() => setShowExport(!showExport)}
          className="px-3 py-1.5 text-sm rounded-md flex items-center gap-1 border text-gray-600 hover:bg-slate-50 border-transparent hover:border-slate-200"
          title="Export"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
        </button>

        {showExport && (
          <div className="absolute right-0 top-full mt-1 bg-white rounded-lg shadow-xl border border-gray-200 z-50 animate-slide-in min-w-[160px]">
            <div className="p-2 border-b border-gray-200">
              <span className="text-xs font-semibold text-gray-500 uppercase">Export Data</span>
            </div>
            <div className="p-1">
              <button
                onClick={() => handleExport('csv')}
                className="dropdown-item w-full text-left flex items-center gap-2"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                Download CSV
              </button>
              <button
                onClick={() => handleExport('tsv')}
                className="dropdown-item w-full text-left flex items-center gap-2"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                Download TSV
              </button>
            </div>
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
