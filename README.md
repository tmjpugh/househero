# HouseHero - Home Maintenance Tracker

HouseHero is a comprehensive home maintenance tracking system with a React frontend and Go backend. Manage tickets, inventory, and maintenance schedules for your homes.

## Features

- 📋 **Ticket Management**: Create and track maintenance tickets with priority levels and status
- 📦 **Inventory Tracking**: Keep detailed records of home appliances and systems
- 📄 **Document Storage**: Upload and store manuals and receipts for inventory items
- 👤 **People Management**: Track who is responsible for each task
- 🏠 **Multi-Home Support**: Manage multiple properties
- 🔄 **Ticket Dependencies**: Mark tickets as blocking/waiting on other tickets
- 💬 **Comments & History**: Track all changes with automatic comments; latest comment shown inline on ticket cards
- 📊 **Dashboard**: Quick overview of open tickets and priorities
- 🔌 **Home Assistant / MQTT Integration**: Publish events and accept commands over MQTT

## Tech Stack

- **Frontend**: React 18 with Vanilla CSS
- **Backend**: Go 1.26
- **Database**: PostgreSQL 15
- **Containerization**: Docker & Docker Compose
- **MQTT** (optional): Eclipse Paho client; any MQTT v3.1/v5 broker (e.g. Mosquitto)

---

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/tmjpugh/househero.git
cd househero

# Copy and review environment variables (see Configuration below)
cp .env.example .env

# Start the application
docker-compose up
```

The application is then available at **http://localhost:8080**.

### Ports

| Service    | Host port | Container port | Purpose                  |
|------------|-----------|----------------|--------------------------|
| `app`      | 8080      | 8080           | HTTP API + web UI        |
| `postgres` | 5432      | 5432           | PostgreSQL database      |

### Mounted Volumes

| Volume name      | Mounted at (container) | Purpose                             |
|------------------|------------------------|-------------------------------------|
| `postgres_data`  | `/var/lib/postgresql/data` | Persistent database storage     |
| `uploads_data`   | `/app/uploads`         | Uploaded photos, manuals, receipts  |

> **Tip**: To persist data between `docker-compose down` / `up` cycles, the named volumes above are created automatically. To completely reset all data run `docker-compose down -v`.

---

## Configuration

All settings are read from environment variables (or a `.env` file).

| Variable        | Default          | Description                                      |
|-----------------|------------------|--------------------------------------------------|
| `DB_HOST`       | `localhost`      | PostgreSQL hostname                              |
| `DB_PORT`       | `5432`           | PostgreSQL port                                  |
| `DB_USER`       | `househero`      | PostgreSQL username                              |
| `DB_PASSWORD`   | `househero_dev`  | PostgreSQL password                              |
| `DB_NAME`       | `househero_db`   | PostgreSQL database name                         |
| `PORT`          | `8080`           | HTTP port the Go server listens on               |
| `MQTT_BROKER`   | *(empty)*        | MQTT broker URL, e.g. `tcp://mosquitto:1883`; leave blank to disable MQTT |
| `MQTT_CLIENT_ID`| `househero`      | MQTT client identifier                           |
| `MQTT_USERNAME` | *(empty)*        | MQTT username (if broker requires auth)          |
| `MQTT_PASSWORD` | *(empty)*        | MQTT password (if broker requires auth)          |

---

## REST API Reference

All endpoints are under the path `/api/`. Requests and responses use JSON unless noted.

