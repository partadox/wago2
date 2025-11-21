# Docker Deployment Guide - Multi-Account WhatsApp Gateway

## Prerequisites

- Docker dan Docker Compose terinstall
- Minimal 1GB RAM
- Minimal 2GB disk space untuk multiple accounts

## Quick Start

### 1. Clone & Setup

```bash
cd /path/to/wago2

# Copy environment example
cp src/.env.example src/.env

# Edit .env sesuai kebutuhan
nano src/.env
```

### 2. Buat Directory untuk Persistent Storage

```bash
# Buat directories yang diperlukan
mkdir -p storages/accounts
mkdir -p statics/qrcode
mkdir -p statics/senditems
mkdir -p statics/media

# Set permissions (opsional, tergantung OS)
chmod -R 755 storages statics
```

### 3. Build & Run

```bash
# Build image
docker-compose build

# Run service
docker-compose up -d

# Check logs
docker-compose logs -f
```

### 4. Verify Service Running

```bash
curl http://localhost:3000/
# atau buka di browser: http://localhost:3000
```

## Volume Mapping

Docker Compose akan membuat volume mapping berikut:

```yaml
volumes:
  - ./storages:/app/storages              # Account databases
  - ./statics/qrcode:/app/statics/qrcode  # QR codes
  - ./statics/senditems:/app/statics/senditems  # Temp files
  - ./statics/media:/app/statics/media    # Media files
```

### Struktur Storage:

```
storages/
├── accounts.db                  # Account registry database
├── accounts/                    # Per-account data
│   ├── account1/
│   │   ├── whatsapp.db         # WhatsApp session data
│   │   └── keys.db             # Encryption keys
│   ├── account2/
│   │   ├── whatsapp.db
│   │   └── keys.db
│   └── ...
└── chatstorage.db              # Chat history storage

statics/
├── qrcode/                      # QR code images untuk login
│   ├── scan-account1-*.png
│   └── scan-account2-*.png
├── senditems/                   # Temporary files untuk send operations
└── media/                       # Downloaded media files
```

## Environment Variables

Edit `src/.env` untuk konfigurasi:

```bash
# Application Settings
APP_PORT=3000
APP_DEBUG=false
APP_OS=Chrome
APP_BASIC_AUTH=user1:pass1,user2:pass2
APP_BASE_PATH=
APP_TRUSTED_PROXIES=0.0.0.0/0

# Database Settings (tidak perlu diubah untuk Docker)
DB_URI="file:storages/whatsapp.db?_foreign_keys=on"
DB_KEYS_URI="file::memory:?cache=shared&_foreign_keys=on"

# WhatsApp Settings
WHATSAPP_AUTO_REPLY=""
WHATSAPP_AUTO_MARK_READ=false
WHATSAPP_AUTO_DOWNLOAD_MEDIA=true
WHATSAPP_WEBHOOK=https://your-webhook-url.com/webhook
WHATSAPP_WEBHOOK_SECRET=your-secret-key
WHATSAPP_ACCOUNT_VALIDATION=true
```

## Docker Commands

### Basic Operations

```bash
# Start service
docker-compose up -d

# Stop service
docker-compose stop

# Restart service
docker-compose restart

# View logs
docker-compose logs -f

# View logs for last 100 lines
docker-compose logs --tail=100 -f

# Stop and remove containers
docker-compose down

# Rebuild image (setelah code changes)
docker-compose build --no-cache
docker-compose up -d
```

### Accessing Container

```bash
# Execute command in container
docker-compose exec whatsapp_go /bin/sh

# Check running processes
docker-compose exec whatsapp_go ps aux

# Check disk usage
docker-compose exec whatsapp_go df -h
```

## Multi-Account Usage via Docker

### 1. Create Account

```bash
curl -X POST http://localhost:3000/accounts \
  -H "Content-Type: application/json" \
  -d '{"account_id": "account1"}'
```

Response:
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

### 2. Login with QR Code

```bash
curl -X POST http://localhost:3000/accounts/account1/login
```

Response:
```json
{
  "code": "SUCCESS",
  "message": "Please scan the QR code",
  "results": {
    "image_path": "statics/qrcode/scan-account1-1732234567.png",
    "duration": 60000000000,
    "code": "..."
  }
}
```

**Cara Scan QR:**
1. QR code tersimpan di `./statics/qrcode/scan-account1-*.png`
2. Copy file ke komputer lokal atau akses via HTTP
3. Scan dengan WhatsApp di HP

**Akses QR via Browser:**
```
http://localhost:3000/statics/qrcode/scan-account1-1732234567.png
```

### 3. Login with Pairing Code

```bash
curl -X POST http://localhost:3000/accounts/account1/login-with-code \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "6281234567890"}'
```

Response:
```json
{
  "code": "SUCCESS",
  "message": "Login code generated successfully",
  "results": {
    "code": "ABCD-EFGH",
    "message": "Please enter this code in your WhatsApp app"
  }
}
```

