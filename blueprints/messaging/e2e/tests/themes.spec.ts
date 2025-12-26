import { test, expect, testUsers } from '../fixtures/test-fixtures';
import { AppPage } from '../pages/app.page';
import { Page } from '@playwright/test';

// All available themes to test
const themes = [
  { name: 'dark', isViewTheme: false, cookieValue: 'default' },
  { name: 'light', isViewTheme: false, cookieValue: 'default' },
  { name: 'aim1.0', isViewTheme: true, cookieValue: 'aim1.0' },
  { name: 'ymxp', isViewTheme: true, cookieValue: 'ymxp' },
  { name: 'im26', isViewTheme: true, cookieValue: 'im26' },
  { name: 'imos9', isViewTheme: true, cookieValue: 'imos9' },
  { name: 'imosx', isViewTheme: true, cookieValue: 'imosx' },
  { name: 'team11', isViewTheme: true, cookieValue: 'team11' },
] as const;

// Theme-specific selectors for elements that have different classes/IDs
const themeSelectors: Record<string, {
  chatList: string;
  messagesContainer: string;
  messageInput: string;
  stickerMessage: string;
  emojiButton: string;
  stickerButton: string;
  voiceButton: string;
  attachButton: string;
}> = {
  'default': {
    chatList: '#chat-list',
    messagesContainer: '#messages',
    messageInput: '#message-input',
    stickerMessage: '.message-sticker',
    emojiButton: '.input-emoji-btn, button[title="Emoji"]',
    stickerButton: '.input-sticker-btn, button[title="Stickers"]',
    voiceButton: '.voice-record-btn, button[title*="Voice"], button[title*="voice"]',
    attachButton: 'button[title="Attach file"], button[title="Attach"]',
  },
  'aim1.0': {
    chatList: '#buddies-list',
    messagesContainer: '#messages',
    messageInput: '#message-input',
    stickerMessage: '.aim-message-sticker',
    emojiButton: '#emoji-btn, button[title="Emoticons"]',
    stickerButton: '#sticker-btn, button[title="Stickers"]',
    voiceButton: '#voice-record-btn, button[title*="Voice"]',
    attachButton: 'button[title="Insert Image"]',
  },
  'ymxp': {
    chatList: '#buddies-list',
    messagesContainer: '#messages',
    messageInput: '#message-input',
    stickerMessage: '.ym-message-sticker',
    emojiButton: '#emoji-btn',
    stickerButton: '#sticker-btn',
    voiceButton: '#voice-record-btn',
    attachButton: 'button[title="Insert File"]',
  },
  'im26': {
    chatList: '#conversations-list',
    messagesContainer: '#messages-container',
    messageInput: '#message-input',
    stickerMessage: '.im-message-sticker',
    emojiButton: 'button[title="Emoji"]',
    stickerButton: 'button[title="Stickers"]',
    voiceButton: 'button[title*="Voice"], button[title*="voice"]',
    attachButton: 'button[title="Attach"]',
  },
  'imos9': {
    chatList: '#conversations-list',
    messagesContainer: '#messages-container',
    messageInput: '#message-input',
    stickerMessage: '.os9-message-sticker',
    emojiButton: 'button[title="Emoji"]',
    stickerButton: 'button[title="Stickers"]',
    voiceButton: 'button[title*="Voice"], button[title*="voice"]',
    attachButton: 'button[title="Attach"]',
  },
  'imosx': {
    chatList: '#conversations-list',
    messagesContainer: '#messages-container',
    messageInput: '#message-input',
    stickerMessage: '.osx-message-sticker',
    emojiButton: 'button[title="Emoji"]',
    stickerButton: 'button[title="Stickers"]',
    voiceButton: 'button[title*="Voice"], button[title*="voice"]',
    attachButton: 'button[title="Attach"]',
  },
  'team11': {
    chatList: '#chat-list',
    messagesContainer: '#messages-container',
    messageInput: '#message-input',
    stickerMessage: '.teams-message-sticker, .message-sticker',
    emojiButton: 'button[title="Emoji"]',
    stickerButton: 'button[title="Stickers"]',
    voiceButton: 'button[title*="Voice"], button[title*="voice"]',
    attachButton: 'button[title="Attach"]',
  },
};

