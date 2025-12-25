# Messaging Blueprint - E2E Testing Specification

## Overview

This document defines comprehensive end-to-end (E2E) test cases for the Messaging Blueprint application. The tests verify all user flows work correctly from a user's perspective, covering authentication, messaging, real-time updates, and settings management.

## UI/UX Review

### Current Design Analysis

The messaging app follows a **WhatsApp-inspired design** with a modern dark theme:

**Strengths:**
- Clean two-panel layout (sidebar + main content)
- Consistent color palette with accent green (#25D366)
- Dark/light theme support with CSS custom properties
- Responsive design considerations (mobile back button)
- Message status indicators (sent/delivered/read)
- Typing indicators and presence display

**Areas for Enhancement (Modern shadcn + ChatGPT Style):**
1. **Typography**: Consider variable font weights, better hierarchy
2. **Spacing**: More generous padding, breathing room
3. **Animations**: Subtle transitions for message appearance, modals
4. **Empty States**: More engaging illustrations instead of emoji
5. **Loading States**: Skeleton loaders for chat list and messages
6. **Message Bubbles**: Rounded corners, subtle shadows
7. **Input Area**: Floating input with backdrop blur effect
8. **Avatars**: Gradient backgrounds, better fallback initials

### Design Tokens (Current)

```css
Dark Theme:
- Background Primary: #0f0f0f
- Background Secondary: #1a1a1a
- Background Tertiary: #252525
- Text Primary: #f5f5f5
- Text Secondary: #a0a0a0
- Accent: #25D366
- Sent Bubble: #005c4b
- Received Bubble: #202c33

Light Theme:
- Background Primary: #ffffff
- Background Secondary: #f0f2f5
- Sent Bubble: #d9fdd3
- Received Bubble: #ffffff
```

---

## Test Categories

### 1. Authentication Flow Tests

#### TC-AUTH-001: User Registration - Happy Path
**Description**: New user successfully creates an account

**Preconditions**:
- User is not logged in
- User is on the home page

**Steps**:
1. Navigate to `/register`
2. Fill in display name: "Test User"
3. Fill in username: "testuser_<timestamp>"
4. Fill in email: "test@example.com" (optional)
5. Fill in password: "password123"
6. Click "Create Account" button
7. Wait for redirect

**Expected Results**:
- Form submits successfully
- User is redirected to `/app`
- Session cookie is set
- User appears in sidebar header

**Test Data**:
```json
{
  "display_name": "Test User",
  "username": "testuser_{{timestamp}}",
  "email": "test@example.com",
  "password": "password123"
}
```

---

#### TC-AUTH-002: User Registration - Duplicate Username
**Description**: Registration fails with existing username

**Steps**:
1. Register a user with username "duplicate_user"
2. Logout
3. Navigate to `/register`
4. Attempt to register with same username

**Expected Results**:
- Error message displays: "Username already taken" or similar
- User remains on registration page
- No session is created

---

#### TC-AUTH-003: User Registration - Validation Errors
**Description**: Form validation prevents invalid submissions

**Test Cases**:
| Field | Input | Expected Error |
|-------|-------|----------------|
| username | empty | "Username is required" |
| username | "ab" | "Username too short" |
| password | empty | "Password is required" |
| password | "12345" | "Password must be at least 6 characters" |
| email | "invalid" | "Invalid email format" |

---

#### TC-AUTH-004: User Login - Happy Path
**Description**: Existing user logs in successfully

**Preconditions**:
- User account exists (username: alice, password: password123)

**Steps**:
1. Navigate to `/login`
2. Enter login: "alice"
3. Enter password: "password123"
4. Click "Sign In" button

**Expected Results**:
- User is redirected to `/app`
- User's name appears in sidebar
- Chat list loads

---

#### TC-AUTH-005: User Login - Invalid Credentials
**Description**: Login fails with wrong password

**Steps**:
1. Navigate to `/login`
2. Enter login: "alice"
3. Enter password: "wrongpassword"
4. Click "Sign In"

**Expected Results**:
- Error message: "Invalid credentials" or similar
- User remains on login page
- Password field may be cleared

---

#### TC-AUTH-006: User Login - Non-existent User
**Description**: Login fails for unknown username

**Steps**:
1. Navigate to `/login`
2. Enter login: "nonexistent_user_12345"
3. Enter password: "anypassword"
4. Click "Sign In"

**Expected Results**:
- Error message displays
- No authentication occurs

---

#### TC-AUTH-007: Session Persistence
**Description**: User remains logged in after page refresh

**Steps**:
1. Login as existing user
2. Navigate to `/app`
3. Refresh the page (F5)

**Expected Results**:
- User remains on `/app`
- User information still displayed
- Chat list reloads

---

#### TC-AUTH-008: Logout Flow
**Description**: User can logout successfully

**Preconditions**: User is logged in

**Steps**:
1. Navigate to `/settings`
2. Scroll to "Danger Zone"
3. Click "Log Out" button

**Expected Results**:
- User is redirected to `/` (home)
- Session cookie is cleared
- Navigating to `/app` redirects to `/login`

---

#### TC-AUTH-009: Protected Route Access
**Description**: Unauthenticated users cannot access app

**Preconditions**: User is not logged in

**Steps**:
1. Navigate directly to `/app`

**Expected Results**:
- User is redirected to `/login`

---

### 2. Chat List & Navigation Tests

#### TC-CHAT-001: Empty Chat List
**Description**: New user sees empty state

**Preconditions**:
- User just registered, no chats exist

**Steps**:
1. Login as new user
2. Observe chat list area

**Expected Results**:
- "No chats yet" message displays
- "Start a new conversation!" hint shown

---

#### TC-CHAT-002: Chat List Population
**Description**: Existing chats display correctly

**Preconditions**:
- User has existing chats with messages

**Steps**:
1. Login as user with chats
2. Observe chat list

**Expected Results**:
- Each chat shows:
  - Avatar (first letter of name)
  - Chat name (other user's name for direct, group name for group)
  - Last message preview (truncated)
  - Timestamp of last message
  - Unread count badge (if applicable)
- Chats sorted by last message time (most recent first)

---

#### TC-CHAT-003: Chat Selection
**Description**: Clicking a chat opens conversation

**Steps**:
1. Login with existing chats
2. Click on a chat in the list

**Expected Results**:
- Empty state hides
- Chat view appears
- Chat header shows name and status
- Messages load and display
- Selected chat is highlighted in list

---

#### TC-CHAT-004: Chat Search/Filter
**Description**: Search filters chat list

**Steps**:
1. Login with multiple chats
2. Type partial name in search input

**Expected Results**:
- Chat list filters to matching chats
- Non-matching chats are hidden
- Clearing search shows all chats

---

#### TC-CHAT-005: New Chat Modal
**Description**: User can start new conversation

**Steps**:
1. Click "+" button in sidebar header
2. Modal opens with contact search

**Expected Results**:
- Modal displays contact list
- Search input to filter contacts
- "Create New Group" button visible
- Clicking outside or X closes modal

---

#### TC-CHAT-006: Start Direct Chat
**Description**: Create new 1-on-1 conversation

**Preconditions**:
- At least one contact exists

**Steps**:
1. Open new chat modal
2. Click on a contact

**Expected Results**:
- Modal closes
- New chat appears in list (or existing chat selected)
- Chat view opens for that conversation

---

### 3. Messaging Tests

#### TC-MSG-001: Send Text Message
**Description**: User sends a message in conversation

**Preconditions**:
- User is in an open chat

**Steps**:
1. Type "Hello, this is a test message" in input
2. Click send button (or press Enter)

**Expected Results**:
- Input clears
- Message appears in chat immediately
- Message shows as "sent" (single checkmark)
- Message has correct timestamp
- Chat list updates with new last message

---

#### TC-MSG-002: Message Status Indicators
**Description**: Messages show correct delivery status

**Status States**:
- **Sent** (✓): Message sent to server
- **Delivered** (✓✓ gray): Message received by recipient
- **Read** (✓✓ blue): Recipient has read the message

**Steps**:
1. Send a message
2. Observe status icon progression

**Expected Results**:
- Initially shows single checkmark
- Updates to double checkmark when delivered
- Blue checkmarks when read (if recipient has read receipts on)

---

#### TC-MSG-003: Multi-line Messages
**Description**: User can send messages with line breaks

**Steps**:
1. Type "Line 1"
2. Press Shift+Enter
3. Type "Line 2"
4. Click send

**Expected Results**:
- Message renders with line breaks preserved
- Whitespace is maintained

---

#### TC-MSG-004: Long Message Handling
**Description**: Long messages display correctly

**Steps**:
1. Type a message exceeding 500 characters
2. Send message

**Expected Results**:
- Message displays with proper word-wrap
- No horizontal scrolling
- Bubble width respects max-width constraint

---

#### TC-MSG-005: Empty Message Prevention
**Description**: Cannot send blank messages

**Steps**:
1. Leave message input empty
2. Click send button

**Expected Results**:
- No message is sent
- No API call made
- Input remains focused

---

#### TC-MSG-006: Whitespace-only Prevention
**Description**: Cannot send whitespace-only messages

**Steps**:
1. Type only spaces or newlines
2. Click send

**Expected Results**:
- Message is not sent
- Input may be cleared or show validation

---

#### TC-MSG-007: Message History Loading
**Description**: Previous messages load when opening chat

**Preconditions**:
- Chat has 50+ messages

**Steps**:
1. Open the chat

**Expected Results**:
- Most recent messages load first
- Messages sorted chronologically (oldest to newest within view)
- Scroll position at bottom (most recent visible)

---

#### TC-MSG-008: Date Separators
**Description**: Messages grouped by date

**Steps**:
1. Open chat with messages across multiple days

**Expected Results**:
- Date headers appear between day groups
- "Today" for current day messages
- "Yesterday" for previous day
- Formatted date for older messages

---

#### TC-MSG-009: Sender Identification in Groups
**Description**: Group messages show sender name

**Preconditions**:
- User is in a group chat with multiple participants

**Steps**:
1. Open group chat
2. View messages from different senders

**Expected Results**:
- Other users' messages show sender name in accent color
- Own messages don't show name (right-aligned bubbles)
- Avatar or initial shown for each sender (optional)

---

#### TC-MSG-010: Auto-scroll on New Message
**Description**: Chat scrolls to show new messages

**Steps**:
1. Open chat at bottom
2. Receive new message (or send one)

**Expected Results**:
- View auto-scrolls to show new message
- Smooth scroll animation (optional)

---

### 4. Real-time & WebSocket Tests

#### TC-RT-001: WebSocket Connection
**Description**: WebSocket establishes on app load

**Steps**:
1. Login and navigate to `/app`
2. Open browser DevTools > Network > WS

**Expected Results**:
- WebSocket connection to `/ws` established
- Connection status: open
- No connection errors in console

---

#### TC-RT-002: Real-time Message Receipt
**Description**: Incoming messages appear instantly

**Setup**: Two browser sessions (Alice and Bob)

**Steps**:
1. Alice and Bob both logged in
2. Alice opens chat with Bob
3. Bob opens chat with Alice
4. Bob sends message "Hello Alice"

**Expected Results**:
- Message appears in Alice's chat immediately (< 500ms)
- No page refresh needed
- Chat list updates on Alice's side

---

#### TC-RT-003: Typing Indicator Display
**Description**: Show when other user is typing

**Setup**: Two sessions

**Steps**:
1. Alice opens chat with Bob
2. Bob starts typing in chat with Alice

**Expected Results**:
- "Bob is typing..." indicator appears for Alice
- Indicator disappears after Bob stops (3 second timeout)

---

#### TC-RT-004: Presence Updates
**Description**: Online status updates in real-time

**Steps**:
1. Alice opens chat with Bob
2. Bob logs out

**Expected Results**:
- Status changes from "Online" to "Offline" for Alice

---

#### TC-RT-005: WebSocket Reconnection
**Description**: Connection recovers after disconnect

**Steps**:
1. Simulate network disconnect
2. Reconnect network

**Expected Results**:
- Console shows "WebSocket disconnected, reconnecting..."
- Connection re-establishes within 3 seconds
- Missed messages sync after reconnection

---

#### TC-RT-006: Message Ordering with Real-time
**Description**: Messages maintain correct order

**Steps**:
1. Send rapid messages: "1", "2", "3"
2. Observe message order

**Expected Results**:
- Messages appear in sent order
- No race conditions cause reordering

---

### 5. Settings & Profile Tests

#### TC-SET-001: Profile Form Population
**Description**: Settings page shows current user data

**Steps**:
1. Navigate to `/settings`

**Expected Results**:
- Display name field populated
- Username field shows current username (readonly)
- Email field populated (if set)
- Bio field populated (if set)
- Avatar shows initial

---

#### TC-SET-002: Profile Update
**Description**: User can update display name and bio

**Steps**:
1. Go to settings
2. Change display name to "New Name"
3. Change bio to "New bio text"
4. Click "Save Changes"

**Expected Results**:
- Success message displays
- Changes persist after refresh
- Sidebar in app shows new display name

---

#### TC-SET-003: Privacy Settings
**Description**: Privacy controls save correctly

**Privacy Options**:
- Last Seen: Everyone / Contacts / Nobody
- Profile Photo: Everyone / Contacts / Nobody
- Read Receipts: Toggle on/off

**Steps**:
1. Change "Last Seen" to "Nobody"
2. Toggle off "Read Receipts"

**Expected Results**:
- Settings save to localStorage
- Settings persist after refresh

---

#### TC-SET-004: Notification Settings
**Description**: Notification preferences save

**Steps**:
1. Toggle off "Message Notifications"
2. Toggle off "Sound"
3. Refresh page

**Expected Results**:
- Settings persist
- Toggles show correct state

---

#### TC-SET-005: Theme Toggle
**Description**: Dark/Light mode switches correctly

**Steps**:
1. Toggle theme switch in settings (or header)

**Expected Results**:
- `data-theme` attribute changes on HTML element
- Colors update immediately (no flash)
- Theme preference saves to localStorage
- Persists across sessions

---

#### TC-SET-006: Password Change - Success
**Description**: User can change password

**Steps**:
1. Enter current password correctly
2. Enter new password (6+ chars)
3. Confirm new password (matching)
4. Click "Change Password"

**Expected Results**:
- Success message displays
- Form clears
- Can login with new password

---

#### TC-SET-007: Password Change - Mismatch
**Description**: Password confirmation must match

**Steps**:
1. Enter current password
2. Enter new password: "newpass1"
3. Enter confirm password: "newpass2"
4. Click "Change Password"

**Expected Results**:
- Error: "Passwords do not match"
- Form not submitted

---

#### TC-SET-008: Account Deletion
**Description**: User can delete account

**Steps**:
1. Scroll to Danger Zone
2. Click "Delete Account"
3. Confirm first dialog
4. Confirm second dialog

**Expected Results**:
- Account deleted
- User redirected to home
- Cannot login with deleted credentials

---

### 6. Group Chat Tests

#### TC-GRP-001: Group Chat Display
**Description**: Group chats show correctly in list

**Expected Results**:
- Group name displayed (not participant names)
- Member count in status: "X members"
- Avatar shows group icon or first letter

---

#### TC-GRP-002: Group Message with Sender Name
**Description**: Group messages identify sender

**Steps**:
1. Open group chat

**Expected Results**:
- Messages from others show sender name
- Name in accent color above message content
- Own messages don't show name

---

#### TC-GRP-003: Create New Group (Future)
**Description**: Create new group chat

*Note: Currently shows "Coming soon" alert*

---

### 7. Error Handling Tests

#### TC-ERR-001: Network Error on Send
**Description**: Handle message send failure

**Steps**:
1. Open chat
2. Disable network
3. Send message

**Expected Results**:
- Error indication shown
- Message not lost (pending state)
- Retry possible when online

---

#### TC-ERR-002: API Error Display
**Description**: API errors show user-friendly messages

**Test Cases**:
- Login failure: "Invalid credentials"
- Registration failure: "Username already taken"
- Session expired: Redirect to login

---

#### TC-ERR-003: WebSocket Error Recovery
**Description**: Graceful WebSocket error handling

**Steps**:
1. Force WebSocket error
2. Observe console and UI

**Expected Results**:
- Console error logged
- Automatic reconnection attempted
- No UI crash

---

### 8. Accessibility Tests

#### TC-A11Y-001: Keyboard Navigation
**Description**: All actions accessible via keyboard

**Test Points**:
- Tab through form fields
- Enter to submit forms
- Escape to close modals
- Arrow keys in lists (optional)

---

#### TC-A11Y-002: Focus Management
**Description**: Focus moves appropriately

**Scenarios**:
- Modal open: focus trapped inside
- Modal close: focus returns to trigger
- After submit: focus on first error or success

---

#### TC-A11Y-003: Screen Reader Labels
**Description**: Elements have accessible names

**Check**:
- Buttons have aria-labels or text
- Images have alt text
- Form inputs have labels
- Icons have title attributes

---

### 9. Responsive Design Tests

#### TC-RWD-001: Mobile Chat Navigation
**Description**: Sidebar/chat toggle on mobile

**Steps**:
1. View app at 375px width
2. Select a chat

**Expected Results**:
- Sidebar hides or slides away
- Chat view fills screen
- Back button visible to return to list

---

#### TC-RWD-002: Message Input Resize
**Description**: Input grows with content

**Steps**:
1. Type multiple lines in input

**Expected Results**:
- Textarea expands (up to max-height)
- Scrollbar appears when exceeding max

---

---

## Test Data Requirements

### Seed Users
```
Username: alice
Display Name: Alice Smith
Password: password123

Username: bob
Display Name: Bob Jones
Password: password123

Username: charlie
Display Name: Charlie Brown
Password: password123
```

### Seed Chats
- Direct chat: Alice <-> Bob (5 messages)
- Group chat: "Friends Group" with Alice, Bob, Charlie (4 messages)

---

## E2E Test Framework

### Technology Stack
- **Framework**: Playwright
- **Language**: TypeScript
- **Browsers**: Chromium, Firefox, WebKit
- **Parallelization**: 3 workers

### Directory Structure
```
blueprints/messaging/e2e/
├── playwright.config.ts
├── package.json
├── tests/
│   ├── auth.spec.ts
│   ├── chat.spec.ts
│   ├── messaging.spec.ts
│   ├── realtime.spec.ts
│   ├── settings.spec.ts
│   └── accessibility.spec.ts
├── fixtures/
│   └── test-fixtures.ts
├── pages/
│   ├── login.page.ts
│   ├── register.page.ts
│   ├── app.page.ts
│   └── settings.page.ts
└── helpers/
    ├── api.ts
    └── websocket.ts
```

---

## Execution Plan

### Priority 1 - Critical Path
1. TC-AUTH-001 to TC-AUTH-009 (Authentication)
2. TC-CHAT-003 (Chat Selection)
3. TC-MSG-001 (Send Message)
4. TC-RT-002 (Real-time Message)

### Priority 2 - Core Features
1. TC-CHAT-001 to TC-CHAT-006 (Chat List)
2. TC-MSG-002 to TC-MSG-010 (Messaging)
3. TC-RT-003 to TC-RT-006 (Real-time)

### Priority 3 - Settings & Edge Cases
1. TC-SET-001 to TC-SET-008 (Settings)
2. TC-ERR-001 to TC-ERR-003 (Error Handling)
3. TC-GRP-001 to TC-GRP-003 (Groups)

### Priority 4 - Quality
1. TC-A11Y-001 to TC-A11Y-003 (Accessibility)
2. TC-RWD-001 to TC-RWD-002 (Responsive)

---

## Success Criteria

- All Priority 1 tests pass: **100%**
- All Priority 2 tests pass: **95%+**
- Overall test pass rate: **90%+**
- No critical/blocking bugs
- All flows complete within reasonable time (< 5s per action)
