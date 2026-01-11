import { test, expect } from '@playwright/test';
import { login, openViewMenu, selectBase } from './utils';

test('create base/table and edit records in grid view', async ({ page }) => {
  await login(page);

  const baseName = `E2E Base ${Date.now()}`;

  await page.getByRole('button', { name: '+ New' }).click();
  await page.getByPlaceholder('Base name').fill(baseName);
  await page.getByRole('button', { name: 'Create' }).click();
  await expect(page.getByRole('button', { name: baseName }).first()).toBeVisible();

  await selectBase(page, 'Project Tracker');
  await expect(page.getByRole('button', { name: 'Add table' })).toBeVisible();

  await page.getByRole('button', { name: 'Add table' }).click();
  await page.getByPlaceholder('Table name').fill('E2E Table');
  await page.getByRole('button', { name: 'Create' }).click();

  await openViewMenu(page);
  await page.getByRole('button', { name: 'Create view' }).click();
  await page.getByPlaceholder('View name').fill('Main Grid');
  await page.locator('.modal-content').getByRole('button', { name: 'Create view' }).click();

  await page.getByRole('button', { name: 'Add field' }).click();
  await page.getByRole('button', { name: /Single line text/i }).click();
  await page.getByPlaceholder('Enter field name').fill('Title');
  await page.getByRole('button', { name: 'Create field' }).click();

  await page.getByRole('button', { name: 'Add field' }).click();
  await page.getByRole('button', { name: 'Date Calendar date' }).click();
  await page.getByPlaceholder('Enter field name').fill('Due Date');
  await page.getByRole('button', { name: 'Create field' }).click();

  await page.getByRole('button', { name: 'Add row' }).click();
  await page.getByRole('button', { name: 'Add row' }).click();

  const rows = page.locator('table tbody tr');
  const firstRecordRow = rows.nth(0);
  const secondRecordRow = rows.nth(1);

  const firstTitleCell = firstRecordRow.locator('td').nth(1);
  await firstTitleCell.dblclick();
  await expect(firstTitleCell.locator('input')).toBeVisible();
  await firstTitleCell.locator('input').fill('Beta');
  await firstTitleCell.locator('input').press('Enter');

  const secondTitleCell = secondRecordRow.locator('td').nth(1);
  await secondTitleCell.dblclick();
  await expect(secondTitleCell.locator('input')).toBeVisible();
  await secondTitleCell.locator('input').fill('Alpha');
  await secondTitleCell.locator('input').press('Enter');

  await page.getByRole('button', { name: 'Sort' }).click();
  await page.getByRole('button', { name: 'Add sort' }).click();
  await page.getByRole('button', { name: 'Apply sorts' }).click();
  await expect(firstTitleCell).toContainText('Alpha');

  await page.getByRole('button', { name: 'Filter' }).click();
  await page.getByRole('button', { name: 'Add filter' }).click();
  await page.getByPlaceholder('Value').fill('Alpha');
  await page.getByRole('button', { name: 'Apply filters' }).click();
  await expect(page.getByText('Beta')).toHaveCount(0);

  await firstRecordRow.hover();
  await firstRecordRow.locator('button').first().click();
  await page.getByRole('button', { name: /Comments/ }).click();
  await page.getByPlaceholder('Add a comment...').fill('First comment');
  await page.getByRole('button', { name: 'Post' }).click();
  await expect(page.getByText('First comment')).toBeVisible();
});
