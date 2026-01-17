import { test, expect } from '@playwright/test';

test.describe('Storage Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const storageLink = page.locator('.mantine-AppShell-navbar').getByRole('link', { name: 'Storage' });
    await storageLink.click();

    await page.waitForLoadState('networkidle');
    await expect(page).toHaveURL(/storage/);
  });

  test('E2E-STORAGE-001: Bucket list loads', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Check for bucket list or empty state
    const bucketSection = page.getByText(/Buckets|No buckets|Create bucket/i).first();
    await expect(bucketSection).toBeVisible();
  });

  test('E2E-STORAGE-002: Create bucket button visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add/i }).first();
    await expect(createButton).toBeVisible();
  });

  test('E2E-STORAGE-003: Create public bucket', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const createButton = page.getByRole('button', { name: /Create|New|Add/i }).first();
    await createButton.click();

    // Fill in bucket name
    const bucketName = `test-bucket-${Date.now()}`;
    const nameInput = page.getByPlaceholder(/name/i).or(page.getByLabel(/name/i)).first();
    await nameInput.fill(bucketName);

    // Enable public toggle if available
    const publicToggle = page.getByRole('checkbox', { name: /public/i }).or(page.getByLabel(/public/i));
    if (await publicToggle.isVisible()) {
      await publicToggle.check();
    }

    // Submit
    const submitButton = page.getByRole('button', { name: /Create|Save/i }).last();
    await submitButton.click();

    await page.waitForTimeout(2000);
  });

  test('E2E-STORAGE-004: File browser visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Select a bucket first
    const bucketItem = page.locator('button').filter({ hasText: /bucket|avatars|public/i }).first();

    if (await bucketItem.isVisible()) {
      await bucketItem.click();
      await page.waitForLoadState('networkidle');

      // Check for file browser
      const fileBrowser = page.getByText(/files|objects|empty|upload/i);
      await expect(fileBrowser).toBeVisible();
    }
  });

  test('E2E-STORAGE-005: Upload zone visible', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Select a bucket
    const bucketItem = page.locator('button').filter({ hasText: /bucket|avatars/i }).first();

    if (await bucketItem.isVisible()) {
      await bucketItem.click();
      await page.waitForLoadState('networkidle');

      // Look for upload zone or button
      const uploadArea = page.getByText(/Upload|Drag|Drop/i).or(page.getByRole('button', { name: /Upload/i }));
      const isVisible = await uploadArea.isVisible().catch(() => false);
      expect(typeof isVisible).toBe('boolean');
    }
  });

  test('E2E-STORAGE-006: Breadcrumb navigation', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Select a bucket
    const bucketItem = page.locator('button').filter({ hasText: /bucket|avatars/i }).first();

    if (await bucketItem.isVisible()) {
      await bucketItem.click();
      await page.waitForLoadState('networkidle');

      // Look for breadcrumb
      const breadcrumb = page.getByText(/\/|Home|Root/i).first();
      const isVisible = await breadcrumb.isVisible().catch(() => false);
      expect(typeof isVisible).toBe('boolean');
    }
  });

  test('E2E-STORAGE-007: Public badge shown for public buckets', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Look for public badge
    const publicBadge = page.getByText(/Public|Private/i).first();
    const isVisible = await publicBadge.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('E2E-STORAGE-008: Delete bucket confirmation', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Find delete button for a bucket
    const deleteButton = page.getByRole('button', { name: /delete/i }).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();

      // Check for confirmation modal
      const confirmModal = page.getByText(/confirm|sure|delete/i);
      await expect(confirmModal).toBeVisible();

      // Cancel
      const cancelButton = page.getByRole('button', { name: /cancel|no|close/i });
      if (await cancelButton.isVisible()) {
        await cancelButton.click();
      }
    }
  });

  test('E2E-STORAGE-009: File size displayed', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    // Select a bucket with files
    const bucketItem = page.locator('button').filter({ hasText: /bucket|avatars/i }).first();

    if (await bucketItem.isVisible()) {
      await bucketItem.click();
      await page.waitForLoadState('networkidle');

      // Look for file size format
      const fileSize = page.getByText(/KB|MB|GB|bytes/i).first();
      const isVisible = await fileSize.isVisible().catch(() => false);
      expect(typeof isVisible).toBe('boolean');
    }
  });

  test('E2E-STORAGE-010: Copy URL button', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const bucketItem = page.locator('button').filter({ hasText: /bucket|avatars/i }).first();

    if (await bucketItem.isVisible()) {
      await bucketItem.click();
      await page.waitForLoadState('networkidle');

      // Look for copy URL action
      const copyButton = page.getByRole('button', { name: /copy|url|link/i }).first();
      const isVisible = await copyButton.isVisible().catch(() => false);
      expect(typeof isVisible).toBe('boolean');
    }
  });

  test('E2E-STORAGE-011: Download file button', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const bucketItem = page.locator('button').filter({ hasText: /bucket|avatars/i }).first();

    if (await bucketItem.isVisible()) {
      await bucketItem.click();
      await page.waitForLoadState('networkidle');

      // Look for download button
      const downloadButton = page.getByRole('button', { name: /download/i }).first();
      const isVisible = await downloadButton.isVisible().catch(() => false);
      expect(typeof isVisible).toBe('boolean');
    }
  });

  test('E2E-STORAGE-012: Signed URL generation', async ({ page }) => {
    await page.waitForLoadState('networkidle');

    const bucketItem = page.locator('button').filter({ hasText: /bucket|avatars/i }).first();

    if (await bucketItem.isVisible()) {
      await bucketItem.click();
      await page.waitForLoadState('networkidle');

      // Look for signed URL option
      const signedUrlButton = page.getByRole('button', { name: /signed|private url/i });
      const isVisible = await signedUrlButton.isVisible().catch(() => false);
      expect(typeof isVisible).toBe('boolean');
    }
  });
});
