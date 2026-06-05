# ▲ stream — VIM-First TUI Calendar & Task Management Engine

`stream` is a terminal-native, keyboard-driven calendar and productivity engine. It is designed under the philosophy that every calendar event is a task and every task is a schedulable work item. 

Its design is heavily inspired by **Linear's** project management workspace and the **Arc Browser's** vertical sidebar layout, featuring true 24-bit color depth, elevated background layers, padded widget cards, and a docked focus session utility.

---

## ⚡ Quick Start & Installation Guide

### 📋 Prerequisites

Before installing `stream`, ensure you have the following toolchains installed:
* **Go**: Version 1.18 or higher (verify with `go version`).
* **Git**: To clone the repository.

### 📥 1. Clone the Repository
Clone the codebase from the remote repository:
```bash
git clone https://github.com/mudoker/stream.git stream
cd stream
```

### 🔨 2. Build the Optimized Binary
Compile the Go project into a single native executable. We use linker flags (`-s -w`) to strip debug symbols and reduce the binary footprint to just **17MB**:
```bash
go build -ldflags="-s -w" -o stream
```

### 📦 3. Install Globally
Move the compiled binary to your local executables path (e.g., `/usr/local/bin` or `~/.local/bin`) so it can be launched from anywhere:
```bash
# Move to system path
sudo mv stream /usr/local/bin/

# Alternatively, move to local user path
mkdir -p ~/.local/bin
mv stream ~/.local/bin/
```
Ensure your shell's `PATH` variable includes the directory where the binary is moved.

### 🚀 4. Run the Application
Launch the TUI interface in fullscreen raw mode:
```bash
stream
```

---

## 🧪 Running Unit Tests

To verify the functional correctness of the Pomodoro partition engine and layout calculators, run the unit test suites:
```bash
go test ./...
```

---

## 📅 Google Calendar OAuth2 Setup

`stream` comes with a delta-sync daemon that operates fully offline, syncing local changes back to Google Calendar when connection is restored.

1. Go to the **[Google Cloud Console](https://console.cloud.google.com/)** and create a project.
2. Search for and enable the **Google Calendar API**.
3. Configure your **OAuth Consent Screen** (select user type as Internal or External).
4. Go to **Credentials** -> **Create Credentials** -> **OAuth Client ID**.
5. Set Application Type to **Desktop Application**.
6. Download the credentials file, rename it to `client_secrets.json`, and place it in the configuration folder:
   * **Linux/macOS:** `~/.config/stream/client_secrets.json`
7. Start `stream`, open the command palette by typing `:`, write `auth`, and press `Enter`.
8. Copy the local authorization loop URL, open it in your browser, and authorize permissions. The sync engine will automatically initialize and save credentials to `~/.config/stream/credentials.json`.

---

## 📂 Configuration & DB Directory

All local databases and configuration credentials reside in the following folder:
* **Path:** `~/.config/stream/`
  * `data.json` — Main local task and calendar database.
  * `ledger.json` — Offline transaction ledger.
  * `credentials.json` — OAuth2 access and refresh tokens.
  * `client_secrets.json` — OAuth2 client secrets.

---

## ⌨️ Modal Keybindings

### NORMAL Mode (Navigation & Split Control)
* `1` — Dashboard View
* `2` — Month Grid View
* `3` — Week Columns View
* `4` — Day Timeline View
* `5` — Analytics View
* `h` / `j` / `k` / `l` — Navigate calendar cells, hours, or backlog tasks
* `Tab` — Day View: toggle focus between Day Timeline and Todo Shelf
* `i` — Open Task Creation Wizard
* `z` — Launch selected task in Zen Mode Focus Session
* `Enter` — Slide-over detailed task inspector panel
* `x` — Complete selected task
* `d` — Delete selected task
* `:` — Open Command Palette

### WIZARD Mode (Task Form Wizard)
* `Tab` / `Down` — Move focus to next input field
* `Shift+Tab` / `Up` — Move focus to previous input field
* `Enter` — Advance to next field, or submit when on `[SUBMIT]`
* `Esc` — Dismiss form wizard and return to NORMAL mode

### ZEN Mode (Pomodoro countdown)
* `Space` — Pause / Resume countdown timer (automatically logs interruptions)
* `+` — Inject $+5$ minutes into active focus session
* `b` — Force break (skip active block)
* `Esc` — Terminate focus session early (saves elapsed work metrics)

---

## 🛠️ Command System Palette (`:`)

Open the Raycast-style command palette by pressing `:` in `NORMAL` mode.
* `:create <task title>` — Create a fixed scheduled task for today at 9:00 AM.
* `:todo <task title>` — Create an unscheduled floating task on the Todo Shelf.
* `:review` — Launch the Daily Shutdown Review (defer unfinished tasks to tomorrow).
* `:sync` — Force run the Google Calendar delta-sync daemon.
* `:auth` — Launch the local authorization callback server.
* `:quit` / `:q` — Exit the application.
