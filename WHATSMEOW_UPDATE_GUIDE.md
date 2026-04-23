# Panduan Update Library whatsmeow

Dokumen ini berisi konteks penting yang perlu dipelajari setiap kali library `go.mau.fi/whatsmeow` diupdate.

## Versi Saat Ini

```
go.mau.fi/whatsmeow v0.0.0-20260327181659-02ec817e7cf4
```

Cek versi terbaru: https://pkg.go.dev/go.mau.fi/whatsmeow

---

## File-File Kritis yang Perlu Dicek

| File | Ketergantungan whatsmeow |
|------|--------------------------|
| `src/infrastructure/whatsapp/init.go` | `whatsmeow.Client`, `AddEventHandler`, `NewClient`, `SendPresence`, `MarkRead` |
| `src/infrastructure/whatsapp/event_message.go` | `events.Message`, `types.ParseJID`, `cli.Store.LIDs.GetPNForLID` |
| `src/infrastructure/whatsapp/event_receipt.go` | `events.Receipt`, `types.ReceiptType*` |
| `src/infrastructure/whatsapp/event_group.go` | `events.GroupInfo`, `types.JID` |
| `src/infrastructure/whatsapp/event_delete.go` | `events.DeleteForMe` |
| `src/infrastructure/whatsapp/webhook.go` | `http` (tidak langsung tergantung whatsmeow) |
| `src/pkg/utils/whatsapp.go` | `whatsmeow.DownloadableMessage`, `client.Download`, `waE2E.*` |
| `src/usecase/account/account.go` | `whatsmeow.Client`, `events.Message`, `types.ParseJID`, `Store.LIDs` |
| `src/usecase/account/account.go` (LoginAccount) | `client.GetQRChannel`, `client.Connect`, `client.IsLoggedIn` |

---

## Area yang Paling Sering Berubah

### 1. Sistem LID (Linked Device ID)

WhatsApp menggunakan LID sebagai identitas internal, berbeda dari nomor HP.

**Fungsi yang dipakai:**
```go
// Resolve LID → nomor HP
pn, err := client.Store.LIDs.GetPNForLID(ctx, lid)

// Resolve nomor HP → LID
lid, err := client.Store.LIDs.GetLIDForPN(ctx, pn)
```

**File terkait:**
- `src/infrastructure/whatsapp/event_message.go` — resolusi LID di payload webhook global
- `src/usecase/account/account.go` (`buildWebhookPayload`) — resolusi LID di payload webhook per-account

**Apa yang perlu dicek saat update:**
- Apakah `store.LIDStore` interface berubah (tambah/hapus method)
- Apakah `types.HiddenUserServer` masih `"lid"`
- Apakah `CachedLIDMap` di `store/sqlstore/lidmap.go` masih implement interface yang sama

---

### 2. Event Types

Semua event yang di-handle ada di `handler()` dalam `init.go`:

```go
case *events.DeleteForMe
case *events.AppStateSyncComplete
case *events.PairSuccess
case *events.LoggedOut
case *events.Connected
case *events.PushNameSetting
case *events.StreamReplaced
case *events.Message       // ← paling penting untuk webhook
case *events.Receipt       // ← untuk ack/read webhook
case *events.Presence
case *events.HistorySync
case *events.AppState
case *events.GroupInfo     // ← untuk group webhook
```

**Apa yang perlu dicek saat update:**
- Apakah ada event type baru yang relevan (misal: `events.FBMessage` untuk Facebook-linked accounts)
- Apakah field di `events.Message` berubah (terutama `Info`, `Message`, `IsViewOnce`)
- Apakah `events.Receipt` masih embed `types.MessageSource` (untuk akses `.Chat`, `.Sender`)
- Apakah `MessageSource.SourceString()` masih return format `"sender in chat"` atau `"chat"`

---

### 3. Download Media

**Fungsi yang dipakai:**
```go
// Di src/pkg/utils/whatsapp.go
data, err := client.Download(ctx, mediaFile)
```

**Interface `DownloadableMessage`** — implementasi yang di-support:
- `*waE2E.ImageMessage`
- `*waE2E.AudioMessage`
- `*waE2E.VideoMessage`
- `*waE2E.DocumentMessage`
- `*waE2E.StickerMessage`

**Apa yang perlu dicek saat update:**
- Apakah signature `client.Download(ctx, msg)` berubah
- Apakah ada tipe media baru yang perlu ditambahkan ke switch di `ExtractMedia()`
- `DownloadAny` sudah deprecated — jangan dipakai

