import { useState, useCallback } from 'react';
import {
  Modal,
  TextInput,
  Select,
  Button,
  Group,
  Stack,
  Tabs,
  Switch,
  NumberInput,
  Text,
  Paper,
} from '@mantine/core';
import {
  BarChart3,
  LineChart,
  PieChart,
  AreaChart,
  ScatterChart,
} from 'lucide-react';
import type {
  Chart,
  ChartType,
  CreateChartRequest,
  UpdateChartRequest,
  Selection,
  DataRange,
} from '../../types';

interface ChartEditorDialogProps {
  opened: boolean;
  onClose: () => void;
  chart?: Chart;
  selection?: Selection;
  sheetId: string;
  onSave: (data: CreateChartRequest | UpdateChartRequest) => Promise<void>;
}

const CHART_TYPES: { value: ChartType; label: string; icon: React.ReactNode }[] = [
  { value: 'line', label: 'Line', icon: <LineChart size={20} /> },
  { value: 'bar', label: 'Bar', icon: <BarChart3 size={20} style={{ transform: 'rotate(90deg)' }} /> },
  { value: 'column', label: 'Column', icon: <BarChart3 size={20} /> },
  { value: 'pie', label: 'Pie', icon: <PieChart size={20} /> },
  { value: 'doughnut', label: 'Doughnut', icon: <PieChart size={20} /> },
  { value: 'area', label: 'Area', icon: <AreaChart size={20} /> },
  { value: 'scatter', label: 'Scatter', icon: <ScatterChart size={20} /> },
  { value: 'stacked_bar', label: 'Stacked Bar', icon: <BarChart3 size={20} style={{ transform: 'rotate(90deg)' }} /> },
  { value: 'stacked_column', label: 'Stacked Column', icon: <BarChart3 size={20} /> },
  { value: 'stacked_area', label: 'Stacked Area', icon: <AreaChart size={20} /> },
  { value: 'radar', label: 'Radar', icon: <PieChart size={20} /> },
  { value: 'combo', label: 'Combo', icon: <BarChart3 size={20} /> },
];

