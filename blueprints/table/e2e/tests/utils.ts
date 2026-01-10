import { expect, type Page } from '@playwright/test';

const DEFAULT_USER = {
  email: 'alice@example.com',
  password: 'password123',
};

export async function login(page: Page, user = DEFAULT_USER) {
  await page.goto('/');
  await page.getByPlaceholder('you@example.com').fill(user.email);
  await page.getByPlaceholder('********').fill(user.password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByText('Bases', { exact: true })).toBeVisible();
}

export async function openViewMenu(page: Page) {
  await page.getByTestId('view-selector').click();
}

export async function selectBase(page: Page, baseName: string) {
  const baseButton = page.getByRole('button', { name: baseName }).first();
  await expect(baseButton).toBeVisible();
  await baseButton.click();
}

export async function selectTable(page: Page, tableName: string) {
  const tableButton = page.getByRole('button', { name: tableName, exact: true });
  await expect(tableButton).toBeVisible();
  await tableButton.click();
}
