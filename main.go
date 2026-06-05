package main

import (
	"fmt"
	"os"

	"stream/internal/db"
	"stream/internal/sync"
	"stream/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// 1. Initialize local JSON DB
	database, err := db.NewJSONDB()
	if err != nil {
		fmt.Printf("Error initializing local database: %v\n", err)
		os.Exit(1)
	}

	// 2. Setup GCal sync engine with log streaming
	var program *tea.Program
	logChan := func(msg string) {
		if program != nil {
			program.Send(tui.SyncLogMsg{Message: msg})
		}
	}

	syncEngine, err := sync.NewSyncEngine(database, logChan)
	if err != nil {
		fmt.Printf("Error setting up sync engine: %v\n", err)
		os.Exit(1)
	}

	// 3. Start sync daemon
	syncEngine.StartDaemon()
	defer syncEngine.Stop()

	// 4. Initialize and run TUI
	model := tui.NewModel(database, syncEngine)
	program = tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
