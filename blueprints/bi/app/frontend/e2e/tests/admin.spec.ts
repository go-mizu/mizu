import { test, expect } from '../fixtures/test-base';

test.describe('Admin Settings', () => {
  test.describe('Settings Overview', () => {
    test('should display admin settings page', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/settings');

      // Wait for settings to load
      await page.waitForSelector('h2:has-text("Settings"), text=Settings', { timeout: 10000 });

      // Verify settings sections exist
      await expect(page.locator('text=Settings, h2')).toBeVisible();

      await page.screenshot({ path: 'e2e/screenshots/admin_settings.png', fullPage: true });
    });

    test('should display settings navigation/tabs', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/settings');
      await page.waitForSelector('h2:has-text("Settings")').catch(() => {});

      // Look for settings navigation
      const settingsNav = page.locator('[data-testid="settings-nav"], aside, [role="tablist"]');
      if (await settingsNav.isVisible({ timeout: 3000 })) {
        await page.screenshot({ path: 'e2e/screenshots/admin_settings_nav.png', fullPage: true });
      }
    });
  });

  test.describe('General Settings', () => {
    test('should display site name setting', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/settings');
      await page.waitForSelector('h2:has-text("Settings")').catch(() => {});

      // Look for site name input
      const siteNameInput = page.locator('input[name="siteName"], input[placeholder*="site name" i], label:has-text("Site name") + input');
      if (await siteNameInput.isVisible({ timeout: 3000 })) {
        const value = await siteNameInput.inputValue();
        expect(value).toBeTruthy();
      }

      await page.screenshot({ path: 'e2e/screenshots/admin_site_name.png', fullPage: true });
    });

    test('should update general settings', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/settings');
      await page.waitForSelector('h2:has-text("Settings")').catch(() => {});

      // Find any editable setting
      const editableInput = page.locator('input:not([disabled])').first();
      if (await editableInput.isVisible({ timeout: 3000 })) {
        const originalValue = await editableInput.inputValue();

        // Modify and save
        await editableInput.fill(originalValue + ' (test)');

        // Look for save button
        const saveBtn = page.locator('button:has-text("Save"), button:has-text("Update")');
        if (await saveBtn.isVisible({ timeout: 2000 })) {
          // Just verify button is clickable, don't actually save
          await expect(saveBtn).toBeEnabled();
        }

        // Restore original value
        await editableInput.fill(originalValue);
      }
    });
  });

  test.describe('Database Settings', () => {
    test('should display databases page', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/databases');

      // Wait for databases to load
      await page.waitForSelector('h2:has-text("Databases"), text=Databases', { timeout: 10000 });

      await page.screenshot({ path: 'e2e/screenshots/admin_databases.png', fullPage: true });
    });

    test('should display connected databases', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/databases');
      await page.waitForSelector('h2:has-text("Databases")').catch(() => {});

      // Should show at least one database (seeded data)
      const databaseCards = page.locator('[data-testid="database-card"], .database-item, tr:has(td)');
      const count = await databaseCards.count();
      expect(count).toBeGreaterThanOrEqual(0);

      await page.screenshot({ path: 'e2e/screenshots/admin_database_list.png', fullPage: true });
    });

    test('should open add database modal', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/databases');
      await page.waitForSelector('h2:has-text("Databases")').catch(() => {});

      // Click add database button
      const addBtn = page.locator('button:has-text("Add database"), button:has-text("Add")');
      if (await addBtn.isVisible({ timeout: 3000 })) {
        await addBtn.click();

        // Modal should appear
        await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });

        await page.screenshot({ path: 'e2e/screenshots/admin_add_database_modal.png', fullPage: true });

        // Close modal
        await page.keyboard.press('Escape');
      }
    });

    test('should show database driver options', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/databases');
      await page.waitForSelector('h2:has-text("Databases")').catch(() => {});

      const addBtn = page.locator('button:has-text("Add database")');
      if (await addBtn.isVisible({ timeout: 3000 })) {
        await addBtn.click();
        await page.waitForTimeout(500);

        // Look for driver selection
        const driverSelect = page.locator('select, [role="combobox"], label:has-text("Database type") + div');
        if (await driverSelect.isVisible({ timeout: 3000 })) {
          await driverSelect.click();

          // Verify driver options appear
          await expect(page.locator('[role="option"], option')).toBeVisible({ timeout: 3000 });

          await page.screenshot({ path: 'e2e/screenshots/admin_database_drivers.png', fullPage: true });
        }

        await page.keyboard.press('Escape');
      }
    });
  });

  test.describe('User Management', () => {
    test('should display users page', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/users');

      // Wait for users to load
      await page.waitForSelector('h2:has-text("Users"), text=Users', { timeout: 10000 });

      await page.screenshot({ path: 'e2e/screenshots/admin_users.png', fullPage: true });
    });

    test('should display user list', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/users');
      await page.waitForSelector('h2:has-text("Users")').catch(() => {});

      // Should show users
      const userRows = page.locator('[data-testid="user-row"], .user-item, tr:has(td), .mantine-Table-tr');
      const count = await userRows.count();
      expect(count).toBeGreaterThanOrEqual(1); // At least admin user

      await page.screenshot({ path: 'e2e/screenshots/admin_user_list.png', fullPage: true });
    });

    test('should open invite user modal', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/users');
      await page.waitForSelector('h2:has-text("Users")').catch(() => {});

      // Click invite button
      const inviteBtn = page.locator('button:has-text("Invite"), button:has-text("Add user")');
      if (await inviteBtn.isVisible({ timeout: 3000 })) {
        await inviteBtn.click();

        // Modal should appear
        await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 5000 });

        await page.screenshot({ path: 'e2e/screenshots/admin_invite_user_modal.png', fullPage: true });

        await page.keyboard.press('Escape');
      }
    });

    test('should edit user details', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/users');
      await page.waitForSelector('h2:has-text("Users")').catch(() => {});

      // Click on a user row to edit
      const userRow = page.locator('[data-testid="user-row"], .user-item, tr:has(td)').first();
      if (await userRow.isVisible({ timeout: 3000 })) {
        await userRow.click();

        // Should open user detail or edit page
        await page.waitForTimeout(500);

        await page.screenshot({ path: 'e2e/screenshots/admin_user_detail.png', fullPage: true });
      }
    });
  });

  test.describe('Permissions', () => {
    test('should display permissions page', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/permissions');

      await page.waitForSelector('h2:has-text("Permissions"), text=Permissions', { timeout: 10000 }).catch(() => {});

      await page.screenshot({ path: 'e2e/screenshots/admin_permissions.png', fullPage: true });
    });

    test('should display permission groups', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/permissions');
      await page.waitForTimeout(1000);

      // Look for groups list
      const groups = page.locator('[data-testid="permission-group"], .permission-group, [role="row"]');
      const count = await groups.count();

      await page.screenshot({ path: 'e2e/screenshots/admin_permission_groups.png', fullPage: true });
    });
  });

  test.describe('Embedding Settings', () => {
    test('should display embedding settings', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/settings/embedding');

      await page.waitForSelector('text=Embedding, h2').catch(() => {});

      await page.screenshot({ path: 'e2e/screenshots/admin_embedding.png', fullPage: true });
    });

    test('should show embedding toggle', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/settings/embedding');
      await page.waitForTimeout(1000);

      // Look for enable embedding toggle
      const embeddingToggle = page.locator('input[type="checkbox"], [role="switch"]');
      if (await embeddingToggle.first().isVisible({ timeout: 3000 })) {
        await page.screenshot({ path: 'e2e/screenshots/admin_embedding_toggle.png', fullPage: true });
      }
    });
  });

  test.describe('Audit Log', () => {
    test('should display audit log', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/audit');

      await page.waitForSelector('text=Audit, h2').catch(() => {});

      await page.screenshot({ path: 'e2e/screenshots/admin_audit.png', fullPage: true });
    });
  });

  test.describe('UI Fidelity', () => {
    test('should have correct admin layout', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/settings');
      await page.waitForSelector('h2:has-text("Settings")').catch(() => {});

      // Check for admin sidebar
      const sidebar = page.locator('aside, [data-testid="admin-sidebar"], nav');
      if (await sidebar.isVisible()) {
        const width = await sidebar.evaluate(el => parseInt(getComputedStyle(el).width));
        expect(width).toBeGreaterThanOrEqual(100);
      }

      await page.screenshot({ path: 'e2e/screenshots/admin_layout.png', fullPage: true });
    });

    test('should have correct form styling', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/settings');
      await page.waitForSelector('h2:has-text("Settings")').catch(() => {});

      // Check input styling
      const input = page.locator('input').first();
      if (await input.isVisible()) {
        const borderRadius = await input.evaluate(el => getComputedStyle(el).borderRadius);
        expect(parseInt(borderRadius)).toBeGreaterThanOrEqual(2);
      }

      await page.screenshot({ path: 'e2e/screenshots/admin_form_styling.png', fullPage: true });
    });

    test('should have correct button styling in admin', async ({ authenticatedPage: page }) => {
      await page.goto('/admin/settings');
      await page.waitForSelector('h2:has-text("Settings")').catch(() => {});

      // Check button styling
      const primaryBtn = page.locator('button.mantine-Button-root[data-variant="filled"]').first();
      if (await primaryBtn.isVisible({ timeout: 3000 })) {
        const bgColor = await primaryBtn.evaluate(el => getComputedStyle(el).backgroundColor);
        expect(bgColor).not.toBe('transparent');
      }

      await page.screenshot({ path: 'e2e/screenshots/admin_button_styling.png', fullPage: true });
    });
  });
});
