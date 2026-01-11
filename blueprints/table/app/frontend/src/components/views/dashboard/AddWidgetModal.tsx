import { useState } from 'react';
import type {
  DashboardWidget,
  WidgetType,
  ChartType,
  AggregationType,
  StackingType,
  Table,
  Field
} from '../../../types';

interface AddWidgetModalProps {
  tables: Table[];
  fields: Field[];
  currentTableId: string;
  editingWidget?: DashboardWidget;
  onClose: () => void;
  onAdd: (widget: DashboardWidget) => void;
}

const widgetTypes: { type: WidgetType; label: string; description: string; icon: React.ReactNode }[] = [
  {
    type: 'chart',
    label: 'Chart',
    description: 'Visualize data with bar, line, or pie charts',
    icon: (
      <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
    ),
  },
  {
    type: 'number',
    label: 'Number',
    description: 'Show a single metric like count, sum, or average',
    icon: (
      <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 20l4-16m2 16l4-16M6 9h14M4 15h14" />
      </svg>
    ),
  },
  {
    type: 'list',
    label: 'List',
    description: 'Display a list of records from your table',
    icon: (
      <svg className="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
      </svg>
    ),
  },
];

const chartTypes: { type: ChartType; label: string }[] = [
  { type: 'bar', label: 'Bar Chart' },
  { type: 'line', label: 'Line Chart' },
  { type: 'pie', label: 'Pie Chart' },
  { type: 'donut', label: 'Donut Chart' },
  { type: 'area', label: 'Area Chart' },
  { type: 'scatter', label: 'Scatter Chart' },
];

const aggregationTypes: { type: AggregationType; label: string }[] = [
  { type: 'count', label: 'Count' },
  { type: 'sum', label: 'Sum' },
  { type: 'avg', label: 'Average' },
  { type: 'min', label: 'Minimum' },
  { type: 'max', label: 'Maximum' },
  { type: 'count_filled', label: 'Count (filled)' },
  { type: 'count_empty', label: 'Count (empty)' },
  { type: 'percent_filled', label: 'Percent filled' },
  { type: 'percent_empty', label: 'Percent empty' },
];

const stackingTypes: { type: StackingType; label: string; description: string }[] = [
  { type: 'none', label: 'None', description: 'No stacking' },
  { type: 'standard', label: 'Standard', description: 'Stack values on top of each other' },
  { type: 'percent', label: 'Percent', description: 'Stack as percentage of total (100%)' },
];

