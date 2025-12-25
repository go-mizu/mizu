import { Page, Locator, expect } from '@playwright/test';

export class SettingsPage {
  readonly page: Page;

  // Header
  readonly backButton: Locator;
  readonly heading: Locator;

  // Profile form
  readonly profileForm: Locator;
  readonly avatarPreview: Locator;
  readonly changePhotoButton: Locator;
  readonly displayNameInput: Locator;
  readonly usernameInput: Locator;
  readonly emailInput: Locator;
  readonly bioInput: Locator;
  readonly saveProfileButton: Locator;
  readonly profileMessage: Locator;

  // Privacy settings
  readonly privacyLastSeenSelect: Locator;
  readonly privacyPhotoSelect: Locator;
  readonly readReceiptsToggle: Locator;

  // Notification settings
  readonly notifyMessagesToggle: Locator;
  readonly notifyGroupsToggle: Locator;
  readonly notifySoundToggle: Locator;

  // Appearance
  readonly darkModeToggle: Locator;

  // Password form
  readonly passwordForm: Locator;
  readonly currentPasswordInput: Locator;
  readonly newPasswordInput: Locator;
  readonly confirmPasswordInput: Locator;
  readonly changePasswordButton: Locator;
  readonly passwordMessage: Locator;

  // Danger zone
  readonly logoutButton: Locator;
  readonly deleteAccountButton: Locator;

  constructor(page: Page) {
    this.page = page;

    // Header
    this.backButton = page.locator('a[href="/app"]');
    this.heading = page.locator('h1:has-text("Settings")');

    // Profile form
    this.profileForm = page.locator('#profile-form');
    this.avatarPreview = page.locator('#avatar-preview');
    this.changePhotoButton = page.locator('button:has-text("Change Photo")');
    this.displayNameInput = page.locator('#display_name');
    this.usernameInput = page.locator('#username');
    this.emailInput = page.locator('#email');
    this.bioInput = page.locator('#bio');
    this.saveProfileButton = page.locator('#profile-form button[type="submit"]');
    this.profileMessage = page.locator('#profile-message');

    // Privacy settings
    this.privacyLastSeenSelect = page.locator('#privacy-last-seen');
    this.privacyPhotoSelect = page.locator('#privacy-photo');
    this.readReceiptsToggle = page.locator('#privacy-read-receipts');

    // Notification settings
    this.notifyMessagesToggle = page.locator('#notify-messages');
    this.notifyGroupsToggle = page.locator('#notify-groups');
    this.notifySoundToggle = page.locator('#notify-sound');

    // Appearance
    this.darkModeToggle = page.locator('#dark-mode');

    // Password form
    this.passwordForm = page.locator('#password-form');
    this.currentPasswordInput = page.locator('#current_password');
    this.newPasswordInput = page.locator('#new_password');
    this.confirmPasswordInput = page.locator('#confirm_password');
    this.changePasswordButton = page.locator('#password-form button[type="submit"]');
    this.passwordMessage = page.locator('#password-message');

    // Danger zone
    this.logoutButton = page.locator('button:has-text("Log Out")');
    this.deleteAccountButton = page.locator('button:has-text("Delete Account")');
  }

  async goto(): Promise<void> {
    await this.page.goto('/settings');
  }

  async waitForLoad(): Promise<void> {
    await expect(this.heading).toBeVisible();
    await expect(this.displayNameInput).toBeVisible();
  }

  // Profile
  async updateProfile(data: {
    displayName?: string;
    email?: string;
    bio?: string;
  }): Promise<void> {
    if (data.displayName !== undefined) {
      await this.displayNameInput.fill(data.displayName);
    }
    if (data.email !== undefined) {
      await this.emailInput.fill(data.email);
    }
    if (data.bio !== undefined) {
      await this.bioInput.fill(data.bio);
    }
    await this.saveProfileButton.click();
  }

  async expectProfileSuccess(): Promise<void> {
    await expect(this.profileMessage).toBeVisible();
    await expect(this.profileMessage).toContainText('success');
  }

  async expectProfileError(message?: string): Promise<void> {
    await expect(this.profileMessage).toBeVisible();
    if (message) {
      await expect(this.profileMessage).toContainText(message);
    }
  }

  // Privacy
  async setPrivacyLastSeen(value: 'everyone' | 'contacts' | 'nobody'): Promise<void> {
    await this.privacyLastSeenSelect.selectOption(value);
  }

  async setPrivacyPhoto(value: 'everyone' | 'contacts' | 'nobody'): Promise<void> {
    await this.privacyPhotoSelect.selectOption(value);
  }

  async toggleReadReceipts(): Promise<void> {
    await this.readReceiptsToggle.click();
  }

  // Notifications
  async toggleMessageNotifications(): Promise<void> {
    await this.notifyMessagesToggle.click();
  }

  async toggleGroupNotifications(): Promise<void> {
    await this.notifyGroupsToggle.click();
  }

  async toggleSound(): Promise<void> {
    await this.notifySoundToggle.click();
  }

  // Appearance
  async toggleDarkMode(): Promise<void> {
    await this.darkModeToggle.click();
  }

  // Password
  async changePassword(
    currentPassword: string,
    newPassword: string,
    confirmPassword: string
  ): Promise<void> {
    await this.currentPasswordInput.fill(currentPassword);
    await this.newPasswordInput.fill(newPassword);
    await this.confirmPasswordInput.fill(confirmPassword);
    await this.changePasswordButton.click();
  }

  async expectPasswordSuccess(): Promise<void> {
    await expect(this.passwordMessage).toBeVisible();
    await expect(this.passwordMessage).toContainText('success');
  }

  async expectPasswordError(message?: string): Promise<void> {
    await expect(this.passwordMessage).toBeVisible();
    if (message) {
      await expect(this.passwordMessage).toContainText(message);
    }
  }

  // Logout
  async logout(): Promise<void> {
    await this.logoutButton.click();
    await this.page.waitForURL('/');
  }

  // Delete account
  async deleteAccount(): Promise<void> {
    this.page.on('dialog', async (dialog) => {
      await dialog.accept();
    });
    await this.deleteAccountButton.click();
  }

  // Navigation
  async goBack(): Promise<void> {
    await this.backButton.click();
    await this.page.waitForURL('/app');
  }
}
