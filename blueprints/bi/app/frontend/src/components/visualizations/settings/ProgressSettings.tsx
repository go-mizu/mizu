import { Stack, NumberInput, ColorInput, Switch } from '@mantine/core'
import { chartColors } from '../../../theme'
import type { BaseSettingsProps } from './types'

export default function ProgressSettings({
  settings,
  onChange,
}: BaseSettingsProps) {
  return (
    <Stack gap="md">
      <NumberInput
        label="Goal"
        value={settings.goal || 100}
        onChange={(v) => onChange('goal', v)}
        min={0}
      />
      <ColorInput
        label="Bar color"
        value={settings.color || chartColors[0]}
        onChange={(v) => onChange('color', v)}
        format="hex"
        swatches={chartColors.slice(0, 8)}
      />
      <Switch
        label="Show percentage"
        checked={settings.showPercentage ?? true}
        onChange={(e) => onChange('showPercentage', e.currentTarget.checked)}
      />
    </Stack>
  )
}
