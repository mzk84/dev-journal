#!/bin/bash

# --- Go Project Directory Structure Script ---

# 1. Define the base project structure and files
# Directories to create
DIRS=(
    cmd/app/
    internal/config/
    internal/content/
    internal/database/
    internal/server/
    web/templates/
    web/static/img/
    content/
)

# Files to touch (relative to the script execution directory)
FILES=(
    cmd/app/main.go
    internal/config/config.go
    internal/content/sync.go
    internal/database/database.go
    internal/server/handlers.go
    internal/server/middleware.go
    Dockerfile
    docker-compose.yml
    go.mod
    go.sum
    README.md
)

# 2. Create all necessary directories
echo "Creating directories..."
for dir in "${DIRS[@]}"; do
    mkdir -p "$dir"
    if [ $? -eq 0 ]; then
        echo "✅ Created directory: $dir"
    else
        echo "❌ Failed to create directory: $dir" >&2
    fi
done

# 3. Create all specified files
echo
echo "Touching files..."
for file in "${FILES[@]}"; do
    touch "$file"
    if [ $? -eq 0 ]; then
        echo "✅ Created file: $file"
    else
        echo "❌ Failed to create file: $file" >&2
    fi
done

echo
echo "✨ Project structure created successfully! ✨"