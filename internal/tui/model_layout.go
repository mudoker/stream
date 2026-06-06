package tui

// Layout holds all pre-computed column dimensions for the current terminal size.
// It is the single source of truth — view functions must NOT recompute these.
type Layout struct {
	SidebarW   int // Arc sidebar width
	TimelineW  int // Day timeline column width (day view)
	TodoW      int // Todo shelf column width (day view)
	WorkspaceW int // Full workspace width (non-day views = TimelineW + TodoW)
	Height     int // Total terminal height
}

// computeLayout calculates all column widths from terminal dimensions.
// Sidebar: 15% of width, clamped [22, 28]
// Todo shelf: 22% of workspace
// Timeline: remaining workspace space
func computeLayout(w, h int) Layout {
	sidebarW := w * 22 / 100
	if sidebarW < 22 {
		sidebarW = 22
	} else if sidebarW > 28 {
		sidebarW = 28
	}

	workspaceW := w - sidebarW - 1 // 1 for the sidebar right border
	if workspaceW < 40 {
		workspaceW = 40
	}

	todoW := workspaceW * 22 / 100
	if todoW < 22 {
		todoW = 22
	} else if todoW > 36 {
		todoW = 36
	}

	timelineW := workspaceW - todoW - 2 // 2 for gutter between columns
	if timelineW < 30 {
		timelineW = 30
	}

	return Layout{
		SidebarW:   sidebarW,
		TimelineW:  timelineW,
		TodoW:      todoW,
		WorkspaceW: workspaceW,
		Height:     h,
	}
}
