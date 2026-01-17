import { useEffect, useState, useCallback, useRef } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
  type Node,
  type Edge,
  type NodeTypes,
  ReactFlowProvider,
  useReactFlow,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import dagre from 'dagre';
import { toPng } from 'html-to-image';
import {
  Box,
  Button,
  Group,
  Select,
  Text,
  Loader,
  Center,
  ActionIcon,
  Tooltip,
} from '@mantine/core';
import {
  IconDownload,
  IconCopy,
  IconLayoutDistributeHorizontal,
  IconKey,
  IconHash,
  IconFingerprint,
  IconDiamond,
  IconDiamondFilled,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../../components/layout/PageContainer';
import { databaseApi } from '../../../api';
import { pgmetaApi } from '../../../api/pgmeta';
import TableNode from './TableNode';

const NODE_WIDTH = 280;
const NODE_HEIGHT_BASE = 50;
const NODE_HEIGHT_PER_COLUMN = 28;

interface SchemaTable {
  id: number;
  schema: string;
  name: string;
  comment?: string;
  columns: Array<{
    name: string;
    type: string;
    is_nullable: boolean;
    is_primary_key: boolean;
    is_unique: boolean;
    is_identity: boolean;
  }>;
}

interface SchemaRelationship {
  id: number;
  source_schema: string;
  source_table: string;
  source_columns: string[];
  target_schema: string;
  target_table: string;
  target_columns: string[];
  constraint_name: string;
}

const nodeTypes: NodeTypes = {
  tableNode: TableNode,
};

// Dagre layout algorithm
function getLayoutedElements(
  nodes: Node[],
  edges: Edge[],
  direction: 'TB' | 'LR' = 'LR'
): { nodes: Node[]; edges: Edge[] } {
  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));
  dagreGraph.setGraph({ rankdir: direction, nodesep: 80, ranksep: 150 });

  nodes.forEach((node) => {
    const data = node.data as { columns?: unknown[] };
    const columnsLen = data?.columns?.length ?? 0;
    const height = NODE_HEIGHT_BASE + columnsLen * NODE_HEIGHT_PER_COLUMN;
    dagreGraph.setNode(node.id, { width: NODE_WIDTH, height });
  });

  edges.forEach((edge) => {
    dagreGraph.setEdge(edge.source, edge.target);
  });

  dagre.layout(dagreGraph);

  const layoutedNodes = nodes.map((node) => {
    const nodeWithPosition = dagreGraph.node(node.id);
    const data = node.data as { columns?: unknown[] };
    const columnsLen = data?.columns?.length ?? 0;
    const height = NODE_HEIGHT_BASE + columnsLen * NODE_HEIGHT_PER_COLUMN;

    return {
      ...node,
      position: {
        x: nodeWithPosition.x - NODE_WIDTH / 2,
        y: nodeWithPosition.y - height / 2,
      },
    };
  });

  return { nodes: layoutedNodes, edges };
}

