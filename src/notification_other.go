//go:build !linux

package main

import (
	"fyne.io/fyne/v2"
)

func showNotification(title, message string) {
	if fyneApp != nil {
		fyneApp.SendNotification(fyne.NewNotification(title, message))
	}
}
