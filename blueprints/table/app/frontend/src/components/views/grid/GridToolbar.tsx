import { useState, useRef, useCallback } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import { FilterBuilder } from './FilterBuilder';
import { SortBuilder } from './SortBuilder';
import type { Field } from '../../../types';

type RowHeightKey = 'short' | 'medium' | 'tall' | 'extra_tall';

interface GridToolbarProps {
  rowHeight: RowHeightKey;
  onRowHeightChange: (height: RowHeightKey) => void;
  showSummaryBar: boolean;
  onShowSummaryBarChange: (show: boolean) => void;
  rowColorFieldId: string | null;
  onRowColorFieldIdChange: (fieldId: string | null) => void;
  visibleFields: Field[];
  onToggleFieldVisibility: (fieldId: string) => void;
  onExport: (format: 'csv' | 'json') => void;
}

const ROW_HEIGHT_OPTIONS: { value: RowHeightKey; label: string; icon: string }[] = [
  { value: 'short', label: 'Short', icon: 'M4 6h16' },
  { value: 'medium', label: 'Medium', icon: 'M4 6h16M4 12h16' },
  { value: 'tall', label: 'Tall', icon: 'M4 6h16M4 10h16M4 14h16' },
  { value: 'extra_tall', label: 'Extra tall', icon: 'M4 5h16M4 9h16M4 13h16M4 17h16' },
];

