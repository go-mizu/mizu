import { test, expect, testUsers, generateTestUsername } from '../fixtures/test-fixtures';
import { SettingsPage } from '../pages/settings.page';
import { AppPage } from '../pages/app.page';
import { LoginPage } from '../pages/login.page';

test.describe('Settings & Profile', () => {
  test.describe('Settings Page Access', () => {
    test('settings page loads when authenticated', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      await expect(settingsPage.heading).toBeVisible();
    });

    test('settings page redirects when not authenticated', async ({ page }) => {
      await page.context().clearCookies();
      await page.goto('/settings');
      await page.waitForURL('/login');
    });
  });

  test.describe('Profile Management', () => {
    test('TC-SET-001: profile form shows current user data', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Username should be populated and readonly
      await expect(settingsPage.usernameInput).toHaveValue(testUsers.alice.username);
      await expect(settingsPage.usernameInput).toHaveAttribute('readonly', '');

      // Avatar should show initial
      await expect(settingsPage.avatarPreview).not.toBeEmpty();
    });

    test('TC-SET-002: can update display name and bio', async ({ page, registerUser }) => {
      // Register a fresh user
      const username = generateTestUsername();
      await registerUser({
        username,
        password: 'password123',
      });

      await page.waitForURL('/app');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Update profile
      const newDisplayName = 'Updated Name';
      const newBio = 'This is my updated bio';

      await settingsPage.updateProfile({
        displayName: newDisplayName,
        bio: newBio,
      });

      // Wait for success message
      await settingsPage.expectProfileSuccess();

      // Refresh and verify persistence
      await page.reload();
      await settingsPage.waitForLoad();

      await expect(settingsPage.displayNameInput).toHaveValue(newDisplayName);
      await expect(settingsPage.bioInput).toHaveValue(newBio);
    });

    test('username field is readonly', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Try to change username
      await expect(settingsPage.usernameInput).toHaveAttribute('readonly', '');

      // Info text should mention username cannot be changed
      await expect(page.locator('text=Username cannot be changed')).toBeVisible();
    });
  });

  test.describe('Privacy Settings', () => {
    test('TC-SET-003: privacy settings can be changed', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Change last seen privacy
      await settingsPage.setPrivacyLastSeen('nobody');

      // Change profile photo privacy
      await settingsPage.setPrivacyPhoto('contacts');

      // Refresh and verify persistence (saved to localStorage)
      await page.reload();
      await settingsPage.waitForLoad();

      await expect(settingsPage.privacyLastSeenSelect).toHaveValue('nobody');
      await expect(settingsPage.privacyPhotoSelect).toHaveValue('contacts');
    });

    test('read receipts can be toggled', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Get initial state
      const initialChecked = await settingsPage.readReceiptsToggle.isChecked();

      // Toggle
      await settingsPage.toggleReadReceipts();

      // Verify toggled
      const newChecked = await settingsPage.readReceiptsToggle.isChecked();
      expect(newChecked).toBe(!initialChecked);
    });
  });

  test.describe('Notification Settings', () => {
    test('TC-SET-004: notification settings can be changed', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Toggle message notifications off
      const initialMessages = await settingsPage.notifyMessagesToggle.isChecked();
      await settingsPage.toggleMessageNotifications();
      const newMessages = await settingsPage.notifyMessagesToggle.isChecked();
      expect(newMessages).toBe(!initialMessages);

      // Toggle sound
      const initialSound = await settingsPage.notifySoundToggle.isChecked();
      await settingsPage.toggleSound();
      const newSound = await settingsPage.notifySoundToggle.isChecked();
      expect(newSound).toBe(!initialSound);
    });

    test('notification settings persist after reload', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Turn off all notifications
      if (await settingsPage.notifyMessagesToggle.isChecked()) {
        await settingsPage.toggleMessageNotifications();
      }
      if (await settingsPage.notifyGroupsToggle.isChecked()) {
        await settingsPage.toggleGroupNotifications();
      }

      // Reload
      await page.reload();
      await settingsPage.waitForLoad();

      // Should still be off
      expect(await settingsPage.notifyMessagesToggle.isChecked()).toBe(false);
      expect(await settingsPage.notifyGroupsToggle.isChecked()).toBe(false);
    });
  });

  test.describe('Appearance Settings', () => {
    test('TC-SET-005: dark mode toggle works', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Toggle dark mode
      const initialDark = await settingsPage.darkModeToggle.isChecked();
      await settingsPage.toggleDarkMode();

      // Theme should change
      const html = page.locator('html');
      if (initialDark) {
        await expect(html).toHaveAttribute('data-theme', 'light');
      } else {
        await expect(html).toHaveAttribute('data-theme', 'dark');
      }
    });

    test('theme preference persists', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Set to light mode
      if (await settingsPage.darkModeToggle.isChecked()) {
        await settingsPage.toggleDarkMode();
      }

      // Reload
      await page.reload();
      await settingsPage.waitForLoad();

      // Should still be light
      expect(await settingsPage.darkModeToggle.isChecked()).toBe(false);
    });
  });

  test.describe('Password Change', () => {
    test('TC-SET-006: can change password successfully', async ({ page, registerUser }) => {
      const username = generateTestUsername();
      await registerUser({
        username,
        password: 'password123',
      });

      await page.waitForURL('/app');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Change password
      await settingsPage.changePassword('password123', 'newpassword456', 'newpassword456');

      await settingsPage.expectPasswordSuccess();

      // Logout and login with new password
      await settingsPage.logout();

      const loginPage = new LoginPage(page);
      await loginPage.goto();
      await loginPage.login(username, 'newpassword456');
      await loginPage.expectSuccessfulLogin();
    });

    test('TC-SET-007: password change fails with mismatched confirmation', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Try with mismatched passwords
      await settingsPage.changePassword('password123', 'newpass1', 'newpass2');

      await settingsPage.expectPasswordError('match');
    });

    test('password fields are required', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Password fields should have minlength
      await expect(settingsPage.newPasswordInput).toHaveAttribute('minlength', '6');
      await expect(settingsPage.confirmPasswordInput).toHaveAttribute('minlength', '6');
    });
  });

  test.describe('Account Danger Zone', () => {
    test('TC-AUTH-008: logout works', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      await settingsPage.logout();

      await expect(page).toHaveURL('/');
    });

    test('TC-SET-008: account deletion works', async ({ page, registerUser }) => {
      // Create a throwaway user
      const username = generateTestUsername();
      await registerUser({
        username,
        password: 'password123',
      });

      await page.waitForURL('/app');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Delete account (page object handles dialogs)
      await settingsPage.deleteAccount();

      // Wait for redirect
      await page.waitForTimeout(2000);

      // Should be redirected to home
      await expect(page).toHaveURL('/');

      // Try to login with deleted account - should fail
      const loginPage = new LoginPage(page);
      await loginPage.goto();
      await loginPage.login(username, 'password123');
      await loginPage.expectErrorMessage();
    });

    test('delete account requires confirmation', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Listen for dialog and dismiss it
      let dialogCount = 0;
      page.on('dialog', async (dialog) => {
        dialogCount++;
        await dialog.dismiss();
      });

      await settingsPage.deleteAccountButton.click();

      // Should have shown at least one confirmation dialog
      await page.waitForTimeout(500);
      expect(dialogCount).toBeGreaterThanOrEqual(1);
    });
  });

  test.describe('Navigation', () => {
    test('back button returns to app', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      await settingsPage.goBack();

      await expect(page).toHaveURL('/app');
    });
  });

  test.describe('Blocked Contacts', () => {
    test('blocked contacts section is visible', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Blocked contacts section should exist
      await expect(page.locator('text=Blocked Contacts')).toBeVisible();
    });

    test('shows "No blocked contacts" when empty', async ({ page, loginAs }) => {
      await loginAs('alice');

      const settingsPage = new SettingsPage(page);
      await settingsPage.goto();
      await settingsPage.waitForLoad();

      // Look for the no blocked message (unless user has blocked someone)
      const noBlockedText = page.locator('#no-blocked');
      const blockedList = page.locator('#blocked-list');

      const listContent = await blockedList.textContent();
      if (listContent?.trim() === '') {
        await expect(noBlockedText).toBeVisible();
      }
    });
  });
});