### Homes

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/api/homes` | List all homes |
| `GET`  | `/api/homes/{id}` | Get a single home |
| `POST` | `/api/homes` | Create a home |
| `PUT`  | `/api/homes/{id}` | Update a home |
| `DELETE` | `/api/homes/{id}` | Delete a home |
| `GET`  | `/api/homes/{id}/settings` | Get per-home settings |
| `PUT`  | `/api/homes/{id}/settings` | Update per-home settings |

### Tickets

| Method | Path | Description |
|--------|------|-------------|
| `GET`    | `/api/tickets?home_id={id}` | List tickets for a home (includes comments) |
| `GET`    | `/api/tickets/{id}` | Get a single ticket with comments and photos |
| `POST`   | `/api/tickets` | Create a ticket |
| `PUT`    | `/api/tickets/{id}` | Update a ticket |
| `DELETE` | `/api/tickets/{id}?home_id={home_id}` | Delete a ticket |
| `POST`   | `/api/tickets/{id}/comments` | Add a comment |
| `POST`   | `/api/tickets/{id}/photos` | Upload a photo |
| `POST`   | `/api/tickets/{id}/documents` | Upload a document |

**Create ticket body** (`POST /api/tickets`):

```json
{
  "home_id": 1,
  "title": "Leaky faucet",
  "type": "maintenance",
  "priority": "medium",
  "status": "open",
  "requester": "Alice",
  "room": "Bathroom",
  "description": "Dripping every 5 seconds",
  "estimated_cost": "85.00",
  "inventory_item_id": 3
}
```

**Add comment body** (`POST /api/tickets/{id}/comments`):

```json
{
  "text": "Ordered replacement washer, arrives Thursday.",
  "author": "Alice",
  "is_system": false
}
```

### Inventory

| Method | Path | Description |
|--------|------|-------------|
| `GET`    | `/api/inventory?home_id={id}` | List inventory items for a home |
| `GET`    | `/api/inventory/{id}` | Get a single inventory item with documents and notes |
| `POST`   | `/api/inventory` | Create an inventory item |
| `PUT`    | `/api/inventory/{id}` | Update an inventory item |
| `DELETE` | `/api/inventory/{id}` | Delete an inventory item |
| `POST`   | `/api/inventory/{id}/receipts` | Upload a receipt |
| `POST`   | `/api/inventory/{id}/manuals` | Upload a manual |
| `DELETE` | `/api/inventory/{id}/documents/{docId}` | Delete a document |

**Create inventory item body** (`POST /api/inventory`):

```json
{
  "home_id": 1,
  "name": "Kitchen Refrigerator",
  "type": "Appliance",
  "make": "Samsung",
  "model": "RF28R7351SR",
  "room": "Kitchen",
  "serial_number": "SN12345",
  "purchase_date": "2022-06-15T00:00:00Z",
  "warranty_expires": "2027-06-15T00:00:00Z"
}
```

---

## MQTT Integration

HouseHero can connect to any MQTT v3.1/v5 broker (e.g. [Eclipse Mosquitto](https://mosquitto.org/)) to:

1. **Publish events** whenever a ticket or inventory item is created, updated, or commented on.
2. **Accept commands** to create tickets or retrieve ticket details — useful for Home Assistant automations.

MQTT is **opt-in**. Set `MQTT_BROKER` in your `.env` / environment to enable it.

### Published Events

| Topic | Trigger | Payload |
|-------|---------|---------|
| `househero/tickets/created` | New ticket saved | Full ticket JSON |
| `househero/tickets/updated` | Ticket updated | Full ticket JSON |
| `househero/tickets/comment_added` | Comment added | Comment + full ticket JSON |
| `househero/inventory/created` | New inventory item saved | Full item JSON |
| `househero/inventory/updated` | Inventory item updated | Full item JSON |

### Command Topics (Incoming)

Send a JSON payload to one of these topics; the response is published to `househero/responses/{request_id}` (or `househero/responses/default` when `request_id` is omitted).

#### Create a ticket

**Topic:** `househero/commands/tickets/create`

| Field | Required | Description |
|-------|----------|-------------|
| `home_id` | ✅ | Numeric home ID |
| `title` | ✅ | Ticket title |
| `request_id` | — | Optional; used to route the response |
| `type` | — | Default: `maintenance` |
| `priority` | — | Default: `medium` |
| `requester` | — | Person requesting the work |
| `room` | — | Room/location |
| `description` | — | Longer description |
| `estimated_cost` | — | Cost estimate string, e.g. `"150.00"` |
| `inventory_item_id` | — | Numeric ID of a linked inventory item; the item name is resolved automatically |
| `inventory_item` | — | Free-text inventory item name (used when `inventory_item_id` is not provided) |

All string fields are sanitized to remove HTML tags and control characters before storage. Invalid or missing optional fields are silently left blank rather than causing an error.

Example (by inventory item ID):

```json
{
  "request_id": "ha-auto-001",
  "home_id": 1,
  "title": "Replace HVAC filter",
  "type": "maintenance",
  "priority": "low",
  "requester": "Home Assistant",
  "room": "Utility Room",
  "description": "Monthly filter replacement reminder",
  "inventory_item_id": 7
}
```

Example (by free-text inventory item name):

```json
{
  "request_id": "ha-auto-002",
  "home_id": 1,
  "title": "Water leak detected – Kitchen",
  "type": "emergency",
  "priority": "critical",
  "requester": "Home Assistant",
  "room": "Kitchen",
  "inventory_item": "Kitchen Sink"
}
```

Response (published to `househero/responses/ha-auto-001`):

```json
{
  "id": 42,
  "ticket_number": 17,
  "home_id": 1,
  "title": "Replace HVAC filter",
  ...
}
```

#### Get ticket details

**Topic:** `househero/commands/tickets/detail`

| Field | Required | Description |
|-------|----------|-------------|
| `home_id` | ✅ | Numeric home ID |
| `ticket_number` | ✅ | Ticket number within that home |
| `request_id` | — | Optional; used to route the response |

Example:

```json
{
  "request_id": "ha-query-005",
  "home_id": 1,
  "ticket_number": 17
}
```

Response (published to `househero/responses/ha-query-005`):

```json
{
  "id": 42,
  "ticket_number": 17,
  "home_id": 1,
  "title": "Replace HVAC filter",
  "status": "open",
  "comments": [ ... ],
  ...
}
```

On error the response contains `{"error": "..."}`.

### Home Assistant Examples

#### Example 1 — Create a ticket from a sensor trigger

A minimal automation that creates a HouseHero ticket when a sensor triggers:

```yaml
automation:
  - alias: "Create HouseHero ticket from sensor"
    trigger:
      - platform: state
        entity_id: binary_sensor.water_leak_kitchen
        to: "on"
    action:
      - service: mqtt.publish
        data:
          topic: househero/commands/tickets/create
          payload: >
            {
              "request_id": "ha-leak-{{ now().timestamp() | int }}",
              "home_id": 1,
              "title": "Water leak detected – Kitchen",
              "type": "emergency",
              "priority": "critical",
              "requester": "Home Assistant",
              "room": "Kitchen"
            }
