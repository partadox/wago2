# 🐳 Docker Deployment - Quick Start

## Cara Deploy dengan Docker (5 Menit)

### 1️⃣ Clone & Setup
```bash
cd /path/to/wago2

# Copy environment file
cp src/.env.example src/.env
```

### 2️⃣ Deploy!
```bash
# One command deployment
./deploy.sh
```

Script akan otomatis:
- ✅ Check Docker & Docker Compose
- ✅ Buat directories yang diperlukan
- ✅ Build Docker image
- ✅ Start services
- ✅ Health check

### 3️⃣ Test Multi-Account
```bash
./test-multi-account.sh
```

## Manual Deployment

Jika prefer manual:

```bash
# 1. Buat directories
mkdir -p storages/accounts statics/{qrcode,senditems,media}

# 2. Build & Run
docker-compose build
docker-compose up -d

# 3. Check logs
docker-compose logs -f

# 4. Test
curl http://localhost:3000/
```

## Quick Commands

```bash
# View logs
docker-compose logs -f

# Stop service
docker-compose stop

# Start service
docker-compose start

# Restart service
docker-compose restart

# Stop & remove
docker-compose down

# Rebuild after code changes
docker-compose build --no-cache
docker-compose up -d
```

## 📱 Cara Pakai Multi-Account

### Create Account
```bash
curl -X POST http://localhost:3000/accounts \
  -H "Content-Type: application/json" \
  -d '{"account_id": "myaccount"}'
```

### Login dengan QR
```bash
# Generate QR
curl -X POST http://localhost:3000/accounts/myaccount/login

# QR tersimpan di: statics/qrcode/scan-myaccount-*.png
# Buka file tersebut dan scan dengan WhatsApp
```

### Login dengan Pairing Code
```bash
curl -X POST http://localhost:3000/accounts/myaccount/login-with-code \
  -H "Content-Type: application/json" \
  -d '{"phone_number": "6281234567890"}'

# Masukkan code yang muncul di WhatsApp HP
```

### Send Message
```bash
curl -X POST http://localhost:3000/send/message \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "myaccount",
    "phone": "6281234567890",
    "message": "Hello from Docker!"
  }'
```

### List Accounts
```bash
curl http://localhost:3000/accounts
```

### Check Account Status
```bash
curl http://localhost:3000/accounts/myaccount
```

## 📂 Struktur Data

```
wago2/
├── storages/              # ← PERSISTENT DATA (backup ini!)
│   ├── accounts.db       # Account registry
│   ├── accounts/         # Per-account databases
│   │   ├── myaccount/
│   │   │   ├── whatsapp.db
│   │   │   └── keys.db
│   │   └── account2/
│   │       ├── whatsapp.db
│   │       └── keys.db
│   └── chatstorage.db    # Chat history
│
├── statics/              # ← PERSISTENT FILES
│   ├── qrcode/          # QR codes untuk login
│   ├── senditems/       # Temporary send files
│   └── media/           # Downloaded media
│
└── docker-compose.yml
```

## 🔧 Troubleshooting

### "No server available" when using environment variables in Coolify/Docker

**Problem**: Service returns "no server available" when environment variables are set, but works without them.

**Root Causes**:
1. **Quotes in environment variables**: The old `.env.example` had quotes around values (like `DB_URI="file:..."`). When copied to Coolify, quotes become part of the value, causing database failures.
2. **Basic Auth blocking health check**: When `APP_BASIC_AUTH` is set, it blocked the health check endpoint, making Coolify think the service is down.

**Solution**:
```bash
# ❌ WRONG (with quotes - copied from old .env.example)
DB_URI="file:storages/whatsapp.db?_foreign_keys=on"

# ✅ CORRECT (without quotes - use this in Coolify)
DB_URI=file:storages/whatsapp.db?_foreign_keys=on
```

