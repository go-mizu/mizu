import { test, expect, testUsers, generateTestUsername } from '../fixtures/test-fixtures';
import { AppPage } from '../pages/app.page';

test.describe('QR Code Friend Feature', () => {
  test.describe('QR Modal', () => {
    test('TC-QR-001: QR modal opens and shows user code', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Click QR button
      await page.click('button[title="QR Code"]');

      // Modal should be visible
      const modal = page.locator('#qr-modal');
      await expect(modal).toBeVisible();

      // Should show "My Code" tab by default
      const myCodeTab = page.locator('#qr-tab-my-code');
      await expect(myCodeTab).toHaveClass(/border-\[#25D366\]/);

      // QR code container should be visible
      const qrContainer = page.locator('#qr-code-container');
      await expect(qrContainer).toBeVisible();

      // Wait for code to load
      await page.waitForTimeout(1000);

      // Code text should be displayed
      const codeText = page.locator('#qr-code-text');
      await expect(codeText).not.toBeEmpty();
      const code = await codeText.textContent();
      expect(code).toMatch(/^MIZU-[A-Z0-9]+$/);
    });

    test('TC-QR-002: Copy code button works', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Open QR modal
      await page.click('button[title="QR Code"]');
      await expect(page.locator('#qr-modal')).toBeVisible();

      // Wait for code to load
      await page.waitForTimeout(1000);

      // Mock clipboard API
      await page.evaluate(() => {
        (window as any).clipboardText = '';
        (navigator.clipboard as any).writeText = (text: string) => {
          (window as any).clipboardText = text;
          return Promise.resolve();
        };
      });

      // Click copy code button
      await page.click('button:has-text("Copy Code")');

      // Check success notification
      await expect(page.locator('.bg-green-500:has-text("Code copied")')).toBeVisible();
    });

    test('TC-QR-003: Switch to Add Friend tab', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Open QR modal
      await page.click('button[title="QR Code"]');
      await expect(page.locator('#qr-modal')).toBeVisible();

      // Click Add Friend tab
      await page.click('#qr-tab-scan');

      // Add Friend tab should be active
      const scanTab = page.locator('#qr-tab-scan');
      await expect(scanTab).toHaveClass(/border-\[#25D366\]/);

      // Input field should be visible
      const input = page.locator('#friend-code-input');
      await expect(input).toBeVisible();
      await expect(input).toBeFocused();
    });

    test('TC-QR-004: Close modal with X button', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Open QR modal
      await page.click('button[title="QR Code"]');
      await expect(page.locator('#qr-modal')).toBeVisible();

      // Close modal
      await page.click('#qr-modal button:has(svg path[d*="M6 18L18 6"])');

      // Modal should be hidden
      await expect(page.locator('#qr-modal')).toBeHidden();
    });
  });

  test.describe('Friend Code Resolution', () => {
    test('TC-QR-005: Resolve valid friend code shows user info', async ({ page, loginAs, context }) => {
      // First, get a friend code from bob
      const bobPage = await context.newPage();
      await bobPage.goto('/login');
      await bobPage.fill('#login', testUsers.bob.username);
      await bobPage.fill('#password', testUsers.bob.password);
      await bobPage.click('button[type="submit"]');
      await bobPage.waitForURL('/app');

      // Open QR modal as bob and get code
      await bobPage.click('button[title="QR Code"]');
      await bobPage.waitForTimeout(1000);
      const bobCode = await bobPage.locator('#qr-code-text').textContent();
      await bobPage.close();

      // Now login as alice and try to add bob
      await loginAs('alice');
      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Open QR modal and switch to Add Friend tab
      await page.click('button[title="QR Code"]');
      await page.click('#qr-tab-scan');

      // Enter bob's code
      await page.fill('#friend-code-input', bobCode!);
      await page.click('button:has-text("Look Up Code")');

      // Wait for friend preview modal
      await expect(page.locator('#friend-preview-modal')).toBeVisible({ timeout: 5000 });

      // Should show bob's info
      await expect(page.locator('#friend-preview-name')).toContainText('Bob');
      await expect(page.locator('#friend-preview-username')).toContainText('@bob');
    });

    test('TC-QR-006: Invalid code shows error', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Open QR modal and switch to Add Friend tab
      await page.click('button[title="QR Code"]');
      await page.click('#qr-tab-scan');

      // Enter invalid code
      await page.fill('#friend-code-input', 'INVALID-CODE');
      await page.click('button:has-text("Look Up Code")');

      // Should show error
      const result = page.locator('#friend-code-result');
      await expect(result).toBeVisible();
      await expect(result).toContainText(/Invalid|expired/i);
    });
  });

  test.describe('Add Friend Flow', () => {
    test('TC-QR-007: Add friend via code creates contact', async ({ page, loginAs, context }) => {
      // Create a new user to add
      const newUsername = generateTestUsername();
      const newPage = await context.newPage();
      await newPage.goto('/register');
      await newPage.fill('#username', newUsername);
      await newPage.fill('#password', 'password123');
      await newPage.click('button[type="submit"]');
      await newPage.waitForURL('/app');

      // Get friend code
      await newPage.click('button[title="QR Code"]');
      await newPage.waitForTimeout(1000);
      const friendCode = await newPage.locator('#qr-code-text').textContent();
      await newPage.close();

      // Login as alice
      await loginAs('alice');
      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Open QR modal and add friend
      await page.click('button[title="QR Code"]');
      await page.click('#qr-tab-scan');
      await page.fill('#friend-code-input', friendCode!);
      await page.click('button:has-text("Look Up Code")');

      // Wait for preview modal
      await expect(page.locator('#friend-preview-modal')).toBeVisible({ timeout: 5000 });

      // Confirm add friend
      await page.click('button:has-text("Add Friend")');

      // Should show success
      await expect(page.locator('.bg-green-500:has-text("Friend added")')).toBeVisible({ timeout: 5000 });

      // Friend preview modal should close
      await expect(page.locator('#friend-preview-modal')).toBeHidden();
    });

    test('TC-QR-008: Cannot add self via own code', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Get own code
      await page.click('button[title="QR Code"]');
      await page.waitForTimeout(1000);
      const ownCode = await page.locator('#qr-code-text').textContent();

      // Switch to Add Friend tab
      await page.click('#qr-tab-scan');

      // Enter own code
      await page.fill('#friend-code-input', ownCode!);
      await page.click('button:has-text("Look Up Code")');

      // Wait for preview modal and try to add
      await expect(page.locator('#friend-preview-modal')).toBeVisible({ timeout: 5000 });
      await page.click('button:has-text("Add Friend")');

      // Should show error
      await expect(page.locator('.bg-red-500:has-text("yourself")')).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('Deep Link', () => {
    test('TC-QR-009: Deep link redirects authenticated user to app', async ({ page, loginAs }) => {
      await loginAs('alice');

      // Navigate to a friend code deep link (using a fake code for redirect test)
      await page.goto('/add-friend/MIZU-TESTCODE');

      // Should redirect to /app with add-friend param
      await expect(page).toHaveURL(/\/app/);
    });

    test('TC-QR-010: Deep link redirects unauthenticated user to login', async ({ page }) => {
      // Navigate to add-friend link without being logged in
      await page.goto('/add-friend/MIZU-TESTCODE');

      // Should redirect to login with next param
      await expect(page).toHaveURL(/\/login.*next=/);
    });
  });

  test.describe('Code Revocation', () => {
    test('TC-QR-011: Generate code produces new code each session', async ({ page, loginAs }) => {
      await loginAs('alice');

      const appPage = new AppPage(page);
      await appPage.waitForLoad();

      // Get first code
      await page.click('button[title="QR Code"]');
      await page.waitForTimeout(1000);
      const firstCode = await page.locator('#qr-code-text').textContent();

      // Code should be in expected format
      expect(firstCode).toMatch(/^MIZU-[A-Z0-9]+$/);

      // Close and reopen - should show same code (cached within expiration)
      await page.click('#qr-modal button:has(svg path[d*="M6 18L18 6"])');
      await page.click('button[title="QR Code"]');
      await page.waitForTimeout(1000);
      const secondCode = await page.locator('#qr-code-text').textContent();

      // Should be same code (not expired yet)
      expect(secondCode).toBe(firstCode);
    });
  });
});
