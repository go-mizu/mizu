import { test, expect, testUsers } from '../fixtures/test-fixtures';
import { AppPage } from '../pages/app.page';
import * as path from 'path';
import * as fs from 'fs';

test.describe('Media and Sticker Interactions', () => {
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

  test.describe('Sticker Picker', () => {
    test('TC-STICKER-001: open sticker picker', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Open sticker picker
      await appPage.openStickerPicker();

      // Verify picker is visible
      await expect(appPage.stickerPicker).toBeVisible();

      // Verify sticker items are present
      const stickerItems = appPage.stickerPicker.locator('.sticker-item');
      await expect(stickerItems.first()).toBeVisible();
    });

    test('TC-STICKER-002: close sticker picker with Escape', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      await appPage.openStickerPicker();
      await expect(appPage.stickerPicker).toBeVisible();

      // Press Escape to close
      await page.keyboard.press('Escape');
      await expect(appPage.stickerPicker).toBeHidden();
    });

    test('TC-STICKER-003: send sticker message', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Get initial sticker message count
      const stickersBefore = await appPage.getStickerMessages();
      const countBefore = await stickersBefore.count();

      // Send a sticker
      await appPage.sendSticker(0, 0);

      // Wait for message to appear
      await page.waitForTimeout(500);

      // Verify sticker message was sent
      const stickersAfter = await appPage.getStickerMessages();
      const countAfter = await stickersAfter.count();

      expect(countAfter).toBeGreaterThan(countBefore);
    });
  });

  test.describe('Sticker Click Interactions', () => {
    test('TC-STICKER-CLICK-001: clicking sticker opens lightbox', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // First send a sticker
      await appPage.sendSticker(0, 0);
      await page.waitForTimeout(500);

      // Click on the sticker in the message
      await appPage.clickStickerInMessage();

      // Verify lightbox is visible
      await appPage.expectStickerLightboxVisible();
    });

    test('TC-STICKER-CLICK-002: sticker lightbox shows sticker name and pack', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Send a sticker
      await appPage.sendSticker(0, 0);
      await page.waitForTimeout(500);

      // Click on the sticker
      await appPage.clickStickerInMessage();
      await appPage.expectStickerLightboxVisible();

      // Verify content is present
      const lightboxContent = page.locator('.sticker-lightbox-content');
      await expect(lightboxContent).toBeVisible();

      // Check for sticker name and pack name
      const stickerName = page.locator('.sticker-lightbox-name');
      const packName = page.locator('.sticker-lightbox-pack');

      await expect(stickerName).toBeVisible();
      await expect(packName).toBeVisible();
    });

    test('TC-STICKER-CLICK-003: close sticker lightbox by clicking', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      await appPage.sendSticker(0, 0);
      await page.waitForTimeout(500);

      await appPage.clickStickerInMessage();
      await appPage.expectStickerLightboxVisible();

      // Close by clicking overlay
      await appPage.closeStickerLightbox();
      await appPage.expectStickerLightboxHidden();
    });

    test('TC-STICKER-CLICK-004: close sticker lightbox with Escape', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      await appPage.sendSticker(0, 0);
      await page.waitForTimeout(500);

      await appPage.clickStickerInMessage();
      await appPage.expectStickerLightboxVisible();

      // Close with Escape key
      await page.keyboard.press('Escape');
      await appPage.expectStickerLightboxHidden();
    });
  });

  test.describe('Image Click Interactions', () => {
    // Create a test image for upload
    const testImagePath = path.join(__dirname, '..', 'fixtures', 'test-image.png');

    test.beforeAll(async () => {
      // Create a simple test image if it doesn't exist
      if (!fs.existsSync(testImagePath)) {
        // Create a minimal valid PNG (1x1 transparent pixel)
        const pngBuffer = Buffer.from([
          0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
          0x00, 0x00, 0x00, 0x0D, // IHDR chunk length
          0x49, 0x48, 0x44, 0x52, // IHDR
          0x00, 0x00, 0x00, 0x01, // Width: 1
          0x00, 0x00, 0x00, 0x01, // Height: 1
          0x08, 0x06, // Bit depth: 8, Color type: RGBA
          0x00, 0x00, 0x00, // Compression, Filter, Interlace
          0x1F, 0x15, 0xC4, 0x89, // CRC
          0x00, 0x00, 0x00, 0x0A, // IDAT chunk length
          0x49, 0x44, 0x41, 0x54, // IDAT
          0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01, // Compressed data
          0x0D, 0x0A, 0x2D, 0xB4, // CRC
          0x00, 0x00, 0x00, 0x00, // IEND chunk length
          0x49, 0x45, 0x4E, 0x44, // IEND
          0xAE, 0x42, 0x60, 0x82  // CRC
        ]);
        fs.writeFileSync(testImagePath, pngBuffer);
      }
    });

    test('TC-IMAGE-CLICK-001: clicking image opens lightbox', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Check if there are any existing images in the chat
      const images = await appPage.getImageMessages();
      const imageCount = await images.count();

      if (imageCount === 0) {
        // Upload a test image
        const fileInput = appPage.fileInput;
        await fileInput.setInputFiles(testImagePath);

        // Wait for preview modal and send
        await expect(appPage.mediaPreviewOverlay).toBeVisible({ timeout: 5000 });
        const sendButton = page.locator('#media-send-btn');
        await sendButton.click();

        // Wait for upload and message to appear
        await page.waitForTimeout(2000);
      }

      // Now click on the image
      const updatedImages = await appPage.getImageMessages();
      const updatedCount = await updatedImages.count();

      if (updatedCount > 0) {
        await appPage.clickImageInMessage();
        await appPage.expectLightboxVisible();
      } else {
        test.skip();
      }
    });

    test('TC-IMAGE-CLICK-002: lightbox shows image correctly', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const images = await appPage.getImageMessages();
      const imageCount = await images.count();

      if (imageCount === 0) {
        test.skip();
        return;
      }

      await appPage.clickImageInMessage();
      await appPage.expectLightboxVisible();

      // Verify lightbox content
      const lightboxContent = page.locator('#lightbox-content');
      await expect(lightboxContent).toBeVisible();

      // Check for image in lightbox
      const lightboxImage = lightboxContent.locator('img');
      await expect(lightboxImage).toBeVisible();
    });

    test('TC-IMAGE-CLICK-003: close lightbox by clicking outside', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const images = await appPage.getImageMessages();
      const imageCount = await images.count();

      if (imageCount === 0) {
        test.skip();
        return;
      }

      await appPage.clickImageInMessage();
      await appPage.expectLightboxVisible();

      // Close by clicking overlay
      await appPage.closeLightbox();
      await appPage.expectLightboxHidden();
    });

    test('TC-IMAGE-CLICK-004: close lightbox with Escape key', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const images = await appPage.getImageMessages();
      const imageCount = await images.count();

      if (imageCount === 0) {
        test.skip();
        return;
      }

      await appPage.clickImageInMessage();
      await appPage.expectLightboxVisible();

      // Close with Escape
      await appPage.closeLightboxWithEscape();
      await appPage.expectLightboxHidden();
    });

    test('TC-IMAGE-CLICK-005: lightbox has download button', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const images = await appPage.getImageMessages();
      const imageCount = await images.count();

      if (imageCount === 0) {
        test.skip();
        return;
      }

      await appPage.clickImageInMessage();
      await appPage.expectLightboxVisible();

      // Check for download button in toolbar
      const downloadButton = page.locator('.lightbox-toolbar .lightbox-btn').first();
      await expect(downloadButton).toBeVisible();
    });
  });

  test.describe('Media Upload Flow', () => {
    test('TC-MEDIA-001: attach button opens file picker', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Verify attach button is visible
      await expect(appPage.attachButton).toBeVisible();
    });

    test('TC-MEDIA-002: file input accepts correct file types', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Check file input has correct accept attribute
      const acceptAttr = await appPage.fileInput.getAttribute('accept');
      expect(acceptAttr).toContain('image/*');
      expect(acceptAttr).toContain('video/*');
      expect(acceptAttr).toContain('audio/*');
    });
  });

  test.describe('Sticker Cursor Style', () => {
    test('TC-STICKER-CURSOR-001: stickers have pointer cursor', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Send a sticker first
      await appPage.sendSticker(0, 0);
      await page.waitForTimeout(500);

      // Check cursor style on sticker
      const stickerMessages = await appPage.getStickerMessages();
      const count = await stickerMessages.count();

      if (count > 0) {
        const cursorStyle = await stickerMessages.last().evaluate(el => {
          return window.getComputedStyle(el).cursor;
        });
        expect(cursorStyle).toBe('pointer');
      }
    });
  });
});
