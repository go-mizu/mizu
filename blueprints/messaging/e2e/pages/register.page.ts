import { Page, Locator, expect } from '@playwright/test';

export class RegisterPage {
  readonly page: Page;
  readonly displayNameInput: Locator;
  readonly usernameInput: Locator;
  readonly emailInput: Locator;
  readonly passwordInput: Locator;
  readonly submitButton: Locator;
  readonly errorMessage: Locator;
  readonly loginLink: Locator;
  readonly heading: Locator;

  constructor(page: Page) {
    this.page = page;
    this.displayNameInput = page.locator('#display_name');
    this.usernameInput = page.locator('#username');
    this.emailInput = page.locator('#email');
    this.passwordInput = page.locator('#password');
    this.submitButton = page.locator('button[type="submit"]');
    this.errorMessage = page.locator('#error-message');
    this.loginLink = page.locator('a[href="/login"]');
    this.heading = page.locator('h1');
  }

  async goto(): Promise<void> {
    await this.page.goto('/register');
  }

  async register(data: {
    displayName?: string;
    username: string;
    email?: string;
    password: string;
  }): Promise<void> {
    if (data.displayName) {
      await this.displayNameInput.fill(data.displayName);
    }
    await this.usernameInput.fill(data.username);
    if (data.email) {
      await this.emailInput.fill(data.email);
    }
    await this.passwordInput.fill(data.password);
    await this.submitButton.click();
  }

  async expectErrorMessage(message?: string): Promise<void> {
    await expect(this.errorMessage).toBeVisible();
    if (message) {
      await expect(this.errorMessage).toContainText(message);
    }
  }

  async expectSuccessfulRegistration(): Promise<void> {
    await this.page.waitForURL('/app');
  }

  async expectHeading(): Promise<void> {
    await expect(this.heading).toContainText('Create an account');
  }
}