export function AddWidgetModal({
  tables,
  fields,
  currentTableId,
  editingWidget,
  onClose,
  onAdd,
}: AddWidgetModalProps) {
  const isEditing = !!editingWidget;
  const [step, setStep] = useState<'type' | 'config'>(isEditing ? 'config' : 'type');
  const [selectedType, setSelectedType] = useState<WidgetType | null>(editingWidget?.type || null);
  const [title, setTitle] = useState(editingWidget?.title || '');
  const [tableId, setTableId] = useState(editingWidget?.config.table_id || currentTableId);
  const [chartType, setChartType] = useState<ChartType>(editingWidget?.config.chart_type || 'bar');
  const [groupByField, setGroupByField] = useState(editingWidget?.config.group_by_field || '');
  const [aggregation, setAggregation] = useState<AggregationType>(editingWidget?.config.aggregation || 'count');
  const [valueField, setValueField] = useState(editingWidget?.config.field_id || '');
  const [stacking, setStacking] = useState<StackingType>(editingWidget?.config.stacking || 'none');
  const [secondaryGroup, setSecondaryGroup] = useState(editingWidget?.config.secondary_group || '');

  const tableFields = fields.filter(f => f.table_id === tableId);
  const selectFields = tableFields.filter(f =>
    f.type === 'single_select' || f.type === 'multi_select'
  );
  const numericFields = tableFields.filter(f =>
    f.type === 'number' || f.type === 'currency' || f.type === 'percent'
  );

  const handleSelectType = (type: WidgetType) => {
    setSelectedType(type);
    setStep('config');

    // Set default title
    switch (type) {
      case 'chart':
        setTitle('Chart');
        break;
      case 'number':
        setTitle('Count');
        break;
      case 'list':
        setTitle('Recent Records');
        break;
    }
  };

  const handleCreate = () => {
    if (!selectedType) return;

    const widget: DashboardWidget = {
      id: editingWidget?.id || `widget-${Date.now()}`,
      type: selectedType,
      title,
      position: editingWidget?.position || { row: 0, col: 0 },
      size: editingWidget?.size || getDefaultSize(selectedType),
      config: {
        table_id: tableId,
        aggregation,
        ...(selectedType === 'chart' && {
          chart_type: chartType,
          group_by_field: groupByField,
          show_legend: true,
          ...(chartType === 'bar' && stacking !== 'none' && {
            stacking,
            secondary_group: secondaryGroup || undefined,
          }),
        }),
        ...(selectedType === 'number' && {
          field_id: valueField || undefined,
        }),
        ...(selectedType === 'list' && {
          limit: editingWidget?.config.limit || 5,
          sort_field: editingWidget?.config.sort_field,
          sort_direction: editingWidget?.config.sort_direction,
        }),
        // Preserve filters from editing widget
        ...(editingWidget?.config.filters && {
          filters: editingWidget.config.filters,
        }),
      },
    };

    onAdd(widget);
  };

  const getDefaultSize = (type: WidgetType) => {
    switch (type) {
      case 'chart':
        return { width: 6, height: 4 };
      case 'number':
        return { width: 3, height: 2 };
      case 'list':
        return { width: 6, height: 4 };
      default:
        return { width: 4, height: 3 };
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b">
          <h2 className="text-lg font-semibold">
            {isEditing ? `Edit ${selectedType} Widget` : (step === 'type' ? 'Add Widget' : `Configure ${selectedType}`)}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {step === 'type' ? (
            <div className="grid grid-cols-1 gap-3">
              {widgetTypes.map(({ type, label, description, icon }) => (
                <button
                  key={type}
                  onClick={() => handleSelectType(type)}
                  className="flex items-center gap-4 p-4 border rounded-lg hover:border-primary hover:bg-blue-50 transition-colors text-left"
                >
                  <div className="text-gray-500">{icon}</div>
                  <div>
                    <div className="font-medium text-gray-900">{label}</div>
                    <div className="text-sm text-gray-500">{description}</div>
                  </div>
                </button>
              ))}
            </div>
          ) : (
            <div className="space-y-4">
              {/* Title */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Title
                </label>
                <input
                  type="text"
                  value={title}
                  onChange={e => setTitle(e.target.value)}
                  className="w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
                  placeholder="Widget title"
                />
              </div>

              {/* Table */}
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Data Source
                </label>
                <select
                  value={tableId}
                  onChange={e => setTableId(e.target.value)}
                  className="w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
                >
                  {tables.map(table => (
                    <option key={table.id} value={table.id}>
                      {table.name}
                    </option>
                  ))}
                </select>
              </div>

              {/* Chart specific */}
              {selectedType === 'chart' && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Chart Type
                    </label>
                    <select
                      value={chartType}
                      onChange={e => setChartType(e.target.value as ChartType)}
                      className="w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
                    >
                      {chartTypes.map(({ type, label }) => (
                        <option key={type} value={type}>
                          {label}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Group By
                    </label>
                    <select
                      value={groupByField}
                      onChange={e => setGroupByField(e.target.value)}
                      className="w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
                    >
                      <option value="">Select a field</option>
                      {selectFields.map(field => (
                        <option key={field.id} value={field.id}>
                          {field.name}
                        </option>
                      ))}
                    </select>
                    {selectFields.length === 0 && (
                      <p className="text-sm text-gray-500 mt-1">
                        No select fields available. Add a single or multi-select field to group by.
                      </p>
                    )}
                  </div>

                  {/* Stacking options - only for bar charts */}
                  {chartType === 'bar' && (
                    <>
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">
                          Stacking
                        </label>
                        <select
                          value={stacking}
                          onChange={e => setStacking(e.target.value as StackingType)}
                          className="w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
                        >
                          {stackingTypes.map(({ type, label }) => (
                            <option key={type} value={type}>
                              {label}
                            </option>
                          ))}
                        </select>
                        <p className="text-xs text-gray-500 mt-1">
                          {stackingTypes.find(s => s.type === stacking)?.description}
                        </p>
                      </div>

                      {stacking !== 'none' && (
                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-1">
                            Stack By (Secondary Group)
                          </label>
                          <select
                            value={secondaryGroup}
                            onChange={e => setSecondaryGroup(e.target.value)}
                            className="w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
                          >
                            <option value="">Select a field</option>
                            {selectFields
                              .filter(f => f.id !== groupByField)
                              .map(field => (
                                <option key={field.id} value={field.id}>
                                  {field.name}
                                </option>
                              ))}
                          </select>
                          <p className="text-xs text-gray-500 mt-1">
                            Choose a second field to create stacked segments
                          </p>
                        </div>
                      )}
                    </>
                  )}
                </>
              )}

              {/* Number specific */}
              {selectedType === 'number' && (
                <>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Aggregation
                    </label>
                    <select
                      value={aggregation}
                      onChange={e => setAggregation(e.target.value as AggregationType)}
                      className="w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
                    >
                      {aggregationTypes.map(({ type, label }) => (
                        <option key={type} value={type}>
                          {label}
                        </option>
                      ))}
                    </select>
                  </div>

                  {(aggregation === 'sum' || aggregation === 'avg' || aggregation === 'min' || aggregation === 'max') && (
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        Field
                      </label>
                      <select
                        value={valueField}
                        onChange={e => setValueField(e.target.value)}
                        className="w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-primary focus:border-primary"
                      >
                        <option value="">Select a field</option>
                        {numericFields.map(field => (
                          <option key={field.id} value={field.id}>
                            {field.name}
                          </option>
                        ))}
                      </select>
                    </div>
                  )}
                </>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t bg-gray-50">
          {step === 'config' && !isEditing && (
            <button
              onClick={() => setStep('type')}
              className="px-4 py-2 text-sm text-gray-600 hover:text-gray-900"
            >
              Back
            </button>
          )}
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-gray-600 hover:text-gray-900"
          >
            Cancel
          </button>
          {step === 'config' && (
            <button
              onClick={handleCreate}
              disabled={!title || (selectedType === 'chart' && !groupByField)}
              className="px-4 py-2 text-sm bg-primary text-white rounded-md hover:bg-primary-dark disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isEditing ? 'Save Changes' : 'Add Widget'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
