package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

var (
	fyneApp      fyne.App
	dashboardWin fyne.Window
	overlayWin   fyne.Window
	sm           *StateMachine
	appIcon      fyne.Resource
	trayIcon     fyne.Resource

	lastTrayMode    Mode = ModeIdle
	lastTrayMinutes int  = -1

	// UI elements that need dynamic updates
	statusLabel       *widget.Label
	timerLabel        *widget.Label
	cycleLabel        *widget.Label
	overlayTimerLabel *widget.Label

	// Settings edits
	editFocusDuration        int
	editShortRestDuration    int
	editLongRestDuration     int
	editCyclesBeforeLongRest int

	// Focus history calendar grid rectangles
	calendarRects      [42]*canvas.Rectangle
	selectedMonth      time.Time
	monthFilterButtons []*widget.Button
)

func RunUI(stateMachine *StateMachine) {
	sm = stateMachine

	// Load local settings edits
	editFocusDuration = sm.Config.FocusDuration
	editShortRestDuration = sm.Config.ShortRestDuration
	editLongRestDuration = sm.Config.LongRestDuration
	editCyclesBeforeLongRest = sm.Config.CyclesBeforeLongRest

	fyneApp = app.NewWithID("com.restreminder.app")
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	selectedMonth = time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.Local)

	if iconRes, err := fyne.LoadResourceFromPath("assets/icon-512.png"); err != nil {
		log.Printf("Warning: failed to load app icon: %v", err)
	} else {
		appIcon = iconRes
		fyneApp.SetIcon(appIcon)
	}
	if tIcon, err := fyne.LoadResourceFromPath("assets/icon-192.png"); err == nil {
		trayIcon = tIcon
	} else if appIcon != nil {
		trayIcon = appIcon
	}
	dashboardWin = fyneApp.NewWindow("Desktop Rest Reminder")
	if appIcon != nil {
		dashboardWin.SetIcon(appIcon)
	}
	dashboardWin.Resize(fyne.NewSize(600, 480))

	// Intercept dashboard close to hide instead of quit
	dashboardWin.SetCloseIntercept(func() {
		dashboardWin.Hide()
	})

	// Construct System Tray Menu & Icon
	if desk, ok := fyneApp.(desktop.App); ok {
		menu := fyne.NewMenu("Rest Reminder",
			fyne.NewMenuItem("Start Focus", func() {
				sm.Start()
			}),
			fyne.NewMenuItem("Start Short Break", func() {
				sm.StartShortRest()
			}),
			fyne.NewMenuItem("Start Long Break", func() {
				sm.StartLongRest()
			}),
			fyne.NewMenuItem("Open Dashboard", func() {
				dashboardWin.Show()
				dashboardWin.RequestFocus()
			}),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayWindow(dashboardWin)
		updateTrayIcon(desk)
	}

	// 1. Dashboard UI Layout
	statusLabel = widget.NewLabel("Status: IDLE")
	timerLabel = widget.NewLabel("Time  : 00:00")
	cycleLabel = widget.NewLabel("Cycle : 1 / 4")

	startBtn := widget.NewButton("START", func() {
		sm.Start()
	})
	stopBtn := widget.NewButton("STOP", func() {
		sm.Stop()
	})

	controls := container.NewHBox(startBtn, stopBtn)

	// 2. Settings Row Panel
	focusRow := makeSettingRow("Focus Duration", &editFocusDuration, 1, 180)
	shortRow := makeSettingRow("Short Rest", &editShortRestDuration, 1, 60)
	longRow := makeSettingRow("Long Rest", &editLongRestDuration, 1, 120)
	cyclesRow := makeSettingRow("Cycles Count", &editCyclesBeforeLongRest, 1, 12)

	saveBtn := widget.NewButton("SAVE SETTINGS", func() {
		sm.Config.FocusDuration = editFocusDuration
		sm.Config.ShortRestDuration = editShortRestDuration
		sm.Config.LongRestDuration = editLongRestDuration
		sm.Config.CyclesBeforeLongRest = editCyclesBeforeLongRest
		if err := SaveConfig(sm.Config); err != nil {
			log.Printf("Error saving settings: %v", err)
		} else {
			log.Printf("Settings saved: %+v", sm.Config)
		}
	})

	settingsPanel := container.NewVBox(
		widget.NewLabelWithStyle("SETTINGS", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		focusRow,
		shortRow,
		longRow,
		cyclesRow,
		saveBtn,
	)

	// 3. Calendar Grid Panel
	calendarContainer := buildCalendarGrid()
	monthFilter := buildMonthFilter()
	calendarAndFilter := container.NewHBox(
		calendarContainer,
		widget.NewSeparator(),
		monthFilter,
	)
	calendarPanel := container.NewVBox(
		widget.NewLabelWithStyle("FOCUS HISTORY", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		calendarAndFilter,
	)

	// Combine all dashboard layout containers
	dashboardContent := container.NewHBox(
		container.NewVBox(
			widget.NewLabelWithStyle("REMINDER CONTROL", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			statusLabel,
			timerLabel,
			cycleLabel,
			controls,
			widget.NewSeparator(),
			settingsPanel,
			widget.NewSeparator(),
		),
		calendarPanel,
	)

	mainContent := container.New(layout.NewCenterLayout(), dashboardContent)

	dashboardWin.SetContent(mainContent)

	// Define State Machine Callback
	sm.OnModeChange = func(oldMode, newMode Mode) {
		// Run all UI transitions on the main Fyne thread
		fyne.Do(func() {
			if desk, ok := fyneApp.(desktop.App); ok {
				updateTrayIcon(desk)
			}

			if newMode == ModeShortRest || newMode == ModeLongRest {
				showRestOverlay(newMode)
			} else {
				dismissRestOverlay()
				if oldMode == ModeShortRest || oldMode == ModeLongRest {
					ShowNotification("Focus Started", "Time to focus again!")
				}
			}

			updateUIElements()
		})
	}

	// Dynamic UI Ticker Loop (1 Hz)
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for range ticker.C {
			sm.Tick()
			// Schedule updates on the main Fyne thread to prevent thread warning logs
			fyne.Do(func() {
				updateUIElements()
			})
		}
	}()

	// Initialize UI Display Elements
	updateUIElements()

	// Run Application Loop
	fyneApp.Run()
}

func makeSettingRow(name string, valPtr *int, min, max int) fyne.CanvasObject {
	label := widget.NewLabel(fmt.Sprintf("%-20s:", name))
	valLabel := widget.NewLabel(fmt.Sprintf("%d", *valPtr))

	decBtn := widget.NewButton("-", func() {
		if *valPtr > min {
			*valPtr--
			valLabel.SetText(fmt.Sprintf("%d", *valPtr))
		}
	})

	incBtn := widget.NewButton("+", func() {
		if *valPtr < max {
			*valPtr++
			valLabel.SetText(fmt.Sprintf("%d", *valPtr))
		}
	})

	return container.NewHBox(label, decBtn, valLabel, incBtn)
}

func buildCalendarGrid() fyne.CanvasObject {
	// Create headers row for day name initials
	headers := container.NewGridWithColumns(7)
	dayNames := []string{"S", "M", "T", "W", "T", "F", "S"}
	for _, name := range dayNames {
		lbl := widget.NewLabelWithStyle(name, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
		headers.Add(lbl)
	}

	// Create calendar cells grid (6 rows of 7 columns = 42 cells)
	grid := container.NewGridWithColumns(7)
	for col := 0; col < 42; col++ {
		rect := canvas.NewRectangle(color.NRGBA{R: 239, G: 242, B: 245, A: 255})
		rect.SetMinSize(fyne.NewSize(22, 22))
		rect.StrokeColor = color.NRGBA{R: 128, G: 119, B: 105, A: 0}
		rect.StrokeWidth = 0.5
		rect.CornerRadius = 1.5
		calendarRects[col] = rect
		grid.Add(rect)
	}

	return container.NewVBox(headers, grid)
}

func updateCalendarGrid() {
	logs, err := LoadHistory()
	if err != nil {
		logs = []FocusLog{}
	}

	historyMap := make(map[int]int)
	targetYear := selectedMonth.Year()
	targetMonth := selectedMonth.Month()

	for _, l := range logs {
		if l.Status == "completed" && len(l.Timestamp) >= 10 {
			t, err := time.Parse("2006-01-02", l.Timestamp[:10])
			if err == nil {
				if t.Year() == targetYear && t.Month() == targetMonth {
					historyMap[t.Day()]++
				}
			}
		}
	}

	// Calculate alignment
	firstDay := time.Date(targetYear, targetMonth, 1, 0, 0, 0, 0, time.Local)
	startWeekday := int(firstDay.Weekday()) // 0 = Sunday, 1 = Monday, ...

	nextMonth := firstDay.AddDate(0, 1, 0)
	lastDay := nextMonth.AddDate(0, 0, -1)
	daysInMonth := lastDay.Day()

	for i := 0; i < 42; i++ {
		rect := calendarRects[i]
		if i < startWeekday || i >= startWeekday+daysInMonth {
			rect.FillColor = color.Transparent
			rect.Refresh()
			continue
		}

		day := i - startWeekday + 1
		count := historyMap[day]

		var fill color.Color
		switch {
		case count == 0:
			fill = color.NRGBA{R: 239, G: 242, B: 245, A: 255}
		case count == 1:
			fill = color.NRGBA{R: 172, G: 238, B: 187, A: 255}
		case count == 2:
			fill = color.NRGBA{R: 74, G: 194, B: 107, A: 255}
		case count == 3:
			fill = color.NRGBA{R: 45, G: 164, B: 78, A: 255}
		case count >= 4:
			fill = color.NRGBA{R: 17, G: 99, B: 41, A: 255}
		}
		rect.FillColor = fill
		rect.Refresh()
	}
}

func buildMonthFilter() fyne.CanvasObject {
	box := container.NewVBox()

	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	firstOfCurrent := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.Local)

	for i := 0; i < 12; i++ {
		monthDate := firstOfCurrent.AddDate(0, -i, 0)
		label := monthDate.Format("Jan 2006")

		mDate := monthDate

		var btn *widget.Button
		btn = widget.NewButton(label, func() {
			selectedMonth = mDate
			updateCalendarGrid()
			for idx, b := range monthFilterButtons {
				targetDate := firstOfCurrent.AddDate(0, -idx, 0)
				if targetDate.Year() == selectedMonth.Year() && targetDate.Month() == selectedMonth.Month() {
					b.Disable()
				} else {
					b.Enable()
				}
			}
		})

		if i == 0 {
			btn.Disable()
		}

		monthFilterButtons = append(monthFilterButtons, btn)
		box.Add(btn)
	}

	scroll := container.NewVScroll(box)
	scroll.SetMinSize(fyne.NewSize(110, 240))
	return scroll
}

func updateUIElements() {
	minutes := sm.RemainingSeconds / 60
	seconds := sm.RemainingSeconds % 60

	// Update Dashboard Labels
	timerLabel.SetText(fmt.Sprintf("Time  : %02d:%02d", minutes, seconds))

	modeText := "Status: IDLE"
	switch sm.Mode {
	case ModeFocus:
		modeText = "Status: FOCUS"
	case ModeShortRest:
		modeText = "Status: SHORT REST"
	case ModeLongRest:
		modeText = "Status: LONG REST"
	}
	statusLabel.SetText(modeText)

	cycleLabel.SetText(fmt.Sprintf("Cycle : %d / %d", sm.CurrentCycle, sm.Config.CyclesBeforeLongRest))

	// Update Dashboard Window Title
	var title string
	if sm.Mode == ModeIdle {
		title = "Desktop Rest Reminder"
	} else {
		title = fmt.Sprintf("[%02d:%02d] Rest Reminder (%s)", minutes, seconds, sm.Mode)
	}
	dashboardWin.SetTitle(title)

	// Update Overlay Timer if active
	if overlayWin != nil && overlayTimerLabel != nil {
		overlayTimerLabel.SetText(fmt.Sprintf("%02d:%02d", minutes, seconds))
	}

	// Update history activity grid colors
	updateCalendarGrid()

	// Update system tray icon dynamically if available
	if desk, ok := fyneApp.(desktop.App); ok {
		updateTrayIcon(desk)
	}
}

func showRestOverlay(mode Mode) {
	dismissRestOverlay()

	overlayWin = fyneApp.NewWindow("Rest Overlay")
	if appIcon != nil {
		overlayWin.SetIcon(appIcon)
	}
	overlayWin.SetFullScreen(true)

	labelText := "REST IN PROGRESS"
	if mode == ModeLongRest {
		labelText = "LONG REST IN PROGRESS"
	}
	overlayTitleLabel := widget.NewLabelWithStyle(labelText, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	overlayTimerLabel = widget.NewLabelWithStyle("00:00", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	stopBtn := widget.NewButton("STOP BREAK", func() {
		sm.Stop()
	})

	overlayContent := container.NewCenter(
		container.NewVBox(
			overlayTitleLabel,
			overlayTimerLabel,
			stopBtn,
		),
	)

	overlayWin.SetContent(overlayContent)
	overlayWin.Show()

	ShowNotification("Rest Started", fmt.Sprintf("Time to rest for %d minutes!", sm.RemainingSeconds/60))
}

func dismissRestOverlay() {
	if overlayWin != nil {
		overlayWin.Close()
		overlayWin = nil
		overlayTimerLabel = nil
	}
}

var font3x5 = [10][5]uint16{
	{0b111, 0b101, 0b101, 0b101, 0b111}, // 0
	{0b010, 0b110, 0b010, 0b010, 0b111}, // 1
	{0b111, 0b001, 0b111, 0b100, 0b111}, // 2
	{0b111, 0b001, 0b111, 0b001, 0b111}, // 3
	{0b101, 0b101, 0b111, 0b001, 0b001}, // 4
	{0b111, 0b100, 0b111, 0b001, 0b111}, // 5
	{0b111, 0b100, 0b111, 0b101, 0b111}, // 6
	{0b111, 0b001, 0b001, 0b001, 0b001}, // 7
	{0b111, 0b101, 0b111, 0b101, 0b111}, // 8
	{0b111, 0b101, 0b111, 0b001, 0b111}, // 9
}

func drawDigit(img *image.RGBA, digit int, offsetX, offsetY int, col color.Color) {
	if digit < 0 || digit > 9 {
		return
	}
	bitmap := font3x5[digit]
	for row := 0; row < 5; row++ {
		line := bitmap[row]
		for colIdx := 0; colIdx < 3; colIdx++ {
			shift := 2 - colIdx
			bit := (line >> shift) & 1
			if bit == 1 {
				img.Set(offsetX+colIdx, offsetY+row, col)
			}
		}
	}
}

func createTrayIcon(mode Mode, minutes int) fyne.Resource {
	if mode == ModeIdle {
		return trayIcon
	}

	// Cap display minutes to 99 to fit on the icon
	displayMin := minutes
	if displayMin > 99 {
		displayMin = 99
	}

	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	
	// Determine background color and text color based on mode
	var bgFill color.Color
	var textCol color.Color = color.White

	switch mode {
	case ModeFocus:
		bgFill = color.RGBA{90, 72, 139, 255} // app icon bg color (purple/indigo)
	case ModeShortRest, ModeLongRest:
		bgFill = color.RGBA{166, 227, 161, 255} // green
		textCol = color.RGBA{30, 30, 46, 255}    // dark text for contrast on green
	default:
		bgFill = color.RGBA{88, 91, 112, 255}  // grey
	}

	// Draw circular background shape (radius ~7.5 px)
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			dx := float64(x) - 7.5
			dy := float64(y) - 7.5
			dist := dx*dx + dy*dy
			if dist <= 56.25 {
				img.Set(x, y, bgFill)
			} else {
				img.Set(x, y, color.Transparent)
			}
		}
	}

	// Draw countdown digits
	if displayMin >= 0 {
		if displayMin < 10 {
			// Center single digit
			drawDigit(img, displayMin, 6, 5, textCol)
		} else {
			// Center two digits
			drawDigit(img, displayMin/10, 4, 5, textCol)
			drawDigit(img, displayMin%10, 8, 5, textCol)
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	// Bypasses Fyne's cache using mode & minutes in the name
	return fyne.NewStaticResource(fmt.Sprintf("tray_icon_%s_%d.png", mode, displayMin), buf.Bytes())
}

func updateTrayIcon(desk desktop.App) {
	minutes := 0
	if sm.Mode != ModeIdle {
		// Use ceiling division so we display the minute in progress (e.g. 24m59s shows "25")
		minutes = (sm.RemainingSeconds + 59) / 60
	}

	if sm.Mode == lastTrayMode && minutes == lastTrayMinutes {
		return
	}

	lastTrayMode = sm.Mode
	lastTrayMinutes = minutes

	icon := createTrayIcon(sm.Mode, minutes)
	if icon != nil {
		desk.SetSystemTrayIcon(icon)
	}
}

func ShowNotification(title, message string) {
	if fyneApp != nil {
		fyneApp.SendNotification(fyne.NewNotification(title, message))
	}
}