function SchemaVisualizerCanvas() {
  const [schemas, setSchemas] = useState<string[]>([]);
  const [selectedSchema, setSelectedSchema] = useState('public');
  const [tables, setTables] = useState<SchemaTable[]>([]);
  const [relationships, setRelationships] = useState<SchemaRelationship[]>([]);
  const [loading, setLoading] = useState(true);
  const [nodes, setNodes, onNodesChange] = useNodesState([] as Node[]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([] as Edge[]);
  const { fitView, getNodes } = useReactFlow();
  const flowRef = useRef<HTMLDivElement>(null);

  // Fetch schemas
  useEffect(() => {
    databaseApi.listSchemas().then((data) => {
      setSchemas(data ?? []);
    });
  }, []);

  // Fetch schema visualization data
  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const data = await pgmetaApi.getSchemaVisualization(selectedSchema);
      setTables(data.tables ?? []);
      setRelationships(data.relationships ?? []);
    } catch (error: unknown) {
      const err = error as Error;
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to load schema data',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [selectedSchema]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // Convert tables and relationships to nodes and edges
  useEffect(() => {
    if (tables.length === 0) {
      setNodes([]);
      setEdges([]);
      return;
    }

    // Create nodes
    const initialNodes: Node[] = tables.map((table, index) => ({
      id: `${table.schema}.${table.name}`,
      type: 'tableNode',
      position: { x: (index % 4) * 350, y: Math.floor(index / 4) * 400 },
      data: {
        label: table.name,
        schema: table.schema,
        columns: table.columns,
      },
    }));

    // Create edges for relationships
    const initialEdges: Edge[] = relationships.map((rel) => ({
      id: `${rel.constraint_name}`,
      source: `${rel.source_schema}.${rel.source_table}`,
      target: `${rel.target_schema}.${rel.target_table}`,
      sourceHandle: `${rel.source_columns[0]}-source`,
      targetHandle: `${rel.target_columns[0]}-target`,
      type: 'smoothstep',
      animated: false,
      style: {
        stroke: 'var(--supabase-border)',
        strokeWidth: 1.5,
        strokeDasharray: '5,5',
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        width: 15,
        height: 15,
        color: 'var(--supabase-border)',
      },
      label: rel.constraint_name,
      labelStyle: { fontSize: 10, fill: 'var(--mantine-color-dimmed)' },
      labelBgStyle: { fill: 'var(--supabase-bg-surface)', fillOpacity: 0.8 },
    }));

    // Apply auto-layout
    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
      initialNodes,
      initialEdges,
      'LR'
    );

    setNodes(layoutedNodes);
    setEdges(layoutedEdges);

    // Fit view after layout
    setTimeout(() => fitView({ padding: 0.2 }), 100);
  }, [tables, relationships, setNodes, setEdges, fitView]);

  // Auto-layout handler
  const handleAutoLayout = useCallback(() => {
    const { nodes: layoutedNodes, edges: layoutedEdges } = getLayoutedElements(
      getNodes(),
      edges,
      'LR'
    );
    setNodes(layoutedNodes);
    setEdges(layoutedEdges);
    setTimeout(() => fitView({ padding: 0.2 }), 100);
  }, [edges, getNodes, setNodes, setEdges, fitView]);

  // Copy as SQL handler
  const handleCopySQL = useCallback(async () => {
    try {
      const sql = await pgmetaApi.getSchemaSQL(selectedSchema);
      await navigator.clipboard.writeText(sql);
      notifications.show({
        title: 'Copied',
        message: 'SQL schema copied to clipboard',
        color: 'green',
      });
    } catch (error: unknown) {
      const err = error as Error;
      notifications.show({
        title: 'Error',
        message: err.message || 'Failed to copy SQL',
        color: 'red',
      });
    }
  }, [selectedSchema]);

  // Export as PNG handler
  const handleExportPNG = useCallback(() => {
    if (!flowRef.current) return;

    const flowElement = flowRef.current.querySelector('.react-flow__viewport');
    if (!flowElement) return;

    toPng(flowElement as HTMLElement, {
      backgroundColor: 'var(--supabase-bg)',
      width: flowElement.scrollWidth,
      height: flowElement.scrollHeight,
    })
      .then((dataUrl) => {
        const link = document.createElement('a');
        link.download = `${selectedSchema}-schema.png`;
        link.href = dataUrl;
        link.click();
        notifications.show({
          title: 'Exported',
          message: 'Schema diagram exported as PNG',
          color: 'green',
        });
      })
      .catch(() => {
        notifications.show({
          title: 'Error',
          message: 'Failed to export image',
          color: 'red',
        });
      });
  }, [selectedSchema]);

  return (
    <PageContainer
      title="Schema Visualizer"
      description="Visualize your database schema as an interactive diagram"
      fullWidth
      noPadding
    >
      <Box style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 140px)' }}>
        {/* Toolbar */}
        <Box
          px="md"
          py="xs"
          style={{
            borderBottom: '1px solid var(--supabase-border)',
            backgroundColor: 'var(--supabase-bg-surface)',
          }}
        >
          <Group justify="space-between">
            <Group gap="md">
              <Text size="sm" c="dimmed">
                schema
              </Text>
              <Select
                size="xs"
                value={selectedSchema}
                onChange={(value) => value && setSelectedSchema(value)}
                data={schemas.map((s) => ({ value: s, label: s }))}
                style={{ width: 150 }}
              />
            </Group>
            <Group gap="xs">
              <Tooltip label="Copy as SQL">
                <Button
                  variant="default"
                  size="xs"
                  leftSection={<IconCopy size={14} />}
                  onClick={handleCopySQL}
                >
                  Copy as SQL
                </Button>
              </Tooltip>
              <Tooltip label="Download as PNG">
                <ActionIcon variant="default" size="md" onClick={handleExportPNG}>
                  <IconDownload size={16} />
                </ActionIcon>
              </Tooltip>
              <Button
                variant="default"
                size="xs"
                leftSection={<IconLayoutDistributeHorizontal size={14} />}
                onClick={handleAutoLayout}
              >
                Auto layout
              </Button>
            </Group>
          </Group>
        </Box>

        {/* Canvas */}
        <Box ref={flowRef} style={{ flex: 1, position: 'relative' }}>
          {loading ? (
            <Center style={{ height: '100%' }}>
              <Loader size="lg" />
            </Center>
          ) : tables.length === 0 ? (
            <Center style={{ height: '100%' }}>
              <Text c="dimmed">No tables in schema "{selectedSchema}"</Text>
            </Center>
          ) : (
            <ReactFlow
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              nodeTypes={nodeTypes}
              fitView
              minZoom={0.1}
              maxZoom={2}
              defaultEdgeOptions={{
                type: 'smoothstep',
              }}
              proOptions={{ hideAttribution: true }}
            >
              <Background color="var(--supabase-border)" gap={20} size={1} />
              <Controls />
              <MiniMap
                nodeColor="var(--supabase-brand)"
                maskColor="rgba(0, 0, 0, 0.3)"
                style={{ background: 'var(--supabase-bg-surface)' }}
              />
            </ReactFlow>
          )}
        </Box>

        {/* Legend */}
        <Box
          px="md"
          py="xs"
          style={{
            borderTop: '1px solid var(--supabase-border)',
            backgroundColor: 'var(--supabase-bg-surface)',
          }}
        >
          <Group gap="lg" justify="center">
            <Group gap={6}>
              <IconKey size={14} color="var(--supabase-brand)" />
              <Text size="xs" c="dimmed">
                Primary key
              </Text>
            </Group>
            <Group gap={6}>
              <IconHash size={14} color="#666" />
              <Text size="xs" c="dimmed">
                Identity
              </Text>
            </Group>
            <Group gap={6}>
              <IconFingerprint size={14} color="#666" />
              <Text size="xs" c="dimmed">
                Unique
              </Text>
            </Group>
            <Group gap={6}>
              <IconDiamond size={14} color="#666" />
              <Text size="xs" c="dimmed">
                Nullable
              </Text>
            </Group>
            <Group gap={6}>
              <IconDiamondFilled size={14} color="#666" />
              <Text size="xs" c="dimmed">
                Non-Nullable
              </Text>
            </Group>
          </Group>
        </Box>
      </Box>
    </PageContainer>
  );
}

export function SchemaVisualizerPage() {
  return (
    <ReactFlowProvider>
      <SchemaVisualizerCanvas />
    </ReactFlowProvider>
  );
}
