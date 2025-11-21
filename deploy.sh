#!/bin/bash

# Deploy script untuk WhatsApp Multi-Account Gateway
set -e

echo "=========================================="
echo "WhatsApp Multi-Account Gateway - Deployment"
echo "=========================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
}

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

print_success "Docker and Docker Compose are installed"

# Check if .env exists
if [ ! -f "src/.env" ]; then
    print_info "Creating .env file from .env.example..."
    cp src/.env.example src/.env
    print_success ".env file created"
    print_info "Please edit src/.env file to configure your settings"
    echo ""
    echo "Press Enter to continue after editing .env file..."
    read
fi

# Create required directories
print_info "Creating required directories..."
mkdir -p storages/accounts
mkdir -p statics/qrcode
mkdir -p statics/senditems
mkdir -p statics/media

# Set permissions
chmod -R 755 storages statics 2>/dev/null || true

print_success "Directories created"

# Build Docker image
print_info "Building Docker image..."
docker-compose build

print_success "Docker image built successfully"

# Start services
print_info "Starting services..."
docker-compose up -d

print_success "Services started"

# Wait for service to be ready
print_info "Waiting for service to be ready..."
sleep 5

# Check if service is running
if docker-compose ps | grep -q "Up"; then
    print_success "Service is running"

    echo ""
    echo "=========================================="
    echo "Deployment Successful! 🎉"
    echo "=========================================="
    echo ""
    echo "Service URL: http://localhost:3000"
    echo ""
    echo "Quick Start Commands:"
    echo "  - View logs:          docker-compose logs -f"
    echo "  - Stop service:       docker-compose stop"
    echo "  - Restart service:    docker-compose restart"
    echo "  - View status:        docker-compose ps"
    echo ""
    echo "Create your first account:"
    echo "  curl -X POST http://localhost:3000/accounts \\"
    echo "    -H 'Content-Type: application/json' \\"
    echo "    -d '{\"account_id\": \"account1\"}'"
    echo ""
    echo "Login with QR code:"
    echo "  curl -X POST http://localhost:3000/accounts/account1/login"
    echo ""
    echo "For full documentation, see: DOCKER_DEPLOYMENT.md"
    echo ""
else
    print_error "Service failed to start. Check logs with: docker-compose logs"
    exit 1
fi