---

### 4. Struktur `events.Message`

Field yang dipakai dalam kode:

```go
evt.Info.ID              // message ID
evt.Info.Sender          // JID pengirim
evt.Info.Chat            // JID chat
evt.Info.PushName        // nama pengirim
evt.Info.Timestamp       // waktu pesan
evt.Info.IsFromMe        // apakah dari diri sendiri
evt.Info.IsGroup         // apakah grup
evt.Info.SourceString()  // "sender in chat" atau "chat"

evt.Message.GetConversation()        // teks biasa
evt.Message.GetExtendedTextMessage() // teks dengan konteks (reply, dll)
evt.Message.GetProtocolMessage()     // revoke / edit / ephemeral
evt.Message.GetReactionMessage()     // emoji reaction
evt.Message.GetImageMessage()
evt.Message.GetAudioMessage()
evt.Message.GetVideoMessage()
evt.Message.GetDocumentMessage()
evt.Message.GetStickerMessage()

evt.IsViewOnce       // view once message
evt.IsEphemeral      // ephemeral/disappearing message
```

---

### 5. Client Initialization

**Fungsi yang dipakai saat init:**
```go
// Database
storeContainer, err := sqlstore.New(ctx, "sqlite3", dbURI, dbLog)
device, err := storeContainer.GetFirstDevice(ctx)

// Client
cli = whatsmeow.NewClient(device, log)
cli.EnableAutoReconnect = true
cli.AutoTrustIdentity = true
cli.AddEventHandler(handlerFunc)

// Login via QR
qrChan, err := client.GetQRChannel(ctx)
err = client.Connect()
```

**Apa yang perlu dicek saat update:**
- Apakah `sqlstore.New` signature berubah
- Apakah field `EnableAutoReconnect` / `AutoTrustIdentity` masih ada
- Apakah `GetQRChannel` masih harus dipanggil **sebelum** `Connect()`
- Apakah `store.DeviceProps` masih bisa dikonfigurasi

---

### 6. ReceiptType Constants

Dipakai di `event_receipt.go` untuk filter webhook:

```go
types.ReceiptTypeRead       // "read"
types.ReceiptTypeReadSelf   // "read-self"
types.ReceiptTypeDelivered  // "" (string kosong)
types.ReceiptTypeSender     // "sender"
types.ReceiptTypePlayed     // "played"
types.ReceiptTypePlayedSelf // "played-self"
```

**Perhatian:** `ReceiptTypeDelivered = ""` (string kosong) — ini tidak intuitif, jangan ganti dengan string `"delivered"`.

---

## Cara Cek Breaking Changes Setelah Update

### Langkah 1 — Build
```bash
cd src && go build ./...
```
Kalau ada error compile, ini paling mudah diidentifikasi.

### Langkah 2 — Check Interface LID
```bash
grep -rn "\.LIDs\." src/
```
Pastikan semua method yang dipanggil masih ada di interface `store.LIDStore`.

### Langkah 3 — Check Event Types
```bash
grep -rn "events\." src/infrastructure/whatsapp/init.go
```
Bandingkan dengan `types/events/events.go` di versi baru.

### Langkah 4 — Test Fungsional
Setelah rebuild Docker:
1. Login akun via QR
2. Kirim pesan text dari **mobile** → cek webhook diterima
3. Kirim pesan text dari **desktop** → cek webhook diterima, format `from` konsisten
4. Kirim pesan media (gambar) → cek download dan webhook
5. Cek read receipt webhook
6. Test revoke pesan

---

## Riwayat Update dan Perubahan

| Tanggal | Versi whatsmeow | Perubahan / Catatan |
|---------|-----------------|---------------------|
| 2026-04-08 | `v0.0.0-20260327181659` | Update library; per-account `forwardToWebhook` diimplementasikan (sebelumnya stub); fix LID resolution untuk sender dari WhatsApp Desktop (device suffix `:47` di-strip agar konsisten dengan Mobile) |
| ~2025-11 | `v0.0.0-20251116104239` | Versi sebelumnya |

---

## Referensi

- Source library: https://pkg.go.dev/go.mau.fi/whatsmeow
- Event types: `go.mau.fi/whatsmeow/types/events`
- Store interfaces: `go.mau.fi/whatsmeow/store`
- LID map impl: `go.mau.fi/whatsmeow/store/sqlstore/lidmap.go`
