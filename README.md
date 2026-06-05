# VIM-First TUI Calendar & Task Management Engine (`tuical`)

`tuical` is a terminal-native calendar and focus-tracking productivity engine built for developers, system administrators, and keyboard-driven power users. It features custom Tokyonight/Catppuccin styling, fluid layout grids, a greedy Pomodoro partition engine, and an offline-first bi-directional Google Calendar sync daemon.

---

## 🚀 Key Features

* **VIM Modal Nav:** Zero-mouse dependency with distinct `NORMAL`, `INSERT`, `ZEN`, `WIZARD`, and `COMMAND` states.
* **Tokyo Night Aesthetics:** True 24-bit color depth, rounded Unicode panels, and dynamic progress bar styling.
* **Priority Execution Engine:** Automatic sorting based on `Weight = (Priority * 1000) + Story Points`. High-effort P0 tasks bubble up dynamically.
* **Greedy Pomodoro Slicing:** Automatically segments task durations into custom focus and rest intervals (e.g. 90/20, 50/10, 25/5).
* **Network-Isolation Safeguards:** Offline local transaction ledger that queues delta changes and syncs them automatically when connection resumes.
* **Private Metadata Shielding:** Saves structured details like Story Points and Priorities in remote events using GCal extended properties.

---

## 🛠️ Installation & Building

To build the optimized native binary (approx. 17MB, zero runtime dependencies):

```bash
# Clone/navigate to project
cd tuical

# Build optimized binary
go build -ldflags="-s -w" -o tuical

# Run the TUI
./tuical
```

---

## ⌨️ Modal Keybindings

### NORMAL Mode (Default Navigation)
* `1` - Dashboard View
* `2` - Month Grid View
* `3` - Week Columnar View
* `4` - Day Timeline View (Primary workspace)
* `5` - Analytics View
* `h` / `j` / `k` / `l` - Navigate cells/hours/tasks
* `tab` - Day View: toggle focus between Timeline and Todo Shelf
* `i` - Open Task Creation Wizard
* `z` - Launch selected task in Zen Mode Focus Session
* `Enter` - Slide out the Detail Panel
* `x` - Mark selected task completed
* `d` - Delete selected task
* `:` - Open Command Palette

### WIZARD Mode (Task Form Creation)
* `Tab` / `Down` - Move focus to next input field
* `Shift+Tab` / `Up` - Move focus to previous input field
* `Enter` - Advance field or submit when focused on `[SUBMIT]`
* `Esc` - Cancel and return to NORMAL mode

### ZEN Mode (Pomodoro Timer)
* `Space` - Pause / Resume countdown timer
* `+` - Inject +5 minutes into current session duration
* `b` - Skip current focus/break block
* `Esc` - Terminate focus session early (saves elapsed work time)

---

## 📅 Google Calendar OAuth2 Setup

To sync your TUI with Google Calendar:

1. Go to the **[Google Cloud Console](https://console.cloud.google.com/)** and create a new project.
2. Enable the **Google Calendar API** for your project.
3. Configure the OAuth Consent Screen (Internal or External).
4. Go to **Credentials**, click **Create Credentials** -> **OAuth Client ID**, select **Desktop Application** as Application Type.
5. Download the client secret JSON file.
6. Rename it to `client_secrets.json` and save it to:
   * **Linux:** `~/.config/tuical/client_secrets.json`
7. Start `tuical`, open the command palette (`:`), type `auth`, and press `Enter`.
8. Copy the link shown, paste it in your browser, approve permissions, and you are fully authorized!

---

## 📂 Config Directory Files

All config and local storage files reside in `~/.config/tuical/`:
* `data.json` - Core task database.
* `ledger.json` - Offline transaction ledger logs.
* `credentials.json` - Google OAuth2 access/refresh tokens.
* `client_secrets.json` - Google OAuth2 client credentials (manually created).