```

#### Example 2 — Notify when a new inventory item is added

Subscribes to `househero/inventory/created` and sends a mobile notification:

```yaml
automation:
  - alias: "HouseHero: New inventory item added"
    trigger:
      - platform: mqtt
        topic: househero/inventory/created
    action:
      - service: notify.mobile_app_your_phone
        data:
          title: "New Inventory Item"
          message: >
            [Home] added {{ trigger.payload_json.name }}
            ({{ trigger.payload_json.type }}) in {{ trigger.payload_json.room }}.
```

> Replace `notify.mobile_app_your_phone` with your actual notification service.
> Substitute `[Home]` with a static home name or map it from `trigger.payload_json.home_id` using a template helper.

#### Example 3 — Notify when a ticket is created

Subscribes to `househero/tickets/created` and sends a mobile notification:

```yaml
automation:
  - alias: "HouseHero: New ticket created"
    trigger:
      - platform: mqtt
        topic: househero/tickets/created
    action:
      - service: notify.mobile_app_your_phone
        data:
          title: "New Ticket"
          message: >
            [Home] added #{{ trigger.payload_json.ticket_number }}
            {{ trigger.payload_json.title }}
```

> Substitute `[Home]` with a static home name or map it from `trigger.payload_json.home_id` using a template helper.

#### Example 4 — Notify on ticket status change or human comment

Subscribes to both `househero/tickets/updated` (status changes) and
`househero/tickets/comment_added` (new comments), skipping system-generated
comments and updates that do not change status:

```yaml
automation:
alias: "HouseHero: Ticket status or comment"
description: Notifies on ticket status changes or new comments
triggers:
  - topic: househero/tickets/updated
    id: status_changed
    trigger: mqtt
  - topic: househero/tickets/comment_added
    id: comment_added
    trigger: mqtt
conditions:
  - condition: template
    value_template: |
      {% if trigger.id == 'status_changed' %}
        {{ trigger.payload_json.status_new != '' and
           trigger.payload_json.status_old != trigger.payload_json.status_new }}
      {% else %}
        {{ not trigger.payload_json.is_system }}
      {% endif %}
actions:
  - action: notify.mobile
    data:
      title: HouseHero Ticket Update
      message: |
        {% if trigger.id == 'status_changed' %}
          [Home] #{{ trigger.payload_json.ticket_number }}
          {{ trigger.payload_json.title }} | {{ trigger.payload_json.status_new }}
        {% else %}
          [Home] #{{ trigger.payload_json.ticket_number }}
          {{ trigger.payload_json.title }} | {{ trigger.payload_json.status }} |
          {{ trigger.payload_json.author }} commented: {{ trigger.payload_json.text }}
        {% endif %}
      data:
        tag: househero_comment
mode: single


```

> Substitute `[Home]` with a static home name or map it from `trigger.payload_json.home_id` using a template helper.

#### Example 5 — Create a ticket with a voice command

Say **"Househero, create a ticket *\<title\>*"** to Home Assistant's voice
assistant and have it publish a create-ticket command to MQTT.

Home Assistant (2023.5+) supports inline conversation sentences directly in
automations — no separate intent file is needed:

```yaml
alias: "HouseHero: Create ticket via voice"
description: ""
triggers:
  - command:
      - Househero create [a] ticket {title}
    trigger: conversation
actions:
  - action: mqtt.publish
    data:
      topic: househero/commands/tickets/create
      payload: |
        {
          "request_id": "ha-voice-{{ now().timestamp() | int }}",
          "home_id": 1,
          "title": "{{ trigger.slots.title }}",
          "requester": "Voice Command",
          "priority": "medium",
          "room": "other",
        }
```

> Set `home_id` to the numeric ID of your home (visible in the HouseHero URL
> when viewing that home, or via `GET /api/homes`).  The captured phrase is
> available as `trigger.slots.title`.

### Running Mosquitto with Docker Compose

Uncomment the `mosquitto` service block in `docker-compose.yaml` and add a minimal `mosquitto.conf`:

```
# mosquitto.conf
listener 1883
allow_anonymous true
```

Then set `MQTT_BROKER=tcp://mosquitto:1883` in your environment.

---

## Development

```bash
# Run locally (requires a PostgreSQL instance)
cp .env.example .env
# Edit .env with your local DB credentials
go run main.go
```

### Building

```bash
go build -o househero ./...
```

