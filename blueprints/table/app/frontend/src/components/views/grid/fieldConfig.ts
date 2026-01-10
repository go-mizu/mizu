import type { Field, FieldConfig } from '../../../types';

export interface NormalizedFieldConfig extends FieldConfig {
  field_id: string;
  position: number;
  width: number;
  visible: boolean;
}

export function normalizeFieldConfig(
  fields: Field[],
  fieldConfig?: FieldConfig[] | null
): NormalizedFieldConfig[] {
  const configMap = new Map<string, FieldConfig>();
  (fieldConfig || []).forEach((config) => {
    configMap.set(config.field_id, config);
  });

  return fields
    .map((field, index) => {
      const config = configMap.get(field.id);
      const visible = config?.visible ?? !field.is_hidden;
      const width = typeof config?.width === 'number' ? config.width : field.width || 200;
      const position = typeof config?.position === 'number' ? config.position : field.position ?? index;

      return {
        field_id: field.id,
        visible,
        width,
        position,
      };
    })
    .sort((a, b) => a.position - b.position);
}

