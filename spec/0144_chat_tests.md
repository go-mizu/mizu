# 0144 Chat Blueprint Tests

## Overview

This document outlines the comprehensive testing strategy for the Chat Blueprint application, covering:
1. Store layer tests (DuckDB stores)
2. CLI command tests (using fang for better UI)
3. End-to-end API tests

## 1. Store/DuckDB Tests

All stores will be tested with in-memory DuckDB databases for fast, isolated tests.

### Test Helper (`store/duckdb/testutil_test.go`)

```go
// setupTestDB creates an in-memory DuckDB for testing
func setupTestDB(t *testing.T) *sql.DB
// teardownTestDB closes the test database
func teardownTestDB(t *testing.T, db *sql.DB)
```

### 1.1 store_test.go
- `TestNew` - Store creation
- `TestEnsure` - Schema initialization
- `TestDB` - DB accessor
- `TestOpen` - Database opening

### 1.2 users_store_test.go
- `TestUsersStore_Insert` - Create user
- `TestUsersStore_GetByID` - Retrieve by ID
- `TestUsersStore_GetByIDs` - Batch retrieval
- `TestUsersStore_GetByUsername` - Retrieve by username
- `TestUsersStore_GetByEmail` - Retrieve by email
- `TestUsersStore_Update` - Update user fields
- `TestUsersStore_ExistsUsername` - Username existence check
- `TestUsersStore_ExistsEmail` - Email existence check
- `TestUsersStore_GetPasswordHash` - Password hash retrieval
- `TestUsersStore_Search` - User search
- `TestUsersStore_GetNextDiscriminator` - Discriminator generation
- `TestUsersStore_CreateSession` - Session creation
- `TestUsersStore_GetSession` - Session retrieval
- `TestUsersStore_DeleteSession` - Session deletion
- `TestUsersStore_DeleteExpiredSessions` - Expired session cleanup
- `TestUsersStore_UpdateStatus` - Status update

### 1.3 servers_store_test.go
- `TestServersStore_Insert` - Create server
- `TestServersStore_GetByID` - Retrieve by ID
- `TestServersStore_GetByInviteCode` - Retrieve by invite code
- `TestServersStore_Update` - Update server
- `TestServersStore_Delete` - Delete server
- `TestServersStore_ListByUser` - List user's servers
- `TestServersStore_ListPublic` - List public servers
- `TestServersStore_IncrementMemberCount` - Member count increment
- `TestServersStore_SetDefaultChannel` - Set default channel

### 1.4 channels_store_test.go
- `TestChannelsStore_Insert` - Create channel
- `TestChannelsStore_GetByID` - Retrieve by ID
- `TestChannelsStore_Update` - Update channel
- `TestChannelsStore_Delete` - Delete channel
- `TestChannelsStore_ListByServer` - List server channels
- `TestChannelsStore_ListDMsByUser` - List user DMs
- `TestChannelsStore_GetDMChannel` - Get DM channel
- `TestChannelsStore_AddRecipient` - Add DM recipient
- `TestChannelsStore_RemoveRecipient` - Remove DM recipient
- `TestChannelsStore_GetRecipients` - Get channel recipients
- `TestChannelsStore_UpdateLastMessage` - Update last message
- `TestChannelsStore_InsertCategory` - Create category
- `TestChannelsStore_GetCategory` - Retrieve category
- `TestChannelsStore_ListCategories` - List categories
- `TestChannelsStore_DeleteCategory` - Delete category

### 1.5 messages_store_test.go
- `TestMessagesStore_Insert` - Create message
- `TestMessagesStore_GetByID` - Retrieve by ID
- `TestMessagesStore_Update` - Update message
- `TestMessagesStore_Delete` - Delete message
- `TestMessagesStore_List` - List with pagination (before/after/around)
- `TestMessagesStore_Search` - Message search
- `TestMessagesStore_Pin` - Pin message
- `TestMessagesStore_Unpin` - Unpin message
- `TestMessagesStore_ListPinned` - List pinned messages
- `TestMessagesStore_AddReaction` - Add reaction
- `TestMessagesStore_RemoveReaction` - Remove reaction
- `TestMessagesStore_GetReactionUsers` - Get reaction users
- `TestMessagesStore_InsertAttachment` - Add attachment
- `TestMessagesStore_InsertEmbed` - Add embed

### 1.6 members_store_test.go
- `TestMembersStore_Insert` - Add member
- `TestMembersStore_Get` - Get member
- `TestMembersStore_Update` - Update member
- `TestMembersStore_Remove` - Remove member
- `TestMembersStore_ListByServer` - List server members
- `TestMembersStore_ListByUser` - List user's memberships

### 1.7 roles_store_test.go
- `TestRolesStore_Insert` - Create role
- `TestRolesStore_GetByID` - Retrieve by ID
- `TestRolesStore_Update` - Update role
- `TestRolesStore_Delete` - Delete role
- `TestRolesStore_ListByServer` - List server roles
- `TestRolesStore_AddMemberRole` - Assign role to member
- `TestRolesStore_RemoveMemberRole` - Remove role from member
- `TestRolesStore_GetMemberRoles` - Get member's roles
- `TestRolesStore_RemoveMemberRoles` - Remove all member roles

### 1.8 presence_store_test.go
- `TestPresenceStore_Upsert` - Create/update presence
- `TestPresenceStore_Get` - Get presence
- `TestPresenceStore_GetMultiple` - Batch presence retrieval
- `TestPresenceStore_UpdateStatus` - Status update
- `TestPresenceStore_Delete` - Delete presence

