import { useState, useEffect, useMemo, useRef, useCallback } from 'react';
import type { Field, View, Table, CellValue, FormFieldConfig, Attachment } from '../types';

interface PublicFormData {
  view: View;
  table: Table;
  fields: Field[];
}

interface PublicFormPageProps {
  viewId: string;
}

export function PublicFormPage({ viewId }: PublicFormPageProps) {
  const [formData, setFormData] = useState<PublicFormData | null>(null);
  const [formValues, setFormValues] = useState<Record<string, CellValue>>({});
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitSuccess, setSubmitSuccess] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loadError, setLoadError] = useState<string | null>(null);

  // Load form data
  useEffect(() => {
    const loadForm = async () => {
      try {
        const response = await fetch(`/api/v1/public/forms/${viewId}`);
        if (!response.ok) {
          throw new Error('Form not found');
        }
        const data = await response.json();
        setFormData(data);
      } catch (err) {
        setLoadError(err instanceof Error ? err.message : 'Failed to load form');
      } finally {
        setIsLoading(false);
      }
    };
    loadForm();
  }, [viewId]);

  // Parse view config
  const viewConfig = useMemo(() => {
    if (!formData?.view?.config) return {};
    const config = typeof formData.view.config === 'string'
      ? JSON.parse(formData.view.config)
      : formData.view.config;
    return config;
  }, [formData?.view?.config]);

  // Get field configs
  const fieldConfigs = useMemo<FormFieldConfig[]>(() => {
    const settings = formData?.view?.settings;
    if (settings?.form_field_configs && settings.form_field_configs.length > 0) {
      return settings.form_field_configs;
    }
    // Default: show all non-computed fields
    return (formData?.fields || []).map((f, idx) => ({
      field_id: f.id,
      visible: !f.is_computed && !['created_time', 'last_modified_time', 'created_by', 'last_modified_by', 'autonumber'].includes(f.type),
      required: f.is_primary,
      position: idx,
    }));
  }, [formData?.view?.settings, formData?.fields]);

  // Visible fields sorted by position
  const visibleFields = useMemo(() => {
    if (!formData?.fields) return [];
    const configMap = new Map(fieldConfigs.map(c => [c.field_id, c]));
    return formData.fields
      .filter(f => {
        const config = configMap.get(f.id);
        if (!config?.visible) return false;
        // Check conditional visibility
        if (config.conditions && config.conditions.length > 0) {
          return config.conditions.every(cond => {
            const condValue = formValues[cond.field_id];
            switch (cond.operator) {
              case 'equals':
                return condValue === cond.value;
              case 'not_equals':
                return condValue !== cond.value;
              case 'contains':
                return String(condValue || '').includes(String(cond.value || ''));
              case 'is_empty':
                return condValue === null || condValue === undefined || condValue === '';
              case 'is_not_empty':
                return condValue !== null && condValue !== undefined && condValue !== '';
              default:
                return true;
            }
          });
        }
        return true;
      })
      .sort((a, b) => {
        const posA = configMap.get(a.id)?.position ?? 999;
        const posB = configMap.get(b.id)?.position ?? 999;
        return posA - posB;
      });
  }, [formData?.fields, fieldConfigs, formValues]);

  // Initialize default values and URL prefill
  useEffect(() => {
    if (!formData?.fields) return;

    const defaults: Record<string, CellValue> = {};

    // Apply default values
    fieldConfigs.forEach(config => {
      if (config.default_value !== undefined && config.default_value !== null) {
        defaults[config.field_id] = config.default_value as CellValue;
      }
    });

    // Apply URL prefill
    const params = new URLSearchParams(window.location.search);
    formData.fields.forEach(field => {
      const value = params.get(`prefill_${field.id}`) || params.get(field.name.toLowerCase().replace(/\s+/g, '_'));
      if (value) {
        defaults[field.id] = value;
      }
    });

    if (Object.keys(defaults).length > 0) {
      setFormValues(prev => ({ ...defaults, ...prev }));
    }
  }, [formData?.fields, fieldConfigs]);

  const handleValueChange = useCallback((fieldId: string, value: CellValue) => {
    setFormValues(prev => ({ ...prev, [fieldId]: value }));
    setErrors(prev => {
      const next = { ...prev };
      delete next[fieldId];
      return next;
    });
  }, []);

  const validateForm = useCallback(() => {
    const newErrors: Record<string, string> = {};
    fieldConfigs.forEach(config => {
      if (config.required && config.visible) {
        const value = formValues[config.field_id];
        if (value === null || value === undefined || value === '' || (Array.isArray(value) && value.length === 0)) {
          const field = formData?.fields.find(f => f.id === config.field_id);
          newErrors[config.field_id] = `${field?.name || 'This field'} is required`;
        }
      }
    });
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [fieldConfigs, formValues, formData?.fields]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);
    try {
      const response = await fetch(`/api/v1/public/forms/${viewId}/submit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ values: formValues }),
      });

      if (!response.ok) {
        throw new Error('Failed to submit form');
      }

      setSubmitSuccess(true);

      // Handle redirect
      const redirectUrl = viewConfig.redirect_url || formData?.view?.settings?.redirect_url;
      if (redirectUrl) {
        setTimeout(() => {
          window.location.href = redirectUrl;
        }, 1500);
      }
    } catch {
      setErrors({ _form: 'Failed to submit form. Please try again.' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const resetForm = () => {
    setSubmitSuccess(false);
    setFormValues({});
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-100">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Loading form...</p>
        </div>
      </div>
    );
  }

  // Error state
  if (loadError) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-100">
        <div className="text-center">
          <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h2 className="text-xl font-semibold text-gray-900 mb-2">Form Not Found</h2>
          <p className="text-gray-600">{loadError}</p>
        </div>
      </div>
    );
  }

  if (!formData) return null;

  const formTitle = viewConfig.title || formData.view?.settings?.title || `${formData.table?.name || 'Table'} Form`;
  const formDescription = viewConfig.description || formData.view?.settings?.description || '';
  const submitButtonText = viewConfig.submit_button_text || formData.view?.settings?.submit_button_text || 'Submit';
  const successMessage = viewConfig.success_message || formData.view?.settings?.success_message || 'Thank you! Your response has been recorded.';
  const themeColor = viewConfig.theme_color || formData.view?.settings?.theme_color || '#2563eb';
  const coverImageUrl = viewConfig.cover_image_url || formData.view?.settings?.cover_image_url;
  const logoUrl = viewConfig.logo_url || formData.view?.settings?.logo_url;
  const showBranding = viewConfig.show_branding !== false;
  const allowMultiple = viewConfig.allow_multiple_submissions !== false;

  // Success state
  if (submitSuccess) {
    return (
      <div className="min-h-screen bg-slate-100 py-8 px-4">
        <div className="max-w-2xl mx-auto">
          <div className="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
            {coverImageUrl && (
              <div className="h-40 bg-cover bg-center" style={{ backgroundImage: `url(${coverImageUrl})` }} />
            )}
            <div className="p-8 text-center">
              <div
                className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
                style={{ backgroundColor: themeColor + '20' }}
              >
                <svg className="w-8 h-8" style={{ color: themeColor }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <h2 className="text-2xl font-semibold text-gray-900 mb-2">Success!</h2>
              <p className="text-slate-600 mb-6">{successMessage}</p>
              {allowMultiple && (
                <button
                  onClick={resetForm}
                  className="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50"
                >
                  Submit another response
                </button>
              )}
            </div>
          </div>
          {showBranding && (
            <div className="mt-4 text-center text-xs text-slate-400">
              Powered by <span className="font-medium">Table</span>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-100 py-8 px-4">
      <div className="max-w-2xl mx-auto">
        <form onSubmit={handleSubmit} className="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
          {/* Cover image */}
          {coverImageUrl && (
            <div
              className="h-40 bg-cover bg-center"
              style={{ backgroundImage: `url(${coverImageUrl})` }}
            />
          )}

          {/* Form header */}
          <div
            className="p-6 border-b border-slate-200"
            style={{ borderTopColor: themeColor, borderTopWidth: coverImageUrl ? 0 : 4 }}
          >
            {logoUrl && (
              <img src={logoUrl} alt="Logo" className="h-12 mb-4" />
            )}
            <h1 className="text-2xl font-semibold text-gray-900">{formTitle}</h1>
            {formDescription && (
              <p className="mt-2 text-slate-600">{formDescription}</p>
            )}
          </div>

          {/* Form error */}
          {errors._form && (
            <div className="mx-6 mt-4 p-3 bg-red-50 text-red-700 rounded-lg text-sm">
              {errors._form}
            </div>
          )}

          {/* Form fields */}
          <div className="p-6 space-y-6">
            {visibleFields.map(field => {
              const config = fieldConfigs.find(c => c.field_id === field.id);
              const isRequired = config?.required || false;
              const helpText = config?.help_text;
              const placeholder = config?.placeholder;

              return (
                <PublicFormField
                  key={field.id}
                  field={field}
                  value={formValues[field.id] ?? null}
                  onChange={(value) => handleValueChange(field.id, value)}
                  required={isRequired}
                  error={errors[field.id]}
                  helpText={helpText}
                  placeholder={placeholder}
                  themeColor={themeColor}
                />
              );
            })}

            {visibleFields.length === 0 && (
              <p className="text-center text-gray-500 py-8">
                This form has no fields configured.
              </p>
            )}
          </div>

          {/* Submit button */}
          <div className="p-6 border-t border-slate-200 bg-slate-50 rounded-b-xl">
            <button
              type="submit"
              disabled={isSubmitting}
              className="w-full py-3 px-4 rounded-lg font-medium text-white transition-colors"
              style={{ backgroundColor: isSubmitting ? '#9ca3af' : themeColor }}
            >
              {isSubmitting ? 'Submitting...' : submitButtonText}
            </button>
          </div>
        </form>

        {/* Branding */}
        {showBranding && (
          <div className="mt-4 text-center text-xs text-slate-400">
            Powered by <span className="font-medium">Table</span>
          </div>
        )}
      </div>
    </div>
  );
}

// Simplified Form Field Component for Public Form
interface PublicFormFieldProps {
  field: Field;
  value: CellValue;
  onChange: (value: CellValue) => void;
  required: boolean;
  error?: string;
  helpText?: string;
  placeholder?: string;
  themeColor?: string;
}

function PublicFormField({ field, value, onChange, required, error, helpText, placeholder, themeColor = '#2563eb' }: PublicFormFieldProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragActive, setDragActive] = useState(false);

  const inputClasses = `w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 ${
    error ? 'border-red-500' : 'border-gray-300'
  }`;

  const renderInput = () => {
    switch (field.type) {
      case 'text':
      case 'single_line_text':
        return (
          <input
            type="text"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={inputClasses}
            placeholder={placeholder || `Enter ${field.name.toLowerCase()}`}
          />
        );

      case 'long_text':
        return (
          <textarea
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={`${inputClasses} min-h-[120px]`}
            placeholder={placeholder || `Enter ${field.name.toLowerCase()}`}
          />
        );

      case 'number':
        return (
          <input
            type="number"
            value={(value as number) ?? ''}
            onChange={(e) => onChange(e.target.value ? parseFloat(e.target.value) : null)}
            className={inputClasses}
            placeholder={placeholder || '0'}
          />
        );

      case 'currency':
        const currencySymbol = field.options?.currency_symbol || '$';
        return (
          <div className="relative">
            <span className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">{currencySymbol}</span>
            <input
              type="number"
              value={(value as number) ?? ''}
              onChange={(e) => onChange(e.target.value ? parseFloat(e.target.value) : null)}
              className={`${inputClasses} pl-7`}
              placeholder={placeholder || '0.00'}
              step="0.01"
            />
          </div>
        );

      case 'percent':
        return (
          <div className="relative">
            <input
              type="number"
              value={(value as number) ?? ''}
              onChange={(e) => onChange(e.target.value ? parseFloat(e.target.value) : null)}
              className={`${inputClasses} pr-8`}
              placeholder={placeholder || '0'}
              min="0"
              max="100"
            />
            <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500">%</span>
          </div>
        );

      case 'single_select':
        return (
          <select
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value || null)}
            className={inputClasses}
          >
            <option value="">{placeholder || 'Select an option'}</option>
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
          <div className={`space-y-2 p-3 border rounded-lg ${error ? 'border-red-500' : 'border-gray-200'}`}>
            {field.options?.choices?.map(choice => (
              <label key={choice.id} className="flex items-center gap-3 cursor-pointer">
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
                  style={{ accentColor: themeColor }}
                />
                <span
                  className="px-2 py-1 rounded text-sm font-medium"
                  style={{
                    backgroundColor: choice.color + '20',
                    color: choice.color,
                  }}
                >
                  {choice.name}
                </span>
              </label>
            ))}
          </div>
        );

      case 'checkbox':
        return (
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={(value as boolean) || false}
              onChange={(e) => onChange(e.target.checked)}
              className="w-5 h-5 rounded border-gray-300"
              style={{ accentColor: themeColor }}
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
            className={inputClasses}
          />
        );

      case 'datetime':
        return (
          <input
            type="datetime-local"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value || null)}
            className={inputClasses}
          />
        );

      case 'email':
        return (
          <input
            type="email"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={inputClasses}
            placeholder={placeholder || 'email@example.com'}
          />
        );

      case 'phone':
        return (
          <input
            type="tel"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={inputClasses}
            placeholder={placeholder || '+1 (555) 000-0000'}
          />
        );

      case 'url':
        return (
          <input
            type="url"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={inputClasses}
            placeholder={placeholder || 'https://'}
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
                className="text-3xl hover:scale-110 transition-transform focus:outline-none"
                style={{ color: i < currentRating ? themeColor : '#d1d5db' }}
              >
                {i < currentRating ? '★' : '☆'}
              </button>
            ))}
          </div>
        );

      case 'attachment':
        const attachments = (value as Attachment[]) || [];

        const handleFiles = (files: FileList) => {
          const newAttachments: Attachment[] = Array.from(files).map((file, idx) => ({
            id: `file-${Date.now()}-${idx}`,
            filename: file.name,
            size: file.size,
            mime_type: file.type,
            url: URL.createObjectURL(file),
          }));
          onChange([...attachments, ...newAttachments]);
        };

        const handleDrag = (e: React.DragEvent) => {
          e.preventDefault();
          e.stopPropagation();
          if (e.type === 'dragenter' || e.type === 'dragover') {
            setDragActive(true);
          } else if (e.type === 'dragleave') {
            setDragActive(false);
          }
        };

        const handleDrop = (e: React.DragEvent) => {
          e.preventDefault();
          e.stopPropagation();
          setDragActive(false);
          if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
            handleFiles(e.dataTransfer.files);
          }
        };

        const removeAttachment = (id: string) => {
          onChange(attachments.filter(a => a.id !== id));
        };

        return (
          <div className="space-y-3">
            <div
              onDragEnter={handleDrag}
              onDragLeave={handleDrag}
              onDragOver={handleDrag}
              onDrop={handleDrop}
              onClick={() => fileInputRef.current?.click()}
              className={`border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors ${
                dragActive
                  ? 'border-blue-500 bg-blue-50'
                  : error
                    ? 'border-red-300 hover:border-red-400'
                    : 'border-gray-300 hover:border-gray-400'
              }`}
            >
              <svg className="w-10 h-10 mx-auto text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
              </svg>
              <p className="text-sm text-gray-600 font-medium">
                {dragActive ? 'Drop files here' : 'Click to upload or drag and drop'}
              </p>
              <p className="text-xs text-gray-400 mt-1">PNG, JPG, PDF, etc.</p>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                onChange={(e) => e.target.files && handleFiles(e.target.files)}
                className="hidden"
              />
            </div>

            {attachments.length > 0 && (
              <div className="space-y-2">
                {attachments.map(att => (
                  <div key={att.id} className="flex items-center gap-3 p-2 bg-slate-50 rounded-lg">
                    {att.mime_type.startsWith('image/') ? (
                      <img src={att.url} alt={att.filename} className="w-10 h-10 object-cover rounded" />
                    ) : (
                      <div className="w-10 h-10 bg-slate-200 rounded flex items-center justify-center">
                        <svg className="w-5 h-5 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                        </svg>
                      </div>
                    )}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-gray-700 truncate">{att.filename}</p>
                      <p className="text-xs text-gray-400">{(att.size / 1024).toFixed(1)} KB</p>
                    </div>
                    <button
                      type="button"
                      onClick={() => removeAttachment(att.id)}
                      className="text-gray-400 hover:text-red-500 p-1"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        );

      default:
        return (
          <input
            type="text"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={inputClasses}
            placeholder={placeholder}
          />
        );
    }
  };

  return (
    <div>
      <label className="block text-sm font-semibold text-slate-700 mb-2">
        {field.name}
        {required && <span className="text-red-500 ml-1">*</span>}
      </label>
      {renderInput()}
      {helpText && (
        <p className="mt-1.5 text-xs text-slate-500">{helpText}</p>
      )}
      {field.description && !helpText && (
        <p className="mt-1.5 text-xs text-slate-500">{field.description}</p>
      )}
      {error && (
        <p className="mt-1.5 text-xs text-red-500">{error}</p>
      )}
    </div>
  );
}
