# Multi-Account Implementation Progress

## ✅ Completed

### 1. Response Format
- ✅ Created `/src/pkg/utils/response.go` dengan format response seperti wago lama
```json
{
  "code": "SUCCESS",
  "message": "...",
  "results": {...}
}
```

### 2. Account Management Infrastructure
- ✅ `/src/domains/account/interfaces.go` - Account domain interfaces & models
- ✅ `/src/infrastructure/account/manager.go` - AccountManager untuk manage multiple clients
- ✅ `/src/infrastructure/account/repository.go` - SQLite repository untuk account data
- ✅ `/src/usecase/account/account.go` - Account business logic
- ✅ `/src/ui/rest/account.go` - REST API handlers untuk account management

### 3. Account Management Endpoints
Semua endpoint account sudah tersedia dan terintegrasi:
- ✅ `POST /accounts` - Create account
- ✅ `GET /accounts` - List all accounts
- ✅ `GET /accounts/:id` - Get account details
- ✅ `DELETE /accounts/:id` - Delete account
- ✅ `POST /accounts/:id/login` - Login dengan QR code
- ✅ `POST /accounts/:id/login-with-code` - Login dengan pairing code
- ✅ `POST /accounts/:id/logout` - Logout
- ✅ `POST /accounts/:id/reconnect` - Reconnect
- ✅ `POST /accounts/:id/webhook` - Set webhook per account
- ✅ `GET /accounts/:id/webhook` - Get webhook config

### 4. Send Usecase - Multi-Account Support
- ✅ Updated `serviceSend` struct dengan `accountManager`
- ✅ Added `getClient(accountID)` helper method
- ✅ Updated all Send* methods:
  - ✅ SendText
  - ✅ SendImage
  - ✅ SendFile
  - ✅ SendVideo
  - ✅ SendAudio
  - ✅ SendContact
  - ✅ SendLink
  - ✅ SendLocation
  - ✅ SendPoll
  - ✅ SendSticker
  - ✅ SendPresence
  - ✅ SendChatPresence
- ✅ Added `account_id` field to `BaseRequest` and `PresenceRequest`
- ✅ Updated `wrapSendMessage` dan `uploadMedia` untuk accept client parameter
- ✅ Updated `getMentionFromText` untuk accept client parameter

### 5. Integration
- ✅ Updated `/src/cmd/root.go` untuk initialize account infrastructure
- ✅ Updated `/src/cmd/rest.go` untuk register account endpoints
- ✅ Build berhasil tanpa error

## ⏳ To Do

### 1. Chat Usecase & Endpoints
Update untuk multi-account support:
- [ ] `/src/usecase/chat.go`
- [ ] `/src/ui/rest/chat.go`
- [ ] `GET /chat/list?account_id=...`
- [ ] `GET /chat/messages?account_id=...`

### 2. Group Usecase & Endpoints
Update untuk multi-account support:
- [ ] `/src/usecase/group.go`
- [ ] `/src/ui/rest/group.go`
- [ ] `GET /group/list?account_id=...`
- [ ] `POST /group/create` dengan `account_id` in body
- [ ] `GET /group/info?account_id=...`

### 3. User Usecase & Endpoints
Update untuk multi-account support:
- [ ] `/src/usecase/user.go`
- [ ] `/src/ui/rest/user.go`
- [ ] User info endpoints

### 4. Message Usecase & Endpoints
Update untuk multi-account support:
- [ ] `/src/usecase/message.go`
- [ ] `/src/ui/rest/message.go`

### 5. Testing
- [ ] Test account creation
- [ ] Test login with QR
- [ ] Test login with pairing code
- [ ] Test sending messages from multiple accounts
- [ ] Test webhooks per account
- [ ] Test account deletion

### 6. Docker & Deployment
- [ ] Update docker-compose.yml untuk volumes
- [ ] Test deployment dengan Docker
- [ ] Documentation untuk deployment

## Architecture

```
Multi-Account Architecture:
├── Account Manager (singleton)
│   ├── Account 1 → WhatsApp Client 1 → DB 1
│   ├── Account 2 → WhatsApp Client 2 → DB 2
│   └── Account N → WhatsApp Client N → DB N
│
├── Account Repository (SQLite)
│   └── storages/accounts.db
│
└── Per-Account Data
    └── storages/accounts/{account_id}/
        ├── whatsapp.db (WhatsApp session data)
        └── keys.db (encryption keys)
```

## API Changes

### Before (Single Account)
```bash
POST /send/message
{
  "phone": "6281234567890",
  "message": "Hello"
}
```

### After (Multi-Account)
```bash
POST /send/message
{
  "account_id": "account1",  # NEW FIELD
  "phone": "6281234567890",
  "message": "Hello"
}
```

### Backward Compatibility
Jika `account_id` tidak disediakan, system akan menggunakan global WhatsApp client (untuk backward compatibility dengan single-account mode).

## Response Format

### Success Response
```json
{
  "code": "SUCCESS",
  "message": "Account created successfully",
  "results": {
    "account_id": "account1",
    "message": "Account created successfully"
  }
}
```

### Error Response
```json
{
  "code": "ERROR",
  "message": "Account not found or not logged in: account1"
}
```

## Testing Guide

### 1. Create Account
```bash
curl -X POST http://localhost:3000/accounts \
  -H "Content-Type: application/json" \
  -d '{"account_id": "account1"}'
```

### 2. Login with QR
```bash
curl -X POST http://localhost:3000/accounts/account1/login
# Response will contain QR code image path
```

### 3. Send Message
```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "account1",
    "phone": "6281234567890",
    "message": "Hello from multi-account!"
  }'
```

### 4. List Accounts
```bash
curl http://localhost:3000/accounts
```

## Notes

- Setiap account memiliki database WhatsApp terpisah di `storages/accounts/{account_id}/`
- Setiap account bisa memiliki webhook configuration sendiri
- Multiple accounts bisa login dan kirim pesan secara concurrent
- Auto-reconnect works per account
- Event handling works per account (messages, receipts, group events, etc.)
