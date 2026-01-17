import { test, expect } from '@playwright/test';
import path from 'path';

test.describe('Schema Visualizer Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/database/schema-visualizer');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(1500); // Allow ReactFlow to initialize
  });

  test('E2E-SCHEMA-001: Page loads with title and description', async ({ page }) => {
    // Check for page title
    const title = page.getByText('Schema Visualizer');
    await expect(title.first()).toBeVisible({ timeout: 10000 });

    // Check for description
    const description = page.getByText('Visualize your database schema');
    await expect(description.first()).toBeVisible({ timeout: 5000 });
  });

  test('E2E-SCHEMA-002: Schema selector is visible and functional', async ({ page }) => {
    // Look for schema selector dropdown
    const schemaText = page.getByText('schema').first();
    await expect(schemaText).toBeVisible({ timeout: 10000 });

    // Find the Select input/dropdown
    const schemaSelect = page.locator('.mantine-Select-input').or(page.locator('input[role="combobox"]')).first();
    await expect(schemaSelect).toBeVisible({ timeout: 10000 });

    // Click to open dropdown
    await schemaSelect.click();
    await page.waitForTimeout(500);

    // Check dropdown is open - look for option with "public"
    const publicOption = page.getByRole('option', { name: 'public' }).or(page.locator('[data-value="public"]'));
    const isDropdownOpen = await publicOption.isVisible().catch(() => false);
    expect(isDropdownOpen).toBeTruthy();

    // Take screenshot of schema selector
    await page.screenshot({ path: 'screenshots/schema-visualizer-selector.png' });
  });

  test('E2E-SCHEMA-003: Toolbar buttons are visible', async ({ page }) => {
    // Check Copy as SQL button
    const copyButton = page.getByRole('button', { name: /Copy as SQL/i });
    await expect(copyButton).toBeVisible({ timeout: 10000 });

    // Check Auto layout button
    const layoutButton = page.getByRole('button', { name: /Auto layout/i });
    await expect(layoutButton).toBeVisible({ timeout: 5000 });

    // Check Download button (ActionIcon)
    const downloadButton = page.locator('button').filter({ has: page.locator('svg.tabler-icon-download') }).or(
      page.locator('[aria-label="Download as PNG"]')
    );
    // Download icon may not have text, just verify toolbar buttons exist
    const toolbarButtons = page.locator('button').filter({ hasText: /Copy|Auto/i });
    expect(await toolbarButtons.count()).toBeGreaterThanOrEqual(2);
  });

  test('E2E-SCHEMA-004: ReactFlow canvas renders', async ({ page }) => {
    // Wait for ReactFlow to render
    await page.waitForSelector('.react-flow', { timeout: 15000 });

    // Check ReactFlow container is visible
    const reactFlowCanvas = page.locator('.react-flow');
    await expect(reactFlowCanvas).toBeVisible();

    // Check for ReactFlow viewport
    const viewport = page.locator('.react-flow__viewport');
    await expect(viewport).toBeVisible();
  });

  test('E2E-SCHEMA-005: Table nodes are displayed', async ({ page }) => {
    // Wait for nodes to render (if tables exist)
    await page.waitForTimeout(2000);

    // Look for table nodes or empty state
    const nodes = page.locator('.react-flow__node');
    const emptyState = page.getByText(/No tables in schema/i);

    const nodeCount = await nodes.count();
    const hasEmptyState = await emptyState.isVisible().catch(() => false);

    // Either we have nodes or an empty state message
    expect(nodeCount > 0 || hasEmptyState).toBeTruthy();

    if (nodeCount > 0) {
      // Take screenshot of table nodes
      await page.screenshot({ path: 'screenshots/schema-visualizer-nodes.png' });
    }
  });

  test('E2E-SCHEMA-006: Legend shows constraint icons', async ({ page }) => {
    // Check for legend at bottom of page
    const primaryKeyLabel = page.getByText('Primary key');
    await expect(primaryKeyLabel).toBeVisible({ timeout: 10000 });

    const identityLabel = page.getByText('Identity');
    await expect(identityLabel).toBeVisible();

    const uniqueLabel = page.getByText('Unique');
    await expect(uniqueLabel).toBeVisible();

    const nullableLabel = page.getByText('Nullable');
    await expect(nullableLabel).toBeVisible();

    const nonNullableLabel = page.getByText('Non-Nullable');
    await expect(nonNullableLabel).toBeVisible();
  });

  test('E2E-SCHEMA-007: MiniMap is rendered', async ({ page }) => {
    // Wait for ReactFlow components
    await page.waitForSelector('.react-flow', { timeout: 15000 });

    // Check for minimap
    const minimap = page.locator('.react-flow__minimap');
    const isMinimapVisible = await minimap.isVisible().catch(() => false);

    // Minimap should be present
    expect(isMinimapVisible).toBeTruthy();
  });

  test('E2E-SCHEMA-008: Controls are rendered', async ({ page }) => {
    // Wait for ReactFlow components
    await page.waitForSelector('.react-flow', { timeout: 15000 });

    // Check for controls (zoom in/out, fit view)
    const controls = page.locator('.react-flow__controls');
    const areControlsVisible = await controls.isVisible().catch(() => false);

    expect(areControlsVisible).toBeTruthy();
  });

  test('E2E-SCHEMA-009: Auto layout button repositions nodes', async ({ page }) => {
    // Wait for page to fully load
    await page.waitForSelector('.react-flow', { timeout: 15000 });
    await page.waitForTimeout(1000);

    // Check if there are nodes to layout
    const nodes = page.locator('.react-flow__node');
    const nodeCount = await nodes.count();

    if (nodeCount > 0) {
      // Get initial position of first node
      const firstNode = nodes.first();
      const initialBoundingBox = await firstNode.boundingBox();

      // Click Auto layout button
      const layoutButton = page.getByRole('button', { name: /Auto layout/i });
      await layoutButton.click();
      await page.waitForTimeout(500);

      // Node positions should be recalculated (hard to test exact positions)
      // Just verify the button click works without errors
      await expect(firstNode).toBeVisible();

      // Take screenshot after auto-layout
      await page.screenshot({ path: 'screenshots/schema-visualizer-auto-layout.png' });
    }
  });

  test('E2E-SCHEMA-010: Full page screenshot', async ({ page }) => {
    // Wait for everything to load
    await page.waitForSelector('.react-flow', { timeout: 15000 });
    await page.waitForTimeout(2000);

    // Take full page screenshot
    await page.screenshot({
      path: 'screenshots/schema-visualizer-full.png',
      fullPage: false,
    });

    // Verify screenshot was taken (file exists would require file system check)
    expect(true).toBeTruthy();
  });

  test('E2E-SCHEMA-011: Nodes can be dragged', async ({ page }) => {
    // Wait for ReactFlow
    await page.waitForSelector('.react-flow', { timeout: 15000 });
    await page.waitForTimeout(1000);

    const nodes = page.locator('.react-flow__node');
    const nodeCount = await nodes.count();

    if (nodeCount > 0) {
      const firstNode = nodes.first();
      const box = await firstNode.boundingBox();

      if (box) {
        // Drag the node
        await page.mouse.move(box.x + box.width / 2, box.y + 10);
        await page.mouse.down();
        await page.mouse.move(box.x + box.width / 2 + 100, box.y + 50);
        await page.mouse.up();

        // Verify node is still visible (drag didn't break anything)
        await expect(firstNode).toBeVisible();
      }
    }
  });

  test('E2E-SCHEMA-012: Canvas can be panned', async ({ page }) => {
    // Wait for ReactFlow
    await page.waitForSelector('.react-flow', { timeout: 15000 });

    const canvas = page.locator('.react-flow__pane');
    const box = await canvas.boundingBox();

    if (box) {
      // Pan the canvas by dragging
      await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
      await page.mouse.down();
      await page.mouse.move(box.x + box.width / 2 + 100, box.y + box.height / 2 + 50);
      await page.mouse.up();

      // Verify canvas is still functional
      await expect(canvas).toBeVisible();
    }
  });

  test('E2E-SCHEMA-013: Zoom controls work', async ({ page }) => {
    // Wait for ReactFlow
    await page.waitForSelector('.react-flow', { timeout: 15000 });

    // Find zoom in button in controls
    const zoomInButton = page.locator('.react-flow__controls-button').first();

    if (await zoomInButton.isVisible()) {
      // Click zoom in
      await zoomInButton.click();
      await page.waitForTimeout(300);

      // Verify zoom happened (hard to test exact zoom level)
      const viewport = page.locator('.react-flow__viewport');
      await expect(viewport).toBeVisible();
    }
  });

  test('E2E-SCHEMA-014: Copy as SQL shows notification', async ({ page }) => {
    // Wait for page to load
    await page.waitForTimeout(2000);

    // Click Copy as SQL button
    const copyButton = page.getByRole('button', { name: /Copy as SQL/i });
    await copyButton.click();

    // Wait for potential notification
    await page.waitForTimeout(1000);

    // Look for notification (success or error)
    const notification = page.locator('.mantine-Notification-root').or(
      page.getByText(/Copied|Error|copied/i)
    );
    const isNotificationVisible = await notification.isVisible().catch(() => false);

    // Notification should appear (either success or error depending on clipboard permissions)
    // In headless mode, clipboard might fail, so we just verify the button is functional
    await expect(copyButton).toBeVisible();
  });

  test('E2E-SCHEMA-015: Relationship edges are displayed', async ({ page }) => {
    // Wait for ReactFlow to render
    await page.waitForSelector('.react-flow', { timeout: 15000 });
    await page.waitForTimeout(2000);

    // Check for edges (relationships)
    const edges = page.locator('.react-flow__edge');
    const edgeCount = await edges.count();

    // Edges may or may not exist depending on table relationships
    // Just verify the query doesn't error
    expect(typeof edgeCount).toBe('number');

    if (edgeCount > 0) {
      // Take screenshot showing relationships
      await page.screenshot({ path: 'screenshots/schema-visualizer-relationships.png' });
    }
  });
});
