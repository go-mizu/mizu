import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class LoginPage {
  constructor(private page: Page) {}

  async goto() {
    await this.page.goto(URLS.login);
  }

  async gotoLegacy() {
    await this.page.goto(URLS.legacy.login);
  }

  async fillCredentials(email: string, password: string) {
    await this.page.fill(SELECTORS.loginUsername, email);
    await this.page.fill(SELECTORS.loginPassword, password);
  }

  async submit() {
    await this.page.click(SELECTORS.loginSubmit);
  }

  async login(email: string, password: string) {
    await this.fillCredentials(email, password);
    await this.submit();
  }

  async expectLoginForm() {
    await expect(this.page.locator(SELECTORS.loginForm)).toBeVisible();
  }

  async expectError() {
    await expect(this.page.locator(SELECTORS.loginError)).toBeVisible();
  }

  async expectErrorMessage(message: string) {
    await expect(this.page.locator(SELECTORS.loginError)).toContainText(message);
  }

  async expectRedirectToDashboard() {
    await this.page.waitForURL(/\/wp-admin\//);
  }
}
