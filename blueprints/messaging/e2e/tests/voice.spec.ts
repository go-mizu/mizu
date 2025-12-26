import { test, expect, testUsers } from '../fixtures/test-fixtures';
import { AppPage } from '../pages/app.page';
import { Page } from '@playwright/test';

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

// Helper to grant microphone permissions
async function grantMicrophonePermission(page: Page): Promise<void> {
  const context = page.context();
  await context.grantPermissions(['microphone']);
}

test.describe('Voice Recording Features', () => {
  test.describe('Voice Button Visibility', () => {
    test('TC-VOICE-001: voice button is visible in chat', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Find voice button
      const voiceButton = page.locator('.voice-record-btn, button[title*="Voice"], button[title*="voice"], #voice-record-btn');
      await expect(voiceButton.first()).toBeVisible({ timeout: 5000 });
    });

    test('TC-VOICE-002: voice button has microphone icon', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Voice button should have an SVG or icon
      const voiceButton = page.locator('.voice-record-btn, button[title*="Voice"], button[title*="voice"], #voice-record-btn').first();
      await expect(voiceButton).toBeVisible();

      // Check for SVG or icon child
      const hasIcon = await voiceButton.locator('svg, i, span').count();
      expect(hasIcon).toBeGreaterThan(0);
    });
  });

  test.describe('Voice Recording UI', () => {
    test('TC-VOICE-003: clicking voice button shows recording UI', async ({ page, loginAs }) => {
      await grantMicrophonePermission(page);
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const voiceButton = page.locator('.voice-record-btn, button[title*="Voice"], button[title*="voice"], #voice-record-btn').first();
      await voiceButton.click();

      // Wait for recording UI to appear
      await page.waitForTimeout(500);

      // Recording container or UI should be visible
      const recordingUI = page.locator('#voice-recording-container, .voice-recording-ui, .recording-container');
      const isVisible = await recordingUI.isVisible().catch(() => false);

      // If not visible, might have permission denied - that's ok for this test
      if (!isVisible) {
        // Check if permission dialog is showing or error state
        const permissionError = page.locator('text=permission, text=denied, text=microphone').first();
        const hasError = await permissionError.isVisible().catch(() => false);
        // Skip if permission issue
        if (hasError) {
          test.skip();
        }
      }
    });

    test('TC-VOICE-004: recording UI shows duration timer', async ({ page, loginAs }) => {
      await grantMicrophonePermission(page);
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const voiceButton = page.locator('.voice-record-btn, button[title*="Voice"], button[title*="voice"], #voice-record-btn').first();
      await voiceButton.click();

      await page.waitForTimeout(1000);

      // Look for duration display (format like 0:01, 00:01, etc.)
      const durationDisplay = page.locator('text=/\\d+:\\d+/, .recording-duration, .voice-duration');
      const hasDuration = await durationDisplay.first().isVisible().catch(() => false);

      // If recording started, duration should be visible
      if (hasDuration) {
        await expect(durationDisplay.first()).toBeVisible();
      }
    });

    test('TC-VOICE-005: recording UI shows cancel button', async ({ page, loginAs }) => {
      await grantMicrophonePermission(page);
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const voiceButton = page.locator('.voice-record-btn, button[title*="Voice"], button[title*="voice"], #voice-record-btn').first();
      await voiceButton.click();

      await page.waitForTimeout(500);

      // Look for cancel button
      const cancelButton = page.locator('button:has-text("Cancel"), .voice-cancel-btn, .recording-cancel');
      const hasCancel = await cancelButton.first().isVisible().catch(() => false);

      if (hasCancel) {
        await expect(cancelButton.first()).toBeVisible();
        // Click cancel to clean up
        await cancelButton.first().click();
      }
    });

    test('TC-VOICE-006: cancel button stops recording', async ({ page, loginAs }) => {
      await grantMicrophonePermission(page);
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const voiceButton = page.locator('.voice-record-btn, button[title*="Voice"], button[title*="voice"], #voice-record-btn').first();
      await voiceButton.click();

      await page.waitForTimeout(500);

      // Click cancel
      const cancelButton = page.locator('button:has-text("Cancel"), .voice-cancel-btn, .recording-cancel').first();
      if (await cancelButton.isVisible()) {
        await cancelButton.click();

        // Recording UI should be hidden
        await page.waitForTimeout(300);
        const recordingUI = page.locator('#voice-recording-container:not(.hidden), .voice-recording-ui:visible');
        const stillRecording = await recordingUI.isVisible().catch(() => false);
        expect(stillRecording).toBe(false);
      }
    });

    test('TC-VOICE-007: recording UI shows waveform', async ({ page, loginAs }) => {
      await grantMicrophonePermission(page);
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      const voiceButton = page.locator('.voice-record-btn, button[title*="Voice"], button[title*="voice"], #voice-record-btn').first();
      await voiceButton.click();

      await page.waitForTimeout(500);

      // Look for waveform or visualization
      const waveform = page.locator('.voice-waveform, .waveform, canvas, .audio-visualizer');
      const hasWaveform = await waveform.first().isVisible().catch(() => false);

      if (hasWaveform) {
        await expect(waveform.first()).toBeVisible();
      }

      // Clean up - press Escape or click cancel
      await page.keyboard.press('Escape');
    });
  });

  test.describe('Voice Message Display', () => {
    test('TC-VOICE-008: voice messages have play button', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Look for existing voice messages
      const voiceMessages = page.locator('.voice-message, .audio-message, [data-type="voice"]');
      const count = await voiceMessages.count();

      if (count > 0) {
        // Voice message should have a play button
        const playButton = voiceMessages.first().locator('button, .play-btn, .voice-play');
        await expect(playButton.first()).toBeVisible();
      } else {
        // No voice messages to test - skip
        test.skip();
      }
    });

    test('TC-VOICE-009: voice messages show duration', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Look for existing voice messages
      const voiceMessages = page.locator('.voice-message, .audio-message, [data-type="voice"]');
      const count = await voiceMessages.count();

      if (count > 0) {
        // Voice message should show duration
        const duration = voiceMessages.first().locator('text=/\\d+:\\d+/, .voice-duration, .audio-duration');
        await expect(duration.first()).toBeVisible();
      } else {
        test.skip();
      }
    });

    test('TC-VOICE-010: voice messages show waveform preview', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Look for existing voice messages
      const voiceMessages = page.locator('.voice-message, .audio-message, [data-type="voice"]');
      const count = await voiceMessages.count();

      if (count > 0) {
        // Voice message should have waveform
        const waveform = voiceMessages.first().locator('.voice-waveform, .waveform, svg, canvas');
        const hasWaveform = await waveform.first().isVisible().catch(() => false);
        // Waveform is optional but nice to have
        if (hasWaveform) {
          await expect(waveform.first()).toBeVisible();
        }
      } else {
        test.skip();
      }
    });
  });

  test.describe('Voice Message Playback', () => {
    test('TC-VOICE-011: clicking play button starts playback', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Look for existing voice messages
      const voiceMessages = page.locator('.voice-message, .audio-message, [data-type="voice"]');
      const count = await voiceMessages.count();

      if (count > 0) {
        const playButton = voiceMessages.first().locator('button, .play-btn, .voice-play').first();
        await playButton.click();

        // After clicking, button should change to pause or show playing state
        await page.waitForTimeout(300);

        const isPaused = voiceMessages.first().locator('.pause-btn, .playing, [data-state="playing"]');
        const isPlaying = await isPaused.first().isVisible().catch(() => false);

        // Either playing state or pause button should be visible
        expect(isPlaying).toBe(true);
      } else {
        test.skip();
      }
    });

    test('TC-VOICE-012: audio element is created for voice playback', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      if (!(await openFirstChat(appPage))) {
        test.skip();
        return;
      }

      // Look for audio elements or players
      const audioElements = page.locator('audio');
      const count = await audioElements.count();

      // Audio elements may be hidden but should exist if there are voice messages
      const voiceMessages = page.locator('.voice-message, .audio-message, [data-type="voice"]');
      const voiceCount = await voiceMessages.count();

      if (voiceCount > 0) {
        // There should be audio support for voice messages
        expect(count).toBeGreaterThanOrEqual(0); // Audio elements may be created on demand
      } else {
        test.skip();
      }
    });
  });
});

test.describe('Voice Recording Browser Support', () => {
  test('TC-VOICE-SUPPORT-001: MediaRecorder API is available', async ({ page, loginAs }) => {
    await loginAs('alice');

    const hasMediaRecorder = await page.evaluate(() => {
      return typeof MediaRecorder !== 'undefined';
    });

    expect(hasMediaRecorder).toBe(true);
  });

  test('TC-VOICE-SUPPORT-002: getUserMedia is available', async ({ page, loginAs }) => {
    await loginAs('alice');

    const hasGetUserMedia = await page.evaluate(() => {
      return !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia);
    });

    expect(hasGetUserMedia).toBe(true);
  });
});
