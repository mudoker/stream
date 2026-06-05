# stream - VIM-First TUI Calendar & Task Management Engine

stream is a terminal-native calendar and productivity engine. It treats calendar events as executable tasks that move through a defined lifecycle.

Its layout features a left-side vertical sidebar inspired by Arc, showing workspaces and docked focus widgets, with a right-side dashboard and timeline styled after Linear.

---

## Installation Guide

### Prerequisites

* Go (version 1.18 or higher)
* Git

### 1. Clone the Repository

```bash
git clone https://github.com/mudoker/stream.git
cd stream
```

### 2. Build the Optimized Binary

```bash
go build -ldflags="-s -w" -o stream
```

### 3. Install Globally

```bash
# Move to system path
sudo mv stream /usr/local/bin/

# Or move to local user path
mkdir -p ~/.local/bin
mv stream ~/.local/bin/
```
Ensure your shell's PATH includes the destination directory.

### 4. Run the Application

```bash
stream
```

---

## Running Unit Tests

```bash
go test ./...
```

---

## Google Calendar OAuth2 Setup

stream runs a delta-sync daemon that operates offline and syncs local updates back to Google Calendar when connection is restored.

1. Create a Google Cloud project.
2. Enable the Google Calendar API.
3. Configure the OAuth Consent Screen and create Desktop Client Credentials.
4. Download the secrets JSON, rename it to client_secrets.json, and save it to:
   * ~/.config/stream/client_secrets.json
5. Start stream, press : to open the command palette, type auth, and press Enter.
6. Open the browser callback link to complete authorization. Tokens will save to ~/.config/stream/credentials.json.

---

## Configuration Directory

All data files reside in ~/.config/stream/:
* data.json - Local task and calendar database.
* ledger.json - Offline transaction ledger logs.
* credentials.json - Google OAuth2 tokens.
* client_secrets.json - Google Calendar client credentials.

---

## Modal Keybindings

### NORMAL Mode
* 1-5 - Switch views (Dashboard, Month Grid, Week Columns, Day Timeline, Analytics)
* h / j / k / l - Navigate cells, hours, or backlog tasks
* Tab - Toggle focus between Day Timeline and Todo Shelf
* i - Open Task Creation Wizard
* z - Launch selected task in Zen Mode Focus Session
* Enter - Slide out detailed task inspector panel
* x - Complete selected task
* d - Delete selected task
* : - Open Command Palette

### WIZARD Mode
* Tab / Down - Move to next field
* Shift+Tab / Up - Move to previous field
* Enter - Advance or submit
* Esc - Dismiss wizard

### ZEN Mode
* Space - Pause / Resume countdown timer
* + - Inject +5 minutes to active focus session
* b - Skip current focus/break block
* Esc - Terminate focus session early

---

## Command System Palette (:)

* :create <title> - Create fixed scheduled task for today at 9:00 AM.
* :todo <title> - Create unscheduled floating task on Todo Shelf.
* :review - Open Daily Shutdown Review to defer unfinished tasks.
* :sync - Force run the Google Calendar sync daemon.
* :auth - Launch authorization callback server.
* :quit / :q - Exit stream.