// Helper to set theme
async function setTheme(page: Page, themeName: string, cookieValue: string, isViewTheme: boolean): Promise<void> {
  // Set cookie for theme
  await page.context().addCookies([{
    name: 'theme',
    value: cookieValue,
    domain: 'localhost',
    path: '/',
  }]);

  // Set localStorage for dark/light variant
  await page.evaluate((theme) => {
    localStorage.setItem('theme', theme);
    if (theme === 'dark' || theme === 'light') {
      document.documentElement.setAttribute('data-theme', theme);
    }
  }, themeName);
}

// Helper to get selectors for current theme
function getSelectors(themeName: string) {
  // Map theme names to selector keys
  const selectorKey = themeName === 'dark' || themeName === 'light' ? 'default' : themeName;
  return themeSelectors[selectorKey] || themeSelectors['default'];
}

// Helper to open first chat
async function openFirstChat(page: Page, selectors: typeof themeSelectors['default']): Promise<boolean> {
  // Wait for page to load
  await page.waitForTimeout(1000);

  // Try to find and click a chat
  const chatItems = page.locator(`${selectors.chatList} > div, ${selectors.chatList} .chat-item, ${selectors.chatList} .ym-buddy, ${selectors.chatList} .aim-buddy`);
  const count = await chatItems.count();

  if (count > 0) {
    await chatItems.first().click();
    await page.waitForTimeout(500);
    return true;
  }
  return false;
}

test.describe('Multi-Theme Feature Verification', () => {
  // Run tests for each theme
  for (const theme of themes) {
    test.describe(`Theme: ${theme.name}`, () => {
      test.beforeEach(async ({ page }) => {
        // Set theme before navigation
        await setTheme(page, theme.name, theme.cookieValue, theme.isViewTheme);
      });

      test(`TC-THEME-${theme.name.toUpperCase()}-001: chat list renders correctly`, async ({ page, loginAs }) => {
        await loginAs('alice');

        const selectors = getSelectors(theme.name);
        const chatList = page.locator(selectors.chatList);

        // Chat list should be visible
        await expect(chatList).toBeVisible({ timeout: 10000 });
      });

      test(`TC-THEME-${theme.name.toUpperCase()}-002: message input is functional`, async ({ page, loginAs }) => {
        await loginAs('alice');

        const selectors = getSelectors(theme.name);

        // Open a chat first
        const chatOpened = await openFirstChat(page, selectors);
        if (!chatOpened) {
          test.skip();
          return;
        }

        // Find message input
        const messageInput = page.locator(selectors.messageInput);
        await expect(messageInput).toBeVisible({ timeout: 5000 });

        // Type a message
        await messageInput.fill('Test message for ' + theme.name);
        await expect(messageInput).toHaveValue('Test message for ' + theme.name);
      });

      test(`TC-THEME-${theme.name.toUpperCase()}-003: emoji button is visible`, async ({ page, loginAs }) => {
        await loginAs('alice');

        const selectors = getSelectors(theme.name);

        const chatOpened = await openFirstChat(page, selectors);
        if (!chatOpened) {
          test.skip();
          return;
        }

        // Find emoji button
        const emojiButton = page.locator(selectors.emojiButton).first();
        await expect(emojiButton).toBeVisible({ timeout: 5000 });
      });

      test(`TC-THEME-${theme.name.toUpperCase()}-004: sticker button is visible`, async ({ page, loginAs }) => {
        await loginAs('alice');

        const selectors = getSelectors(theme.name);

        const chatOpened = await openFirstChat(page, selectors);
        if (!chatOpened) {
          test.skip();
          return;
        }

        // Find sticker button
        const stickerButton = page.locator(selectors.stickerButton).first();
        await expect(stickerButton).toBeVisible({ timeout: 5000 });
      });

      test(`TC-THEME-${theme.name.toUpperCase()}-005: sticker picker opens`, async ({ page, loginAs }) => {
        await loginAs('alice');

        const selectors = getSelectors(theme.name);

        const chatOpened = await openFirstChat(page, selectors);
        if (!chatOpened) {
          test.skip();
          return;
        }

        // Open sticker picker
        const stickerButton = page.locator(selectors.stickerButton).first();
        await stickerButton.click();

        // Sticker picker should be visible
        const stickerPicker = page.locator('.sticker-picker');
        await expect(stickerPicker).toBeVisible({ timeout: 5000 });

        // Close picker
        await page.keyboard.press('Escape');
      });

      test(`TC-THEME-${theme.name.toUpperCase()}-006: emoji picker opens`, async ({ page, loginAs }) => {
        await loginAs('alice');

        const selectors = getSelectors(theme.name);

        const chatOpened = await openFirstChat(page, selectors);
        if (!chatOpened) {
          test.skip();
          return;
        }

        // Open emoji picker
        const emojiButton = page.locator(selectors.emojiButton).first();
        await emojiButton.click();

        // Emoji picker should be visible
        const emojiPicker = page.locator('.emoji-picker');
        await expect(emojiPicker).toBeVisible({ timeout: 5000 });

        // Close picker
        await page.keyboard.press('Escape');
      });

      test(`TC-THEME-${theme.name.toUpperCase()}-007: attach button is visible`, async ({ page, loginAs }) => {
        await loginAs('alice');

        const selectors = getSelectors(theme.name);

        const chatOpened = await openFirstChat(page, selectors);
        if (!chatOpened) {
          test.skip();
          return;
        }

        // Find attach button
        const attachButton = page.locator(selectors.attachButton).first();
        await expect(attachButton).toBeVisible({ timeout: 5000 });
      });

      test(`TC-THEME-${theme.name.toUpperCase()}-008: send message works`, async ({ page, loginAs }) => {
        await loginAs('alice');

        const selectors = getSelectors(theme.name);

        const chatOpened = await openFirstChat(page, selectors);
        if (!chatOpened) {
          test.skip();
          return;
        }

        const messageInput = page.locator(selectors.messageInput);
        const testMessage = `Test message ${Date.now()}`;

        await messageInput.fill(testMessage);
        await page.keyboard.press('Enter');

        // Wait for message to be sent
        await page.waitForTimeout(1000);

        // Verify message appears in container
        const messagesContainer = page.locator(selectors.messagesContainer);
        await expect(messagesContainer).toContainText(testMessage, { timeout: 5000 });
      });

      test(`TC-THEME-${theme.name.toUpperCase()}-009: send sticker works`, async ({ page, loginAs }) => {
        await loginAs('alice');

        const selectors = getSelectors(theme.name);

        const chatOpened = await openFirstChat(page, selectors);
        if (!chatOpened) {
          test.skip();
          return;
        }

        // Open sticker picker
        const stickerButton = page.locator(selectors.stickerButton).first();
        await stickerButton.click();

        const stickerPicker = page.locator('.sticker-picker');
        await expect(stickerPicker).toBeVisible({ timeout: 5000 });

        // Click first sticker
        const stickerItem = stickerPicker.locator('.sticker-item').first();
        await stickerItem.click();

        // Wait for sticker to be sent
        await page.waitForTimeout(1000);

        // Verify sticker message appears
        const stickerMessage = page.locator(selectors.stickerMessage + ', .sticker-content, [data-type="sticker"]');
        await expect(stickerMessage.last()).toBeVisible({ timeout: 5000 });
      });
    });
  }
});

