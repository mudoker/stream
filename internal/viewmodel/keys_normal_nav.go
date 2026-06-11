package viewmodel

func (m *Model) handleDashboardOrAnalyticsNav(key string) {
	if m.CurrentView == DashboardView {
		switch key {
		case "h", "left":
			m.DashboardFocusCol = 0
		case "l", "right":
			m.DashboardFocusCol = 1
		case "k", "up":
			m.DashboardFocusRow--
			if m.DashboardFocusRow < 0 {
				m.DashboardFocusRow = 2
			}
		case "j", "down":
			m.DashboardFocusRow = (m.DashboardFocusRow + 1) % 3
		}

		availH := m.Height - 11
		if availH < 10 {
			availH = 10
		}

		var rowHeights []int
		if availH > 45 {
			contentH := availH - 6
			if contentH < 3 {
				contentH = 3
			}
			rowHeights = PartitionHeights(contentH, 3)
			for i := range rowHeights {
				rowHeights[i] += 2
			}
		} else {
			rowHeights = []int{15, 15, 15}
		}

		var yStart, yEnd int
		for i := 0; i < m.DashboardFocusRow; i++ {
			yStart += rowHeights[i]
		}
		yEnd = yStart + rowHeights[m.DashboardFocusRow]

		if yStart < m.ScrollOffset {
			m.ScrollOffset = yStart
		} else if yEnd > m.ScrollOffset+availH {
			m.ScrollOffset = yEnd - availH
		}
	} else if m.CurrentView == AnalyticsView {
		totalLayers := 6
		switch key {
		case "h", "left":
			m.AnalyticsFocusCol = 0
		case "l", "right":
			m.AnalyticsFocusCol = 1
		case "k", "up":
			m.AnalyticsFocusRow--
			if m.AnalyticsFocusRow < 0 {
				m.AnalyticsFocusRow = totalLayers - 1
			}
		case "j", "down":
			m.AnalyticsFocusRow = (m.AnalyticsFocusRow + 1) % totalLayers
		}

		yStart := m.AnalyticsFocusRow * 13
		yEnd := (m.AnalyticsFocusRow + 1) * 13

		gridHeight := m.Height - 11
		if gridHeight < 10 {
			gridHeight = 10
		}

		defaultTotalH := 78
		rowHeights := make([]int, totalLayers)
		if gridHeight > defaultTotalH {
			contentH := gridHeight - 12
			if contentH < 6 {
				contentH = 6
			}
			rowHeights = PartitionHeights(contentH, totalLayers)
			for i := range rowHeights {
				rowHeights[i] += 2
			}
		} else {
			for i := range rowHeights {
				rowHeights[i] = 13
			}
		}

		var yStartAcc int
		for i := 0; i < m.AnalyticsFocusRow; i++ {
			yStartAcc += rowHeights[i]
		}
		yStart = yStartAcc
		yEnd = yStartAcc + rowHeights[m.AnalyticsFocusRow]

		if yStart < m.ScrollOffset {
			m.ScrollOffset = yStart
		} else if yEnd > m.ScrollOffset+gridHeight {
			m.ScrollOffset = yEnd - gridHeight
		}
	}
}

func (m *Model) handleMonthNav(key string) {
	switch key {
	case "h":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
	case "l":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
	case "j":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 7)
	case "k":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -7)
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, -1, 0)
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 1, 0)
	case "enter":
		m.CurrentView = DayView
	}
}

func (m *Model) handleWeekNav(key string) {
	switch key {
	case "h", "left":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
	case "l", "right":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -7)
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 7)
	case "j", "down":
		m.ScrollOffset++
	case "k", "up":
		m.ScrollOffset--
		if m.ScrollOffset < 0 {
			m.ScrollOffset = 0
		}
	}
}

func (m *Model) handleDayNav(key string) {
	switch key {
	case "h":
		if !m.TodoShelfFocus {
			m.NavigateHorizontal(-1)
			m.AutoScrollToSelectedTask()
		}
	case "l":
		if !m.TodoShelfFocus {
			m.NavigateHorizontal(1)
			m.AutoScrollToSelectedTask()
		}
	case "j":
		if m.TodoShelfFocus {
			m.MoveTaskSelection(1)
		} else {
			m.NavigateVertical(1)
			m.AutoScrollToSelectedTask()
		}
	case "k":
		if m.TodoShelfFocus {
			m.MoveTaskSelection(-1)
		} else {
			m.NavigateVertical(-1)
			m.AutoScrollToSelectedTask()
		}
	case "J":
		if !m.TodoShelfFocus {
			m.TimelineHour = (m.TimelineHour + 1) % 24
			m.selectFirstTaskInCurrentHour()
		}
	case "K":
		if !m.TodoShelfFocus {
			m.TimelineHour = (m.TimelineHour - 1 + 24) % 24
			m.selectFirstTaskInCurrentHour()
		}
	case "H":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, -1)
		m.selectDefaultTaskForSelectedDay()
		m.AutoScrollToSelectedTask()
	case "L":
		m.SelectedDay = m.SelectedDay.AddDate(0, 0, 1)
		m.selectDefaultTaskForSelectedDay()
		m.AutoScrollToSelectedTask()
	}
}