export function GridToolbar({
  rowHeight,
  onRowHeightChange,
  showSummaryBar,
  onShowSummaryBarChange,
  rowColorFieldId,
  onRowColorFieldIdChange,
  visibleFields,
  onToggleFieldVisibility,
  onExport,
}: GridToolbarProps) {
  const { currentView, fields, groupBy, setGroupBy } = useBaseStore();
  const [showFilterBuilder, setShowFilterBuilder] = useState(false);
  const [showSortBuilder, setShowSortBuilder] = useState(false);
  const [showFieldsMenu, setShowFieldsMenu] = useState(false);
  const [showRowHeightMenu, setShowRowHeightMenu] = useState(false);
  const [showGroupMenu, setShowGroupMenu] = useState(false);
  const [showMoreMenu, setShowMoreMenu] = useState(false);
  const [showRowColorMenu, setShowRowColorMenu] = useState(false);
  const [showExportMenu, setShowExportMenu] = useState(false);

  const filterButtonRef = useRef<HTMLButtonElement>(null);
  const sortButtonRef = useRef<HTMLButtonElement>(null);
  const fieldsButtonRef = useRef<HTMLButtonElement>(null);
  const rowHeightButtonRef = useRef<HTMLButtonElement>(null);
  const groupButtonRef = useRef<HTMLButtonElement>(null);
  const moreButtonRef = useRef<HTMLButtonElement>(null);
  const rowColorButtonRef = useRef<HTMLButtonElement>(null);
  const exportButtonRef = useRef<HTMLButtonElement>(null);

  const filterCount = currentView?.filters?.length || 0;
  const sortCount = currentView?.sorts?.length || 0;

  const hiddenFieldCount = fields.length - visibleFields.length;

  const groupableFields = fields.filter(f =>
    ['single_select', 'multi_select', 'checkbox', 'user', 'collaborator'].includes(f.type)
  );

  const colorableFields = fields.filter(f => f.type === 'single_select');

  const closeAllMenus = useCallback(() => {
    setShowFieldsMenu(false);
    setShowRowHeightMenu(false);
    setShowGroupMenu(false);
    setShowMoreMenu(false);
    setShowRowColorMenu(false);
    setShowExportMenu(false);
  }, []);

  return (
    <div className="flex items-center gap-1 px-4 py-2 border-b border-slate-200 bg-slate-50">
      {/* Filter button */}
      <button
        ref={filterButtonRef}
        onClick={() => {
          closeAllMenus();
          setShowFilterBuilder(!showFilterBuilder);
          setShowSortBuilder(false);
        }}
        className={`flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md transition-colors ${
          filterCount > 0
            ? 'bg-primary-100 text-primary-700 hover:bg-primary-200'
            : 'text-slate-600 hover:bg-slate-100'
        }`}
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
        </svg>
        Filter
        {filterCount > 0 && (
          <span className="px-1.5 py-0.5 text-xs bg-primary text-white rounded-full">
            {filterCount}
          </span>
        )}
      </button>

      {/* Sort button */}
      <button
        ref={sortButtonRef}
        onClick={() => {
          closeAllMenus();
          setShowSortBuilder(!showSortBuilder);
          setShowFilterBuilder(false);
        }}
        className={`flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md transition-colors ${
          sortCount > 0
            ? 'bg-primary-100 text-primary-700 hover:bg-primary-200'
            : 'text-slate-600 hover:bg-slate-100'
        }`}
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12" />
        </svg>
        Sort
        {sortCount > 0 && (
          <span className="px-1.5 py-0.5 text-xs bg-primary text-white rounded-full">
            {sortCount}
          </span>
        )}
      </button>

      {/* Group button */}
      <div className="relative">
        <button
          ref={groupButtonRef}
          onClick={() => {
            closeAllMenus();
            setShowGroupMenu(!showGroupMenu);
            setShowFilterBuilder(false);
            setShowSortBuilder(false);
          }}
          className={`flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md transition-colors ${
            groupBy
              ? 'bg-primary-100 text-primary-700 hover:bg-primary-200'
              : 'text-slate-600 hover:bg-slate-100'
          }`}
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          Group
        </button>

        {showGroupMenu && (
          <>
            <div className="fixed inset-0 z-40" onClick={() => setShowGroupMenu(false)} />
            <div className="absolute top-full left-0 mt-1 z-50 bg-white rounded-lg shadow-xl border border-slate-200 min-w-[200px]">
              <div className="p-2">
                <button
                  onClick={() => {
                    setGroupBy(null);
                    setShowGroupMenu(false);
                  }}
                  className={`w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md ${
                    !groupBy ? 'bg-primary-50 text-primary-700' : 'text-slate-600 hover:bg-slate-100'
                  }`}
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16" />
                  </svg>
                  No grouping
                </button>
                <hr className="my-2" />
                {groupableFields.map(field => (
                  <button
                    key={field.id}
                    onClick={() => {
                      setGroupBy(field.id);
                      setShowGroupMenu(false);
                    }}
                    className={`w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md ${
                      groupBy === field.id ? 'bg-primary-50 text-primary-700' : 'text-slate-600 hover:bg-slate-100'
                    }`}
                  >
                    {field.name}
                    {groupBy === field.id && (
                      <svg className="w-4 h-4 ml-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </button>
                ))}
                {groupableFields.length === 0 && (
                  <p className="px-3 py-2 text-sm text-slate-500 italic">
                    No fields available for grouping
                  </p>
                )}
              </div>
            </div>
          </>
        )}
      </div>

      {/* Hide fields button */}
      <div className="relative">
        <button
          ref={fieldsButtonRef}
          onClick={() => {
            closeAllMenus();
            setShowFieldsMenu(!showFieldsMenu);
            setShowFilterBuilder(false);
            setShowSortBuilder(false);
          }}
          className={`flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md transition-colors ${
            hiddenFieldCount > 0
              ? 'bg-amber-100 text-amber-700 hover:bg-amber-200'
              : 'text-slate-600 hover:bg-slate-100'
          }`}
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-7 0-10-7-10-7a13.717 13.717 0 012.64-4.136M6.3 6.3A10.05 10.05 0 0112 5c7 0 10 7 10 7a13.717 13.717 0 01-2.64 4.136M6.3 6.3L3 3m3.3 3.3l3.64 3.64M17.7 17.7L21 21m-3.3-3.3l-3.64-3.64M17.7 17.7l-3.64-3.64M6.3 6.3l3.64 3.64" />
          </svg>
          Hide fields
          {hiddenFieldCount > 0 && (
            <span className="px-1.5 py-0.5 text-xs bg-amber-600 text-white rounded-full">
              {hiddenFieldCount}
            </span>
          )}
        </button>

        {showFieldsMenu && (
          <>
            <div className="fixed inset-0 z-40" onClick={() => setShowFieldsMenu(false)} />
            <div className="absolute top-full left-0 mt-1 z-50 bg-white rounded-lg shadow-xl border border-slate-200 min-w-[220px] max-h-[400px] overflow-y-auto">
              <div className="p-2">
                <p className="px-2 py-1 text-xs font-medium text-slate-500 uppercase">Fields</p>
                {fields.map(field => {
                  const isVisible = visibleFields.some(f => f.id === field.id);
                  return (
                    <button
                      key={field.id}
                      onClick={() => onToggleFieldVisibility(field.id)}
                      className="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md text-slate-600 hover:bg-slate-100"
                    >
                      <div className={`w-4 h-4 rounded border flex items-center justify-center ${
                        isVisible ? 'bg-primary border-primary' : 'border-slate-300'
                      }`}>
                        {isVisible && (
                          <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                          </svg>
                        )}
                      </div>
                      <span className={!isVisible ? 'text-slate-400' : ''}>{field.name}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          </>
        )}
      </div>

      <div className="flex-1" />

      {/* Row color */}
      <div className="relative">
        <button
          ref={rowColorButtonRef}
          onClick={() => {
            closeAllMenus();
            setShowRowColorMenu(!showRowColorMenu);
            setShowFilterBuilder(false);
            setShowSortBuilder(false);
          }}
          className={`flex items-center gap-1.5 px-2.5 py-1.5 text-sm rounded-md transition-colors ${
            rowColorFieldId
              ? 'bg-violet-100 text-violet-700 hover:bg-violet-200'
              : 'text-slate-600 hover:bg-slate-100'
          }`}
          title="Row coloring"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
          </svg>
        </button>

        {showRowColorMenu && (
          <>
            <div className="fixed inset-0 z-40" onClick={() => setShowRowColorMenu(false)} />
            <div className="absolute top-full right-0 mt-1 z-50 bg-white rounded-lg shadow-xl border border-slate-200 min-w-[200px]">
              <div className="p-3">
                <p className="text-xs font-medium text-slate-500 uppercase mb-2">Color rows by</p>
                <button
                  onClick={() => {
                    onRowColorFieldIdChange(null);
                    setShowRowColorMenu(false);
                  }}
                  className={`w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md ${
                    !rowColorFieldId ? 'bg-primary-50 text-primary-700' : 'text-slate-600 hover:bg-slate-100'
                  }`}
                >
                  None
                </button>
                {colorableFields.map(field => (
                  <button
                    key={field.id}
                    onClick={() => {
                      onRowColorFieldIdChange(field.id);
                      setShowRowColorMenu(false);
                    }}
                    className={`w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md ${
                      rowColorFieldId === field.id ? 'bg-primary-50 text-primary-700' : 'text-slate-600 hover:bg-slate-100'
                    }`}
                  >
                    {field.name}
                  </button>
                ))}
              </div>
            </div>
          </>
        )}
      </div>

      {/* Row height */}
      <div className="relative">
        <button
          ref={rowHeightButtonRef}
          onClick={() => {
            closeAllMenus();
            setShowRowHeightMenu(!showRowHeightMenu);
            setShowFilterBuilder(false);
            setShowSortBuilder(false);
          }}
          className="flex items-center gap-1.5 px-2.5 py-1.5 text-sm text-slate-600 hover:bg-slate-100 rounded-md"
          title="Row height"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8h16M4 16h16" />
          </svg>
        </button>

        {showRowHeightMenu && (
          <>
            <div className="fixed inset-0 z-40" onClick={() => setShowRowHeightMenu(false)} />
            <div className="absolute top-full right-0 mt-1 z-50 bg-white rounded-lg shadow-xl border border-slate-200 min-w-[160px]">
              <div className="p-2">
                <p className="px-2 py-1 text-xs font-medium text-slate-500 uppercase">Row height</p>
                {ROW_HEIGHT_OPTIONS.map(option => (
                  <button
                    key={option.value}
                    onClick={() => {
                      onRowHeightChange(option.value);
                      setShowRowHeightMenu(false);
                    }}
                    className={`w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md ${
                      rowHeight === option.value ? 'bg-primary-50 text-primary-700' : 'text-slate-600 hover:bg-slate-100'
                    }`}
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d={option.icon} />
                    </svg>
                    {option.label}
                    {rowHeight === option.value && (
                      <svg className="w-4 h-4 ml-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </button>
                ))}
              </div>
            </div>
          </>
        )}
      </div>

      {/* Export */}
      <div className="relative">
        <button
          ref={exportButtonRef}
          onClick={() => {
            closeAllMenus();
            setShowExportMenu(!showExportMenu);
            setShowFilterBuilder(false);
            setShowSortBuilder(false);
          }}
          className="flex items-center gap-1.5 px-2.5 py-1.5 text-sm text-slate-600 hover:bg-slate-100 rounded-md"
          title="Export"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
        </button>

        {showExportMenu && (
          <>
            <div className="fixed inset-0 z-40" onClick={() => setShowExportMenu(false)} />
            <div className="absolute top-full right-0 mt-1 z-50 bg-white rounded-lg shadow-xl border border-slate-200 min-w-[160px]">
              <div className="p-2">
                <p className="px-2 py-1 text-xs font-medium text-slate-500 uppercase">Export as</p>
                <button
                  onClick={() => {
                    onExport('csv');
                    setShowExportMenu(false);
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md text-slate-600 hover:bg-slate-100"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                  CSV file
                </button>
                <button
                  onClick={() => {
                    onExport('json');
                    setShowExportMenu(false);
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md text-slate-600 hover:bg-slate-100"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                  </svg>
                  JSON file
                </button>
              </div>
            </div>
          </>
        )}
      </div>

      {/* More options */}
      <div className="relative">
        <button
          ref={moreButtonRef}
          onClick={() => {
            closeAllMenus();
            setShowMoreMenu(!showMoreMenu);
            setShowFilterBuilder(false);
            setShowSortBuilder(false);
          }}
          className="flex items-center gap-1.5 px-2.5 py-1.5 text-sm text-slate-600 hover:bg-slate-100 rounded-md"
          title="More options"
        >
          <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
          </svg>
        </button>

        {showMoreMenu && (
          <>
            <div className="fixed inset-0 z-40" onClick={() => setShowMoreMenu(false)} />
            <div className="absolute top-full right-0 mt-1 z-50 bg-white rounded-lg shadow-xl border border-slate-200 min-w-[180px]">
              <div className="p-2">
                <button
                  onClick={() => {
                    onShowSummaryBarChange(!showSummaryBar);
                    setShowMoreMenu(false);
                  }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm rounded-md text-slate-600 hover:bg-slate-100"
                >
                  <div className={`w-4 h-4 rounded border flex items-center justify-center ${
                    showSummaryBar ? 'bg-primary border-primary' : 'border-slate-300'
                  }`}>
                    {showSummaryBar && (
                      <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </div>
                  Show summary bar
                </button>
              </div>
            </div>
          </>
        )}
      </div>

      {/* Filter Builder Panel */}
      <FilterBuilder
        isOpen={showFilterBuilder}
        onClose={() => setShowFilterBuilder(false)}
        anchorRef={filterButtonRef}
      />

      {/* Sort Builder Panel */}
      <SortBuilder
        isOpen={showSortBuilder}
        onClose={() => setShowSortBuilder(false)}
        anchorRef={sortButtonRef}
      />
    </div>
  );
}
