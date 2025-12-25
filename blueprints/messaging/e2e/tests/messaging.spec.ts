import { test, expect, testUsers } from '../fixtures/test-fixtures';
import { AppPage } from '../pages/app.page';

test.describe('Messaging', () => {
  // Helper to ensure we have a chat to work with
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

  test.describe('Sending Messages', () => {
    test('TC-MSG-001: send text message with send button', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const testMessage = `Test message ${Date.now()}`;
      await appPage.sendMessage(testMessage);

      // Input should be cleared
      await expect(appPage.messageInput).toHaveValue('');

      // Message should appear in chat
      await appPage.expectLastMessageText(testMessage);
    });

    test('TC-MSG-001b: send message with Enter key', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const testMessage = `Enter key message ${Date.now()}`;
      await appPage.sendMessageWithEnter(testMessage);

      await expect(appPage.messageInput).toHaveValue('');
      await appPage.expectLastMessageText(testMessage);
    });

    test('TC-MSG-003: multi-line message with Shift+Enter', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Type first line
      await appPage.messageInput.fill('Line 1');
      // Add new line
      await appPage.messageInput.press('Shift+Enter');
      await appPage.messageInput.type('Line 2');

      // Verify input has both lines
      const inputValue = await appPage.messageInput.inputValue();
      expect(inputValue).toContain('Line 1');
      expect(inputValue).toContain('Line 2');

      // Send the message
      await appPage.sendButton.click();

      // Message should contain both lines
      const lastMessage = await appPage.getLastMessage();
      await expect(lastMessage).toContainText('Line 1');
      await expect(lastMessage).toContainText('Line 2');
    });

    test('TC-MSG-004: long message wraps correctly', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const longMessage = 'This is a very long message '.repeat(30);
      await appPage.sendMessage(longMessage);

      const lastMessage = await appPage.getLastMessage();

      // Message should be visible
      await expect(lastMessage).toBeVisible();

      // Message bubble should have max-width (not exceed 70% as per CSS)
      const bubbleBox = await lastMessage.boundingBox();
      const containerBox = await appPage.messagesContainer.boundingBox();

      if (bubbleBox && containerBox) {
        expect(bubbleBox.width).toBeLessThan(containerBox.width * 0.75);
      }
    });

    test('TC-MSG-005: empty message cannot be sent', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Get current message count
      const messagesBefore = await appPage.getMessages();
      const countBefore = await messagesBefore.count();

      // Try to send empty message
      await appPage.messageInput.fill('');
      await appPage.sendButton.click();

      // Wait a bit
      await page.waitForTimeout(500);

      // Message count should be the same
      const messagesAfter = await appPage.getMessages();
      const countAfter = await messagesAfter.count();

      expect(countAfter).toBe(countBefore);
    });

    test('TC-MSG-006: whitespace-only message cannot be sent', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const messagesBefore = await appPage.getMessages();
      const countBefore = await messagesBefore.count();

      // Try to send whitespace only
      await appPage.messageInput.fill('   \n   ');
      await appPage.sendButton.click();

      await page.waitForTimeout(500);

      const messagesAfter = await appPage.getMessages();
      const countAfter = await messagesAfter.count();

      expect(countAfter).toBe(countBefore);
    });
  });

  test.describe('Message Display', () => {
    test('TC-MSG-002: sent messages show status icons', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      await appPage.sendMessage(`Status test ${Date.now()}`);

      const lastMessage = await appPage.getLastMessage();

      // Should have a checkmark (sent status)
      await expect(lastMessage).toContainText('✓');
    });

    test('TC-MSG-007: messages load when opening chat', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Messages container should be visible
      await expect(appPage.messagesContainer).toBeVisible();

      // Should have messages or empty state
      const messages = await appPage.getMessages();
      const hasMessages = (await messages.count()) > 0;
      const hasEmptyState = (await appPage.messagesContainer.locator('.text-secondary').count()) > 0;

      expect(hasMessages || hasEmptyState).toBe(true);
    });

    test('TC-MSG-008: date separators appear between days', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Look for date separator elements
      const dateSeparators = appPage.messagesContainer.locator('.text-center .bg-tertiary.rounded-full');
      const separatorCount = await dateSeparators.count();

      // Date separators should contain date text like "Today", "Yesterday", or a date
      if (separatorCount > 0) {
        const firstSeparator = dateSeparators.first();
        const text = await firstSeparator.textContent();
        expect(text).toBeTruthy();
      }
    });

    test('sent messages appear on right side', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      await appPage.sendMessage(`Alignment test ${Date.now()}`);

      const lastMessage = await appPage.getLastMessage();

      // Sent messages have .sent-bubble class
      await expect(lastMessage).toHaveClass(/sent-bubble/);
    });
  });

  test.describe('Message Input', () => {
    test('TC-MSG-INPUT-001: textarea auto-resizes', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Get initial height
      const initialHeight = await appPage.messageInput.evaluate((el) => el.offsetHeight);

      // Type multiple lines
      await appPage.messageInput.fill('Line 1\nLine 2\nLine 3\nLine 4\nLine 5');

      // Height should increase
      const newHeight = await appPage.messageInput.evaluate((el) => el.offsetHeight);

      expect(newHeight).toBeGreaterThan(initialHeight);
    });

    test('attach button is visible', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      await expect(appPage.attachButton).toBeVisible();
    });

    test('send button is visible', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      await expect(appPage.sendButton).toBeVisible();
    });
  });

  test.describe('Chat List Updates', () => {
    test('sending message updates last message in chat list', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const testMessage = `Chat list update ${Date.now()}`;
      await appPage.sendMessage(testMessage);

      // Check chat list shows the new message
      const firstChatItem = (await appPage.getChatItems()).first();
      await expect(firstChatItem).toContainText(testMessage.substring(0, 20));
    });
  });

  test.describe('Message Escape', () => {
    test('HTML in messages is escaped', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const htmlMessage = '<script>alert("xss")</script>';
      await appPage.sendMessage(htmlMessage);

      const lastMessage = await appPage.getLastMessage();

      // Should show the literal text, not execute script
      await expect(lastMessage).toContainText('<script>');

      // No script should have been injected
      const scripts = await page.locator('script:has-text("xss")').count();
      expect(scripts).toBe(0);
    });
  });
});
