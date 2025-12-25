# Messaging Blueprint UX Improvements

## Overview

This document outlines UX issues identified in the messaging blueprint and their fixes.

## Critical Issues

### 1. Self-Chat and Mizu Agent Not Appearing for Existing Users

**Problem**: The `setupNewUser()` function is only called during user registration. Users who registered before this feature was added don't get the "Saved Messages" (self-chat) or "Mizu Agent" chat automatically created.

**Impact**: Existing users see an empty chat list and clicking "Saved Messages" button does nothing.

**Fix**:
- Add new API endpoint `POST /api/v1/users/ensure-chats` that creates missing default chats
- Call this endpoint when loading chats on the frontend
- The endpoint checks if Saved Messages and Agent chat exist, creates them if not

### 2. Chat `other_user` Field Not Populated

**Problem**: The frontend relies on `chat.other_user` to detect self-chats and display the other participant's info in direct chats. However, the Go `Chat` struct doesn't have an `OtherUser` field and the API doesn't populate it.

**Impact**:
- Self-chat detection fails (`chat.other_user?.id === currentUser?.id` is always false)
- Chat names show as undefined/empty for direct chats
- Clicking "Saved Messages" creates a new chat but can't find it in the list

**Fix**:
- Add `OtherUser *accounts.User` field to the `chats.Chat` struct
- Modify `chats.List()` and `chats.GetByIDForUser()` to populate `OtherUser` for direct chats
- Modify chat creation to return the chat with `OtherUser` populated

### 3. Saved Messages Click Does Nothing

**Problem**: When clicking "Saved Messages" button, the `openSavedMessages()` function:
1. Searches for existing self-chat using `chat.other_user?.id === currentUser?.id`
2. Since `other_user` is never populated, it always returns undefined
3. Then creates a new self-chat via API
4. The API returns the chat without `other_user` populated
5. Reloads chats but still can't find the self-chat due to issue #2

**Fix**: Addressed by fixing issue #2 (populating `other_user` field)

### 4. New Chat Modal Has No Contact List

**Problem**: The "New Chat" modal shows "Type to search for users" but provides no initial list of contacts or recent users. Users don't know who to search for.

**Impact**: Poor discoverability - users must know exact usernames to start chats

**Fix**:
- Show recent chat contacts when modal opens
- Display "Suggested contacts" section before search results
- Include Mizu Agent in suggestions for quick access

## Moderate Issues

### 5. MarkAsRead Requires Message ID in Body

**Problem**: The `MarkAsRead` endpoint requires a `message_id` in the request body, but the frontend sends an empty POST. This causes JSON parsing errors.

**Fix**: Make `message_id` optional in the handler, allowing empty bodies

### 6. No Loading States

**Problem**: No visual feedback when:
- Loading chats
- Loading messages
- Searching users
- Sending messages

**Fix**: Add loading spinners/skeletons for:
- Initial chat list load
- Message list load
- User search
- Message send button

### 7. No Error Handling UI

**Problem**: When API calls fail, errors are only logged to console. Users see no feedback.

**Fix**: Add toast/notification system for error messages

### 8. Group Creation Not Implemented

**Problem**: "New Group" button shows `alert('Group creation coming soon!')`

**Status**: Documented as future work, not blocking

### 9. Non-functional UI Elements

**Problem**: Several UI elements have no functionality:
- Voice call button
- Video call button
- Attach file button
- More options (3-dot) menu

**Status**: Documented as future work, not blocking for MVP

### 10. No Chat Actions Menu

**Problem**: No way to:
- Archive chats
- Pin chats
- Mute chats
- Delete chats

**Fix**: Add dropdown menu to chat list items with these actions

### 11. No Message Actions

**Problem**: No way to:
- Edit messages
- Delete messages
- React to messages
- Reply to messages
- Forward messages

**Status**: API supports these, UI implementation is future work

## Minor Issues

### 12. Online Status Always Shows "Online"

**Problem**: Direct chats always show "Online" status regardless of actual presence

**Fix**: Connect to WebSocket presence events and update status accordingly

### 13. WebSocket Reconnection Doesn't Reload Chats

**Problem**: When WebSocket reconnects, it doesn't fetch new messages that arrived during disconnect

**Fix**: Reload current chat's messages on WebSocket reconnect

### 14. No Notification Sounds

**Problem**: No audio feedback for new messages

**Status**: Future enhancement

### 15. Mobile Responsiveness Issues

**Problem**:
- Back button shows only on mobile but sidebar doesn't hide
- Chat view doesn't properly fill screen on mobile

**Fix**: Improve responsive layout with proper mobile sidebar handling

### 16. Chat List Doesn't Update Order on New Message

**Problem**: When receiving a new message, the chat moves to top but this can be jarring

**Status**: Current behavior is acceptable, just document it

## Implementation Summary

### API Changes

1. **New endpoint**: `POST /api/v1/users/ensure-chats`
   - Creates Saved Messages and Agent chat if they don't exist
   - Returns list of created chats

2. **Modified**: `GET /api/v1/chats`
   - Populates `other_user` field for direct chats

3. **Modified**: `POST /api/v1/chats`
   - Returns created chat with `other_user` populated

4. **Modified**: `POST /api/v1/chats/{id}/read`
   - Make `message_id` optional

### Frontend Changes

1. Call `ensure-chats` endpoint before loading chats
2. Add loading states for async operations
3. Add error toast notifications
4. Show recent contacts in New Chat modal
5. Improve mobile responsiveness

### Data Model Changes

1. Add `OtherUser` field to `chats.Chat` struct
2. Add `last_message` population in chat list

## Testing

- Verify existing users get Saved Messages and Agent chat
- Verify new users still get default chats
- Verify clicking Saved Messages opens correct chat
- Verify chat names display correctly
- Verify new chat creation works with proper UI updates
