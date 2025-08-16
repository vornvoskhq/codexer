#!/bin/bash

# Exit on error
set -e

# Set environment variables
export DATABASE_URL="postgres://plandex:plandex@localhost:5432/plandex?sslmode=disable"
export GOENV=development
export LOCAL_MODE=1
export PLANDEX_BASE_DIR="$(pwd)"
export OLLAMA_BASE_URL="http://host.docker.internal:11434"

# Create necessary directories
mkdir -p "${PLANDEX_BASE_DIR}/llm-logs"

echo "Starting Plandex server with the following configuration:"
echo "- Database: ${DATABASE_URL}"
echo "- Base Directory: ${PLANDEX_BASE_DIR}"
echo "- Ollama URL: ${OLLAMA_BASE_URL}"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "docker-compose could not be found. Please install it and try again."
    exit 1
fi

# Navigate to the app directory
cd "$(dirname "$0")/app"

# Start just the PostgreSQL database
echo "Starting PostgreSQL database with Docker Compose..."
docker-compose up -d plandex-postgres

echo "Waiting for PostgreSQL to be ready..."
until docker-compose exec -T plandex-postgres pg_isready -U plandex -d plandex > /dev/null 2>&1; do
    echo "Waiting for PostgreSQL to be ready..."
    sleep 2
done

echo "PostgreSQL is ready!"
echo "Starting local Plandex server..."

# Run the local plandex-server executable
cd server
./plandex-server
