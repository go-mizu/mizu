import { Page, Locator, expect } from '@playwright/test';

export class AppPage {
  readonly page: Page;

  // Sidebar elements
  readonly sidebar: Locator;
  readonly userName: Locator;
  readonly userAvatar: Locator;
  readonly newChatButton: Locator;
  readonly themeToggle: Locator;
  readonly settingsLink: Locator;
  readonly searchInput: Locator;
  readonly chatList: Locator;

  // Empty state
  readonly emptyState: Locator;

  // Chat view elements
  readonly chatView: Locator;
  readonly chatHeader: Locator;
  readonly chatName: Locator;
  readonly chatStatus: Locator;
  readonly chatAvatar: Locator;
  readonly backButton: Locator;
  readonly voiceCallButton: Locator;
  readonly videoCallButton: Locator;
  readonly optionsButton: Locator;

  // Messages
  readonly messagesContainer: Locator;
  readonly typingIndicator: Locator;

  // Message input
  readonly messageForm: Locator;
  readonly messageInput: Locator;
  readonly attachButton: Locator;
  readonly sendButton: Locator;

  // New chat modal
  readonly newChatModal: Locator;
  readonly contactSearch: Locator;
  readonly contactList: Locator;
  readonly createGroupButton: Locator;
  readonly closeModalButton: Locator;

  constructor(page: Page) {
    this.page = page;

    // Sidebar
    this.sidebar = page.locator('#sidebar');
    this.userName = page.locator('#user-name');
    this.userAvatar = page.locator('#user-avatar');
    this.newChatButton = page.locator('button[title="New chat"]');
    this.themeToggle = page.locator('button[title="Toggle theme"]');
    this.settingsLink = page.locator('a[href="/settings"]');
    this.searchInput = page.locator('#search-input');
    this.chatList = page.locator('#chat-list');

    // Empty state
    this.emptyState = page.locator('#empty-state');

    // Chat view
    this.chatView = page.locator('#chat-view');
    this.chatHeader = page.locator('#chat-view header, #chat-view > div:first-child');
    this.chatName = page.locator('#chat-name');
    this.chatStatus = page.locator('#chat-status');
    this.chatAvatar = page.locator('#chat-avatar');
    this.backButton = page.locator('#back-btn');
    this.voiceCallButton = page.locator('button[title="Voice call"]');
    this.videoCallButton = page.locator('button[title="Video call"]');
    this.optionsButton = page.locator('button[title="More options"]');

    // Messages
    this.messagesContainer = page.locator('#messages');
    this.typingIndicator = page.locator('#typing-indicator');

    // Message input
    this.messageForm = page.locator('#message-form');
    this.messageInput = page.locator('#message-input');
    this.attachButton = page.locator('button[title="Attach file"]');
    this.sendButton = page.locator('#message-form button[type="submit"]');

    // New chat modal
    this.newChatModal = page.locator('#new-chat-modal');
    this.contactSearch = page.locator('#contact-search');
    this.contactList = page.locator('#contact-list');
    this.createGroupButton = page.locator('button:has-text("Create New Group")');
    this.closeModalButton = this.newChatModal.locator('button').first();
  }

  async goto(): Promise<void> {
    await this.page.goto('/app');
  }

  async waitForLoad(): Promise<void> {
    await expect(this.sidebar).toBeVisible();
    await expect(this.userName).not.toBeEmpty();
  }

  // Chat list interactions
  async getChatItems(): Promise<Locator> {
    return this.chatList.locator('> div[onclick]');
  }

  async selectChat(chatName: string): Promise<void> {
    const chatItem = this.chatList.locator(`div:has-text("${chatName}")`).first();
    await chatItem.click();
  }

  async selectChatByIndex(index: number): Promise<void> {
    const chatItems = await this.getChatItems();
    await chatItems.nth(index).click();
  }

  async expectChatViewVisible(): Promise<void> {
    await expect(this.chatView).toBeVisible();
    await expect(this.emptyState).toBeHidden();
  }

  async expectEmptyStateVisible(): Promise<void> {
    await expect(this.emptyState).toBeVisible();
    await expect(this.chatView).toBeHidden();
  }

  // Messaging
  async sendMessage(text: string): Promise<void> {
    await this.messageInput.fill(text);
    await this.sendButton.click();
  }

  async sendMessageWithEnter(text: string): Promise<void> {
    await this.messageInput.fill(text);
    await this.messageInput.press('Enter');
  }

  async getMessages(): Promise<Locator> {
    return this.messagesContainer.locator('.sent-bubble, .received-bubble');
  }

  async getLastMessage(): Promise<Locator> {
    const messages = await this.getMessages();
    return messages.last();
  }

  async expectMessageCount(count: number): Promise<void> {
    const messages = await this.getMessages();
    await expect(messages).toHaveCount(count);
  }

  async expectLastMessageText(text: string): Promise<void> {
    const lastMessage = await this.getLastMessage();
    await expect(lastMessage).toContainText(text);
  }

  async expectTypingIndicator(visible: boolean): Promise<void> {
    if (visible) {
      await expect(this.typingIndicator).toBeVisible();
    } else {
      await expect(this.typingIndicator).toBeHidden();
    }
  }

  // Search
  async searchChats(query: string): Promise<void> {
    await this.searchInput.fill(query);
  }

  async clearSearch(): Promise<void> {
    await this.searchInput.clear();
  }

  // New chat modal
  async openNewChatModal(): Promise<void> {
    await this.newChatButton.click();
    await expect(this.newChatModal).toBeVisible();
  }

  async closeNewChatModal(): Promise<void> {
    await this.closeModalButton.click();
    await expect(this.newChatModal).toBeHidden();
  }

  async searchContacts(query: string): Promise<void> {
    await this.contactSearch.fill(query);
  }

  async selectContact(name: string): Promise<void> {
    const contact = this.contactList.locator(`div:has-text("${name}")`).first();
    await contact.click();
  }

  // Theme
  async toggleTheme(): Promise<void> {
    await this.themeToggle.click();
  }

  async expectDarkTheme(): Promise<void> {
    await expect(this.page.locator('html')).toHaveAttribute('data-theme', 'dark');
  }

  async expectLightTheme(): Promise<void> {
    await expect(this.page.locator('html')).toHaveAttribute('data-theme', 'light');
  }

  // Navigation
  async goToSettings(): Promise<void> {
    await this.settingsLink.click();
    await this.page.waitForURL('/settings');
  }

  // Mobile back navigation
  async goBackToList(): Promise<void> {
    await this.backButton.click();
    await this.expectEmptyStateVisible();
  }

  // Unread badges
  async getUnreadBadge(chatName: string): Promise<Locator> {
    const chatItem = this.chatList.locator(`div:has-text("${chatName}")`).first();
    return chatItem.locator('.rounded-full.accent');
  }
}
