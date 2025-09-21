#!/bin/bash

# Ethiopia Dating App Startup Script

echo "Starting Ethiopia Dating App..."

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Docker is not running. Please start Docker first."
    exit 1
fi

# Check if .env file exists
if [ ! -f .env ]; then
    echo "Creating .env file from template..."
    cp env.example .env
    echo "Please edit .env file with your configuration before running again."
    exit 1
fi

# Start services with Docker Compose
echo "Starting services with Docker Compose..."
docker-compose up -d

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 10

# Check if services are running
echo "Checking service status..."
docker-compose ps

echo "Services started successfully!"
echo ""
echo "Access points:"
echo "- API: http://localhost:8080"
echo "- MinIO Console: http://localhost:9001 (admin/admin)"
echo "- PostgreSQL: localhost:5432"
echo "- Redis: localhost:6379"
echo ""
echo "To view logs: docker-compose logs -f"
echo "To stop services: docker-compose down"
