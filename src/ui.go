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
	calendarRects [30]*canvas.Rectangle
)

func RunUI(stateMachine *StateMachine) {
	sm = stateMachine

	// Load local settings edits
	editFocusDuration = sm.Config.FocusDuration
	editShortRestDuration = sm.Config.ShortRestDuration
	editLongRestDuration = sm.Config.LongRestDuration
	editCyclesBeforeLongRest = sm.Config.CyclesBeforeLongRest

	fyneApp = app.NewWithID("com.restreminder.app")
	dashboardWin = fyneApp.NewWindow("Desktop Rest Reminder")
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
	calendarPanel := container.NewVBox(
		widget.NewLabelWithStyle("FOCUS HISTORY (LAST 30 DAYS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		calendarContainer,
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
	grid := container.NewGridWithColumns(7)
	for col := 0; col < 30; col++ {
		rect := canvas.NewRectangle(color.NRGBA{R: 239, G: 242, B: 245, A: 255})
		rect.SetMinSize(fyne.NewSize(20, 20))
		rect.StrokeColor = color.NRGBA{R: 128, G: 119, B: 105, A: 0}
		rect.StrokeWidth = 0.5
		rect.CornerRadius = 1.5
		calendarRects[col] = rect
		grid.Add(rect)
	}
	return grid
}

func updateCalendarGrid() {
	logs, err := LoadHistory()
	if err != nil {
		logs = []FocusLog{}
	}

	historyMap := make(map[string]int)
	for _, l := range logs {
		if l.Status == "completed" && len(l.Timestamp) >= 10 {
			dayKey := l.Timestamp[:10]
			historyMap[dayKey]++
		}
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -29) // Last 30 days chronologically

	for col := 0; col < 30; col++ {
		day := startDate.AddDate(0, 0, col)
		dayStr := day.Format("2006-01-02")

		rect := calendarRects[col]
		if day.After(now) {
			rect.FillColor = color.Transparent
			rect.Refresh()
			continue
		}

		count := historyMap[dayStr]
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
}

func showRestOverlay(mode Mode) {
	dismissRestOverlay()

	overlayWin = fyneApp.NewWindow("Rest Overlay")
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

func updateTrayIcon(desk desktop.App) {
	icon := createTrayIcon(sm.Mode)
	desk.SetSystemTrayIcon(icon)
}

func createTrayIcon(mode Mode) fyne.Resource {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	var fill color.Color = color.RGBA{88, 91, 112, 255} // grey
	switch mode {
	case ModeFocus:
		fill = color.RGBA{243, 139, 168, 255} // red
	case ModeShortRest, ModeLongRest:
		fill = color.RGBA{166, 227, 161, 255} // green
	}
	for x := 0; x < 16; x++ {
		for y := 0; y < 16; y++ {
			img.Set(x, y, fill)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return fyne.NewStaticResource("tray_icon.png", buf.Bytes())
}

func ShowNotification(title, message string) {
	if fyneApp != nil {
		fyneApp.SendNotification(fyne.NewNotification(title, message))
	}
}