**Important Environment Variables for Coolify**:
```bash
# Required
APP_PORT=3000
DB_URI=file:storages/whatsapp.db?_foreign_keys=on
DB_KEYS_URI=file::memory:?cache=shared&_foreign_keys=on

# Optional
APP_DEBUG=false
APP_OS=Chrome
WHATSAPP_AUTO_MARK_READ=false
WHATSAPP_AUTO_DOWNLOAD_MEDIA=true
WHATSAPP_ACCOUNT_VALIDATION=true
WHATSAPP_CHAT_STORAGE=true

# Security (Optional - jika ingin protect API)
APP_BASIC_AUTH=user1:pass1
```

**Notes**:
- ⚠️ Do NOT use quotes when setting environment variables in Docker/Coolify UI!
- ✅ Health check endpoint `/health` tidak memerlukan autentikasi (aman untuk Coolify)
- ✅ Jika menggunakan `APP_BASIC_AUTH`, semua endpoint API (kecuali `/health`) akan require autentikasi

### Port already in use
```bash
# Check port 3000
netstat -tulpn | grep 3000

# Change port in docker-compose.yml
ports:
  - "3001:3000"  # Use port 3001 instead
```

### Cannot access QR code
```bash
# Check if file exists
ls -la statics/qrcode/

# Fix permissions
chmod -R 755 statics/
```

### Database locked
```bash
# Stop container
docker-compose stop

# Remove lock files
find storages/ -name "*.db-shm" -delete
find storages/ -name "*.db-wal" -delete

# Start again
docker-compose start
```

### Container keeps restarting
```bash
# Check logs
docker-compose logs --tail=50

# Common issues:
# 1. Port already used
# 2. Invalid .env file
# 3. Missing directories
```

## 🔒 Security untuk Production

### 1. Enable Basic Auth
Edit `src/.env`:
```bash
APP_BASIC_AUTH=admin:your-secure-password
```

Test dengan auth:
```bash
curl -u admin:your-secure-password http://localhost:3000/accounts
```

### 2. Behind Nginx (HTTPS)
```nginx
server {
    listen 443 ssl;
    server_name wa.yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 3. Firewall
```bash
# Allow only your IPs
ufw allow from YOUR_IP to any port 3000
```

## 💾 Backup

### Manual Backup
```bash
# Backup semua data
tar -czf backup-$(date +%Y%m%d).tar.gz storages/

# Backup specific account
tar -czf account1-$(date +%Y%m%d).tar.gz storages/accounts/account1/
```

### Restore
```bash
docker-compose stop
tar -xzf backup-20231122.tar.gz
docker-compose start
```

### Automated Backup (Cron)
```bash
# Add to crontab
0 2 * * * cd /path/to/wago2 && tar -czf /backups/wago-$(date +\%Y\%m\%d).tar.gz storages/
```

## 📊 Monitoring

### Health Check
```bash
# Manual check
curl http://localhost:3000/

# Docker health status
docker-compose ps
```

### Resource Usage
```bash
# Container stats
docker stats

# Disk usage
du -sh storages/
du -sh storages/accounts/*/
```

### Logs
```bash
# Follow logs
docker-compose logs -f

# Last 100 lines
docker-compose logs --tail=100

# Search errors
docker-compose logs | grep -i error

# Export to file
docker-compose logs > logs-$(date +%Y%m%d).log
```

## 🚀 Production Deployment

### Docker Compose Production
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
      - /var/whatsapp/storages:/app/storages
      - /var/whatsapp/statics:/app/statics
    env_file:
      - .env.production
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

Deploy:
```bash
docker-compose -f docker-compose.prod.yml up -d
```

## 📚 Full Documentation

- **Lengkap:** [DOCKER_DEPLOYMENT.md](DOCKER_DEPLOYMENT.md)
- **Multi-Account:** [MULTI_ACCOUNT_PROGRESS.md](MULTI_ACCOUNT_PROGRESS.md)
- **API:** [wago-postman-api-collection.json](wago-postman-api-collection.json)

## 🆘 Support

Jika ada masalah:
1. Check logs: `docker-compose logs -f`
2. Check GitHub issues
3. Read full docs: `DOCKER_DEPLOYMENT.md`

---

Made with ❤️ for Multi-Account WhatsApp Gateway
