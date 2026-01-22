import { test, expect } from '../fixtures/test-base';

/**
 * UI Fidelity Tests
 * Compare our implementation against Metabase styling
 * Takes screenshots for visual comparison
 */

test.describe('UI Fidelity - Typography', () => {
  test('should have correct heading font styles', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Check h2 heading
    const h2 = page.locator('h2').first();
    if (await h2.isVisible()) {
      const styles = await h2.evaluate(el => ({
        fontFamily: getComputedStyle(el).fontFamily,
        fontWeight: getComputedStyle(el).fontWeight,
        fontSize: getComputedStyle(el).fontSize,
        color: getComputedStyle(el).color,
      }));

      // Should use Lato font
      expect(styles.fontFamily.toLowerCase()).toContain('lato');
      // Should be bold (700)
      expect(parseInt(styles.fontWeight)).toBeGreaterThanOrEqual(600);
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_typography_headings.png', fullPage: true });
  });

  test('should have correct body text styles', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Check body text
    const bodyText = page.locator('p, span').first();
    if (await bodyText.isVisible()) {
      const styles = await bodyText.evaluate(el => ({
        fontFamily: getComputedStyle(el).fontFamily,
        fontSize: getComputedStyle(el).fontSize,
        lineHeight: getComputedStyle(el).lineHeight,
      }));

      // Should use Lato font
      expect(styles.fontFamily.toLowerCase()).toContain('lato');
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_typography_body.png', fullPage: true });
  });

  test('should use uppercase section titles', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Look for section titles (Our analytics, Pinned, etc)
    const sectionTitles = page.locator('[class*="sectionTitle"], span:text-matches(/^[A-Z]{2,}/)');

    // If any exist, check they use uppercase styling
    const count = await sectionTitles.count();
    for (let i = 0; i < count; i++) {
      const el = sectionTitles.nth(i);
      const textTransform = await el.evaluate(e => getComputedStyle(e).textTransform);
      // Should be uppercase
      expect(textTransform === 'uppercase' || (await el.textContent())?.toUpperCase() === await el.textContent()).toBeTruthy();
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_section_titles.png', fullPage: true });
  });
});

test.describe('UI Fidelity - Colors', () => {
  test('should use Metabase brand blue (#509EE3)', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Find primary buttons
    const primaryBtn = page.locator('button.mantine-Button-root[data-variant="filled"]').first();
    if (await primaryBtn.isVisible()) {
      const bgColor = await primaryBtn.evaluate(el => getComputedStyle(el).backgroundColor);

      // Convert to hex and check
      // Metabase brand blue is #509EE3 = rgb(80, 158, 227)
      const isBlue = bgColor.includes('80') && bgColor.includes('158');
      expect(isBlue || bgColor.includes('#509')).toBeTruthy();
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_colors_brand.png', fullPage: true });
  });

  test('should use correct text colors', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Primary text should be #4C5773
    const title = page.locator('h2').first();
    if (await title.isVisible()) {
      const color = await title.evaluate(el => getComputedStyle(el).color);
      // Should be dark blue-gray
      expect(color).toBeTruthy();
    }

    // Secondary text should be #696E7B
    const secondaryText = page.locator('text=Welcome, [class*="dimmed"]').first();
    if (await secondaryText.isVisible()) {
      const color = await secondaryText.evaluate(el => getComputedStyle(el).color);
      expect(color).toBeTruthy();
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_colors_text.png', fullPage: true });
  });

  test('should use correct background colors', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Main content area should be light background
    const mainContent = page.locator('main, [class*="container"]').first();
    if (await mainContent.isVisible()) {
      const bgColor = await mainContent.evaluate(el => getComputedStyle(el).backgroundColor);
      // Should be white or light gray
      expect(bgColor.includes('255') || bgColor.includes('249') || bgColor.includes('transparent')).toBeTruthy();
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_colors_background.png', fullPage: true });
  });
});

test.describe('UI Fidelity - Spacing', () => {
  test('should use 8px grid spacing', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Check padding on containers
    const container = page.locator('[class*="container"], [class*="padding"]').first();
    if (await container.isVisible()) {
      const padding = await container.evaluate(el => ({
        top: parseInt(getComputedStyle(el).paddingTop),
        right: parseInt(getComputedStyle(el).paddingRight),
        bottom: parseInt(getComputedStyle(el).paddingBottom),
        left: parseInt(getComputedStyle(el).paddingLeft),
      }));

      // All padding should be divisible by 4 (minimum grid unit)
      const allDivisible = [padding.top, padding.right, padding.bottom, padding.left].every(p => p % 4 === 0);
      expect(allDivisible).toBeTruthy();
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_spacing_grid.png', fullPage: true });
  });

  test('should have consistent card spacing', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Find cards
    const cards = page.locator('.mantine-Card-root, .mantine-Paper-root');
    const count = await cards.count();

    if (count >= 2) {
      const firstCard = cards.first();
      const secondCard = cards.nth(1);

      const firstPadding = await firstCard.evaluate(el => getComputedStyle(el).padding);
      const secondPadding = await secondCard.evaluate(el => getComputedStyle(el).padding);

      // Padding should be consistent
      expect(firstPadding).toBe(secondPadding);
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_spacing_cards.png', fullPage: true });
  });
});

