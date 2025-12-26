import { test, expect, testUsers } from '../fixtures/test-fixtures';
import { AppPage } from '../pages/app.page';

// Helper to open first chat
async function openFirstChat(appPage: AppPage): Promise<boolean> {
  await appPage.waitForLoad();
  const chatItems = await appPage.getChatItems();
  const count = await chatItems.count();

  if (count > 0) {
    await appPage.selectChatByIndex(0);
    await appPage.expectChatViewVisible();
    return true;
  }
  return false;
}

test.describe('Emoji Picker Features', () => {
  test.describe('Emoji Button', () => {
    test('TC-EMOJI-001: emoji button is visible in chat', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Find emoji button
      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]');
      await expect(emojiButton.first()).toBeVisible({ timeout: 5000 });
    });

    test('TC-EMOJI-002: emoji button has emoji icon', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await expect(emojiButton).toBeVisible();

      // Should have SVG or emoji icon
      const hasIcon = await emojiButton.locator('svg, span').count();
      expect(hasIcon).toBeGreaterThan(0);
    });
  });

  test.describe('Emoji Picker Display', () => {
    test('TC-EMOJI-003: clicking emoji button opens picker', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      // Emoji picker should be visible
      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible({ timeout: 5000 });
    });

    test('TC-EMOJI-004: emoji picker has category tabs', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Should have category tabs or buttons
      const categoryTabs = emojiPicker.locator('.emoji-category-tab, .emoji-tab, button[data-category]');
      const tabCount = await categoryTabs.count();
      expect(tabCount).toBeGreaterThan(0);
    });

    test('TC-EMOJI-005: emoji picker has search input', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Should have search input
      const searchInput = emojiPicker.locator('input[type="text"], input[placeholder*="Search"], .emoji-search');
      await expect(searchInput.first()).toBeVisible();
    });

    test('TC-EMOJI-006: emoji picker shows emoji grid', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Should have emoji items
      const emojiItems = emojiPicker.locator('.emoji-item, button.emoji, span.emoji');
      const emojiCount = await emojiItems.count();
      expect(emojiCount).toBeGreaterThan(10);
    });
  });

  test.describe('Emoji Picker Interactions', () => {
    test('TC-EMOJI-007: clicking emoji inserts into input', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const messageInput = page.locator('#message-input');
      const initialValue = await messageInput.inputValue();

      // Open emoji picker
      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Click first emoji
      const firstEmoji = emojiPicker.locator('.emoji-item, button.emoji, span.emoji').first();
      await firstEmoji.click();

      // Input should now contain emoji
      await page.waitForTimeout(300);
      const newValue = await messageInput.inputValue();
      expect(newValue.length).toBeGreaterThan(initialValue.length);
    });

    test('TC-EMOJI-008: emoji picker closes after selection', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Click an emoji
      const firstEmoji = emojiPicker.locator('.emoji-item, button.emoji, span.emoji').first();
      await firstEmoji.click();

      // Picker should close
      await page.waitForTimeout(300);
      await expect(emojiPicker).toBeHidden();
    });

    test('TC-EMOJI-009: escape key closes emoji picker', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Press Escape
      await page.keyboard.press('Escape');

      // Picker should close
      await expect(emojiPicker).toBeHidden();
    });

    test('TC-EMOJI-010: clicking outside closes emoji picker', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Click outside (on the page body)
      await page.click('body', { position: { x: 10, y: 10 } });

      // Picker should close
      await page.waitForTimeout(300);
      await expect(emojiPicker).toBeHidden();
    });
  });

  test.describe('Emoji Category Navigation', () => {
    test('TC-EMOJI-011: clicking category tab changes emoji display', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Get category tabs
      const categoryTabs = emojiPicker.locator('.emoji-category-tab, .emoji-tab, button[data-category]');
      const tabCount = await categoryTabs.count();

      if (tabCount > 1) {
        // Click second category tab
        await categoryTabs.nth(1).click();
        await page.waitForTimeout(200);

        // Category should be marked active
        const activeTab = emojiPicker.locator('.emoji-category-tab.active, .emoji-tab.active, button[data-category].active');
        await expect(activeTab).toBeVisible();
      }
    });

    test('TC-EMOJI-012: each category shows different emojis', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Get first emoji in first category
      const firstEmoji = await emojiPicker.locator('.emoji-item, button.emoji, span.emoji').first().textContent();

      // Get category tabs
      const categoryTabs = emojiPicker.locator('.emoji-category-tab, .emoji-tab, button[data-category]');
      const tabCount = await categoryTabs.count();

      if (tabCount > 1) {
        // Click last category tab
        await categoryTabs.nth(tabCount - 1).click();
        await page.waitForTimeout(200);

        // Get first emoji in new category
        const newFirstEmoji = await emojiPicker.locator('.emoji-item, button.emoji, span.emoji').first().textContent();

        // Emojis should be different
        expect(newFirstEmoji).not.toBe(firstEmoji);
      }
    });
  });

  test.describe('Emoji Search', () => {
    test('TC-EMOJI-013: searching filters emoji display', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Get initial emoji count
      const emojiItems = emojiPicker.locator('.emoji-item, button.emoji, span.emoji');
      const initialCount = await emojiItems.count();

      // Search for something specific
      const searchInput = emojiPicker.locator('input[type="text"], input[placeholder*="Search"], .emoji-search').first();
      await searchInput.fill('smile');

      await page.waitForTimeout(300);

      // Emoji count should be different (filtered)
      const filteredCount = await emojiItems.count();
      expect(filteredCount).toBeLessThan(initialCount);
    });

    test('TC-EMOJI-014: clearing search shows all emojis', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      const emojiItems = emojiPicker.locator('.emoji-item, button.emoji, span.emoji');
      const initialCount = await emojiItems.count();

      // Search then clear
      const searchInput = emojiPicker.locator('input[type="text"], input[placeholder*="Search"], .emoji-search').first();
      await searchInput.fill('heart');
      await page.waitForTimeout(200);

      await searchInput.clear();
      await page.waitForTimeout(200);

      // Should show all emojis again
      const restoredCount = await emojiItems.count();
      expect(restoredCount).toBe(initialCount);
    });

    test('TC-EMOJI-015: no results shows empty state', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Search for something that won't match
      const searchInput = emojiPicker.locator('input[type="text"], input[placeholder*="Search"], .emoji-search').first();
      await searchInput.fill('xyznonexistent123');

      await page.waitForTimeout(300);

      // Should show no results or empty state
      const emojiItems = emojiPicker.locator('.emoji-item, button.emoji, span.emoji');
      const count = await emojiItems.count();
      expect(count).toBe(0);
    });
  });

  test.describe('Recent Emojis', () => {
    test('TC-EMOJI-016: recently used emojis are tracked', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Open emoji picker and select an emoji
      const emojiButton = page.locator('.input-emoji-btn, button[title="Emoji"], #emoji-btn, button[title="Emoticons"]').first();
      await emojiButton.click();

      const emojiPicker = page.locator('.emoji-picker');
      await expect(emojiPicker).toBeVisible();

      // Get a specific emoji
      const targetEmoji = emojiPicker.locator('.emoji-item, button.emoji, span.emoji').first();
      const emojiText = await targetEmoji.textContent();
      await targetEmoji.click();

      // Reopen picker
      await emojiButton.click();
      await expect(emojiPicker).toBeVisible();

      // Check for recent section
      const recentSection = emojiPicker.locator('.emoji-recent, [data-category="recent"]');
      if (await recentSection.isVisible()) {
        const recentEmojis = recentSection.locator('.emoji-item, button.emoji, span.emoji');
        const recentTexts = await recentEmojis.allTextContents();

        // The emoji we just used should be in recent
        expect(recentTexts.some(t => t === emojiText)).toBe(true);
      }
    });
  });
});

