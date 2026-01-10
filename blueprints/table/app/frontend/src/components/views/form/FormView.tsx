import { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import { useBaseStore } from '../../../stores/baseStore';
import type { Field, CellValue, FormFieldConfig, FormSection, Attachment } from '../../../types';

// Default form configuration
const DEFAULT_FORM_CONFIG = {
  title: '',
  description: '',
  submit_button_text: 'Submit',
  success_message: 'Thank you! Your response has been recorded.',
  show_branding: true,
  cover_image_url: '',
  logo_url: '',
  theme_color: '#2563eb',
  is_public: true,
  require_password: false,
  allow_multiple_submissions: true,
  redirect_url: '',
};

export function FormView() {
  const { currentView, currentTable, fields, createRecord, updateViewConfig } = useBaseStore();
  const [formValues, setFormValues] = useState<Record<string, CellValue>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitSuccess, setSubmitSuccess] = useState(false);
  const [showConfig, setShowConfig] = useState(true);
  const [activeConfigTab, setActiveConfigTab] = useState<'fields' | 'appearance' | 'settings' | 'share'>('fields');
  const [copiedLink, setCopiedLink] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Parse view config
  const viewConfig = useMemo(() => {
    if (!currentView?.config) return DEFAULT_FORM_CONFIG;
    const config = typeof currentView.config === 'string'
      ? JSON.parse(currentView.config)
      : currentView.config;
    return { ...DEFAULT_FORM_CONFIG, ...config };
  }, [currentView?.config]);

  // Form field configurations
  const [fieldConfigs, setFieldConfigs] = useState<FormFieldConfig[]>([]);

  // Initialize field configs from view config or defaults
  useEffect(() => {
    const savedConfigs = currentView?.settings?.form_field_configs;
    if (savedConfigs && savedConfigs.length > 0) {
      setFieldConfigs(savedConfigs);
    } else {
      setFieldConfigs(
        fields.map((f, idx) => ({
          field_id: f.id,
          visible: !f.is_computed && f.type !== 'created_time' && f.type !== 'last_modified_time' && f.type !== 'created_by' && f.type !== 'last_modified_by' && f.type !== 'autonumber',
          required: f.is_primary,
          position: idx,
        }))
      );
    }
  }, [fields, currentView?.settings?.form_field_configs]);

  // Form sections (for future section grouping feature)
  const [sections] = useState<FormSection[]>(
    currentView?.settings?.form_sections || []
  );

  // Visible fields sorted by position
  const visibleFields = useMemo(() => {
    const configMap = new Map(fieldConfigs.map(c => [c.field_id, c]));
    return fields
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
  }, [fields, fieldConfigs, formValues]);

  // Initialize default values
  useEffect(() => {
    const defaults: Record<string, CellValue> = {};
    fieldConfigs.forEach(config => {
      if (config.default_value !== undefined && config.default_value !== null) {
        defaults[config.field_id] = config.default_value as CellValue;
      }
    });
    if (Object.keys(defaults).length > 0) {
      setFormValues(prev => ({ ...defaults, ...prev }));
    }
  }, [fieldConfigs]);

  // Handle URL prefill
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const prefillValues: Record<string, CellValue> = {};
    fields.forEach(field => {
      const value = params.get(`prefill_${field.id}`) || params.get(field.name.toLowerCase().replace(/\s+/g, '_'));
      if (value) {
        prefillValues[field.id] = value;
      }
    });
    if (Object.keys(prefillValues).length > 0) {
      setFormValues(prev => ({ ...prev, ...prefillValues }));
    }
  }, [fields]);

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
          const field = fields.find(f => f.id === config.field_id);
          newErrors[config.field_id] = `${field?.name || 'This field'} is required`;
        }
      }
    });
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [fieldConfigs, formValues, fields]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);
    try {
      await createRecord(formValues);
      setFormValues({});
      setSubmitSuccess(true);

      // Handle redirect
      const redirectUrl = viewConfig.redirect_url || currentView?.settings?.redirect_url;
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

  const toggleFieldVisibility = (fieldId: string) => {
    setFieldConfigs(prev => prev.map(c =>
      c.field_id === fieldId ? { ...c, visible: !c.visible } : c
    ));
  };

  const toggleFieldRequired = (fieldId: string) => {
    setFieldConfigs(prev => prev.map(c =>
      c.field_id === fieldId ? { ...c, required: !c.required } : c
    ));
  };

  const updateFieldConfig = (fieldId: string, updates: Partial<FormFieldConfig>) => {
    setFieldConfigs(prev => prev.map(c =>
      c.field_id === fieldId ? { ...c, ...updates } : c
    ));
  };

  const saveConfig = useCallback(() => {
    updateViewConfig({
      ...viewConfig,
      form_field_configs: fieldConfigs,
      form_sections: sections,
    });
  }, [updateViewConfig, viewConfig, fieldConfigs, sections]);

  // Auto-save config on changes
  useEffect(() => {
    const timeout = setTimeout(saveConfig, 500);
    return () => clearTimeout(timeout);
  }, [fieldConfigs, sections, saveConfig]);

  const updateFormSetting = (key: string, value: unknown) => {
    updateViewConfig({ ...viewConfig, [key]: value });
  };

  const copyShareLink = () => {
    const link = `${window.location.origin}/form/${currentView?.id || ''}`;
    navigator.clipboard.writeText(link);
    setCopiedLink(true);
    setTimeout(() => setCopiedLink(false), 2000);
  };

  const getEmbedCode = () => {
    const link = `${window.location.origin}/form/${currentView?.id || ''}`;
    return `<iframe src="${link}" width="100%" height="800" frameborder="0" style="border: none;"></iframe>`;
  };

  const formTitle = viewConfig.title || currentView?.settings?.title || `${currentTable?.name || 'Table'} Form`;
  const formDescription = viewConfig.description || currentView?.settings?.description || '';
  const submitButtonText = viewConfig.submit_button_text || currentView?.settings?.submit_button_text || 'Submit';
  const successMessage = viewConfig.success_message || currentView?.settings?.success_message || 'Thank you! Your response has been recorded.';
  const themeColor = viewConfig.theme_color || currentView?.settings?.theme_color || '#2563eb';
  const coverImageUrl = viewConfig.cover_image_url || currentView?.settings?.cover_image_url;
  const logoUrl = viewConfig.logo_url || currentView?.settings?.logo_url;
  const showBranding = viewConfig.show_branding !== false;

  // Reset success state when starting a new submission
  const resetForm = () => {
    setSubmitSuccess(false);
    setFormValues({});
  };

  if (submitSuccess) {
    return (
      <div className="flex-1 flex overflow-hidden">
        <div className="flex-1 overflow-auto bg-slate-100 p-8">
          <div className="max-w-2xl mx-auto">
            <div className="bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
              {coverImageUrl && (
                <div className="h-40 bg-cover bg-center" style={{ backgroundImage: `url(${coverImageUrl})` }} />
              )}
              <div className="p-8 text-center">
                <div className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4" style={{ backgroundColor: themeColor + '20' }}>
                  <svg className="w-8 h-8" style={{ color: themeColor }} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                  </svg>
                </div>
                <h2 className="text-2xl font-semibold text-gray-900 mb-2">Success!</h2>
                <p className="text-slate-600 mb-6">{successMessage}</p>
                {(viewConfig.allow_multiple_submissions !== false) && (
                  <button
                    onClick={resetForm}
                    className="btn btn-secondary"
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
        {/* Keep config panel visible */}
        <ConfigPanel
          showConfig={showConfig}
          setShowConfig={setShowConfig}
          activeConfigTab={activeConfigTab}
          setActiveConfigTab={setActiveConfigTab}
          fields={fields}
          fieldConfigs={fieldConfigs}
          toggleFieldVisibility={toggleFieldVisibility}
          toggleFieldRequired={toggleFieldRequired}
          updateFieldConfig={updateFieldConfig}
          viewConfig={viewConfig}
          updateFormSetting={updateFormSetting}
          copyShareLink={copyShareLink}
          copiedLink={copiedLink}
          currentView={currentView}
          getEmbedCode={getEmbedCode}
        />
      </div>
    );
  }

  return (
    <div className="flex-1 flex overflow-hidden">
      {/* Form preview */}
      <div className="flex-1 overflow-auto bg-slate-100 p-8">
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
            <div className="p-6 border-b border-slate-200" style={{ borderTopColor: themeColor, borderTopWidth: coverImageUrl ? 0 : 4 }}>
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
                  <FormField
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
                  No fields to display. Configure fields in the sidebar.
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

      {/* Configuration panel */}
      <ConfigPanel
        showConfig={showConfig}
        setShowConfig={setShowConfig}
        activeConfigTab={activeConfigTab}
        setActiveConfigTab={setActiveConfigTab}
        fields={fields}
        fieldConfigs={fieldConfigs}
        toggleFieldVisibility={toggleFieldVisibility}
        toggleFieldRequired={toggleFieldRequired}
        updateFieldConfig={updateFieldConfig}
        viewConfig={viewConfig}
        updateFormSetting={updateFormSetting}
        copyShareLink={copyShareLink}
        copiedLink={copiedLink}
        currentView={currentView}
        getEmbedCode={getEmbedCode}
      />
    </div>
  );
}

// Configuration Panel Component
interface ConfigPanelProps {
  showConfig: boolean;
  setShowConfig: (show: boolean) => void;
  activeConfigTab: 'fields' | 'appearance' | 'settings' | 'share';
  setActiveConfigTab: (tab: 'fields' | 'appearance' | 'settings' | 'share') => void;
  fields: Field[];
  fieldConfigs: FormFieldConfig[];
  toggleFieldVisibility: (fieldId: string) => void;
  toggleFieldRequired: (fieldId: string) => void;
  updateFieldConfig: (fieldId: string, updates: Partial<FormFieldConfig>) => void;
  viewConfig: typeof DEFAULT_FORM_CONFIG;
  updateFormSetting: (key: string, value: unknown) => void;
  copyShareLink: () => void;
  copiedLink: boolean;
  currentView: { id?: string; settings?: { title?: string } } | null;
  getEmbedCode: () => string;
}

function ConfigPanel({
  showConfig,
  setShowConfig,
  activeConfigTab,
  setActiveConfigTab,
  fields,
  fieldConfigs,
  toggleFieldVisibility,
  toggleFieldRequired,
  updateFieldConfig,
  viewConfig,
  updateFormSetting,
  copyShareLink,
  copiedLink,
  currentView,
  getEmbedCode,
}: ConfigPanelProps) {
  const [embedCopied, setEmbedCopied] = useState(false);

  const copyEmbed = () => {
    navigator.clipboard.writeText(getEmbedCode());
    setEmbedCopied(true);
    setTimeout(() => setEmbedCopied(false), 2000);
  };

  return (
    <div className={`${showConfig ? 'w-96' : 'w-12'} border-l border-slate-200 bg-white flex flex-col transition-all duration-200`}>
      <button
        onClick={() => setShowConfig(!showConfig)}
        className="p-3 border-b border-slate-200 hover:bg-slate-50 flex items-center justify-center"
      >
        <svg className={`w-5 h-5 text-gray-600 transform ${showConfig ? 'rotate-180' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
        </svg>
      </button>

      {showConfig && (
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Tabs */}
          <div className="flex border-b border-slate-200">
            {(['fields', 'appearance', 'settings', 'share'] as const).map(tab => (
              <button
                key={tab}
                onClick={() => setActiveConfigTab(tab)}
                className={`flex-1 px-3 py-2 text-xs font-medium capitalize ${
                  activeConfigTab === tab
                    ? 'text-blue-600 border-b-2 border-blue-600'
                    : 'text-slate-500 hover:text-slate-700'
                }`}
              >
                {tab}
              </button>
            ))}
          </div>

          <div className="flex-1 overflow-auto p-4">
            {activeConfigTab === 'fields' && (
              <div className="space-y-3">
                <p className="text-xs text-slate-500 mb-4">
                  Toggle field visibility and set required fields
                </p>
                {fields.map(field => {
                  const config = fieldConfigs.find(c => c.field_id === field.id);
                  const isComputed = field.is_computed || ['created_time', 'last_modified_time', 'created_by', 'last_modified_by', 'autonumber', 'formula', 'rollup', 'count', 'lookup'].includes(field.type);

                  return (
                    <div key={field.id} className={`p-3 rounded-lg border ${config?.visible ? 'border-slate-200 bg-white' : 'border-slate-100 bg-slate-50'}`}>
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <input
                            type="checkbox"
                            checked={config?.visible !== false}
                            onChange={() => toggleFieldVisibility(field.id)}
                            disabled={isComputed}
                            className="w-4 h-4 rounded border-gray-300"
                          />
                          <span className={`text-sm ${config?.visible ? 'text-gray-700' : 'text-gray-400'}`}>
                            {field.name}
                          </span>
                          {isComputed && (
                            <span className="text-xs text-slate-400">(auto)</span>
                          )}
                        </div>
                        <label className="flex items-center gap-1 text-xs text-slate-500">
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

                      {config?.visible && (
                        <div className="mt-3 space-y-2">
                          <div>
                            <label className="block text-xs text-slate-500 mb-1">Help text</label>
                            <input
                              type="text"
                              value={config?.help_text || ''}
                              onChange={(e) => updateFieldConfig(field.id, { help_text: e.target.value })}
                              placeholder="Add helper text..."
                              className="input text-xs py-1"
                            />
                          </div>
                          <div>
                            <label className="block text-xs text-slate-500 mb-1">Placeholder</label>
                            <input
                              type="text"
                              value={config?.placeholder || ''}
                              onChange={(e) => updateFieldConfig(field.id, { placeholder: e.target.value })}
                              placeholder="Add placeholder..."
                              className="input text-xs py-1"
                            />
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}

            {activeConfigTab === 'appearance' && (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Form Title</label>
                  <input
                    type="text"
                    value={viewConfig.title || ''}
                    onChange={(e) => updateFormSetting('title', e.target.value)}
                    placeholder="Enter form title..."
                    className="input"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
                  <textarea
                    value={viewConfig.description || ''}
                    onChange={(e) => updateFormSetting('description', e.target.value)}
                    placeholder="Add a description for your form..."
                    className="input min-h-[80px]"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Submit Button Text</label>
                  <input
                    type="text"
                    value={viewConfig.submit_button_text || 'Submit'}
                    onChange={(e) => updateFormSetting('submit_button_text', e.target.value)}
                    className="input"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Theme Color</label>
                  <div className="flex gap-2">
                    <input
                      type="color"
                      value={viewConfig.theme_color || '#2563eb'}
                      onChange={(e) => updateFormSetting('theme_color', e.target.value)}
                      className="w-10 h-10 rounded cursor-pointer"
                    />
                    <input
                      type="text"
                      value={viewConfig.theme_color || '#2563eb'}
                      onChange={(e) => updateFormSetting('theme_color', e.target.value)}
                      className="input flex-1"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Cover Image URL</label>
                  <input
                    type="url"
                    value={viewConfig.cover_image_url || ''}
                    onChange={(e) => updateFormSetting('cover_image_url', e.target.value)}
                    placeholder="https://..."
                    className="input"
                  />
                  {viewConfig.cover_image_url && (
                    <div className="mt-2 h-20 rounded-lg bg-cover bg-center border" style={{ backgroundImage: `url(${viewConfig.cover_image_url})` }} />
                  )}
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Logo URL</label>
                  <input
                    type="url"
                    value={viewConfig.logo_url || ''}
                    onChange={(e) => updateFormSetting('logo_url', e.target.value)}
                    placeholder="https://..."
                    className="input"
                  />
                  {viewConfig.logo_url && (
                    <img src={viewConfig.logo_url} alt="Logo preview" className="mt-2 h-10 object-contain" />
                  )}
                </div>

                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium text-gray-700">Show Branding</label>
                  <button
                    onClick={() => updateFormSetting('show_branding', !viewConfig.show_branding)}
                    className={`w-10 h-6 rounded-full transition-colors ${viewConfig.show_branding !== false ? 'bg-blue-600' : 'bg-slate-300'}`}
                  >
                    <div className={`w-4 h-4 rounded-full bg-white shadow transform transition-transform ${viewConfig.show_branding !== false ? 'translate-x-5' : 'translate-x-1'}`} />
                  </button>
                </div>
              </div>
            )}

            {activeConfigTab === 'settings' && (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Success Message</label>
                  <textarea
                    value={viewConfig.success_message || ''}
                    onChange={(e) => updateFormSetting('success_message', e.target.value)}
                    placeholder="Thank you for your submission!"
                    className="input min-h-[60px]"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Redirect URL (after submit)</label>
                  <input
                    type="url"
                    value={viewConfig.redirect_url || ''}
                    onChange={(e) => updateFormSetting('redirect_url', e.target.value)}
                    placeholder="https://example.com/thank-you"
                    className="input"
                  />
                  <p className="text-xs text-slate-500 mt-1">Leave empty to show success message</p>
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <label className="text-sm font-medium text-gray-700">Allow Multiple Submissions</label>
                    <p className="text-xs text-slate-500">Let users submit the form more than once</p>
                  </div>
                  <button
                    onClick={() => updateFormSetting('allow_multiple_submissions', !viewConfig.allow_multiple_submissions)}
                    className={`w-10 h-6 rounded-full transition-colors ${viewConfig.allow_multiple_submissions !== false ? 'bg-blue-600' : 'bg-slate-300'}`}
                  >
                    <div className={`w-4 h-4 rounded-full bg-white shadow transform transition-transform ${viewConfig.allow_multiple_submissions !== false ? 'translate-x-5' : 'translate-x-1'}`} />
                  </button>
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <label className="text-sm font-medium text-gray-700">Password Protection</label>
                    <p className="text-xs text-slate-500">Require password to access form</p>
                  </div>
                  <button
                    onClick={() => updateFormSetting('require_password', !viewConfig.require_password)}
                    className={`w-10 h-6 rounded-full transition-colors ${viewConfig.require_password ? 'bg-blue-600' : 'bg-slate-300'}`}
                  >
                    <div className={`w-4 h-4 rounded-full bg-white shadow transform transition-transform ${viewConfig.require_password ? 'translate-x-5' : 'translate-x-1'}`} />
                  </button>
                </div>
              </div>
            )}

            {activeConfigTab === 'share' && (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Share Link</label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={`${window.location.origin}/form/${currentView?.id || ''}`}
                      readOnly
                      className="input flex-1 text-xs"
                    />
                    <button
                      type="button"
                      onClick={copyShareLink}
                      className={`btn ${copiedLink ? 'btn-primary' : 'btn-secondary'} px-3`}
                    >
                      {copiedLink ? (
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                        </svg>
                      ) : (
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                        </svg>
                      )}
                    </button>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">Embed Code</label>
                  <textarea
                    value={getEmbedCode()}
                    readOnly
                    className="input text-xs font-mono h-20"
                  />
                  <button
                    type="button"
                    onClick={copyEmbed}
                    className={`btn ${embedCopied ? 'btn-primary' : 'btn-secondary'} w-full mt-2`}
                  >
                    {embedCopied ? 'Copied!' : 'Copy Embed Code'}
                  </button>
                </div>

                <div className="p-3 bg-blue-50 rounded-lg">
                  <h4 className="text-sm font-medium text-blue-900 mb-1">Prefill Fields</h4>
                  <p className="text-xs text-blue-700">
                    Add URL parameters to prefill form fields:
                  </p>
                  <code className="block mt-2 text-xs bg-blue-100 p-2 rounded">
                    ?prefill_[field_id]=value
                  </code>
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <label className="text-sm font-medium text-gray-700">Public Form</label>
                    <p className="text-xs text-slate-500">Anyone with the link can submit</p>
                  </div>
                  <button
                    onClick={() => updateFormSetting('is_public', !viewConfig.is_public)}
                    className={`w-10 h-6 rounded-full transition-colors ${viewConfig.is_public !== false ? 'bg-blue-600' : 'bg-slate-300'}`}
                  >
                    <div className={`w-4 h-4 rounded-full bg-white shadow transform transition-transform ${viewConfig.is_public !== false ? 'translate-x-5' : 'translate-x-1'}`} />
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// Form Field Component
interface FormFieldProps {
  field: Field;
  value: CellValue;
  onChange: (value: CellValue) => void;
  required: boolean;
  error?: string;
  helpText?: string;
  placeholder?: string;
  themeColor?: string;
}

function FormField({ field, value, onChange, required, error, helpText, placeholder, themeColor = '#2563eb' }: FormFieldProps) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [dragActive, setDragActive] = useState(false);

  const renderInput = () => {
    switch (field.type) {
      case 'text':
      case 'single_line_text':
        return (
          <input
            type="text"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={`input ${error ? 'border-red-500' : ''}`}
            placeholder={placeholder || `Enter ${field.name.toLowerCase()}`}
          />
        );

      case 'long_text':
        return (
          <textarea
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={`input min-h-[120px] ${error ? 'border-red-500' : ''}`}
            placeholder={placeholder || `Enter ${field.name.toLowerCase()}`}
          />
        );

      case 'number':
        return (
          <input
            type="number"
            value={(value as number) ?? ''}
            onChange={(e) => onChange(e.target.value ? parseFloat(e.target.value) : null)}
            className={`input ${error ? 'border-red-500' : ''}`}
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
              className={`input pl-7 ${error ? 'border-red-500' : ''}`}
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
              className={`input pr-8 ${error ? 'border-red-500' : ''}`}
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
            className={`input ${error ? 'border-red-500' : ''}`}
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
              <label key={choice.id} className="flex items-center gap-3 cursor-pointer group">
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
            {(!field.options?.choices || field.options.choices.length === 0) && (
              <p className="text-sm text-gray-400">No options available</p>
            )}
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
            className={`input ${error ? 'border-red-500' : ''}`}
          />
        );

      case 'datetime':
        return (
          <input
            type="datetime-local"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value || null)}
            className={`input ${error ? 'border-red-500' : ''}`}
          />
        );

      case 'email':
        return (
          <input
            type="email"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={`input ${error ? 'border-red-500' : ''}`}
            placeholder={placeholder || 'email@example.com'}
          />
        );

      case 'phone':
        return (
          <input
            type="tel"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={`input ${error ? 'border-red-500' : ''}`}
            placeholder={placeholder || '+1 (555) 000-0000'}
          />
        );

      case 'url':
        return (
          <input
            type="url"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={`input ${error ? 'border-red-500' : ''}`}
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
          // In a real implementation, you'd upload files to a server
          // For now, we'll create local object URLs
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

      case 'duration':
        const durationValue = (value as number) || 0;
        const hours = Math.floor(durationValue / 3600);
        const minutes = Math.floor((durationValue % 3600) / 60);

        return (
          <div className="flex gap-2">
            <div className="flex-1">
              <input
                type="number"
                value={hours || ''}
                onChange={(e) => {
                  const h = parseInt(e.target.value) || 0;
                  onChange(h * 3600 + minutes * 60);
                }}
                className={`input ${error ? 'border-red-500' : ''}`}
                placeholder="0"
                min="0"
              />
              <span className="text-xs text-gray-500 mt-1 block">Hours</span>
            </div>
            <div className="flex-1">
              <input
                type="number"
                value={minutes || ''}
                onChange={(e) => {
                  const m = parseInt(e.target.value) || 0;
                  onChange(hours * 3600 + m * 60);
                }}
                className={`input ${error ? 'border-red-500' : ''}`}
                placeholder="0"
                min="0"
                max="59"
              />
              <span className="text-xs text-gray-500 mt-1 block">Minutes</span>
            </div>
          </div>
        );

      case 'collaborator':
      case 'user':
        // For now, just show a text input for collaborator email
        // In a real app, you'd have a user picker
        return (
          <input
            type="text"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={`input ${error ? 'border-red-500' : ''}`}
            placeholder={placeholder || 'Enter collaborator email'}
          />
        );

      default:
        return (
          <input
            type="text"
            value={(value as string) || ''}
            onChange={(e) => onChange(e.target.value)}
            className={`input ${error ? 'border-red-500' : ''}`}
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
