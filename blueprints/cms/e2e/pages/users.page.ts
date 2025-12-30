import { Page, expect } from '@playwright/test';
import { URLS, SELECTORS } from '../fixtures/test-fixtures';

export class UsersPage {
  constructor(private page: Page) {}

  // Navigation
  async goto() {
    await this.page.goto(URLS.users);
  }

  async gotoNew() {
    await this.page.goto(URLS.usersNew);
  }

  async gotoEdit(id: string) {
    await this.page.goto(URLS.userEdit(id));
  }

  async gotoProfile() {
    await this.page.goto(URLS.profile);
  }

  async gotoLegacy() {
    await this.page.goto(URLS.legacy.users);
  }

  // List page assertions
  async expectListPage() {
    await expect(this.page.locator(SELECTORS.table)).toBeVisible();
  }

  async expectPageTitle(title: string) {
    await expect(this.page.locator(SELECTORS.pageTitle)).toContainText(title);
  }

  async expectUserInList(username: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${username}`)).toBeVisible();
  }

  async expectUserNotInList(username: string) {
    await expect(this.page.locator(`${SELECTORS.table} >> text=${username}`)).not.toBeVisible();
  }

  // Role tabs
  async clickRoleTab(role: string) {
    await this.page.click(`${SELECTORS.statusTabs} >> text=${role}`);
  }

  // Search
  async search(query: string) {
    await this.page.fill(SELECTORS.searchBox, query);
    await this.page.click(SELECTORS.searchSubmit);
  }

  // Row actions
  async clickRowAction(username: string, action: string) {
    const row = this.page.locator(`${SELECTORS.tableRow}:has-text("${username}")`);
    await row.hover();
    await row.locator(`text=${action}`).click();
  }

  async editUser(username: string) {
    await this.clickRowAction(username, 'Edit');
  }

  async deleteUser(username: string) {
    await this.clickRowAction(username, 'Delete');
  }

  // Bulk actions
  async selectUser(index: number) {
    await this.page.locator(`${SELECTORS.tableRow}:nth-child(${index + 1}) ${SELECTORS.tableCheckbox}`).check();
  }

  async selectAllUsers() {
    await this.page.locator('thead ' + SELECTORS.tableCheckbox).check();
  }

  async applyBulkAction(action: string) {
    await this.page.selectOption(SELECTORS.bulkActions, action);
    await this.page.click(SELECTORS.bulkApply);
  }

  // New user form
  async expectNewUserPage() {
    await expect(this.page.locator('#user_login, input[name="user_login"]')).toBeVisible();
  }

  async fillUsername(username: string) {
    await this.page.fill('#user_login, input[name="user_login"]', username);
  }

  async fillEmail(email: string) {
    await this.page.fill(SELECTORS.emailInput, email);
  }

  async fillFirstName(firstName: string) {
    await this.page.fill('#first_name, input[name="first_name"]', firstName);
  }

  async fillLastName(lastName: string) {
    await this.page.fill('#last_name, input[name="last_name"]', lastName);
  }

  async fillPassword(password: string) {
    await this.page.fill(SELECTORS.passwordInput, password);
  }

  async selectRole(role: string) {
    await this.page.selectOption(SELECTORS.roleSelect, role);
  }

  async submitNewUser() {
    await this.page.click('#createusersub, input[value="Add New User"]');
  }

  async createUser(username: string, email: string, password: string, role: string = 'subscriber') {
    await this.gotoNew();
    await this.fillUsername(username);
    await this.fillEmail(email);
    await this.fillPassword(password);
    await this.selectRole(role);
    await this.submitNewUser();
  }

  // Edit user page
  async expectEditPage() {
    await expect(this.page.locator(SELECTORS.emailInput)).toBeVisible();
  }

  async save() {
    await this.page.click(SELECTORS.submitButton);
  }

  async expectSuccessNotice() {
    await expect(this.page.locator('.notice-success, .updated')).toBeVisible();
  }

  // Profile page
  async expectProfilePage() {
    await expect(this.page.locator('#your-profile, [data-testid="profile-form"]')).toBeVisible();
  }

  async selectColorScheme(scheme: string) {
    await this.page.click(`input[name="admin_color"][value="${scheme}"]`);
  }

  async saveProfile() {
    await this.page.click('#submit, input[value="Update Profile"]');
  }
}
