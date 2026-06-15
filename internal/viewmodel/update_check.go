package viewmodel

import (
	"context"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type UpdateCheckMsg struct {
	Commits []string
	Err     error
}

func (m *Model) CheckForUpdatesCmd() tea.Cmd {
	return func() tea.Msg {
		// Verify if we have git and are inside a git repo
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		// 1. Fetch origin to update our tracking branches
		fetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin")
		if err := fetchCmd.Run(); err != nil {
			return UpdateCheckMsg{Err: err}
		}

		// 2. Get the list of commit messages we are behind by compared to origin/main
		logCmd := exec.CommandContext(ctx, "git", "log", "HEAD..origin/main", "--format=%s")
		output, err := logCmd.Output()
		if err != nil {
			return UpdateCheckMsg{Err: err}
		}

		lines := strings.Split(string(output), "\n")
		var commits []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				commits = append(commits, trimmed)
			}
		}

		return UpdateCheckMsg{Commits: commits}
	}
}

func (m *Model) PullAndRestartCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		pullCmd := exec.CommandContext(ctx, "git", "pull")
		_ = pullCmd.Run() // Run git pull

		return tea.Quit()
	}
}

func (m *Model) HandleUpdatePromptKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	if !m.UpdatePromptOpen {
		return false, nil
	}
	keyStr := msg.String()
	switch keyStr {
	case "left", "h", "up", "k":
		m.UpdatePromptSelectedIdx = 0 // Update
		return true, nil
	case "right", "l", "down", "j":
		m.UpdatePromptSelectedIdx = 1 // Snooze
		return true, nil
	case "enter":
		if m.UpdatePromptSelectedIdx == 0 {
			m.UpdatePromptOpen = false
			m.StatusMsg = "Pulling updates..."
			return true, m.PullAndRestartCmd()
		} else {
			// Snooze for 1 hour
			settings := m.DB.GetUserSettings()
			settings.UpdateSnoozedUntil = time.Now().Add(1 * time.Hour)
			_ = m.DB.UpdateUserSettings(settings)
			m.UpdatePromptOpen = false
			m.StatusMsg = "Update snoozed for 1 hour."
			return true, nil
		}
	case "esc", "q":
		m.UpdatePromptOpen = false
		m.StatusMsg = "Update dismissed."
		return true, nil
	}
	return true, nil
}
