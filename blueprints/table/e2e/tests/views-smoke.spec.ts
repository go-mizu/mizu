import { test, expect } from '@playwright/test';
import { login, openViewMenu, selectBase, selectTable } from './utils';

async function chooseView(page: any, viewName: string) {
  await openViewMenu(page);
  await page.getByRole('button', { name: viewName }).click();
}

test('switch between seeded views and create list view', async ({ page }) => {
  await login(page);
  await selectBase(page, 'Project Tracker');
  await selectTable(page, 'Tasks');

  await chooseView(page, 'By Status');
  await expect(page.getByRole('button', { name: 'Add card' }).first()).toBeVisible();

  await chooseView(page, 'Calendar');
  await expect(page.getByRole('button', { name: 'Today' })).toBeVisible();
  await expect(page.getByText('Sun')).toBeVisible();

  await chooseView(page, 'Gallery');
  await expect(page.getByText('Add record')).toBeVisible();

  await chooseView(page, 'Timeline');
  await expect(page.getByText('Scale:')).toBeVisible();

  await chooseView(page, 'Submit Task');
  await expect(page.getByRole('button', { name: 'Submit', exact: true })).toBeVisible();
  await expect(page.getByText('Tasks Form')).toBeVisible();

  await openViewMenu(page);
  await page.getByRole('button', { name: 'Create view' }).click();
  await page.getByPlaceholder('View name').fill('List View');
  await page.locator('.modal-content').getByRole('button', { name: 'List' }).click();
  await page.locator('.modal-content').getByRole('button', { name: 'Create view' }).click();

  await expect(page.getByText(/records$/i)).toBeVisible();
});
