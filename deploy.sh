#!/bin/bash

# BareRTC Deployment Script for CloudPanel
# Usage: ./deploy.sh

set -e

echo "🚀 Starting BareRTC deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
DOMAIN="x.tchatenlive.fr"
APP_NAME="barertc"
DOCKER_COMPOSE_FILE="docker-compose.prod.yml"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [[ $EUID -eq 0 ]]; then
   print_error "This script should not be run as root"
   exit 1
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

print_status "Checking prerequisites..."

# Get server IP
SERVER_IP=$(curl -s ifconfig.me)
if [ -z "$SERVER_IP" ]; then
    print_warning "Could not determine server IP automatically"
    read -p "Please enter your server's public IP address: " SERVER_IP
fi

print_status "Server IP: $SERVER_IP"

# Update TURN server configuration
print_status "Updating TURN server configuration..."
sed -i "s/YOUR_SERVER_IP/$SERVER_IP/g" turnserver.conf

# Update docker-compose file
print_status "Updating docker-compose configuration..."
sed -i "s/YOUR_SERVER_IP/$SERVER_IP/g" docker-compose.prod.yml

# Build and start the application
print_status "Building and starting BareRTC..."
docker-compose -f $DOCKER_COMPOSE_FILE down --remove-orphans
docker-compose -f $DOCKER_COMPOSE_FILE build --no-cache
docker-compose -f $DOCKER_COMPOSE_FILE up -d

# Wait for services to start
print_status "Waiting for services to start..."
sleep 10

# Check if services are running
if docker-compose -f $DOCKER_COMPOSE_FILE ps | grep -q "Up"; then
    print_status "✅ BareRTC is running successfully!"
else
    print_error "❌ Failed to start BareRTC"
    docker-compose -f $DOCKER_COMPOSE_FILE logs
    exit 1
fi

# Show service status
print_status "Service status:"
docker-compose -f $DOCKER_COMPOSE_FILE ps

# Show logs
print_status "Recent logs:"
docker-compose -f $DOCKER_COMPOSE_FILE logs --tail=20

print_status "🎉 Deployment completed successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Configure your domain 'x.tchatenlive.fr' to point to your server IP: $SERVER_IP"
echo "2. Set up SSL certificate in CloudPanel for the domain"
echo "3. Configure reverse proxy in CloudPanel to forward traffic to localhost:9000"
echo "4. Test the application at https://x.tchatenlive.fr"
echo ""
echo "🔧 Useful commands:"
echo "  View logs: docker-compose -f $DOCKER_COMPOSE_FILE logs -f"
echo "  Stop services: docker-compose -f $DOCKER_COMPOSE_FILE down"
echo "  Restart services: docker-compose -f $DOCKER_COMPOSE_FILE restart"
echo "  Update application: ./deploy.sh" 