import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class MenusPage {
  constructor(private page: Page) {}

  // Navigation
  async goto() {
    await this.page.goto(URLS.menus);
  }

  async gotoLegacy() {
    await this.page.goto(URLS.legacy.menus);
  }

  // Page assertions
  async expectMenusPage() {
    await expect(this.page.locator(SELECTORS.menuItemsList).or(this.page.locator('[data-testid="menu-builder"]'))).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  // Menu selection
  async selectMenu(menuName: string) {
    await this.page.selectOption('#select-menu-to-edit, select[name="menu"]', { label: menuName });
    await this.page.click('.submit-select-menu, input[value="Select"]');
  }

  async createNewMenu() {
    await this.page.click('.menu-add-new, a:has-text("create a new menu")');
  }

  // Menu creation
  async fillMenuName(name: string) {
    await this.page.fill('#menu-name, input[name="menu-name"]', name);
  }

  async saveMenu() {
    await this.page.click('#save_menu_header, input[value="Save Menu"]');
  }

  async createMenu(name: string) {
    await this.createNewMenu();
    await this.fillMenuName(name);
    await this.saveMenu();
  }

  // Adding items
  async expandAvailableSection(section: string) {
    await this.page.click(`.accordion-section:has-text("${section}") .accordion-section-title`);
  }

  async addPageToMenu(pageTitle: string) {
    await this.expandAvailableSection('Pages');
    await this.page.check(`input[value="${pageTitle}"], label:has-text("${pageTitle}") input`);
    await this.page.click('#submit-posttype-page, .button-secondary:has-text("Add to Menu")');
  }

  async addPostToMenu(postTitle: string) {
    await this.expandAvailableSection('Posts');
    await this.page.check(`input[value="${postTitle}"], label:has-text("${postTitle}") input`);
    await this.page.click('#submit-posttype-post, .button-secondary:has-text("Add to Menu")');
  }

  async addCategoryToMenu(categoryName: string) {
    await this.expandAvailableSection('Categories');
    await this.page.check(`input[value="${categoryName}"], label:has-text("${categoryName}") input`);
    await this.page.click('#submit-taxonomy-category, .button-secondary:has-text("Add to Menu")');
  }

  async addCustomLink(url: string, label: string) {
    await this.expandAvailableSection('Custom Links');
    await this.page.fill('#custom-menu-item-url, input[name="menu-item-url"]', url);
    await this.page.fill('#custom-menu-item-name, input[name="menu-item-title"]', label);
    await this.page.click('#submit-customlinkdiv, .button-secondary:has-text("Add to Menu")');
  }

  // Menu item operations
  async expectMenuItemInList(label: string) {
    await expect(this.page.locator(`${SELECTORS.menuItemsList} >> text=${label}`)).toBeVisible();
  }

  async expectMenuItemNotInList(label: string) {
    await expect(this.page.locator(`${SELECTORS.menuItemsList} >> text=${label}`)).not.toBeVisible();
  }

  async openMenuItemSettings(label: string) {
    await this.page.click(`li.menu-item:has-text("${label}") .item-edit`);
  }

  async editMenuItemLabel(oldLabel: string, newLabel: string) {
    await this.openMenuItemSettings(oldLabel);
    await this.page.fill(`li.menu-item:has-text("${oldLabel}") input.edit-menu-item-title`, newLabel);
  }

  async removeMenuItem(label: string) {
    await this.openMenuItemSettings(label);
    await this.page.click(`li.menu-item:has-text("${label}") .item-delete`);
  }

  // Menu locations
  async assignMenuLocation(location: string) {
    await this.page.check(`input[name="menu-locations[${location}]"], label:has-text("${location}") input`);
  }

  async unassignMenuLocation(location: string) {
    await this.page.uncheck(`input[name="menu-locations[${location}]"], label:has-text("${location}") input`);
  }

  // Delete menu
  async deleteMenu() {
    await this.page.click('.menu-delete, a:has-text("Delete Menu")');
  }

  async expectSuccessNotice() {
    await expect(this.page.locator('.notice-success, .updated')).toBeVisible();
  }
}