**Cara Pairing:**
1. Buka WhatsApp di HP
2. Settings → Linked Devices → Link a Device
3. Pilih "Link with phone number instead"
4. Masukkan pairing code

### 4. Send Message

```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "account1",
    "phone": "6281234567890",
    "message": "Hello from Docker deployment!"
  }'
```

### 5. List Accounts

```bash
curl http://localhost:3000/accounts
```

### 6. Get Account Status

```bash
curl http://localhost:3000/accounts/account1
```

## Production Deployment

### 1. Use External Volumes (Recommended)

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  whatsapp_go:
    image: your-registry/whatsapp-gateway:latest
    restart: always
    ports:
      - "3000:3000"
    volumes:
      # Use named volumes untuk production
      - whatsapp_storage:/app/storages
      - whatsapp_qrcode:/app/statics/qrcode
      - whatsapp_media:/app/statics/media
    env_file:
      - .env.production
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "5"

volumes:
  whatsapp_storage:
    driver: local
  whatsapp_qrcode:
    driver: local
  whatsapp_media:
    driver: local
```

### 2. Behind Reverse Proxy (Nginx)

```nginx
# /etc/nginx/sites-available/whatsapp-gateway
server {
    listen 80;
    server_name wa-gateway.yourdomain.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name wa-gateway.yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # Increase timeout untuk QR generation
    proxy_read_timeout 300;
    proxy_connect_timeout 300;
    proxy_send_timeout 300;

    # Max upload size untuk media
    client_max_body_size 100M;

    location / {
        proxy_pass http://localhost:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

### 3. With Basic Auth

```bash
# Generate basic auth credentials
echo -n "admin:yourpassword" | base64
# Output: YWRtaW46eW91cnBhc3N3b3Jk
```

Update `.env`:
```bash
APP_BASIC_AUTH=admin:yourpassword
```

Test dengan auth:
```bash
curl -u admin:yourpassword http://localhost:3000/accounts
```

### 4. Resource Limits

```yaml
# docker-compose.prod.yml
services:
  whatsapp_go:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

## Backup & Restore

### Backup

```bash
# Backup all account data
tar -czf whatsapp-backup-$(date +%Y%m%d).tar.gz storages/

# Backup specific account
tar -czf account1-backup-$(date +%Y%m%d).tar.gz storages/accounts/account1/

# Backup to remote
rsync -avz storages/ user@backup-server:/backups/whatsapp/
```

### Restore

```bash
# Stop service
docker-compose stop

# Restore backup
tar -xzf whatsapp-backup-20231122.tar.gz

# Start service
docker-compose start
```

## Monitoring

### Health Check

```bash
# Manual health check
curl http://localhost:3000/

# Check container health
docker-compose ps
```

### Logs Monitoring

```bash
# Follow logs
docker-compose logs -f

# Search errors
docker-compose logs | grep -i error

# Export logs
docker-compose logs > logs-$(date +%Y%m%d).log
```

### Disk Usage

```bash
# Check storage size
du -sh storages/

# Check per account
du -sh storages/accounts/*/

# Docker disk usage
docker system df
```

## Troubleshooting

### Container tidak start

```bash
# Check logs
docker-compose logs

# Check if port already used
netstat -tulpn | grep 3000

# Rebuild
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### QR Code tidak muncul

```bash
# Check permissions
ls -la statics/qrcode/

# Create directory if not exists
mkdir -p statics/qrcode
chmod 755 statics/qrcode

# Check logs
docker-compose logs | grep -i qr
```

### Account tidak bisa login

```bash
# Check database
ls -la storages/accounts/account1/

# Remove and recreate
curl -X DELETE http://localhost:3000/accounts/account1
curl -X POST http://localhost:3000/accounts \
  -H "Content-Type: application/json" \
  -d '{"account_id": "account1"}'
```

### Database locked error

```bash
# Stop container
docker-compose stop

# Remove lock files
find storages/ -name "*.db-shm" -delete
find storages/ -name "*.db-wal" -delete

# Start container
docker-compose start
```

## Security Best Practices

1. **Gunakan HTTPS** di production
2. **Enable Basic Auth** untuk protect API
3. **Set APP_TRUSTED_PROXIES** sesuai proxy Anda
4. **Backup regularly** database accounts
5. **Monitor logs** untuk suspicious activity
6. **Use secrets** untuk webhook secret
7. **Limit resource** dengan deploy constraints
8. **Update regularly** base image

## Scale Multiple Instances

Untuk high availability, deploy multiple instances:

```yaml
# docker-compose.ha.yml
version: '3.8'

services:
  whatsapp_go_1:
    # ... config instance 1
    ports:
      - "3001:3000"
    volumes:
      - ./storages1:/app/storages

  whatsapp_go_2:
    # ... config instance 2
    ports:
      - "3002:3000"
    volumes:
      - ./storages2:/app/storages

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - whatsapp_go_1
      - whatsapp_go_2
```

## Support

Jika ada pertanyaan atau issue:
1. Check logs: `docker-compose logs -f`
2. Check GitHub issues
3. Contact support team