export function ChartEditorDialog({
  opened,
  onClose,
  chart,
  selection,
  sheetId,
  onSave,
}: ChartEditorDialogProps) {
  const isEditing = !!chart;

  // Form state
  const [name, setName] = useState(chart?.name || 'Chart');
  const [chartType, setChartType] = useState<ChartType>(chart?.chartType || 'column');
  const [titleText, setTitleText] = useState(chart?.title?.text || '');
  const [titleFontSize, setTitleFontSize] = useState(chart?.title?.fontSize || 16);
  const [titleBold, setTitleBold] = useState(chart?.title?.bold || false);
  const [legendEnabled, setLegendEnabled] = useState(chart?.legend?.enabled ?? true);
  const [legendPosition, setLegendPosition] = useState<string>(chart?.legend?.position || 'bottom');
  const [animated, setAnimated] = useState(chart?.options?.animated ?? true);
  const [tooltipEnabled, setTooltipEnabled] = useState(chart?.options?.tooltipEnabled ?? true);
  const [hasHeader, setHasHeader] = useState(chart?.dataRanges?.[0]?.hasHeader ?? true);
  const [width, setWidth] = useState(chart?.size.width || 600);
  const [height, setHeight] = useState(chart?.size.height || 400);

  // Y-axis settings
  const [yAxisGridLines, setYAxisGridLines] = useState(chart?.axes?.yAxis?.gridLines ?? true);
  const [yAxisTitle, setYAxisTitle] = useState(chart?.axes?.yAxis?.title?.text || '');

  // X-axis settings
  const [xAxisGridLines, setXAxisGridLines] = useState(chart?.axes?.xAxis?.gridLines ?? false);
  const [xAxisTitle, setXAxisTitle] = useState(chart?.axes?.xAxis?.title?.text || '');

  const [loading, setLoading] = useState(false);

  // Get data range from selection or existing chart
  const dataRange: DataRange = chart?.dataRanges?.[0] || {
    startRow: selection?.startRow ?? 0,
    startCol: selection?.startCol ?? 0,
    endRow: selection?.endRow ?? 10,
    endCol: selection?.endCol ?? 3,
    hasHeader: hasHeader,
  };

  const getColumnName = (col: number): string => {
    let name = '';
    let c = col;
    while (c >= 0) {
      name = String.fromCharCode(65 + (c % 26)) + name;
      c = Math.floor(c / 26) - 1;
    }
    return name;
  };

  const getRangeString = (range: DataRange): string => {
    return `${getColumnName(range.startCol)}${range.startRow + 1}:${getColumnName(range.endCol)}${range.endRow + 1}`;
  };

  const handleSave = useCallback(async () => {
    setLoading(true);
    try {
      const data: CreateChartRequest | UpdateChartRequest = {
        ...(isEditing ? {} : { sheetId }),
        name,
        chartType,
        ...(isEditing ? {} : {
          position: {
            row: dataRange.endRow + 2,
            col: dataRange.startCol,
            offsetX: 0,
            offsetY: 0,
          },
        }),
        size: { width, height },
        dataRanges: [{
          ...dataRange,
          hasHeader,
        }],
        title: titleText ? {
          text: titleText,
          fontSize: titleFontSize,
          bold: titleBold,
        } : undefined,
        legend: {
          enabled: legendEnabled,
          position: legendPosition as 'top' | 'bottom' | 'left' | 'right' | 'none',
        },
        axes: {
          xAxis: {
            gridLines: xAxisGridLines,
            title: xAxisTitle ? { text: xAxisTitle } : undefined,
          },
          yAxis: {
            gridLines: yAxisGridLines,
            title: yAxisTitle ? { text: yAxisTitle } : undefined,
          },
        },
        options: {
          animated,
          tooltipEnabled,
          interactive: true,
        },
      };

      await onSave(data);
      onClose();
    } catch (err) {
      console.error('Failed to save chart:', err);
    } finally {
      setLoading(false);
    }
  }, [
    isEditing,
    sheetId,
    name,
    chartType,
    dataRange,
    hasHeader,
    titleText,
    titleFontSize,
    titleBold,
    legendEnabled,
    legendPosition,
    xAxisGridLines,
    xAxisTitle,
    yAxisGridLines,
    yAxisTitle,
    animated,
    tooltipEnabled,
    width,
    height,
    onSave,
    onClose,
  ]);

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={isEditing ? 'Edit Chart' : 'Insert Chart'}
      size="lg"
    >
      <Tabs defaultValue="setup">
        <Tabs.List>
          <Tabs.Tab value="setup">Setup</Tabs.Tab>
          <Tabs.Tab value="customize">Customize</Tabs.Tab>
          <Tabs.Tab value="axes">Axes</Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="setup" pt="md">
          <Stack gap="md">
            <TextInput
              label="Chart Name"
              value={name}
              onChange={(e) => setName(e.currentTarget.value)}
              placeholder="My Chart"
            />

            <div>
              <Text size="sm" fw={500} mb="xs">Chart Type</Text>
              <Group gap="xs">
                {CHART_TYPES.map((type) => (
                  <Paper
                    key={type.value}
                    p="sm"
                    withBorder
                    style={{
                      cursor: 'pointer',
                      backgroundColor: chartType === type.value ? '#e3f2fd' : 'transparent',
                      borderColor: chartType === type.value ? '#2196F3' : undefined,
                    }}
                    onClick={() => setChartType(type.value)}
                  >
                    <Stack align="center" gap={4}>
                      {type.icon}
                      <Text size="xs">{type.label}</Text>
                    </Stack>
                  </Paper>
                ))}
              </Group>
            </div>

            <Paper p="sm" withBorder>
              <Text size="sm" fw={500} mb="xs">Data Range</Text>
              <Text size="sm" c="dimmed">{getRangeString(dataRange)}</Text>
              <Switch
                label="First row is header"
                checked={hasHeader}
                onChange={(e) => setHasHeader(e.currentTarget.checked)}
                mt="xs"
              />
            </Paper>

            <Group grow>
              <NumberInput
                label="Width (px)"
                value={width}
                onChange={(v) => setWidth(Number(v) || 600)}
                min={200}
                max={1200}
              />
              <NumberInput
                label="Height (px)"
                value={height}
                onChange={(v) => setHeight(Number(v) || 400)}
                min={150}
                max={800}
              />
            </Group>
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="customize" pt="md">
          <Stack gap="md">
            <TextInput
              label="Chart Title"
              value={titleText}
              onChange={(e) => setTitleText(e.currentTarget.value)}
              placeholder="Enter chart title"
            />

            <Group grow>
              <NumberInput
                label="Title Font Size"
                value={titleFontSize}
                onChange={(v) => setTitleFontSize(Number(v) || 16)}
                min={10}
                max={32}
              />
              <Switch
                label="Bold Title"
                checked={titleBold}
                onChange={(e) => setTitleBold(e.currentTarget.checked)}
                mt="xl"
              />
            </Group>

            <Group grow>
              <Switch
                label="Show Legend"
                checked={legendEnabled}
                onChange={(e) => setLegendEnabled(e.currentTarget.checked)}
              />
              <Select
                label="Legend Position"
                value={legendPosition}
                onChange={(v) => setLegendPosition(v || 'bottom')}
                data={[
                  { value: 'top', label: 'Top' },
                  { value: 'bottom', label: 'Bottom' },
                  { value: 'left', label: 'Left' },
                  { value: 'right', label: 'Right' },
                ]}
                disabled={!legendEnabled}
              />
            </Group>

            <Group grow>
              <Switch
                label="Animation"
                checked={animated}
                onChange={(e) => setAnimated(e.currentTarget.checked)}
              />
              <Switch
                label="Show Tooltips"
                checked={tooltipEnabled}
                onChange={(e) => setTooltipEnabled(e.currentTarget.checked)}
              />
            </Group>
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="axes" pt="md">
          <Stack gap="md">
            <Text fw={500}>X-Axis</Text>
            <Group grow>
              <TextInput
                label="X-Axis Title"
                value={xAxisTitle}
                onChange={(e) => setXAxisTitle(e.currentTarget.value)}
                placeholder="X-Axis label"
              />
              <Switch
                label="Show Grid Lines"
                checked={xAxisGridLines}
                onChange={(e) => setXAxisGridLines(e.currentTarget.checked)}
                mt="xl"
              />
            </Group>

            <Text fw={500} mt="md">Y-Axis</Text>
            <Group grow>
              <TextInput
                label="Y-Axis Title"
                value={yAxisTitle}
                onChange={(e) => setYAxisTitle(e.currentTarget.value)}
                placeholder="Y-Axis label"
              />
              <Switch
                label="Show Grid Lines"
                checked={yAxisGridLines}
                onChange={(e) => setYAxisGridLines(e.currentTarget.checked)}
                mt="xl"
              />
            </Group>
          </Stack>
        </Tabs.Panel>
      </Tabs>

      <Group justify="flex-end" mt="xl">
        <Button variant="subtle" onClick={onClose}>
          Cancel
        </Button>
        <Button onClick={handleSave} loading={loading}>
          {isEditing ? 'Update' : 'Insert Chart'}
        </Button>
      </Group>
    </Modal>
  );
}

export default ChartEditorDialog;
