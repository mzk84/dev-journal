# Dev-Journal: A Git-Based Markdown Website Engine

This project provides a complete solution for running a website where the content is managed as Markdown files in a private GitHub repository. It's built on Go and leverages SQLite for metadata and analytics, with HTMX for a dynamic admin interface.

## Features

- **Git-based CMS**: Manage all your content (.md files and images) through a simple git push.
- **Automated Updates**: A GitHub webhook triggers a git pull on the server, automatically syncing new or updated content.
- **Dynamic Rendering**: Markdown files are parsed and rendered into HTML on the fly.
- **Lightweight Analytics**: Tracks page visits for each Markdown file.
- **Admin Dashboard**: A secret admin area to view stats and toggle page visibility.
- **Themable**: Basic branding (colors, logo) can be configured via environment variables.
- **Containerized**: Ready for deployment as a Docker container.

## Tech Stack

- **Backend**: Go
- **Routing**: Chi
- **Database**: SQLite
- **Markdown Parsing**: Goldmark
- **Frontend Interaction**: HTMX
- **Styling**: Tailwind CSS (via CDN)
- **Templating**: Go html/template
- **Deployment**: Docker

## Project Structure

```
├── cmd/app/main.go             # Main application entry point
├── internal/
│   ├── server/                 # Web server, routing, and handlers
│   │   ├── handlers.go
│   │   └── middleware.go
│   ├── database/               # SQLite interaction logic
│   │   └── database.go
│   ├── content/                # Logic for syncing git repo with DB
│   │   └── sync.go
│   └── config/                 # Configuration management
│       └── config.go
├── web/
│   ├── templates/              # HTML templates
│   │   ├── layout.html
│   │   ├── page.html
│   │   ├── admin_login.html
│   │   └── admin_dashboard.html
│   └── static/                 # Static assets (logo, custom css, etc.)
│       └── img/
│           └── logo.svg
├── content/                    # (Created at runtime) Cloned git repository
├── go.mod
├── go.sum
├── Dockerfile                  # For building the production container
└── docker-compose.yml          # For easy deployment
```

## How It Works

### Initialization

The Go application starts, initializes the SQLite database, and clones your private GitHub repository into the `content/` directory for the first time.

### Request Handling

When a user visits a URL (e.g., `/about`), the app looks up `about.md` in its database.

### Rendering

If the page exists and is visible, the app reads `about.md`, converts it to HTML using Goldmark, increments its visit count, and serves it within a themed HTML template.

### Content Update

1. You git push a change to your private content repository.
2. **Webhook Trigger**: GitHub sends a webhook to the `/webhook` endpoint on your server.
3. **Syncing**: The app verifies the webhook, pulls the latest changes from your repo into the `content/` folder, and runs a sync operation to update the SQLite database with any new or changed files.

## Deployment Guide (Hetzner VM)

### Prerequisites

A Hetzner (or any) Linux VM with Docker and Docker Compose installed.

### 1. GitHub Setup

#### a. Create a Private Repository

This repo will hold your content (`home.md`, `blog/post1.md`, `img/logo.png`, etc.).

#### b. Create a Deploy Key

On your local machine, generate a new SSH key pair specifically for this purpose:

```bash
ssh-keygen -t ed25519 -C "your_email@example.com" -f ./gmd_deploy_key
```

This creates `gmd_deploy_key` (private key) and `gmd_deploy_key.pub` (public key).

In your GitHub repo settings, go to **Deploy Keys > Add deploy key**.

- Give it a title (e.g., "Hetzner VM Deploy Key").
- Paste the contents of `gmd_deploy_key.pub`.
- Do not check "Allow write access".

#### c. Set up a Webhook

In your GitHub repo settings, go to **Webhooks > Add webhook**.

- **Payload URL**: `http://<your-server-ip-or-domain>/webhook`
- **Content type**: `application/json`
- **Secret**: Generate a strong random string and save it. You'll need this for the environment variables.
- **Events**: Just the `push` event is fine.

### 2. Server Setup

#### a. Transfer Files to Server

Copy the entire project folder (except the `.git` directory) to your Hetzner VM.

Securely copy the private deploy key (`gmd_deploy_key`) to a safe location on the server, for example, `/root/.ssh/gmd_deploy_key`.

#### b. Create `prod.env` file

In the project root on your server, create a file named `prod.env` and add the following, replacing the placeholder values:

```env
# -- GitHub & Content --
# Use the SSH clone URL from your repo
GIT_REPO_URL=git@github.com:your-username/your-private-repo.git
# Path inside the container where the private deploy key is mounted
GIT_SSH_KEY_PATH=/app/secrets/gmd_deploy_key

# -- Webhook --
# The secret you created in the GitHub webhook settings
GITHUB_WEBHOOK_SECRET=your_super_strong_webhook_secret

# -- Admin --
# A secret password to log into the admin panel
ADMIN_SECRET=your_super_strong_admin_password
# A non-obvious URL for the admin login page
ADMIN_LOGIN_PATH=/secret-admin-area

# -- Theming (Optional) --
THEME_LOGO_URL=/static/img/logo.svg
THEME_COLOR_PRIMARY=#3498db
THEME_FONT_SANS="Inter"
```

#### c. Configure `docker-compose.yml`

Make sure the `volumes` section correctly maps the path to your private deploy key on the host machine. Update `/root/.ssh/gmd_deploy_key` if you stored it elsewhere.

### 3. Build and Run

From your project directory on the server, run:

```bash
docker-compose up --build -d
```

Your site will be running on port `8080`. You should set up a reverse proxy like Nginx or Caddy to handle SSL and forward requests from port `80/443` to `8080`.

### 4. First Run & Accessing the Admin Panel

On the first run, the application will clone your repository.

To access the admin panel, go to `http://<your-domain>/secret-admin-area` (or whatever you set `ADMIN_LOGIN_PATH` to).

Use the password you set in `ADMIN_SECRET` to log in.