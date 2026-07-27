package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	gosync "sync"

	"stream/internal/db"
	"stream/internal/sync"
	"stream/internal/view"
	"stream/internal/viewmodel"

	tea "github.com/charmbracelet/bubbletea"
)

type SafeProgram struct {
	mu   gosync.RWMutex
	prog *tea.Program
}

func (sp *SafeProgram) Set(p *tea.Program) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.prog = p
}

func (sp *SafeProgram) Send(msg tea.Msg) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	if sp.prog != nil {
		sp.prog.Send(msg)
	}
}

//go:embed version.txt
var version string

func main() {
	// 1. Initialize local JSON DB
	database, err := db.NewJSONDB()
	if err != nil {
		fmt.Printf("Error initializing local database: %v\n", err)
		os.Exit(1)
	}

	// 2. Setup GCal sync engine with log streaming
	safeProg := &SafeProgram{}
	logChan := func(msg string) {
		safeProg.Send(viewmodel.SyncLogMsg{Message: msg})
	}

	authCompleteChan := func() {
		safeProg.Send(viewmodel.AuthCompleteMsg{})
	}

	syncEngine, err := sync.NewSyncEngine(database, logChan, authCompleteChan)
	if err != nil {
		fmt.Printf("Error setting up sync engine: %v\n", err)
		os.Exit(1)
	}

	// 3. Start sync daemon
	syncEngine.StartDaemon()
	defer syncEngine.Stop()

	// 4. Initialize and run TUI
	vm := viewmodel.NewModel(database, syncEngine)
	vm.Version = "v" + strings.TrimSpace(version)
	ui := view.NewView(&vm)
	program := tea.NewProgram(ui, tea.WithAltScreen())
	safeProg.Set(program)

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