test.describe('Emoji in Messages', () => {
  test('TC-EMOJI-MSG-001: emoji displays correctly in sent message', async ({ page, loginAs }) => {
    await loginAs('alice');

    const appPage = new AppPage(page);
    if (!(await openFirstChat(appPage))) {
      test.skip();
      return;
    }

    // Type emoji directly
    const messageInput = page.locator('#message-input');
    const testMessage = 'Hello! 😀';
    await messageInput.fill(testMessage);
    await page.keyboard.press('Enter');

    // Wait for message
    await page.waitForTimeout(1000);

    // Message should contain emoji
    const messages = page.locator('#messages, #messages-container');
    await expect(messages).toContainText('😀');
  });

  test('TC-EMOJI-MSG-002: multiple emojis in one message', async ({ page, loginAs }) => {
    await loginAs('alice');

    const appPage = new AppPage(page);
    if (!(await openFirstChat(appPage))) {
      test.skip();
      return;
    }

    const messageInput = page.locator('#message-input');
    const testMessage = '🎉 Party time! 🥳🎊🎈';
    await messageInput.fill(testMessage);
    await page.keyboard.press('Enter');

    await page.waitForTimeout(1000);

    const messages = page.locator('#messages, #messages-container');
    await expect(messages).toContainText('🎉');
    await expect(messages).toContainText('🥳');
  });
});
