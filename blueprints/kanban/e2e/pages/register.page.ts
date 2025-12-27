import { type Page, type Locator, expect } from '@playwright/test';

export class RegisterPage {
  readonly page: Page;
  readonly usernameInput: Locator;
  readonly emailInput: Locator;
  readonly displayNameInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;
  readonly errorMessage: Locator;
  readonly loginLink: Locator;
  readonly logo: Locator;

  constructor(page: Page) {
    this.page = page;
    this.usernameInput = page.locator('#username');
    this.emailInput = page.locator('#email');
    this.displayNameInput = page.locator('#display_name');
    this.passwordInput = page.locator('#password');
    this.submitButton = page.locator('button[type="submit"]');
    this.errorMessage = page.locator('.alert-error');
    this.loginLink = page.locator('a[href="/login"]');
    this.logo = page.locator('.auth-logo');
  }

  async goto(): Promise<void> {
    await this.page.goto('/register');
  }

  async register(data: {
    username: string;
    email: string;
    displayName: string;
    password: string;
  }): Promise<void> {
    await this.usernameInput.fill(data.username);
    await this.emailInput.fill(data.email);
    await this.displayNameInput.fill(data.displayName);
    await this.passwordInput.fill(data.password);
    await this.submitButton.click();
  }

  async expectSuccessfulRegistration(): Promise<void> {
    await this.page.waitForURL(/\/(app|acme)/);
  }

  async expectError(message?: string): Promise<void> {
    await expect(this.errorMessage).toBeVisible();
    if (message) {
      await expect(this.errorMessage).toContainText(message);
    }
  }

  async expectToBeOnRegisterPage(): Promise<void> {
    await expect(this.logo).toContainText('Kanban');
    await expect(this.usernameInput).toBeVisible();
    await expect(this.emailInput).toBeVisible();
  }
}
