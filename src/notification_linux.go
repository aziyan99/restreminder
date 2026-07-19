//go:build linux

package main

import (
	"fyne.io/fyne/v2"
	"github.com/godbus/dbus/v5"
)

func showNotification(title, message string) {
	conn, err := dbus.SessionBus()
	if err == nil {
		obj := conn.Object("org.freedesktop.Notifications", "/org/freedesktop/Notifications")
		call := obj.Call("org.freedesktop.Notifications.Notify", 0,
			"restreminder", // app_name
			uint32(0),      // replaces_id
			"",             // app_icon
			title,          // summary
			message,        // body
			[]string{},     // actions
			map[string]dbus.Variant{
				"transient": dbus.MakeVariant(true),
			}, // hints
			int32(5000),    // expire_timeout in milliseconds (5 seconds)
		)
		if call.Err == nil {
			return
		}
	}

	// Fallback to Fyne's notification
	if fyneApp != nil {
		fyneApp.SendNotification(fyne.NewNotification(title, message))
	}
}
