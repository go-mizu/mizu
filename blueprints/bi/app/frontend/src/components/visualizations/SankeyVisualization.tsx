import { useMemo } from 'react'
import { Box, Text, Paper } from '@mantine/core'
import { chartColors } from '../../theme'

interface SankeyVisualizationProps {
  data: Record<string, any>[]
  columns: { name: string; display_name: string; type: string }[]
  settings?: Record<string, any>
  height: number
}

interface SankeyNode {
  name: string
  layer: number
  index: number
  value: number
  x: number
  y: number
  height: number
  color: string
}

interface SankeyLink {
  source: SankeyNode
  target: SankeyNode
  value: number
  y0: number
  y1: number
}

/**
 * Sankey Diagram Visualization
 *
 * Expects data in one of two formats:
 * 1. Source-Target-Value: { source: string, target: string, value: number }
 * 2. Multi-step: { step1: string, step2: string, step3?: string, ..., value: number }
 */
export default function SankeyVisualization({
  data,
  columns,
  settings = {},
  height,
}: SankeyVisualizationProps) {
  const nodeWidth = settings.nodeWidth ?? 20
  const nodePadding = settings.nodePadding ?? 10
  const linkOpacity = settings.linkOpacity ?? 0.4

  // Process data to create nodes and links
  const { nodes, links, width: chartWidth, svgHeight } = useMemo(() => {
    // Detect data format
    const hasSourceTarget = columns.some(c => c.name.toLowerCase() === 'source') &&
                           columns.some(c => c.name.toLowerCase() === 'target')

    let processedLinks: { source: string; target: string; value: number }[] = []

    if (hasSourceTarget) {
      // Format 1: Source-Target-Value
      const sourceCol = columns.find(c => c.name.toLowerCase() === 'source')?.name || columns[0]?.name
      const targetCol = columns.find(c => c.name.toLowerCase() === 'target')?.name || columns[1]?.name
      const valueCol = columns.find(c =>
        c.name.toLowerCase() === 'value' ||
        c.type === 'number' ||
        c.type === 'integer'
      )?.name || columns[2]?.name

      processedLinks = data.map(row => ({
        source: String(row[sourceCol] ?? ''),
        target: String(row[targetCol] ?? ''),
        value: Number(row[valueCol]) || 1,
      }))
    } else {
      // Format 2: Multi-step - treat string columns as steps, last numeric as value
      const stepCols = columns.filter(c => c.type === 'string' || c.type === 'text' || !['number', 'integer', 'float'].includes(c.type))
      const valueCol = columns.find(c => ['number', 'integer', 'float'].includes(c.type))

      if (stepCols.length >= 2) {
        data.forEach(row => {
          const value = valueCol ? (Number(row[valueCol.name]) || 1) : 1
          for (let i = 0; i < stepCols.length - 1; i++) {
            const source = String(row[stepCols[i].name] ?? '')
            const target = String(row[stepCols[i + 1].name] ?? '')
            if (source && target) {
              processedLinks.push({ source, target, value })
            }
          }
        })
      }
    }

    if (processedLinks.length === 0) {
      return { nodes: [], links: [], width: 0, svgHeight: 0 }
    }

    // Build node layers by analyzing link paths
    const nodeLayerMap = new Map<string, number>()
    const sources = new Set(processedLinks.map(l => l.source))
    const targets = new Set(processedLinks.map(l => l.target))

    // Nodes that are only sources go in layer 0
    sources.forEach(s => {
      if (!targets.has(s)) {
        nodeLayerMap.set(s, 0)
      }
    })

    // Propagate layers
    let changed = true
    let iterations = 0
    while (changed && iterations < 100) {
      changed = false
      iterations++
      processedLinks.forEach(link => {
        const sourceLayer = nodeLayerMap.get(link.source)
        if (sourceLayer !== undefined) {
          const currentTargetLayer = nodeLayerMap.get(link.target)
          const newTargetLayer = sourceLayer + 1
          if (currentTargetLayer === undefined || currentTargetLayer < newTargetLayer) {
            nodeLayerMap.set(link.target, newTargetLayer)
            changed = true
          }
        }
      })
      // Also handle nodes that only appear as targets (put them in layer 1 if not set)
      targets.forEach(t => {
        if (!nodeLayerMap.has(t)) {
          nodeLayerMap.set(t, 1)
        }
      })
    }

    // Get all unique nodes
    const allNodeNames = new Set([...sources, ...targets])
    const maxLayer = Math.max(...Array.from(nodeLayerMap.values()))

    // Calculate node values (sum of incoming/outgoing flows)
    const nodeValues = new Map<string, number>()
    processedLinks.forEach(link => {
      nodeValues.set(link.source, (nodeValues.get(link.source) || 0) + link.value)
      nodeValues.set(link.target, Math.max(nodeValues.get(link.target) || 0, link.value))
    })

    // Create nodes grouped by layer
    const nodesByLayer: SankeyNode[][] = []
    for (let i = 0; i <= maxLayer; i++) {
      nodesByLayer[i] = []
    }

    let nodeIndex = 0
    allNodeNames.forEach(name => {
      const layer = nodeLayerMap.get(name) ?? 0
      nodesByLayer[layer].push({
        name,
        layer,
        index: nodeIndex++,
        value: nodeValues.get(name) || 0,
        x: 0,
        y: 0,
        height: 0,
        color: chartColors[layer % chartColors.length],
      })
    })

    // Calculate positions
    const chartWidth = 800
    const layerWidth = (chartWidth - nodeWidth) / Math.max(maxLayer, 1)
    const totalMaxValue = Math.max(...nodesByLayer.map(layer =>
      layer.reduce((sum, n) => sum + n.value, 0)
    ))

    const availableHeight = height - 40 // Leave room for labels
    const heightScale = availableHeight / (totalMaxValue || 1)

    nodesByLayer.forEach((layer, layerIndex) => {
      const layerTotal = layer.reduce((sum, n) => sum + n.value, 0)
      const layerHeight = layerTotal * heightScale
      const startY = (availableHeight - layerHeight - (layer.length - 1) * nodePadding) / 2 + 20

      let currentY = startY
      layer.forEach((node) => {
        node.x = layerIndex * layerWidth
        node.y = currentY
        node.height = Math.max(4, node.value * heightScale)
        currentY += node.height + nodePadding
      })
    })

    // Create flat node array with lookup
    const nodeArray = nodesByLayer.flat()
    const nodeMap = new Map(nodeArray.map(n => [n.name, n]))

    // Create links with positions
    const sankeyLinks: SankeyLink[] = processedLinks.map(link => {
      const source = nodeMap.get(link.source)!
      const target = nodeMap.get(link.target)!
      return {
        source,
        target,
        value: link.value,
        y0: 0,
        y1: 0,
      }
    })

    // Calculate link Y positions (stacked at each node)
    const sourceOffsets = new Map<string, number>()
    const targetOffsets = new Map<string, number>()

    sankeyLinks.forEach(link => {
      const sourceOffset = sourceOffsets.get(link.source.name) || 0
      const targetOffset = targetOffsets.get(link.target.name) || 0
      const linkHeight = (link.value / link.source.value) * link.source.height

      link.y0 = link.source.y + sourceOffset + linkHeight / 2
      link.y1 = link.target.y + targetOffset + linkHeight / 2

      sourceOffsets.set(link.source.name, sourceOffset + linkHeight)
      targetOffsets.set(link.target.name, targetOffset + linkHeight)
    })

    return {
      nodes: nodeArray,
      links: sankeyLinks,
      width: chartWidth,
      svgHeight: availableHeight + 40,
    }
  }, [data, columns, height, nodeWidth, nodePadding])

  if (nodes.length === 0) {
    return (
      <Paper p="xl" ta="center" bg="gray.0" radius="md">
        <Text c="dimmed">
          No valid Sankey data found. Expected format: source/target/value columns or multi-step flow columns.
        </Text>
      </Paper>
    )
  }

  // Generate link path
  const generateLinkPath = (link: SankeyLink) => {
    const sourceX = link.source.x + nodeWidth
    const targetX = link.target.x
    const curvature = 0.5
    const xi = (targetX - sourceX) * curvature

    const linkHeight = (link.value / link.source.value) * link.source.height
    const halfHeight = linkHeight / 2

    return `
      M ${sourceX} ${link.y0 - halfHeight}
      C ${sourceX + xi} ${link.y0 - halfHeight},
        ${targetX - xi} ${link.y1 - halfHeight},
        ${targetX} ${link.y1 - halfHeight}
      L ${targetX} ${link.y1 + halfHeight}
      C ${targetX - xi} ${link.y1 + halfHeight},
        ${sourceX + xi} ${link.y0 + halfHeight},
        ${sourceX} ${link.y0 + halfHeight}
      Z
    `
  }

  return (
    <Box style={{ width: '100%', height, overflow: 'auto' }}>
      <svg width={chartWidth} height={svgHeight}>
        {/* Links */}
        <g>
          {links.map((link, i) => (
            <path
              key={i}
              d={generateLinkPath(link)}
              fill={link.source.color}
              fillOpacity={linkOpacity}
              stroke={link.source.color}
              strokeWidth={0.5}
              strokeOpacity={0.3}
            >
              <title>{`${link.source.name} → ${link.target.name}: ${link.value.toLocaleString()}`}</title>
            </path>
          ))}
        </g>

        {/* Nodes */}
        <g>
          {nodes.map((node, i) => (
            <g key={i}>
              <rect
                x={node.x}
                y={node.y}
                width={nodeWidth}
                height={node.height}
                fill={node.color}
                stroke="var(--mantine-color-gray-5)"
                strokeWidth={0.5}
                rx={2}
              >
                <title>{`${node.name}: ${node.value.toLocaleString()}`}</title>
              </rect>
              <text
                x={node.layer === 0 ? node.x - 4 : node.x + nodeWidth + 4}
                y={node.y + node.height / 2}
                textAnchor={node.layer === 0 ? 'end' : 'start'}
                dominantBaseline="middle"
                fontSize={11}
                fill="var(--mantine-color-gray-7)"
              >
                {node.name}
              </text>
            </g>
          ))}
        </g>
      </svg>
    </Box>
  )
}
