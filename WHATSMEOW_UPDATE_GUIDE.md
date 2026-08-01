# Panduan Update Library whatsmeow

Dokumen ini berisi konteks penting yang perlu dipelajari setiap kali library `go.mau.fi/whatsmeow` diupdate.

## Versi Saat Ini

```
go.mau.fi/whatsmeow v0.0.0-20260730092514-662ad1dc6900
```

Cek versi terbaru: https://pkg.go.dev/go.mau.fi/whatsmeow

---

## Penyebab #1 Aplikasi Tiba-Tiba Error: Versi Client Kedaluwarsa

Nomor versi WhatsApp Web **hardcoded di dalam library**, di `store/clientpayload.go`:

```go
var waVersion = WAVersionContainer{2, 3000, 1044142122}
```

Kalau WhatsApp menaikkan versi minimum di sisi server, semua client dengan versi lama
langsung ditolak. Gejalanya:

- Akun yang sudah login → `events.ConnectFailure` dengan `Reason = 405` (`ConnectFailureClientOutdated`)
- Saat scan QR → `events.ClientOutdated`, QR channel mengirim event `err-client-outdated`

**Perbaikannya cuma satu: update library-nya.** Tidak ada workaround di sisi aplikasi,
karena versi tidak bisa di-override dari luar (project ini tidak memanggil `SetWAVersion`).

```bash
cd src && go get go.mau.fi/whatsmeow@latest && go mod tidy && go build ./...
```

Sejak update 2026-08-01 event-event ini sudah di-log eksplisit (lihat `handleConnectFailure`
di `src/infrastructure/whatsapp/init.go`), jadi kalau terjadi lagi langsung kelihatan di log.

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

Event diagnostik yang juga di-handle (ditambahkan 2026-08-01):

```go
case *events.ClientOutdated  // versi client ditolak WhatsApp
case *events.ConnectFailure  // koneksi ditolak, ada field Reason + Message
case *events.TemporaryBan    // akun kena ban sementara
case *events.CATRefreshError // gagal refresh client auth token
```

Sebelumnya event-event ini tidak di-handle sama sekali, jadi saat WhatsApp menolak koneksi
aplikasi hanya terlihat "diam" sementara auto-reconnect terus mencoba ulang.

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

### 6. Migrasi Skema Database

`sqlstore.New()` menjalankan `container.Upgrade()` otomatis, jadi migrasi berlaku begitu
aplikasi start — termasuk untuk `DB_KEYS_URI` dan tiap database per-account di
`storages/accounts/<nama>/`.

Cek versi skema sebuah database SQLite:

```bash
sqlite3 storages/whatsapp.db "SELECT version FROM whatsmeow_version;"
```

Migrasi butuh foreign key aktif di SQLite — pastikan URI-nya pakai `?_foreign_keys=on`
(sudah jadi default di `config/settings.go`).

**Perhatian:** migrasi jalan satu arah. Kalau perlu rollback ke versi library lama,
database harus di-restore dari backup — jadi backup folder `storages/` dulu sebelum update.

---

### 7. Passkey Pairing (belum diimplementasikan)

Versi `20260730` menambahkan alur pairing berbasis passkey/WebAuthn:

```go
events.PairPasskeyRequest / PairPasskeyError / PairPasskeyConfirmation
client.SendPasskeyResponse(ctx, resp)
client.SendPasskeyConfirmation(ctx)
```

QR channel bisa mengirim event `passkey-request` dan `passkey-confirmation`. Aplikasi ini
**belum** mengimplementasikannya (butuh authenticator WebAuthn), tapi sudah melaporkan
event tersebut dengan pesan yang jelas alih-alih diam. Kalau suatu saat WhatsApp mewajibkan
passkey untuk linking device, alur ini harus dibangun.

---

### 8. ReceiptType Constants

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

### Langkah 4 — Diff API Antar Versi

Build yang sukses tidak menjamin aman, karena perubahan bisa bersifat runtime. Bandingkan
langsung dua versi di module cache:

```bash
OLD=$(go env GOMODCACHE)/go.mau.fi/whatsmeow@<versi-lama>
NEW=$(go env GOMODCACHE)/go.mau.fi/whatsmeow@<versi-baru>

# Versi client WA (paling penting)
diff "$OLD/store/clientpayload.go" "$NEW/store/clientpayload.go"

# Semua method Client — cek ada yang hilang/berubah signature
sig(){ grep -rh "^func (cli \*Client)" "$1"/*.go | sed 's/ {$//' | sort; }
diff <(sig "$OLD") <(sig "$NEW")

# Event types
diff "$OLD/types/events/events.go" "$NEW/types/events/events.go"

# Migrasi database baru
diff -rq "$OLD/store/sqlstore/upgrades" "$NEW/store/sqlstore/upgrades"
```

### Langkah 5 — Smoke Test Tanpa Menyentuh Sesi Produksi

Jalankan dengan database kosong sementara, lalu minta QR. Kalau QR terbit, berarti
WhatsApp menerima versi client-nya:

```bash
cd src && go build -o /tmp/wa-test . && cd /tmp
./wa-test rest --port=3899 --db-uri="file:/tmp/wa-test.db?_foreign_keys=on" &
curl -s http://localhost:3899/app/login
```

Jangan pakai database produksi untuk tes ini — dua koneksi dengan kredensial device yang
sama akan memicu `StreamReplaced` dan memutus instance yang sedang jalan.

### Langkah 6 — Test Fungsional
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
| 2026-08-01 | `v0.0.0-20260730092514` | Versi client WA naik `2.3000.1035920091` → `2.3000.1044142122` (penyebab error koneksi). Migrasi DB v13→v14 (tabel `whatsmeow_nct_salt`, jalan otomatis). Tidak ada breaking change di level compile — semua perubahan API bersifat additive. Ditambahkan handler `ClientOutdated`/`ConnectFailure`/`TemporaryBan`/`CATRefreshError`, dan fix hang di `/app/login` saat QR channel tutup tanpa emit code |
| 2026-04-08 | `v0.0.0-20260327181659` | Update library; per-account `forwardToWebhook` diimplementasikan (sebelumnya stub); fix LID resolution untuk sender dari WhatsApp Desktop (device suffix `:47` di-strip agar konsisten dengan Mobile) |
| ~2025-11 | `v0.0.0-20251116104239` | Versi sebelumnya |

---

## Referensi

- Source library: https://pkg.go.dev/go.mau.fi/whatsmeow
- Event types: `go.mau.fi/whatsmeow/types/events`
- Store interfaces: `go.mau.fi/whatsmeow/store`
- LID map impl: `go.mau.fi/whatsmeow/store/sqlstore/lidmap.go`