## 2. CLI Command Tests

Using fang for enhanced CLI testing with better UI output verification.

### 2.1 root_test.go
- `TestRootCmd_Execute` - Root command execution
- `TestRootCmd_Version` - Version flag
- `TestRootCmd_Help` - Help output
- `TestRootCmd_GlobalFlags` - Global flag parsing (--data, --addr, --dev)

### 2.2 init_test.go
- `TestInitCmd_Execute` - Init command execution
- `TestInitCmd_CreatesDataDir` - Data directory creation
- `TestInitCmd_CreatesDatabase` - Database initialization
- `TestInitCmd_Idempotent` - Multiple executions are safe

### 2.3 seed_test.go
- `TestSeedCmd_Execute` - Seed command execution
- `TestSeedCmd_CreatesUsers` - Sample user creation
- `TestSeedCmd_CreatesServers` - Sample server creation
- `TestSeedCmd_CreatesChannels` - Sample channel creation
- `TestSeedCmd_CreatesMessages` - Sample message creation
- `TestSeedCmd_RequiresInit` - Requires init first

### 2.4 serve_test.go
- `TestServeCmd_Execute` - Serve command execution
- `TestServeCmd_CustomAddr` - Custom address binding
- `TestServeCmd_DevMode` - Development mode
- `TestServeCmd_GracefulShutdown` - Signal handling

## 3. End-to-End API Tests

Located in `app/web/server_e2e_test.go`.

### Test Setup

```go
// TestServer creates a test server with in-memory database
func setupTestServer(t *testing.T) (*Server, func())
```

### 3.1 Auth Endpoints
- `TestE2E_Auth_Register` - User registration flow
- `TestE2E_Auth_Register_Validation` - Registration validation errors
- `TestE2E_Auth_Login` - Login flow
- `TestE2E_Auth_Login_InvalidCredentials` - Login failure cases
- `TestE2E_Auth_Logout` - Logout flow
- `TestE2E_Auth_Me` - Get current user
- `TestE2E_Auth_UpdateMe` - Profile update

### 3.2 Server Endpoints
- `TestE2E_Servers_Create` - Server creation
- `TestE2E_Servers_Get` - Server retrieval
- `TestE2E_Servers_Update` - Server update
- `TestE2E_Servers_Delete` - Server deletion
- `TestE2E_Servers_List` - List user's servers
- `TestE2E_Servers_ListPublic` - List public servers
- `TestE2E_Servers_Join` - Join server
- `TestE2E_Servers_Leave` - Leave server
- `TestE2E_Servers_JoinByInvite` - Join by invite code

### 3.3 Channel Endpoints
- `TestE2E_Channels_Create` - Channel creation
- `TestE2E_Channels_Get` - Channel retrieval
- `TestE2E_Channels_Update` - Channel update
- `TestE2E_Channels_Delete` - Channel deletion
- `TestE2E_Channels_ListByServer` - List server channels
- `TestE2E_Channels_CreateCategory` - Category creation
- `TestE2E_Channels_ListCategories` - List categories

### 3.4 Message Endpoints
- `TestE2E_Messages_Create` - Send message
- `TestE2E_Messages_Get` - Get message
- `TestE2E_Messages_Update` - Edit message
- `TestE2E_Messages_Delete` - Delete message
- `TestE2E_Messages_List_Pagination` - Message pagination
- `TestE2E_Messages_Search` - Message search
- `TestE2E_Messages_Pin` - Pin message
- `TestE2E_Messages_Unpin` - Unpin message
- `TestE2E_Messages_ListPinned` - List pinned
- `TestE2E_Messages_AddReaction` - Add reaction
- `TestE2E_Messages_RemoveReaction` - Remove reaction
- `TestE2E_Messages_Typing` - Typing indicator

### 3.5 DM Endpoints
- `TestE2E_DMs_Create` - Create DM channel
- `TestE2E_DMs_List` - List DM channels
- `TestE2E_DMs_SendMessage` - Send DM

### 3.6 Member Endpoints
- `TestE2E_Members_List` - List server members
- `TestE2E_Roles_List` - List server roles

### 3.7 User Search
- `TestE2E_Users_Search` - Search users

### 3.8 Integration Scenarios
- `TestE2E_FullFlow_CreateServerAndChat` - Complete server creation and messaging flow
- `TestE2E_FullFlow_DMConversation` - Direct message conversation
- `TestE2E_FullFlow_MultiUserServer` - Multiple users in server

## Implementation Order

1. **Phase 1: Store Tests**
   - testutil_test.go (shared test helpers)
   - store_test.go
   - users_store_test.go
   - servers_store_test.go
   - channels_store_test.go
   - messages_store_test.go
   - members_store_test.go
   - roles_store_test.go
   - presence_store_test.go

2. **Phase 2: CLI Tests**
   - root_test.go
   - init_test.go
   - seed_test.go
   - serve_test.go

3. **Phase 3: E2E Tests**
   - server_e2e_test.go

## Test Running

```bash
# Run all tests
make test

# Run store tests only
go test ./store/duckdb/... -v

# Run CLI tests only
go test ./cli/... -v

# Run E2E tests only
go test ./app/web/... -v -run E2E

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Dependencies

- `github.com/stretchr/testify` - Assertions
- In-memory DuckDB (`:memory:`) for isolated tests
- `net/http/httptest` for E2E testing
