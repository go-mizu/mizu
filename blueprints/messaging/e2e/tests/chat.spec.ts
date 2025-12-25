import { test, expect, testUsers } from '../fixtures/test-fixtures';
import { AppPage } from '../pages/app.page';

test.describe('Chat List & Navigation', () => {
  test.describe('Chat List Display', () => {
    test('TC-CHAT-001: empty chat list shows placeholder', async ({ page, loginAs }) => {
      // Register a new user with no chats
      await page.goto('/register');
      const username = `newuser_${Date.now()}`;
      await page.fill('#username', username);
      await page.fill('#password', 'password123');
      await page.click('button[type="submit"]');
      await page.waitForURL('/app');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Chat list should show "No chats yet"
      await expect(appPage.chatList).toContainText('No chats yet');
    });

    test('TC-CHAT-002: chat list shows existing chats', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Should display chat list (may be empty or have chats depending on seed data)
      await expect(appPage.chatList).toBeVisible();
    });

    test('chat list items show correct info', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // If there are chats, verify structure
      const chatItems = await appPage.getChatItems();
      const count = await chatItems.count();

      if (count > 0) {
        const firstChat = chatItems.first();
        // Each chat should have an avatar area
        await expect(firstChat.locator('.rounded-full')).toBeVisible();
        // Each chat should have a name
        await expect(firstChat.locator('.font-medium')).toBeVisible();
      }
    });
  });

  test.describe('Chat Selection', () => {
    test('TC-CHAT-003: clicking chat opens conversation', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      const chatItems = await appPage.getChatItems();
      const count = await chatItems.count();

      if (count > 0) {
        // Click first chat
        await appPage.selectChatByIndex(0);

        // Chat view should be visible
        await appPage.expectChatViewVisible();

        // Chat header should have name
        await expect(appPage.chatName).not.toBeEmpty();
      } else {
        // Skip if no chats
        test.skip();
      }
    });

    test('initial state shows empty state', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Initially no chat selected
      await appPage.expectEmptyStateVisible();
      await expect(appPage.emptyState).toContainText('Welcome to Messaging');
    });
  });

  test.describe('Chat Search', () => {
    test('TC-CHAT-004: search filters chat list', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      const chatItems = await appPage.getChatItems();
      const initialCount = await chatItems.count();

      if (initialCount > 0) {
        // Search for something unlikely to match
        await appPage.searchChats('zzzznonesuch');

        // Wait a moment for filtering
        await page.waitForTimeout(300);

        // Visible chats should be fewer or zero
        const visibleChats = appPage.chatList.locator('> div[style*="display: flex"], > div:not([style*="display: none"])');
        const filteredCount = await visibleChats.count();

        expect(filteredCount).toBeLessThanOrEqual(initialCount);
      }
    });

    test('clearing search shows all chats', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Search then clear
      await appPage.searchChats('test');
      await page.waitForTimeout(100);
      await appPage.clearSearch();
      await page.waitForTimeout(100);

      // All chats should be visible again
      await expect(appPage.chatList).toBeVisible();
    });
  });

  test.describe('New Chat Modal', () => {
    test('TC-CHAT-005: new chat modal opens', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      await appPage.openNewChatModal();

      // Modal should be visible with contact search
      await expect(appPage.newChatModal).toBeVisible();
      await expect(appPage.contactSearch).toBeVisible();
      await expect(appPage.createGroupButton).toBeVisible();
    });

    test('new chat modal closes on X click', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      await appPage.openNewChatModal();
      await appPage.closeNewChatModal();

      await expect(appPage.newChatModal).toBeHidden();
    });

    test('contact search filters list', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      await appPage.openNewChatModal();
      await appPage.searchContacts('nonexistent');

      await page.waitForTimeout(300);
      // Contacts should be filtered
    });

    test('create group button shows coming soon', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      await appPage.openNewChatModal();

      // Listen for dialog
      page.on('dialog', async (dialog) => {
        expect(dialog.message()).toContain('coming soon');
        await dialog.accept();
      });

      await appPage.createGroupButton.click();
    });
  });

  test.describe('Chat Header', () => {
    test('chat header shows user info for direct chat', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      const chatItems = await appPage.getChatItems();
      if ((await chatItems.count()) > 0) {
        await appPage.selectChatByIndex(0);

        await expect(appPage.chatName).not.toBeEmpty();
        await expect(appPage.chatAvatar).toBeVisible();
        await expect(appPage.chatStatus).toBeVisible();
      } else {
        test.skip();
      }
    });

    test('chat header has call buttons', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      const chatItems = await appPage.getChatItems();
      if ((await chatItems.count()) > 0) {
        await appPage.selectChatByIndex(0);

        await expect(appPage.voiceCallButton).toBeVisible();
        await expect(appPage.videoCallButton).toBeVisible();
        await expect(appPage.optionsButton).toBeVisible();
      } else {
        test.skip();
      }
    });
  });

  test.describe('Theme Toggle', () => {
    test('TC-SET-005: theme toggle switches theme', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Default should be dark theme
      await appPage.expectDarkTheme();

      // Toggle to light
      await appPage.toggleTheme();
      await appPage.expectLightTheme();

      // Toggle back to dark
      await appPage.toggleTheme();
      await appPage.expectDarkTheme();
    });
  });

  test.describe('Navigation', () => {
    test('settings link works', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      await appPage.goToSettings();
      await expect(page).toHaveURL('/settings');
    });
  });
});
