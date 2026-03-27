# HouseHero - Home Maintenance Tracker

HouseHero is a comprehensive home maintenance tracking system with a React frontend and Go backend. Manage tickets, inventory, and maintenance schedules for your homes.

## Features

- 📋 **Ticket Management**: Create and track maintenance tickets with priority levels and status
- 📦 **Inventory Tracking**: Keep detailed records of home appliances and systems
- 📄 **Document Storage**: Upload and store manuals and receipts for inventory items
- 👤 **People Management**: Track who is responsible for each task
- 🏠 **Multi-Home Support**: Manage multiple properties
- 🔄 **Ticket Dependencies**: Mark tickets as blocking/waiting on other tickets
- 💬 **Comments & History**: Track all changes with automatic comments
- 📊 **Dashboard**: Quick overview of open tickets and priorities

## Tech Stack

- **Frontend**: React 18 with Vanilla CSS
- **Backend**: Go 1.21
- **Database**: PostgreSQL 15
- **Reverse Proxy**: Caddy
- **Containerization**: Docker & Docker Compose

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/tmjpugh/househero.git
cd househero

# Start the application
docker-compose up
