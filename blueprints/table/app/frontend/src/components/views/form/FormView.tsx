import { useState, useMemo } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { Field, CellValue } from '../../../types';

interface FormFieldConfig {
  fieldId: string;
  visible: boolean;
  required: boolean;
}

export function FormView() {
  const { currentView, currentTable, fields, createRecord } = useBaseStore();
  const [formValues, setFormValues] = useState<Record<string, CellValue>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitSuccess, setSubmitSuccess] = useState(false);
  const [showConfig, setShowConfig] = useState(false);
  const [fieldConfigs, setFieldConfigs] = useState<FormFieldConfig[]>(() =>
    fields.map(f => ({ fieldId: f.id, visible: true, required: false }))
  );

  // Update field configs when fields change
  useMemo(() => {
    setFieldConfigs(prev => {
      const existing = new Map(prev.map(c => [c.fieldId, c]));
      return fields.map(f => existing.get(f.id) || { fieldId: f.id, visible: true, required: false });
    });
  }, [fields]);

  const visibleFields = useMemo(() => {
    return fields.filter(f => {
      const config = fieldConfigs.find(c => c.fieldId === f.id);
      return config?.visible !== false;
    });
  }, [fields, fieldConfigs]);

  const handleValueChange = (fieldId: string, value: CellValue) => {
    setFormValues(prev => ({ ...prev, [fieldId]: value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validate required fields
    for (const config of fieldConfigs) {
      if (config.required && config.visible) {
        const value = formValues[config.fieldId];
        if (value === null || value === undefined || value === '') {
          const field = fields.find(f => f.id === config.fieldId);
          alert(`${field?.name || 'Field'} is required`);
          return;
        }
      }
    }

    setIsSubmitting(true);
    try {
      await createRecord(formValues);
      setFormValues({});
      setSubmitSuccess(true);
      setTimeout(() => setSubmitSuccess(false), 3000);
    } catch {
      alert('Failed to submit form');
    } finally {
      setIsSubmitting(false);
    }
  };

  const toggleFieldVisibility = (fieldId: string) => {
    setFieldConfigs(prev => prev.map(c =>
      c.fieldId === fieldId ? { ...c, visible: !c.visible } : c
    ));
  };

  const toggleFieldRequired = (fieldId: string) => {
    setFieldConfigs(prev => prev.map(c =>
      c.fieldId === fieldId ? { ...c, required: !c.required } : c
    ));
  };

  const formTitle = currentView?.settings?.title || `${currentTable?.name || 'Table'} Form`;
  const formDescription = currentView?.settings?.description || '';

  return (
    <div className="flex-1 flex overflow-hidden">
      {/* Form preview */}
      <div className="flex-1 overflow-auto bg-gray-100 p-8">
        <div className="max-w-2xl mx-auto">
          <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow-sm border border-gray-200">
            {/* Form header */}
            <div className="p-6 border-b border-gray-200">
              <h1 className="text-2xl font-bold text-gray-900">{formTitle}</h1>
              {formDescription && (
                <p className="mt-2 text-gray-600">{formDescription}</p>
              )}
            </div>

            {/* Form fields */}
            <div className="p-6 space-y-6">
              {visibleFields.map(field => {
                const config = fieldConfigs.find(c => c.fieldId === field.id);
                const isRequired = config?.required || false;

                return (
                  <FormField
                    key={field.id}
                    field={field}
                    value={formValues[field.id] ?? null}
                    onChange={(value) => handleValueChange(field.id, value)}
                    required={isRequired}
                  />
                );
              })}

              {visibleFields.length === 0 && (
                <p className="text-center text-gray-500 py-8">
                  No fields to display. Configure fields in the sidebar.
                </p>
              )}
            </div>

            {/* Submit button */}
            <div className="p-6 border-t border-gray-200 bg-gray-50 rounded-b-lg">
              <button
                type="submit"
                disabled={isSubmitting}
                className="btn btn-primary w-full py-3"
              >
                {isSubmitting ? 'Submitting...' : 'Submit'}
              </button>
            </div>
          </form>

          {/* Success message */}
          {submitSuccess && (
            <div className="mt-4 p-4 bg-success-50 text-success-700 rounded-lg flex items-center gap-2">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
              Record created successfully!
            </div>
          )}
        </div>
      </div>

      {/* Configuration panel */}
      <div className={`${showConfig ? 'w-80' : 'w-12'} border-l border-gray-200 bg-white flex flex-col transition-all duration-200`}>
        <button
          onClick={() => setShowConfig(!showConfig)}
          className="p-3 border-b border-gray-200 hover:bg-gray-50 flex items-center justify-center"
        >
          <svg className={`w-5 h-5 text-gray-600 transform ${showConfig ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
        </button>

        {showConfig && (
          <div className="flex-1 overflow-auto p-4">
            <h3 className="font-medium text-gray-900 mb-4">Form Configuration</h3>

            <div className="space-y-3">
              {fields.map(field => {
                const config = fieldConfigs.find(c => c.fieldId === field.id);

                return (
                  <div key={field.id} className="flex items-center justify-between p-2 rounded hover:bg-gray-50">
                    <div className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={config?.visible !== false}
                        onChange={() => toggleFieldVisibility(field.id)}
                        className="w-4 h-4 rounded border-gray-300"
                      />
                      <span className="text-sm text-gray-700">{field.name}</span>
                    </div>
                    <label className="flex items-center gap-1 text-xs text-gray-500">
                      <input
                        type="checkbox"
                        checked={config?.required || false}
                        onChange={() => toggleFieldRequired(field.id)}
                        disabled={!config?.visible}
                        className="w-3 h-3 rounded border-gray-300"
                      />
                      Required
                    </label>
                  </div>
                );
              })}
            </div>

            <hr className="my-4" />

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Share Link
                </label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={`${window.location.origin}/form/${currentView?.id || ''}`}
                    readOnly
                    className="input flex-1 text-xs"
                  />
                  <button
                    type="button"
                    onClick={() => {
                      navigator.clipboard.writeText(`${window.location.origin}/form/${currentView?.id || ''}`);
                    }}
                    className="btn btn-secondary px-2"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

interface FormFieldProps {
  field: Field;
  value: CellValue;
  onChange: (value: CellValue) => void;
  required: boolean;
}

function FormField({ field, value, onChange, required }: FormFieldProps) {
  const renderInput = () => {
    switch (field.type) {
      case 'text':
        return (
          <input
            type="text"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className="input"
            placeholder={`Enter ${field.name.toLowerCase()}`}
          />
        );

      case 'long_text':
        return (
          <textarea
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className="input min-h-[100px]"
            placeholder={`Enter ${field.name.toLowerCase()}`}
          />
        );

      case 'number':
      case 'currency':
      case 'percent':
        return (
          <input
            type="number"
            value={(value as number) ?? ''}
            onChange={(e) => onChange(e.target.value ? parseFloat(e.target.value) : null)}
            className="input"
            placeholder="0"
            step={field.type === 'currency' ? '0.01' : 'any'}
          />
        );

      case 'single_select':
        return (
          <select
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value || null)}
            className="input"
          >
            <option value="">Select an option</option>
            {field.options?.choices?.map(choice => (
              <option key={choice.id} value={choice.id}>
                {choice.name}
              </option>
            ))}
          </select>
        );

      case 'multi_select':
        const selectedValues = (value as string[]) || [];
        return (
          <div className="space-y-2">
            {field.options?.choices?.map(choice => (
              <label key={choice.id} className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={selectedValues.includes(choice.id)}
                  onChange={(e) => {
                    if (e.target.checked) {
                      onChange([...selectedValues, choice.id]);
                    } else {
                      onChange(selectedValues.filter(v => v !== choice.id));
                    }
                  }}
                  className="w-4 h-4 rounded border-gray-300"
                />
                <span
                  className="px-2 py-0.5 rounded text-sm"
                  style={{ backgroundColor: choice.color + '20', color: choice.color }}
                >
                  {choice.name}
                </span>
              </label>
            ))}
          </div>
        );

      case 'checkbox':
        return (
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={(value as boolean) || false}
              onChange={(e) => onChange(e.target.checked)}
              className="w-5 h-5 rounded border-gray-300"
            />
            <span className="text-sm text-gray-600">Yes</span>
          </label>
        );

      case 'date':
        return (
          <input
            type="date"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value || null)}
            className="input"
          />
        );

      case 'datetime':
        return (
          <input
            type="datetime-local"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value || null)}
            className="input"
          />
        );

      case 'email':
        return (
          <input
            type="email"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className="input"
            placeholder="email@example.com"
          />
        );

      case 'phone':
        return (
          <input
            type="tel"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className="input"
            placeholder="+1 (555) 000-0000"
          />
        );

      case 'url':
        return (
          <input
            type="url"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className="input"
            placeholder="https://"
          />
        );

      case 'rating':
        const maxRating = field.options?.max || 5;
        const currentRating = (value as number) || 0;
        return (
          <div className="flex gap-1">
            {Array.from({ length: maxRating }, (_, i) => (
              <button
                key={i}
                type="button"
                onClick={() => onChange(i + 1 === currentRating ? 0 : i + 1)}
                className="text-2xl"
              >
                {i < currentRating ? '★' : '☆'}
              </button>
            ))}
          </div>
        );

      case 'attachment':
        return (
          <div className="border-2 border-dashed border-gray-300 rounded-lg p-6 text-center">
            <svg className="w-8 h-8 mx-auto text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
            </svg>
            <p className="text-sm text-gray-500">Click to upload or drag and drop</p>
            <input type="file" className="hidden" />
          </div>
        );

      default:
        return (
          <input
            type="text"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className="input"
          />
        );
    }
  };

  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-2">
        {field.name}
        {required && <span className="text-danger ml-1">*</span>}
      </label>
      {renderInput()}
      {field.description && (
        <p className="mt-1 text-xs text-gray-500">{field.description}</p>
      )}
    </div>
  );
}
