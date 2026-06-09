# stream - VIM-First TUI Task Management System 

stream is a terminal-native calendar and productivity engine. It treats calendar events as executable tasks that move through a defined lifecycle.

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
* workspaces.json - Workspace metadata and configuration.
* ledger.json - Offline transaction ledger logs.
* credentials.json - Google OAuth2 tokens.
* client_secrets.json - Google Calendar client credentials.

---

## Modal Keybindings

### NORMAL Mode
* 1-5 - Switch views (Dashboard, Month Grid, Week Columns, Day Timeline, Analytics)
* j / k - Navigate selected task vertically
* h / l - Navigate overlapping tasks horizontally
* J / K - Scroll timeline hours up / down
* H / L - Switch days backward / forward
* t - Jump back to today
* w - Cycle to next workspace →
* W - Cycle to previous workspace ←
* Tab - Toggle focus between Sidebar, Day Timeline, and Todo Shelf
* i - Open Task Creation Wizard
* z - Start selected task focus session or resume a background session
* Enter - Slide out detailed task inspector panel
* e - Edit the selected task (from Detail Inspector)
* x - Complete selected task
* d - Delete selected task (triggers confirmation dialog)
* : - Open Command Palette

### WIZARD Mode (Task Form)
* Tab / Down - Move to next field
* Shift+Tab / Up - Move to previous field
* Left / Right / Space - Cycle selection on dropdown fields (Priority, Story Points, Anchored)
* Enter - Advance fields or submit
* Esc - Dismiss wizard

### WORKSPACE_WIZARD Mode (Workspace Form)
* Tab / Down - Move to next field
* Shift+Tab / Up - Move to previous field
* Enter - Advance fields or submit
* Esc - Dismiss workspace wizard

### ZEN Mode
* Space - Pause / Resume countdown timer
* + - Inject +5 minutes to active focus session
* b - Skip current focus/break block
* Esc - Exit focus mode screen (timer continues running in background)

---

## Command System Palette (:)

* :create <title> - Create fixed scheduled task for today at 9:00 AM.
* :todo <title> - Create unscheduled floating task on Todo Shelf.
* :ws-create - Create a new workspace.
* :ws-edit - Edit the active workspace's name, icon, and badge.
* :ws-delete [name] - Delete active workspace (or workspace with specified name) and all its tasks.
* :ws-switch <name> - Switch active workspace view.
* :review - Open Daily Shutdown Review to defer unfinished tasks.
* :sync - Force run the Google Calendar sync daemon.
* :auth - Launch authorization callback server.
* :stop - Stop / Abort active Zen focus session.
* :quit / :q - Exit stream.