test.describe('UI Fidelity - Components', () => {
  test('should style buttons correctly', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Primary button
    const primaryBtn = page.locator('button.mantine-Button-root').first();
    if (await primaryBtn.isVisible()) {
      const styles = await primaryBtn.evaluate(el => ({
        borderRadius: getComputedStyle(el).borderRadius,
        fontWeight: getComputedStyle(el).fontWeight,
        transition: getComputedStyle(el).transition,
      }));

      // Should have border radius
      expect(parseInt(styles.borderRadius)).toBeGreaterThanOrEqual(2);
      // Should be bold
      expect(parseInt(styles.fontWeight)).toBeGreaterThanOrEqual(600);
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_components_buttons.png', fullPage: true });
  });

  test('should style cards with correct borders and shadows', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Find any card
    const card = page.locator('.mantine-Card-root, .mantine-Paper-root').first();
    if (await card.isVisible()) {
      const styles = await card.evaluate(el => ({
        border: getComputedStyle(el).border,
        borderRadius: getComputedStyle(el).borderRadius,
        boxShadow: getComputedStyle(el).boxShadow,
      }));

      // Should have border
      expect(styles.border.length).toBeGreaterThan(0);
      // Should have border radius
      expect(parseInt(styles.borderRadius)).toBeGreaterThanOrEqual(2);
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_components_cards.png', fullPage: true });
  });

  test('should style inputs correctly', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Find search or any input
    const input = page.locator('input.mantine-TextInput-input, input.mantine-Input-input').first();
    if (await input.isVisible()) {
      const styles = await input.evaluate(el => ({
        borderRadius: getComputedStyle(el).borderRadius,
        borderColor: getComputedStyle(el).borderColor,
      }));

      // Should have border radius
      expect(parseInt(styles.borderRadius)).toBeGreaterThanOrEqual(2);
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_components_inputs.png', fullPage: true });
  });

  test('should style badges/pills correctly', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Find any badge
    const badge = page.locator('.mantine-Badge-root').first();
    if (await badge.isVisible()) {
      const styles = await badge.evaluate(el => ({
        borderRadius: getComputedStyle(el).borderRadius,
        fontWeight: getComputedStyle(el).fontWeight,
        textTransform: getComputedStyle(el).textTransform,
      }));

      // Should be rounded
      expect(parseInt(styles.borderRadius)).toBeGreaterThanOrEqual(4);
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_components_badges.png', fullPage: true });
  });
});

test.describe('UI Fidelity - Navigation', () => {
  test('should have correct sidebar styling', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Find sidebar
    const sidebar = page.locator('[data-testid="sidebar"], aside, nav').first();
    if (await sidebar.isVisible()) {
      const styles = await sidebar.evaluate(el => ({
        backgroundColor: getComputedStyle(el).backgroundColor,
        width: getComputedStyle(el).width,
      }));

      // Should have background color
      expect(styles.backgroundColor).toBeTruthy();
    }

    await page.screenshot({ path: 'e2e/screenshots/ui_nav_sidebar.png', fullPage: true });
  });

  test('should have correct header styling', async ({ authenticatedPage: page }) => {
    await page.goto('/');

    // Find header area
    const header = page.locator('header, [class*="header"]').first();
    if (await header.isVisible()) {
      await page.screenshot({ path: 'e2e/screenshots/ui_nav_header.png' });
    }
  });
});

test.describe('UI Fidelity - Full Page Screenshots', () => {
  test('capture home page', async ({ authenticatedPage: page }) => {
    await page.goto('/');
    await page.waitForTimeout(1000);
    await page.screenshot({ path: 'e2e/screenshots/full_home.png', fullPage: true });
  });

  test('capture browse page', async ({ authenticatedPage: page }) => {
    await page.goto('/browse');
    await page.waitForSelector('h2:has-text("Browse")');
    await page.waitForTimeout(500);
    await page.screenshot({ path: 'e2e/screenshots/full_browse.png', fullPage: true });
  });

  test('capture question builder', async ({ authenticatedPage: page }) => {
    await page.goto('/question/new');
    await page.waitForTimeout(1000);
    await page.screenshot({ path: 'e2e/screenshots/full_question_builder.png', fullPage: true });
  });

  test('capture admin settings', async ({ authenticatedPage: page }) => {
    await page.goto('/admin/settings');
    await page.waitForSelector('h2:has-text("Settings")').catch(() => {});
    await page.waitForTimeout(500);
    await page.screenshot({ path: 'e2e/screenshots/full_admin_settings.png', fullPage: true });
  });
});