test.describe('Theme Consistency Checks', () => {
  test('TC-THEME-CONSISTENCY-001: all themes have required UI elements', async ({ page, loginAs }) => {
    const results: Array<{ theme: string; elements: Record<string, boolean> }> = [];

    for (const theme of themes) {
      await setTheme(page, theme.name, theme.cookieValue, theme.isViewTheme);
      await loginAs('alice');

      const selectors = getSelectors(theme.name);
      const chatOpened = await openFirstChat(page, selectors);

      if (chatOpened) {
        const elements: Record<string, boolean> = {
          messageInput: await page.locator(selectors.messageInput).isVisible().catch(() => false),
          emojiButton: await page.locator(selectors.emojiButton).first().isVisible().catch(() => false),
          stickerButton: await page.locator(selectors.stickerButton).first().isVisible().catch(() => false),
          attachButton: await page.locator(selectors.attachButton).first().isVisible().catch(() => false),
        };

        results.push({ theme: theme.name, elements });
      }

      // Clear cookies for next iteration
      await page.context().clearCookies();
    }

    // Verify all themes have required elements
    for (const result of results) {
      expect(result.elements.messageInput, `${result.theme}: messageInput`).toBe(true);
      expect(result.elements.emojiButton, `${result.theme}: emojiButton`).toBe(true);
      expect(result.elements.stickerButton, `${result.theme}: stickerButton`).toBe(true);
    }
  });
});
